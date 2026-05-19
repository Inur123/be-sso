package service

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"

	"sso.pelajarnumagetan.or.id/internal/config"
	"sso.pelajarnumagetan.or.id/internal/domain"
	"sso.pelajarnumagetan.or.id/internal/repository"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

// AuthorizeRequest — params dari GET /oauth/authorize
type AuthorizeRequest struct {
	ResponseType string `query:"response_type" json:"response_type"`
	ClientID     string `query:"client_id" json:"client_id"`
	RedirectURI  string `query:"redirect_uri" json:"redirect_uri"`
	Scope        string `query:"scope" json:"scope"`
	State        string `query:"state" json:"state"`
}

// AuthorizeInfo — info yang dikembalikan ke FE untuk tampilkan consent page
type AuthorizeInfo struct {
	App         *AppResponse `json:"app"`
	Scope       string       `json:"scope"`
	State       string       `json:"state"`
	RedirectURI string       `json:"redirect_uri"`
}

// TokenRequest — body POST /oauth/token
type TokenRequest struct {
	ClientID     string `json:"client_id" form:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret"`
	Code         string `json:"code" form:"code"`
	RedirectURI  string `json:"redirect_uri" form:"redirect_uri"`
}

// TokenResponse — response /oauth/token
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// RefreshRequest — body POST /oauth/refreshAccessToken
type RefreshRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type"`
	RefreshToken string `json:"refresh_token" form:"refresh_token"`
	ClientID     string `json:"client_id" form:"client_id"`
}

type OAuthService interface {
	// Step 1: validasi params, kembalikan info app untuk consent page
	Authorize(req *AuthorizeRequest) (*AuthorizeInfo, error)
	// Step 2: user setujui consent → generate auth code
	Confirm(userID uuid.UUID, req *AuthorizeRequest) (string, error)
	// Step 3: App tukar code → token
	ExchangeToken(req *TokenRequest) (*TokenResponse, error)
	// Refresh
	RefreshAccessToken(req *RefreshRequest) (*TokenResponse, error)
	// Revoke
	RevokeToken(token string) error
}

type oauthService struct {
	appRepo      repository.AppRepository
	authCodeRepo repository.AuthCodeRepository
	sessionRepo  repository.SessionRepository
	userRepo     repository.UserRepository
	cfg          *config.Config
}

func NewOAuthService(
	appRepo repository.AppRepository,
	authCodeRepo repository.AuthCodeRepository,
	sessionRepo repository.SessionRepository,
	userRepo repository.UserRepository,
) OAuthService {
	return &oauthService{
		appRepo:      appRepo,
		authCodeRepo: authCodeRepo,
		sessionRepo:  sessionRepo,
		userRepo:     userRepo,
		cfg:          config.Get(),
	}
}

// Authorize — validasi client_id & redirect_uri, return info app untuk consent
func (s *oauthService) Authorize(req *AuthorizeRequest) (*AuthorizeInfo, error) {
	if req.ResponseType != "code" {
		return nil, errors.New("response_type harus 'code'")
	}
	if req.ClientID == "" || req.RedirectURI == "" {
		return nil, errors.New("client_id dan redirect_uri wajib diisi")
	}

	app, err := s.appRepo.FindByClientID(req.ClientID)
	if err != nil {
		return nil, errors.New("client_id tidak ditemukan")
	}

	// Cek app sudah verified
	if app.Status != domain.StatusVerified {
		return nil, errors.New("aplikasi belum diverifikasi")
	}

	// Validasi redirect_uri
	if !slices.Contains(app.RedirectURIs, req.RedirectURI) {
		return nil, errors.New("redirect_uri tidak valid")
	}

	return &AuthorizeInfo{
		App:         &AppResponse{ID: app.ID, Name: app.Name, LogoURL: app.LogoURL, ClientID: app.ClientID},
		Scope:       req.Scope,
		State:       req.State,
		RedirectURI: req.RedirectURI,
	}, nil
}

