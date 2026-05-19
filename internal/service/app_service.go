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
	OwnerID      uuid.UUID        `json:"owner_id"`
	CreatedAt    time.Time        `json:"created_at"`
}

type AppCreatedResponse struct {
	AppResponse
	ClientSecret string `json:"client_secret"` // Hanya muncul saat create / regenerate
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
	appRepo repository.AppRepository
}

func NewAppService(appRepo repository.AppRepository) AppService {
	return &appService{appRepo: appRepo}
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

	if err := s.appRepo.Update(app); err != nil {
		return nil, errors.New("gagal update aplikasi")
	}

	resp := toAppResponse(app)
	return &resp, nil
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
