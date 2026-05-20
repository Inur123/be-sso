package service

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"sso.pelajarnumagetan.or.id/internal/config"
	"sso.pelajarnumagetan.or.id/internal/domain"
	"sso.pelajarnumagetan.or.id/internal/repository"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

type RegisterRequest struct {
	Name            string `json:"name" validate:"required,min=3"`
	Email           string `json:"email" validate:"required,email"`
	Phone           string `json:"phone"`
	Gender          string `json:"gender" validate:"required,oneof=laki-laki perempuan"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int         `json:"expires_in"`
	User         *UserInfo   `json:"user"`
}

type UserInfo struct {
	ID    uuid.UUID       `json:"id"`
	Name  string          `json:"name"`
	Email string          `json:"email"`
	Phone string          `json:"phone"`
	Image string          `json:"image"`
	Role  domain.UserRole `json:"role"`
}

type AuthService interface {
	Register(req *RegisterRequest) (*domain.User, error)
	VerifyEmail(token string) error
	Login(req *LoginRequest) (*AuthResponse, error)
	Logout(refreshToken string) error
	RefreshToken(refreshToken string) (*AuthResponse, error)
	GetProfile(userID uuid.UUID) (*domain.User, error)
	UpdateProfile(userID uuid.UUID, name, image, gender, phone string) (*domain.User, error)
	GetMySessions(userID uuid.UUID) ([]domain.UserSession, error)
	ChangePassword(userID uuid.UUID, oldPassword, newPassword string) error
	ForgotPassword(email string) error
	ResetPassword(token, password, confirmPassword string) error
}

type authService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	cfg         *config.Config
}

func NewAuthService(userRepo repository.UserRepository, sessionRepo repository.SessionRepository) AuthService {
	return &authService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		cfg:         config.Get(),
	}
}

func (s *authService) Register(req *RegisterRequest) (*domain.User, error) {
	// Validasi nomor HP wajib diisi
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		return nil, errors.New("nomor HP wajib diisi")
	}

	// Normalisasi nomor HP ke format +62
	var cleanDigits []rune
	for _, r := range req.Phone {
		if r >= '0' && r <= '9' {
			cleanDigits = append(cleanDigits, r)
		}
	}
	phoneStr := string(cleanDigits)
	if len(phoneStr) < 8 {
		return nil, errors.New("nomor HP tidak valid, minimal 8 digit")
	}

	if strings.HasPrefix(phoneStr, "62") {
		req.Phone = "+" + phoneStr
	} else if strings.HasPrefix(phoneStr, "0") {
		req.Phone = "+62" + phoneStr[1:]
	} else if strings.HasPrefix(phoneStr, "8") {
		req.Phone = "+62" + phoneStr
	} else {
		req.Phone = "+" + phoneStr
	}

	// Cek email sudah dipakai
	existing, _ := s.userRepo.FindByEmailUnscoped(req.Email)
	if existing != nil {
		// Jika record ini berstatus soft-deleted (deleted_at valid), otomatis hapus secara permanen (hard-delete)!
		if existing.DeletedAt.Valid {
			_ = s.userRepo.Delete(existing.ID)
		} else {
			return nil, errors.New("email sudah terdaftar")
		}
	}

	// Cek nomor hp sudah dipakai
	existingPhone, _ := s.userRepo.FindByPhoneUnscoped(req.Phone)
	if existingPhone != nil {
		// Jika record ini berstatus soft-deleted (deleted_at valid), otomatis hapus secara permanen (hard-delete)!
		if existingPhone.DeletedAt.Valid {
			_ = s.userRepo.Delete(existingPhone.ID)
		} else {
			return nil, errors.New("nomor HP sudah terdaftar")
		}
	}

	if req.Password != req.ConfirmPassword {
		return nil, errors.New("password dan konfirmasi password tidak cocok")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("gagal memproses password")
	}

	user := &domain.User{
		Name:       req.Name,
		Email:      req.Email,
		Password:   hashedPassword,
		Phone:      req.Phone,
		Gender:     req.Gender,
		Role:       domain.RoleUser,
		IsActive:   true,
		IsVerified: false, // Default awal adalah belum terverifikasi
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, errors.New("gagal membuat akun")
	}

	// Generate email verification token
	token, err := utils.GenerateEmailVerificationToken(user.Email)
	if err == nil {
		// Kirim email secara asynchronous (background goroutine) agar performa register tetap instan
		go func(name, email, tok string) {
			_ = utils.SendVerificationEmail(name, email, tok)
		}(req.Name, user.Email, token)
	}

	return user, nil
}

func (s *authService) VerifyEmail(token string) error {
	email, err := utils.ParseEmailVerificationToken(token)
	if err != nil {
		return errors.New("token verifikasi tidak valid atau telah kadaluarsa")
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return errors.New("pengguna tidak ditemukan")
	}

	if user.IsVerified {
		return nil // Sudah pernah diverifikasi sebelumnya, anggap sukses langsung
	}

	user.IsVerified = true
	return s.userRepo.Update(user)
}

func (s *authService) Login(req *LoginRequest) (*AuthResponse, error) {
	// Cari user
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("email atau password salah")
	}

	// Cek aktif
	if !user.IsActive {
		return nil, errors.New("akun Anda dinonaktifkan")
	}

	// Cek verifikasi email
	if !user.IsVerified {
		return nil, errors.New("email Anda belum diverifikasi")
	}

	// Verifikasi password
	if !utils.CheckPassword(req.Password, user.Password) {
		return nil, errors.New("email atau password salah")
	}

	// Generate access token (scope "dashboard" untuk login SSO sendiri)
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email, user.Name, "dashboard", "sso-dashboard")
	if err != nil {
		return nil, errors.New("gagal generate token")
	}

	// Generate refresh token
	refreshToken := utils.GenerateRefreshToken()

	// Simpan session — AppID nil untuk login dashboard SSO
	session := &domain.UserSession{
		UserID:       user.ID,
		AppID:        nil, // nil = login ke dashboard SSO sendiri
		RefreshToken: refreshToken,
		Scope:        "dashboard",
		ExpiresAt:    time.Now().Add(time.Duration(s.cfg.JWTRefreshExpire) * time.Second),
	}
	if err := s.sessionRepo.Create(session); err != nil {
		return nil, errors.New("gagal menyimpan sesi")
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    s.cfg.JWTAccessExpire,
		User: &UserInfo{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Phone: user.Phone,
			Image: user.Image,
			Role:  user.Role,
		},
	}, nil
}

func (s *authService) Logout(refreshToken string) error {
	return s.sessionRepo.DeleteByRefreshToken(refreshToken)
}

func (s *authService) RefreshToken(refreshToken string) (*AuthResponse, error) {
	// Cari session
	session, err := s.sessionRepo.FindByRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New("refresh token tidak valid")
	}

	// Cek expired
	if session.IsExpired() {
		_ = s.sessionRepo.DeleteByRefreshToken(refreshToken)
		return nil, errors.New("refresh token sudah expired")
	}

	user := session.User

	// Generate access token baru
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email, user.Name, session.Scope, "sso-dashboard")
	if err != nil {
		return nil, errors.New("gagal generate token")
	}

	return &AuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   s.cfg.JWTAccessExpire,
		User: &UserInfo{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Phone: user.Phone,
			Image: user.Image,
			Role:  user.Role,
		},
	}, nil
}

func (s *authService) GetProfile(userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.FindByID(userID)
}

func (s *authService) UpdateProfile(userID uuid.UUID, name, image, gender, phone string) (*domain.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user tidak ditemukan")
	}

	if name != "" {
		user.Name = name
	}
	if image != "" {
		user.Image = image
	}
	if gender != "" {
		user.Gender = gender
	}
	if phone != "" {
		phone = strings.TrimSpace(phone)
		var cleanDigits []rune
		for _, r := range phone {
			if r >= '0' && r <= '9' {
				cleanDigits = append(cleanDigits, r)
			}
		}
		phoneStr := string(cleanDigits)
		if len(phoneStr) < 8 {
			return nil, errors.New("nomor HP tidak valid, minimal 8 digit")
		}

		var normalizedPhone string
		if strings.HasPrefix(phoneStr, "62") {
			normalizedPhone = "+" + phoneStr
		} else if strings.HasPrefix(phoneStr, "0") {
			normalizedPhone = "+62" + phoneStr[1:]
		} else if strings.HasPrefix(phoneStr, "8") {
			normalizedPhone = "+62" + phoneStr
		} else {
			normalizedPhone = "+" + phoneStr
		}

		// Dekripsi phone lama untuk dibandingkan
		decOldPhone, _ := utils.DecryptField(user.Phone, s.cfg.EncryptionKey)
		if normalizedPhone != decOldPhone {
			// Cek apakah nomor HP baru sudah terdaftar oleh user lain
			existingPhone, _ := s.userRepo.FindByPhoneUnscoped(normalizedPhone)
			if existingPhone != nil {
				if existingPhone.DeletedAt.Valid {
					_ = s.userRepo.Delete(existingPhone.ID)
				} else if existingPhone.ID != userID {
					return nil, errors.New("nomor HP sudah terdaftar oleh pengguna lain")
				}
			}
			// Enkripsi phone baru
			encPhone, err := utils.EncryptField(normalizedPhone, s.cfg.EncryptionKey)
			if err != nil {
				return nil, errors.New("gagal memproses nomor HP baru")
			}
			user.Phone = encPhone
		}
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, errors.New("gagal update profil")
	}

	return user, nil
}

func (s *authService) GetMySessions(userID uuid.UUID) ([]domain.UserSession, error) {
	return s.sessionRepo.FindOAuthByUserID(userID)
}

func (s *authService) ChangePassword(userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("user tidak ditemukan")
	}

	// Verifikasi password lama
	if !utils.CheckPassword(oldPassword, user.Password) {
		return errors.New("password lama salah")
	}

	// Hash password baru
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return errors.New("gagal memproses password baru")
	}

	user.Password = hashedPassword
	return s.userRepo.Update(user)
}

func (s *authService) ForgotPassword(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return errors.New("pengguna dengan email tersebut tidak ditemukan")
	}

	if !user.IsActive {
		return errors.New("akun Anda dinonaktifkan")
	}

	token, err := utils.GeneratePasswordResetToken(user.Email)
	if err != nil {
		return errors.New("gagal memproses permintaan reset password")
	}

	// Kirim email reset password secara asynchronous
	go func(name, emailAddr, tok string) {
		_ = utils.SendResetPasswordEmail(name, emailAddr, tok)
	}(user.Name, user.Email, token)

	return nil
}

func (s *authService) ResetPassword(token, password, confirmPassword string) error {
	if password != confirmPassword {
		return errors.New("password baru dan konfirmasi password tidak cocok")
	}

	if len(password) < 8 {
		return errors.New("password baru minimal 8 karakter")
	}

	email, err := utils.ParsePasswordResetToken(token)
	if err != nil {
		return err
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return errors.New("pengguna tidak ditemukan")
	}

	if !user.IsActive {
		return errors.New("akun Anda dinonaktifkan")
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return errors.New("gagal memproses password baru")
	}

	user.Password = hashedPassword
	return s.userRepo.Update(user)
}

