package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/freeradius-manager/backend/internal/database"
	"github.com/freeradius-manager/backend/internal/handlers"
	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/freeradius-manager/backend/pkg/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present (development mode)
	_ = godotenv.Load()

	log := logger.New()

	// Initialize database
	db, err := database.New()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Info("Database connection established")

	// Set Gin mode
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(logger.GinLogger(log))

	// CORS configuration
	corsConfig := cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	// Rate limiter middleware
	rateLimiter := middleware.NewRateLimiter()
	router.Use(rateLimiter.Middleware())

	// Health check (no auth required)
	router.GET("/health", handlers.HealthCheck(db))
	router.GET("/api/v1/version", handlers.Version())

	// Initialize handlers
	h := handlers.New(db, log)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth endpoints (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", h.Login)
			auth.POST("/refresh", h.RefreshToken)
			auth.POST("/logout", middleware.RequireAuth(), h.Logout)
			auth.POST("/change-password", middleware.RequireAuth(), h.ChangePassword)
			auth.POST("/mfa/setup", middleware.RequireAuth(), h.MFASetup)
			auth.POST("/mfa/verify", middleware.RequireAuth(), h.MFAVerify)
		}

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.RequireAuth())
		protected.Use(middleware.AuditLogger(db, log))

		// Dashboard & Statistics
		protected.GET("statistics/dashboard", middleware.RequireRole("operator", "admin", "super_admin"), h.DashboardStats)
		protected.GET("sessions/active", middleware.RequireRole("operator", "admin", "super_admin"), h.ActiveSessions)
		protected.GET("sessions/user/:username", middleware.RequireRole("operator", "admin", "super_admin"), h.UserSessions)
		protected.GET("logs/auth", middleware.RequireRole("operator", "admin", "super_admin"), h.AuthLogs)
		protected.GET("logs/audit", middleware.RequireRole("admin", "super_admin"), h.AuditLogs)

		// Admin user management (super_admin only)
		adminUsers := protected.Group("admin/users")
		adminUsers.Use(middleware.RequireRole("super_admin"))
		{
			adminUsers.GET("", h.ListAdminUsers)
			adminUsers.POST("", h.CreateAdminUser)
			adminUsers.PUT("/:id", h.UpdateAdminUser)
			adminUsers.DELETE("/:id", h.DeleteAdminUser)
		}

		// RADIUS user management
		radiusUsers := protected.Group("radius/users")
		{
			radiusUsers.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListRadiusUsers)
			radiusUsers.POST("", middleware.RequireRole("admin", "super_admin"), h.CreateRadiusUser)
			radiusUsers.GET("/:id", middleware.RequireRole("operator", "admin", "super_admin"), h.GetRadiusUser)
			radiusUsers.PUT("/:id", middleware.RequireRole("admin", "super_admin"), h.UpdateRadiusUser)
			radiusUsers.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteRadiusUser)
			radiusUsers.POST("/:id/reset-password", middleware.RequireRole("operator", "admin", "super_admin"), h.ResetRadiusUserPassword)
			radiusUsers.POST("/:id/suspend", middleware.RequireRole("admin", "super_admin"), h.SuspendRadiusUser)
			radiusUsers.POST("/:id/activate", middleware.RequireRole("admin", "super_admin"), h.ActivateRadiusUser)
			radiusUsers.POST("/:id/disconnect", middleware.RequireRole("admin", "super_admin"), h.DisconnectRadiusUser)
			radiusUsers.GET("/:id/sessions", middleware.RequireRole("operator", "admin", "super_admin"), h.RadiusUserSessions)
		}

		// Bulk operations
		protected.POST("radius/users/import", middleware.RequireRole("admin", "super_admin"), h.ImportRadiusUsers)
		protected.GET("radius/users/export", middleware.RequireRole("admin", "super_admin"), h.ExportRadiusUsers)

		// NAS device management
		nas := protected.Group("nas")
		{
			nas.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListNAS)
			nas.POST("", middleware.RequireRole("admin", "super_admin"), h.CreateNAS)
			nas.GET("/:id", middleware.RequireRole("operator", "admin", "super_admin"), h.GetNAS)
			nas.PUT("/:id", middleware.RequireRole("admin", "super_admin"), h.UpdateNAS)
			nas.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteNAS)
			nas.POST("/:id/test", middleware.RequireRole("admin", "super_admin"), h.TestNAS)
			nas.POST("/discover", middleware.RequireRole("admin", "super_admin"), h.DiscoverNAS)
		}

		// RADIUS test endpoint
		protected.POST("radius/test", middleware.RequireRole("admin", "super_admin"), h.TestRADIUS)

		// System settings (super_admin only)
		system := protected.Group("settings")
		{
			system.GET("", middleware.RequireRole("super_admin"), h.GetSettings)
			system.PUT("", middleware.RequireRole("super_admin"), h.UpdateSettings)
		}

		// Backup/Restore (super_admin only)
		protected.POST("backup", middleware.RequireRole("super_admin"), h.CreateBackup)
		protected.POST("restore", middleware.RequireRole("super_admin"), h.RestoreBackup)
		protected.GET("backups", middleware.RequireRole("super_admin"), h.ListBackups)
	}

	port := os.Getenv("WEB_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Infof("Backend server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("Server forced shutdown: %v", err)
	}
	log.Info("Server stopped")
}
