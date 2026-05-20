package handler

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"sso.pelajarnumagetan.or.id/internal/config"
	"sso.pelajarnumagetan.or.id/internal/middleware"
	"sso.pelajarnumagetan.or.id/internal/service"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
// POST /v1/auth/register
func (h *AuthHandler) Register(c echo.Context) error {
	var req service.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}
	if err := c.Validate(&req); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return utils.BadRequest(c, he.Message.(string))
		}
		return utils.BadRequest(c, "Validasi gagal")
	}

	user, err := h.authService.Register(&req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.Created(c, "Akun berhasil dibuat. Silakan periksa email Anda untuk verifikasi.", map[string]interface{}{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	})
}

// VerifyEmail godoc
// POST /v1/auth/verify-email
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	type verifyReq struct {
		Token string `json:"token" validate:"required"`
	}

	var req verifyReq
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}
	if req.Token == "" {
		return utils.BadRequest(c, "Token verifikasi wajib disertakan")
	}

	if err := h.authService.VerifyEmail(req.Token); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Email berhasil diverifikasi! Silakan login.", nil)
}

// Login godoc
// POST /v1/auth/login
func (h *AuthHandler) Login(c echo.Context) error {
	var req service.LoginRequest
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}
	if err := c.Validate(&req); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return utils.BadRequest(c, he.Message.(string))
		}
		return utils.BadRequest(c, "Validasi gagal")
	}

	resp, err := h.authService.Login(&req)
	if err != nil {
		return utils.Unauthorized(c, err.Error())
	}

	return utils.OK(c, "Login berhasil", resp)
}

// Logout godoc
// POST /v1/auth/logout
func (h *AuthHandler) Logout(c echo.Context) error {
	type logoutReq struct {
		RefreshToken string `json:"refresh_token"`
	}

	var req logoutReq
	if err := c.Bind(&req); err != nil || req.RefreshToken == "" {
		return utils.BadRequest(c, "refresh_token diperlukan")
	}

	if err := h.authService.Logout(req.RefreshToken); err != nil {
		return utils.BadRequest(c, "Gagal logout")
	}

	return utils.OK(c, "Logout berhasil", nil)
}

// RefreshToken godoc
// POST /v1/auth/refresh
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	type refreshReq struct {
		RefreshToken string `json:"refresh_token"`
	}

	var req refreshReq
	if err := c.Bind(&req); err != nil || req.RefreshToken == "" {
		return utils.BadRequest(c, "refresh_token diperlukan")
	}

	resp, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		return utils.Unauthorized(c, err.Error())
	}

	return utils.OK(c, "Token diperbarui", resp)
}

// Me godoc
// GET /v1/user/me
func (h *AuthHandler) Me(c echo.Context) error {
	claims, ok := c.Get(middleware.UserContextKey).(*utils.JWTClaims)
	if !ok {
		return utils.Unauthorized(c, "Unauthorized")
	}

	user, err := h.authService.GetProfile(claims.UserID)
	if err != nil {
		return utils.NotFound(c, "User tidak ditemukan")
	}

	return utils.OK(c, "Profil berhasil diambil", map[string]interface{}{
		"id":          user.ID,
		"name":        user.Name,
		"email":       user.Email,
		"phone":       user.Phone,
		"gender":      user.Gender,
		"image":       user.Image,
		"role":        user.Role,
		"is_verified": user.IsVerified,
	})
}

// UpdateProfile godoc
// POST /v1/user/update
func (h *AuthHandler) UpdateProfile(c echo.Context) error {
	claims, ok := c.Get(middleware.UserContextKey).(*utils.JWTClaims)
	if !ok {
		return utils.Unauthorized(c, "Unauthorized")
	}

	type updateReq struct {
		Name   string `json:"name"`
		Image  string `json:"image"`
		Gender string `json:"gender"`
		Phone  string `json:"phone"`
	}

	var req updateReq
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}

	user, err := h.authService.UpdateProfile(claims.UserID, req.Name, req.Image, req.Gender, req.Phone)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Profil berhasil diperbarui", map[string]interface{}{
		"id":     user.ID,
		"name":   user.Name,
		"email":  user.Email,
		"phone":  user.Phone,
		"gender": user.Gender,
		"image":  user.Image,
	})
}

// MySessions godoc
// GET /v1/user/sessions
func (h *AuthHandler) MySessions(c echo.Context) error {
	claims, ok := c.Get(middleware.UserContextKey).(*utils.JWTClaims)
	if !ok {
		return utils.Unauthorized(c, "Unauthorized")
	}

	sessions, err := h.authService.GetMySessions(claims.UserID)
	if err != nil {
		return utils.BadRequest(c, "Gagal mengambil sesi")
	}

	return utils.OK(c, "Sesi berhasil diambil", sessions)
}

