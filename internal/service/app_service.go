package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"

	"sso.pelajarnumagetan.or.id/internal/domain"
	"sso.pelajarnumagetan.or.id/internal/repository"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

type CreateAppRequest struct {
	Name         string   `json:"name" validate:"required,min=3"`
	Description  string   `json:"description"`
	RedirectURIs []string `json:"redirect_uris" validate:"required,min=1"`
	LogoURL      string   `json:"logo_url"`
}

type UpdateAppRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	RedirectURIs []string `json:"redirect_uris"`
	LogoURL      string   `json:"logo_url"`
	IsRestricted *bool    `json:"is_restricted"`
}

type AppResponse struct {
	ID           uuid.UUID        `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	ClientID     string           `json:"client_id"`
	RedirectURIs []string         `json:"redirect_uris"`
	LogoURL      string           `json:"logo_url"`
	Status       domain.AppStatus `json:"status"`
	IsActive     bool             `json:"is_active"`
	IsRestricted bool             `json:"is_restricted"`
	OwnerID      uuid.UUID        `json:"owner_id"`
	CreatedAt    time.Time        `json:"created_at"`
}

type AppCreatedResponse struct {
	AppResponse
	ClientSecret string `json:"client_secret"` // Hanya muncul saat create / regenerate
}

type UserAccessResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Image     string    `json:"image"`
	HasAccess bool      `json:"has_access"`
}

type AppService interface {
	Create(ownerID uuid.UUID, req *CreateAppRequest) (*AppCreatedResponse, error)
	GetByID(id uuid.UUID, ownerID uuid.UUID) (*AppResponse, error)
	GetByOwner(ownerID uuid.UUID) ([]AppResponse, error)
	GetPublicInfo(appID uuid.UUID) (*AppResponse, error)
	Update(id uuid.UUID, ownerID uuid.UUID, req *UpdateAppRequest) (*AppResponse, error)
	RegenerateSecret(id uuid.UUID, ownerID uuid.UUID) (string, error)
	Delete(id uuid.UUID, ownerID uuid.UUID) error
	ToggleActive(id uuid.UUID, ownerID uuid.UUID) (*AppResponse, error)
	GetAppAccessList(appID, requesterID uuid.UUID) ([]UserAccessResponse, error)
	UpdateAppAccessList(appID, requesterID uuid.UUID, userIDs []uuid.UUID) error
	SearchUserForAccess(appID, requesterID uuid.UUID, searchQuery string) (*UserAccessResponse, error)

	// Admin
	GetAll(page, perPage int, status string) ([]AppResponse, int64, error)
	GetPending() ([]AppResponse, error)
	Approve(id uuid.UUID) error
	Reject(id uuid.UUID) error
	AdminGetByID(id uuid.UUID) (*AppResponse, error)
	AdminUpdate(id uuid.UUID, req *UpdateAppRequest) (*AppResponse, error)
	AdminToggleActive(id uuid.UUID) (*AppResponse, error)
}

type appService struct {
	appRepo       repository.AppRepository
	appAccessRepo repository.AppAccessRepository
	userRepo      repository.UserRepository
}

func NewAppService(
	appRepo repository.AppRepository,
	appAccessRepo repository.AppAccessRepository,
	userRepo repository.UserRepository,
) AppService {
	return &appService{
		appRepo:       appRepo,
		appAccessRepo: appAccessRepo,
		userRepo:      userRepo,
	}
}

func (s *appService) Create(ownerID uuid.UUID, req *CreateAppRequest) (*AppCreatedResponse, error) {
	clientID := generateClientID()
	rawSecret := generateSecret()

	hashedSecret, err := utils.HashPassword(rawSecret)
	if err != nil {
		return nil, errors.New("gagal generate secret")
	}

	app := &domain.Application{
		Name:         req.Name,
		Description:  req.Description,
		ClientID:     clientID,
		ClientSecret: hashedSecret,
		RedirectURIs: req.RedirectURIs,
		LogoURL:      req.LogoURL,
		Status:       domain.StatusPending,
		IsActive:     true,
		OwnerID:      ownerID,
	}

	if err := s.appRepo.Create(app); err != nil {
		return nil, errors.New("gagal membuat aplikasi")
	}

	return &AppCreatedResponse{
		AppResponse:  toAppResponse(app),
		ClientSecret: rawSecret, // Hanya dikembalikan sekali!
	}, nil
}

func (s *appService) GetByID(id uuid.UUID, ownerID uuid.UUID) (*AppResponse, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("aplikasi tidak ditemukan")
	}
	// Pastikan hanya owner yang bisa lihat
	if app.OwnerID != ownerID {
		return nil, errors.New("akses ditolak")
	}
	resp := toAppResponse(app)
	return &resp, nil
}

func (s *appService) GetByOwner(ownerID uuid.UUID) ([]AppResponse, error) {
	apps, err := s.appRepo.FindByOwner(ownerID)
	if err != nil {
		return nil, errors.New("gagal mengambil daftar aplikasi")
	}
	return toAppResponses(apps), nil
}

func (s *appService) GetPublicInfo(appID uuid.UUID) (*AppResponse, error) {
	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, errors.New("aplikasi tidak ditemukan")
	}
	resp := toAppResponse(app)
	return &resp, nil
}

func (s *appService) Update(id uuid.UUID, ownerID uuid.UUID, req *UpdateAppRequest) (*AppResponse, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("aplikasi tidak ditemukan")
	}
	if app.OwnerID != ownerID {
		return nil, errors.New("akses ditolak")
	}

	if req.Name != "" {
		app.Name = req.Name
	}
	if req.Description != "" {
		app.Description = req.Description
	}
	if len(req.RedirectURIs) > 0 {
		app.RedirectURIs = req.RedirectURIs
	}
	if req.LogoURL != "" {
		app.LogoURL = req.LogoURL
	}
	if req.IsRestricted != nil {
		app.IsRestricted = *req.IsRestricted
	}

	if err := s.appRepo.Update(app); err != nil {
		return nil, errors.New("gagal update aplikasi")
	}

	resp := toAppResponse(app)
	return &resp, nil
}

func (s *appService) GetAppAccessList(appID, requesterID uuid.UUID) ([]UserAccessResponse, error) {
	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, errors.New("aplikasi tidak ditemukan")
	}

	user, err := s.userRepo.FindByID(requesterID)
	if err != nil {
		return nil, errors.New("user tidak ditemukan")
	}

	if user.Role != domain.RoleSuperAdmin && app.OwnerID != requesterID {
		return nil, errors.New("akses ditolak")
	}

	assignedUsers, err := s.appAccessRepo.GetAssignedUsers(appID)
	if err != nil {
		return nil, err
	}

	var usersToReturn []domain.User
	if user.Role == domain.RoleSuperAdmin {
		allActive, err := s.userRepo.FindAllActive()
		if err != nil {
			return nil, err
		}
		for _, u := range allActive {
			if u.Role != domain.RoleSuperAdmin {
				usersToReturn = append(usersToReturn, u)
			}
		}
	} else {
		for _, u := range assignedUsers {
			if u.Role != domain.RoleSuperAdmin {
				usersToReturn = append(usersToReturn, u)
			}
		}
	}

	assignedMap := make(map[uuid.UUID]bool)
	for _, u := range assignedUsers {
		assignedMap[u.ID] = true
	}

	var resp []UserAccessResponse
	for _, u := range usersToReturn {
		resp = append(resp, UserAccessResponse{
			UserID:    u.ID,
			Name:      u.Name,
			Email:     u.Email,
			Image:     u.Image,
			HasAccess: assignedMap[u.ID],
		})
	}

	return resp, nil
}

func (s *appService) SearchUserForAccess(appID, requesterID uuid.UUID, searchQuery string) (*UserAccessResponse, error) {
	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return nil, errors.New("aplikasi tidak ditemukan")
	}

	user, err := s.userRepo.FindByID(requesterID)
	if err != nil {
		return nil, errors.New("user tidak ditemukan")
	}

	if user.Role != domain.RoleSuperAdmin && app.OwnerID != requesterID {
		return nil, errors.New("akses ditolak")
	}

	var foundUser *domain.User
	// Cari berdasarkan UUID (User ID) saja
	if parsedUUID, err := uuid.Parse(searchQuery); err == nil {
		foundUser, _ = s.userRepo.FindByID(parsedUUID)
	}

	if foundUser == nil {
		return nil, errors.New("user tidak ditemukan dengan ID tersebut")
	}

	// Jangan munculkan super admin
	if foundUser.Role == domain.RoleSuperAdmin {
		return nil, errors.New("user tidak ditemukan dengan ID tersebut")
	}

	// Pastikan user aktif dan terverifikasi
	if !foundUser.IsActive || !foundUser.IsVerified {
		return nil, errors.New("user tidak aktif atau belum memverifikasi email")
	}

	assignedUsers, err := s.appAccessRepo.GetAssignedUsers(appID)
	if err != nil {
		return nil, err
	}

	hasAccess := false
	for _, u := range assignedUsers {
		if u.ID == foundUser.ID {
			hasAccess = true
			break
		}
	}

	return &UserAccessResponse{
		UserID:    foundUser.ID,
		Name:      foundUser.Name,
		Email:     foundUser.Email,
		Image:     foundUser.Image,
		HasAccess: hasAccess,
	}, nil
}

func (s *appService) UpdateAppAccessList(appID, requesterID uuid.UUID, userIDs []uuid.UUID) error {
	app, err := s.appRepo.FindByID(appID)
	if err != nil {
		return errors.New("aplikasi tidak ditemukan")
	}

	user, err := s.userRepo.FindByID(requesterID)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}

	if user.Role != domain.RoleSuperAdmin && app.OwnerID != requesterID {
		return errors.New("akses ditolak")
	}

	return s.appAccessRepo.AssignUsers(appID, userIDs)
}

func (s *appService) RegenerateSecret(id uuid.UUID, ownerID uuid.UUID) (string, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return "", errors.New("aplikasi tidak ditemukan")
	}
	if app.OwnerID != ownerID {
		return "", errors.New("akses ditolak")
	}

	rawSecret := generateSecret()
	hashedSecret, err := utils.HashPassword(rawSecret)
	if err != nil {
		return "", errors.New("gagal generate secret")
	}

	app.ClientSecret = hashedSecret
	if err := s.appRepo.Update(app); err != nil {
		return "", errors.New("gagal update secret")
	}

	return rawSecret, nil
}

func (s *appService) Delete(id uuid.UUID, ownerID uuid.UUID) error {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return errors.New("aplikasi tidak ditemukan")
	}
	if app.OwnerID != ownerID {
		return errors.New("akses ditolak")
	}
	return s.appRepo.Delete(id)
}

func (s *appService) ToggleActive(id uuid.UUID, ownerID uuid.UUID) (*AppResponse, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("aplikasi tidak ditemukan")
	}
	if app.OwnerID != ownerID {
		return nil, errors.New("akses ditolak")
	}
	app.IsActive = !app.IsActive
	if err := s.appRepo.Update(app); err != nil {
		return nil, errors.New("gagal update status")
	}
	resp := toAppResponse(app)
	return &resp, nil
}

// Admin methods
func (s *appService) GetAll(page, perPage int, status string) ([]AppResponse, int64, error) {
	apps, total, err := s.appRepo.FindAll(page, perPage, status)
	if err != nil {
		return nil, 0, err
	}
	return toAppResponses(apps), total, nil
}

func (s *appService) GetPending() ([]AppResponse, error) {
	apps, err := s.appRepo.FindPending()
	if err != nil {
		return nil, err
	}
	return toAppResponses(apps), nil
}

func (s *appService) Approve(id uuid.UUID) error {
	if _, err := s.appRepo.FindByID(id); err != nil {
		return errors.New("aplikasi tidak ditemukan")
	}
	return s.appRepo.UpdateStatus(id, domain.StatusVerified)
}

func (s *appService) Reject(id uuid.UUID) error {
	if _, err := s.appRepo.FindByID(id); err != nil {
		return errors.New("aplikasi tidak ditemukan")
	}
	return s.appRepo.UpdateStatus(id, domain.StatusRejected)
}

func (s *appService) AdminGetByID(id uuid.UUID) (*AppResponse, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("aplikasi tidak ditemukan")
	}
	resp := toAppResponse(app)
	return &resp, nil
}

func (s *appService) AdminUpdate(id uuid.UUID, req *UpdateAppRequest) (*AppResponse, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("aplikasi tidak ditemukan")
	}

	if req.Name != "" {
		app.Name = req.Name
	}
	if req.Description != "" {
		app.Description = req.Description
	}
	if len(req.RedirectURIs) > 0 {
		app.RedirectURIs = req.RedirectURIs
	}
	if req.LogoURL != "" {
		app.LogoURL = req.LogoURL
	}
	if req.IsRestricted != nil {
		app.IsRestricted = *req.IsRestricted
	}

	if err := s.appRepo.Update(app); err != nil {
		return nil, errors.New("gagal update aplikasi")
	}

	resp := toAppResponse(app)
	return &resp, nil
}

func (s *appService) AdminToggleActive(id uuid.UUID) (*AppResponse, error) {
	app, err := s.appRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("aplikasi tidak ditemukan")
	}
	app.IsActive = !app.IsActive
	if err := s.appRepo.Update(app); err != nil {
		return nil, errors.New("gagal update status")
	}
	resp := toAppResponse(app)
	return &resp, nil
}

// Helper
func generateClientID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func toAppResponse(app *domain.Application) AppResponse {
	return AppResponse{
		ID:           app.ID,
		Name:         app.Name,
		Description:  app.Description,
		ClientID:     app.ClientID,
		RedirectURIs: app.RedirectURIs,
		LogoURL:      app.LogoURL,
		Status:       app.Status,
		IsActive:     app.IsActive,
		IsRestricted: app.IsRestricted,
		OwnerID:      app.OwnerID,
		CreatedAt:    app.CreatedAt,
	}
}

func toAppResponses(apps []domain.Application) []AppResponse {
	result := make([]AppResponse, len(apps))
	for i, app := range apps {
		result[i] = toAppResponse(&app)
	}
	return result
}
