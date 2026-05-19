package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"

	"sso.pelajarnumagetan.or.id/internal/config"
	"sso.pelajarnumagetan.or.id/internal/database"
	"sso.pelajarnumagetan.or.id/internal/domain"
	"sso.pelajarnumagetan.or.id/internal/handler"
	"sso.pelajarnumagetan.or.id/internal/middleware"
	"sso.pelajarnumagetan.or.id/internal/repository"
	"sso.pelajarnumagetan.or.id/internal/service"
)

func main() {
	// Load config
	cfg := config.Load()
	if len(cfg.EncryptionKey) >= 8 {
		log.Printf("🔑 ACTIVE ENCRYPTION KEY: %s...%s (len: %d)", cfg.EncryptionKey[:4], cfg.EncryptionKey[len(cfg.EncryptionKey)-4:], len(cfg.EncryptionKey))
	} else {
		log.Printf("⚠️  ENCRYPTION KEY INVALID OR TOO SHORT: %s", cfg.EncryptionKey)
	}

	// Connect database
	db := database.ConnectPostgres()
	database.ConnectRedis()

	// Auto migrate
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Application{},
		&domain.UserSession{},
	); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("✅ Migration completed")

	// --- Dependency Injection ---
	// Repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	appRepo := repository.NewAppRepository(db)
	authCodeRepo := repository.NewAuthCodeRepository(database.Redis)

	// Services
	authService := service.NewAuthService(userRepo, sessionRepo)
	userService := service.NewUserService(userRepo)
	appService := service.NewAppService(appRepo)
	oauthService := service.NewOAuthService(appRepo, authCodeRepo, sessionRepo, userRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	appHandler := handler.NewAppHandler(appService)
	oauthHandler := handler.NewOAuthHandler(oauthService)

	// Seed superadmin pertama
	if err := service.SeedSuperAdmin(userRepo); err != nil {
		log.Printf("⚠️  Seed superadmin gagal: %v", err)
	} else {
		log.Println("✅ Superadmin ready")
	}

	// --- Echo Setup ---
	e := echo.New()
	e.HideBanner = true
	e.Validator = middleware.NewValidator() // ← validasi input otomatis

	// Global middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status":  "ok",
			"service": cfg.AppName,
		})
	})

	// OAuth routes (sejajar nu.id — public, dipanggil oleh App A/B)
	oauth := e.Group("/oauth")
	oauth.GET("/authorize", oauthHandler.Authorize)
	oauth.POST("/authorize/confirm", oauthHandler.Confirm, middleware.Auth())
	oauth.POST("/token", oauthHandler.Token)
	oauth.POST("/refreshAccessToken", oauthHandler.RefreshAccessToken)
	oauth.POST("/revoke", oauthHandler.Revoke)

	// v1 API routes
	v1 := e.Group("/v1")

	// Auth routes (public)
	auth := v1.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/verify-email", authHandler.VerifyEmail)
	auth.POST("/login", authHandler.Login)
	auth.POST("/logout", authHandler.Logout)
	auth.POST("/refresh", authHandler.RefreshToken)

	// User routes (protected)
	user := v1.Group("/user", middleware.Auth())
	user.GET("/me", authHandler.Me)
	user.POST("/update", authHandler.UpdateProfile)
	user.GET("/sessions", authHandler.MySessions)
	user.POST("/upload-avatar", authHandler.UploadAvatar)

	// Avatar — public endpoint (decrypt on the fly)
	v1.GET("/avatar/:hash", authHandler.ServeAvatar)

	// App routes — developer (protected)
	apps := v1.Group("/apps", middleware.Auth())
	apps.POST("", appHandler.Create)
	apps.GET("", appHandler.GetMyApps)
	apps.GET("/:id", appHandler.GetByID)
	apps.PUT("/:id", appHandler.Update)
	apps.DELETE("/:id", appHandler.Delete)
	apps.PUT("/:id/toggle-active", appHandler.ToggleActive)
	apps.POST("/:id/regenerate", appHandler.RegenerateSecret)
	apps.GET("/:id/info", appHandler.GetPublicInfo)

	// Admin routes — superadmin only (protected)
	admin := v1.Group("/admin", middleware.Auth(), middleware.LoadUserRole(db), middleware.RequireRole(domain.RoleSuperAdmin))
	admin.GET("/apps", appHandler.AdminGetAll)
	admin.GET("/apps/pending", appHandler.AdminGetPending)
	admin.GET("/apps/:id", appHandler.AdminGetByID)
	admin.PUT("/apps/:id", appHandler.AdminUpdate)
	admin.PUT("/apps/:id/toggle-active", appHandler.AdminToggleActive)
	admin.POST("/apps/:id/approve", appHandler.AdminApprove)
	admin.POST("/apps/:id/reject", appHandler.AdminReject)
	admin.GET("/users", userHandler.AdminGetUsers)
	admin.GET("/users/:id", userHandler.AdminGetByID)
	admin.PUT("/users/:id/role", userHandler.AdminUpdateRole)
	admin.PUT("/users/:id/deactivate", userHandler.AdminDeactivate)
	admin.PUT("/users/:id/activate", userHandler.AdminActivate)
	admin.PUT("/users/:id/verify-email", userHandler.AdminVerifyEmail)
	admin.DELETE("/users/:id", userHandler.AdminDelete)

	// Static files — serve uploaded avatars
	e.Static("/uploads", "uploads")

	// Start server
	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("🚀 %s running on %s", cfg.AppName, addr)
	if err := e.Start(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
