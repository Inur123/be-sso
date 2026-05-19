package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"sso.pelajarnumagetan.or.id/internal/domain"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

const UserContextKey = "current_user_claims"
const UserRoleKey = "current_user_role"

// LoadUserRole — middleware untuk memuat role user dari database ke context
func LoadUserRole(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := c.Get(UserContextKey).(*utils.JWTClaims)
			if !ok || claims == nil {
				return utils.Unauthorized(c, "Unauthorized")
			}

			var role string
			if err := db.Model(&domain.User{}).Select("role").Where("id = ?", claims.UserID).Row().Scan(&role); err != nil {
				return utils.Unauthorized(c, "User tidak ditemukan")
			}

			c.Set(UserRoleKey, domain.UserRole(role))
			return next(c)
		}
	}
}

// Auth — validasi Bearer token, simpan claims ke context
func Auth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return utils.Unauthorized(c, "Token tidak ditemukan")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return utils.Unauthorized(c, "Format token tidak valid")
			}

			claims, err := utils.ParseAccessToken(parts[1])
			if err != nil {
				return utils.Unauthorized(c, "Token tidak valid atau sudah expired")
			}

			c.Set(UserContextKey, claims)
			return next(c)
		}
	}
}

// RequireRole — cek role user.
// Gunakan setelah Auth() middleware.
// Role harus di-set ke context dengan key UserRoleKey oleh handler atau middleware LoadUser.
func RequireRole(roles ...domain.UserRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := c.Get(UserContextKey).(*utils.JWTClaims)
			if !ok || claims == nil {
				return utils.Unauthorized(c, "Unauthorized")
			}

			// Ambil role dari context — di-set oleh LoadUserRole middleware
			roleVal := c.Get(UserRoleKey)
			if roleVal == nil {
				return utils.Forbidden(c, "Role tidak dapat diverifikasi")
			}

			userRole, ok := roleVal.(domain.UserRole)
			if !ok {
				return utils.Forbidden(c, "Role tidak valid")
			}

			for _, role := range roles {
				if userRole == role {
					return next(c)
				}
			}

			return utils.Forbidden(c, "Akses ditolak — role tidak mencukupi")
		}
	}
}