// UploadAvatar godoc
// POST /v1/user/upload-avatar
func (h *AuthHandler) UploadAvatar(c echo.Context) error {
	claims, ok := c.Get(middleware.UserContextKey).(*utils.JWTClaims)
	if !ok {
		return utils.Unauthorized(c, "Unauthorized")
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return utils.BadRequest(c, "File tidak ditemukan")
	}

	// Validasi ekstensi
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowed[ext] {
		return utils.BadRequest(c, "Format tidak didukung. Gunakan JPG, PNG, atau WebP")
	}

	// Validasi ukuran (max 2MB)
	if file.Size > 2*1024*1024 {
		return utils.BadRequest(c, "Ukuran file maksimal 2MB")
	}

	src, err := file.Open()
	if err != nil {
		return utils.InternalError(c, "Gagal membuka file")
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return utils.InternalError(c, "Gagal membaca file")
	}

	// Hash nama file (tanpa ekstensi — format tersembunyi)
	h256 := sha256.New()
	h256.Write([]byte(fmt.Sprintf("%s:%d", claims.UserID, time.Now().UnixNano())))
	h256.Write(data)
	hashedName := fmt.Sprintf("%x", h256.Sum(nil)) // no extension

	// Enkripsi file dengan AES-256-GCM
	cfg := config.Get()
	encrypted, err := utils.EncryptAvatar(data, ext, cfg.EncryptionKey)
	if err != nil {
		return utils.InternalError(c, "Gagal mengenkripsi file")
	}

	uploadDir := "uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return utils.InternalError(c, "Gagal membuat direktori")
	}

	destPath := filepath.Join(uploadDir, hashedName)
	if err := os.WriteFile(destPath, encrypted, 0644); err != nil {
		return utils.InternalError(c, "Gagal menyimpan file")
	}

	// Hapus foto lama
	currentUser, _ := h.authService.GetProfile(claims.UserID)
	if currentUser != nil && currentUser.Image != "" {
		// extract hash dari URL lama: /v1/avatar/<hash>
		oldHash := filepath.Base(currentUser.Image)
		oldPath := filepath.Join(uploadDir, oldHash)
		if oldPath != destPath {
			_ = os.Remove(oldPath)
		}
	}

	// Update DB — URL ke endpoint decrypt
	name, gender := "", ""
	if currentUser != nil {
		name = currentUser.Name
		gender = currentUser.Gender
	}
	publicURL := fmt.Sprintf("/v1/avatar/%s", hashedName)
	user, err := h.authService.UpdateProfile(claims.UserID, name, publicURL, gender, "")
	if err != nil {
		return utils.InternalError(c, "Gagal update profil")
	}

	return utils.OK(c, "Avatar berhasil diperbarui", map[string]interface{}{
		"image": user.Image,
	})
}

// ServeAvatar godoc
// GET /v1/avatar/:hash — decrypt dan serve avatar
func (h *AuthHandler) ServeAvatar(c echo.Context) error {
	hash := c.Param("hash")
	if hash == "" {
		return utils.NotFound(c, "Avatar tidak ditemukan")
	}

	filePath := filepath.Join("uploads/avatars", filepath.Base(hash))
	encrypted, err := os.ReadFile(filePath)
	if err != nil {
		return utils.NotFound(c, "Avatar tidak ditemukan")
	}

	cfg := config.Get()
	imageData, ext, err := utils.DecryptAvatar(encrypted, cfg.EncryptionKey)
	if err != nil {
		return utils.InternalError(c, "Gagal mendekripsi avatar")
	}

	// Map ekstensi ke Content-Type
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".webp": "image/webp",
	}
	ct := contentTypes[ext]
	if ct == "" {
		ct = "application/octet-stream"
	}

	// Cache 1 hari
	c.Response().Header().Set("Cache-Control", "public, max-age=86400")
	return c.Blob(200, ct, imageData)
}

// ChangePassword godoc
// POST /v1/user/change-password
func (h *AuthHandler) ChangePassword(c echo.Context) error {
	claims, ok := c.Get(middleware.UserContextKey).(*utils.JWTClaims)
	if !ok {
		return utils.Unauthorized(c, "Unauthorized")
	}

	type changePwdReq struct {
		OldPassword string `json:"old_password" validate:"required"`
		NewPassword string `json:"new_password" validate:"required,min=8"`
	}

	var req changePwdReq
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}
	if err := c.Validate(&req); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return utils.BadRequest(c, he.Message.(string))
		}
		return utils.BadRequest(c, "Validasi gagal")
	}

	if err := h.authService.ChangePassword(claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Password berhasil diubah", nil)
}

// ForgotPassword godoc
// POST /v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	type forgotReq struct {
		Email string `json:"email" validate:"required,email"`
	}

	var req forgotReq
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}
	if err := c.Validate(&req); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return utils.BadRequest(c, he.Message.(string))
		}
		return utils.BadRequest(c, "Validasi gagal")
	}

	if err := h.authService.ForgotPassword(req.Email); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Email pemulihan kata sandi telah dikirim. Silakan periksa kotak masuk Anda.", nil)
}

// ResetPassword godoc
// POST /v1/auth/reset-password
func (h *AuthHandler) ResetPassword(c echo.Context) error {
	type resetReq struct {
		Token           string `json:"token" validate:"required"`
		Password        string `json:"password" validate:"required,min=8"`
		ConfirmPassword string `json:"confirm_password" validate:"required"`
	}

	var req resetReq
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}
	if err := c.Validate(&req); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return utils.BadRequest(c, he.Message.(string))
		}
		return utils.BadRequest(c, "Validasi gagal")
	}

	if err := h.authService.ResetPassword(req.Token, req.Password, req.ConfirmPassword); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Password Anda berhasil diperbarui! Silakan login kembali.", nil)
}