// Confirm — user sudah klik "Izinkan", generate auth code
func (s *oauthService) Confirm(userID uuid.UUID, req *AuthorizeRequest) (string, error) {
	app, err := s.appRepo.FindByClientID(req.ClientID)
	if err != nil {
		return "", errors.New("client_id tidak ditemukan")
	}

	if app.Status != domain.StatusVerified {
		return "", errors.New("aplikasi belum diverifikasi")
	}

	if !slices.Contains(app.RedirectURIs, req.RedirectURI) {
		return "", errors.New("redirect_uri tidak valid")
	}

	// Generate random code
	code := utils.GenerateRefreshToken()

	// Simpan di Redis — expire 5 menit
	codeData := &repository.AuthCodeData{
		UserID:      userID.String(),
		ClientID:    req.ClientID,
		Scope:       req.Scope,
		RedirectURI: req.RedirectURI,
		State:       req.State,
	}

	if err := s.authCodeRepo.Save(code, codeData, 5*time.Minute); err != nil {
		return "", errors.New("gagal menyimpan authorization code")
	}

	return code, nil
}

// ExchangeToken — tukar code → access_token + refresh_token
func (s *oauthService) ExchangeToken(req *TokenRequest) (*TokenResponse, error) {
	// Ambil code dari Redis
	codeData, err := s.authCodeRepo.Find(req.Code)
	if err != nil {
		return nil, errors.New("authorization code tidak valid atau sudah expired")
	}

	// Validasi client_id
	if codeData.ClientID != req.ClientID {
		return nil, errors.New("client_id tidak cocok")
	}

	// Validasi redirect_uri
	if codeData.RedirectURI != req.RedirectURI {
		return nil, errors.New("redirect_uri tidak cocok")
	}

	// Validasi client_secret
	app, err := s.appRepo.FindByClientID(req.ClientID)
	if err != nil {
		return nil, errors.New("aplikasi tidak ditemukan")
	}

	if !utils.CheckPassword(req.ClientSecret, app.ClientSecret) {
		return nil, errors.New("client_secret tidak valid")
	}

	// Hapus code dari Redis (one-time use)
	_ = s.authCodeRepo.Delete(req.Code)

	// Ambil user
	userID, err := uuid.Parse(codeData.UserID)
	if err != nil {
		return nil, errors.New("user tidak valid")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user tidak ditemukan")
	}

	// Generate access token
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email, user.Name, codeData.Scope, app.ClientID)
	if err != nil {
		return nil, errors.New("gagal generate token")
	}

	// Generate refresh token
	refreshToken := utils.GenerateRefreshToken()

	// Simpan session — AppID pointer ke app.ID
	appID := app.ID
	session := &domain.UserSession{
		UserID:       user.ID,
		AppID:        &appID,
		RefreshToken: refreshToken,
		Scope:        codeData.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(s.cfg.JWTRefreshExpire) * time.Second),
	}
	if err := s.sessionRepo.Create(session); err != nil {
		return nil, errors.New("gagal menyimpan sesi")
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    s.cfg.JWTAccessExpire,
		RefreshToken: refreshToken,
	}, nil
}

// RefreshAccessToken — tukar refresh_token → access_token baru
func (s *oauthService) RefreshAccessToken(req *RefreshRequest) (*TokenResponse, error) {
	if req.GrantType != "refresh_token" {
		return nil, errors.New("grant_type harus 'refresh_token'")
	}

	session, err := s.sessionRepo.FindByRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("refresh_token tidak valid")
	}

	if session.IsExpired() {
		_ = s.sessionRepo.DeleteByRefreshToken(req.RefreshToken)
		return nil, errors.New("refresh_token sudah expired")
	}

	user := session.User

	// Validasi client_id — App bisa nil jika session dashboard
	if session.App == nil {
		return nil, errors.New("session ini bukan milik OAuth app")
	}
	app := session.App

	// Validasi client_id
	if app.ClientID != req.ClientID {
		return nil, errors.New("client_id tidak cocok")
	}

	// Generate access token baru
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email, user.Name, session.Scope, app.ClientID)
	if err != nil {
		return nil, errors.New("gagal generate token")
	}

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   s.cfg.JWTAccessExpire,
	}, nil
}

// RevokeToken — hapus session berdasarkan refresh_token
func (s *oauthService) RevokeToken(token string) error {
	if err := s.sessionRepo.DeleteByRefreshToken(token); err != nil {
		return errors.New("token tidak ditemukan")
	}
	return nil
}
