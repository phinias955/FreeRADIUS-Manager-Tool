package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
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

// setupState caches whether setup has been completed so we only hit the DB once.
var (
	setupDone   bool
	setupDoneMu sync.RWMutex
)

// markSetupDone caches the completed state in memory (irreversible).
func markSetupDone() {
	setupDoneMu.Lock()
	setupDone = true
	setupDoneMu.Unlock()
}

// isSetupDone checks in-memory cache first, then DB.
func isSetupDone(db *database.DB) bool {
	setupDoneMu.RLock()
	cached := setupDone
	setupDoneMu.RUnlock()
	if cached {
		return true
	}
	var val string
	db.QueryRow(`SELECT value FROM system_settings WHERE key = 'setup_complete'`).Scan(&val)
	if val == "true" {
		markSetupDone()
		return true
	}
	return false
}

// setupIPRateLimit is a very simple per-IP request counter for the setup endpoint.
var (
	setupIPHits   = map[string]int{}
	setupIPMu     sync.Mutex
	maxSetupHits  = 10
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

	// Health check (no auth required) — available at both paths so nginx proxy
	// can forward /api/v1/health and the Docker health-check can hit /health.
	healthHandler := handlers.HealthCheck(db)
	router.GET("/health", healthHandler)
	router.GET("/api/v1/health", healthHandler)
	router.GET("/api/v1/version", handlers.Version())

	// Initialize handlers
	h := handlers.New(db, log)

	// ── Start Tier 2-7 background workers ───────────────────────────────────
	handlers.StartNASMonitor(db, log)
	handlers.StartAlertWorker(db, log)
	handlers.StartScheduler(db, log)
	handlers.StartHoneypot(db, log)
	handlers.StartCredStuffingDetector(db, log)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// First-run setup wizard endpoints — permanently locked once setup_complete = true
		setup := v1.Group("/setup")
		setup.Use(func(c *gin.Context) {
			// Rate-limit per IP
			ip := c.ClientIP()
			setupIPMu.Lock()
			setupIPHits[ip]++
			hits := setupIPHits[ip]
			setupIPMu.Unlock()
			if hits > maxSetupHits {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
				return
			}

			// Once setup is done the endpoint vanishes (404 — no information leak)
			if isSetupDone(db) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.Next()
		})
		{
			setup.GET("/status", h.SetupStatus)
			setup.POST("/complete", func(c *gin.Context) {
				h.SetupComplete(c)
				// If the handler succeeded, warm the in-memory cache immediately
				if c.Writer.Status() == http.StatusOK {
					markSetupDone()
				}
			})
		}

		// Auth endpoints (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", h.Login)
			auth.POST("/refresh", h.RefreshToken)
			auth.POST("/logout", middleware.RequireAuth(db), h.Logout)
			auth.GET("/profile", middleware.RequireAuth(db), h.GetProfile)
			auth.PUT("/profile", middleware.RequireAuth(db), h.UpdateProfile)
			auth.POST("/change-password", middleware.RequireAuth(db), h.ChangePassword)
			auth.POST("/mfa/setup", middleware.RequireAuth(db), h.MFASetup)
			auth.POST("/mfa/verify", middleware.RequireAuth(db), h.MFAVerify)
		}

		// ── Tier 4 Pro: User Self-Service Portal (public, no JWT) ────────────
		portal := v1.Group("/portal")
		{
			portal.POST("/login", h.PortalLogin)
			portal.POST("/logout", h.PortalLogout)
			portal.GET("/dashboard", h.PortalAuthMiddleware(), h.PortalDashboard)
		}

		// ── Tier 5 Pro: Captive Portal serve (public HTML, no JWT) ───────────
		v1.GET("/captive/serve/:id", h.ServeCaptivePortal)

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.RequireAuth(db))
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

		// Network scanner (nmap)
		network := protected.Group("network")
		{
			network.GET("/scanner/status", middleware.RequireRole("admin", "super_admin"), h.NetworkScannerStatus)
			network.POST("/scan", middleware.RequireRole("admin", "super_admin"), h.StartNetworkScan)
			network.GET("/scans", middleware.RequireRole("admin", "super_admin"), h.ListNetworkScans)
			network.GET("/scans/:id", middleware.RequireRole("admin", "super_admin"), h.GetNetworkScan)
			network.DELETE("/scans/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteNetworkScan)
			network.POST("/scans/hosts/:hostId/import-nas", middleware.RequireRole("admin", "super_admin"), h.ImportScanHostAsNAS)
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

		// ── Tier 1 Pro: Vouchers ─────────────────────────────────────────────
		vouchers := protected.Group("vouchers")
		{
			vouchers.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListVouchers)
			vouchers.POST("/generate", middleware.RequireRole("admin", "super_admin"), h.GenerateVouchers)
			vouchers.GET("/batches", middleware.RequireRole("operator", "admin", "super_admin"), h.GetVoucherBatches)
			vouchers.POST("/:id/disable", middleware.RequireRole("admin", "super_admin"), h.DisableVoucher)
			vouchers.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteVoucher)
			vouchers.GET("/export", middleware.RequireRole("admin", "super_admin"), h.ExportVouchers)
		}

		// ── Tier 1 Pro: Bandwidth Profiles ───────────────────────────────────
		bw := protected.Group("bandwidth-profiles")
		{
			bw.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListBandwidthProfiles)
			bw.POST("", middleware.RequireRole("admin", "super_admin"), h.CreateBandwidthProfile)
			bw.PUT("/:id", middleware.RequireRole("admin", "super_admin"), h.UpdateBandwidthProfile)
			bw.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteBandwidthProfile)
		}
		// Apply bandwidth profile to a RADIUS user
		protected.POST("radius/users/:id/bandwidth", middleware.RequireRole("admin", "super_admin"), h.ApplyBandwidthProfile)

		// ── Tier 1 Pro: Reports ───────────────────────────────────────────────
		reports := protected.Group("reports")
		reports.Use(middleware.RequireRole("operator", "admin", "super_admin"))
		{
			reports.GET("/usage", h.UsageReport)
			reports.GET("/usage/daily", h.DailyUsageReport)
			reports.GET("/auth", h.AuthSuccessReport)
			reports.GET("/nas", h.NASUsageReport)
			reports.GET("/usage/export", h.ExportUsageReport)
		}

		// ── Tier 2 Pro: User Plans ────────────────────────────────────────────
		plans := protected.Group("plans")
		{
			plans.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListPlans)
			plans.POST("", middleware.RequireRole("admin", "super_admin"), h.CreatePlan)
			plans.PUT("/:id", middleware.RequireRole("admin", "super_admin"), h.UpdatePlan)
			plans.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeletePlan)
		}
		protected.POST("radius/users/:id/plan", middleware.RequireRole("admin", "super_admin"), h.AssignPlan)

		// ── Tier 2 Pro: Billing / Invoices ────────────────────────────────────
		billing := protected.Group("invoices")
		{
			billing.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListInvoices)
			billing.POST("", middleware.RequireRole("admin", "super_admin"), h.CreateInvoice)
			billing.PUT("/:id", middleware.RequireRole("admin", "super_admin"), h.UpdateInvoice)
			billing.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteInvoice)
		}

		// ── Tier 2 Pro: NAS Monitor ───────────────────────────────────────────
		protected.GET("nas/status", middleware.RequireRole("operator", "admin", "super_admin"), h.GetNASStatus)
		protected.POST("nas/:id/ping", middleware.RequireRole("admin", "super_admin"), h.PingNASNow)

		// ── Tier 2 Pro: Alert Rules ───────────────────────────────────────────
		alerts := protected.Group("alerts")
		{
			alerts.GET("", middleware.RequireRole("admin", "super_admin"), h.ListAlertRules)
			alerts.POST("", middleware.RequireRole("super_admin"), h.CreateAlertRule)
			alerts.PUT("/:id", middleware.RequireRole("super_admin"), h.UpdateAlertRule)
			alerts.DELETE("/:id", middleware.RequireRole("super_admin"), h.DeleteAlertRule)
			alerts.POST("/test-email", middleware.RequireRole("super_admin"), h.SendTestEmail)
		}

		// ── Tier 3 Pro: IP Pools ──────────────────────────────────────────────
		pools := protected.Group("ip-pools")
		{
			pools.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListIPPools)
			pools.POST("", middleware.RequireRole("admin", "super_admin"), h.CreateIPPool)
			pools.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteIPPool)
			pools.GET("/:id/ips", middleware.RequireRole("operator", "admin", "super_admin"), h.ListPoolIPs)
		}
		protected.POST("ip-pools/assign", middleware.RequireRole("admin", "super_admin"), h.AssignIP)
		protected.POST("ip-pools/release", middleware.RequireRole("admin", "super_admin"), h.ReleaseIP)

		// ── Tier 3 Pro: API Keys ──────────────────────────────────────────────
		apikeys := protected.Group("api-keys")
		{
			apikeys.GET("", middleware.RequireRole("super_admin"), h.ListAPIKeys)
			apikeys.POST("", middleware.RequireRole("super_admin"), h.CreateAPIKey)
			apikeys.POST("/:id/revoke", middleware.RequireRole("super_admin"), h.RevokeAPIKey)
			apikeys.DELETE("/:id", middleware.RequireRole("super_admin"), h.DeleteAPIKey)
			apikeys.GET("/stats", middleware.RequireRole("admin", "super_admin"), h.APIKeyStats)
		}

		// ── Tier 3 Pro: Scheduler ─────────────────────────────────────────────
		sched := protected.Group("scheduler")
		{
			sched.GET("", middleware.RequireRole("admin", "super_admin"), h.ListScheduledTasks)
			sched.POST("/:id/toggle", middleware.RequireRole("super_admin"), h.ToggleTask)
			sched.POST("/:id/run", middleware.RequireRole("super_admin"), h.RunTaskNow)
			sched.PUT("/:id/schedule", middleware.RequireRole("super_admin"), h.UpdateTaskSchedule)
		}

		// ── Tier 3 Pro: Bulk Import / Export (replaces existing handlers) ────
		// NOTE: routes already registered above as radius/users/import & export

		// ── Tier 4 Pro: Hotspot Zones ─────────────────────────────────────────
		zones := protected.Group("zones")
		{
			zones.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListZones)
			zones.POST("", middleware.RequireRole("admin", "super_admin"), h.CreateZone)
			zones.PUT("/:id", middleware.RequireRole("admin", "super_admin"), h.UpdateZone)
			zones.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteZone)
			zones.GET("/:id/stats", middleware.RequireRole("operator", "admin", "super_admin"), h.ZoneStats)
		}
		protected.POST("zones/assign-nas", middleware.RequireRole("admin", "super_admin"), h.AssignNASToZone)

		// ── Tier 4 Pro: Live Stats SSE ────────────────────────────────────────
		protected.GET("live/stats", middleware.RequireRole("operator", "admin", "super_admin"), h.LiveStats)
		protected.GET("live/stats/current", middleware.RequireRole("operator", "admin", "super_admin"), h.GetCurrentStats)

		// ── Tier 4 Pro: SMS ───────────────────────────────────────────────────
		smsGroup := protected.Group("sms")
		{
			smsGroup.POST("/send", middleware.RequireRole("admin", "super_admin"), h.SendSMS)
			smsGroup.GET("/logs", middleware.RequireRole("admin", "super_admin"), h.ListSMSLogs)
			smsGroup.POST("/notify-expiry", middleware.RequireRole("admin", "super_admin"), h.NotifyUserExpiry)
			smsGroup.GET("/config", middleware.RequireRole("super_admin"), h.SMSConfig)
		}

		// ── Tier 5 Pro: Organizations / Resellers ─────────────────────────────
		orgs := protected.Group("organizations")
		{
			orgs.GET("", middleware.RequireRole("admin", "super_admin"), h.ListOrganizations)
			orgs.POST("", middleware.RequireRole("super_admin"), h.CreateOrganization)
			orgs.PUT("/:id", middleware.RequireRole("super_admin"), h.UpdateOrganization)
			orgs.DELETE("/:id", middleware.RequireRole("super_admin"), h.DeleteOrganization)
			orgs.GET("/:id/stats", middleware.RequireRole("admin", "super_admin"), h.OrgStats)
		}
		protected.POST("organizations/assign-user", middleware.RequireRole("admin", "super_admin"), h.AssignUserToOrg)

		// ── Tier 5 Pro: Customers CRM ─────────────────────────────────────────
		customers := protected.Group("customers")
		{
			customers.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListCustomers)
			customers.GET("/:id", middleware.RequireRole("operator", "admin", "super_admin"), h.GetCustomer)
			customers.POST("", middleware.RequireRole("admin", "super_admin"), h.CreateCustomer)
			customers.PUT("/:id", middleware.RequireRole("admin", "super_admin"), h.UpdateCustomer)
			customers.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteCustomer)
		}

		// ── Tier 5 Pro: Support Tickets ───────────────────────────────────────
		tickets := protected.Group("tickets")
		{
			tickets.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListTickets)
			tickets.POST("", middleware.RequireRole("operator", "admin", "super_admin"), h.CreateTicket)
			tickets.PUT("/:id", middleware.RequireRole("operator", "admin", "super_admin"), h.UpdateTicket)
			tickets.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteTicket)
		}

		// ── Tier 5 Pro: Captive Portals (admin) ───────────────────────────────
		captive := protected.Group("captive")
		{
			captive.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListCaptivePortals)
			captive.POST("", middleware.RequireRole("admin", "super_admin"), h.CreateCaptivePortal)
			captive.PUT("/:id", middleware.RequireRole("admin", "super_admin"), h.UpdateCaptivePortal)
			captive.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteCaptivePortal)
		}

		// ── Tier 5 Pro: Webhooks ─────────────────────────────────────────────
		hooks := protected.Group("webhooks")
		{
			hooks.GET("", middleware.RequireRole("admin", "super_admin"), h.ListWebhooks)
			hooks.POST("", middleware.RequireRole("super_admin"), h.CreateWebhook)
			hooks.PUT("/:id", middleware.RequireRole("super_admin"), h.UpdateWebhook)
			hooks.DELETE("/:id", middleware.RequireRole("super_admin"), h.DeleteWebhook)
			hooks.POST("/:id/test", middleware.RequireRole("super_admin"), h.TestWebhook)
			hooks.GET("/:id/logs", middleware.RequireRole("admin", "super_admin"), h.ListWebhookLogs)
		}

		// ── Tier 6 Pro: Payments ─────────────────────────────────────────────
		payments := protected.Group("payments")
		{
			payments.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListPayments)
			payments.POST("", middleware.RequireRole("admin", "super_admin"), h.CreatePayment)
			payments.DELETE("/:id", middleware.RequireRole("super_admin"), h.DeletePayment)
			payments.GET("/:id/receipt", middleware.RequireRole("operator", "admin", "super_admin"), h.GetPaymentReceipt)
			payments.GET("/summary", middleware.RequireRole("admin", "super_admin"), h.PaymentSummary)
		}

		// ── Tier 6 Pro: RADIUS Attribute Templates ────────────────────────────
		tmpl := protected.Group("templates")
		{
			tmpl.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListTemplates)
			tmpl.POST("", middleware.RequireRole("admin", "super_admin"), h.CreateTemplate)
			tmpl.PUT("/:id", middleware.RequireRole("admin", "super_admin"), h.UpdateTemplate)
			tmpl.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteTemplate)
			tmpl.POST("/:id/apply", middleware.RequireRole("admin", "super_admin"), h.ApplyTemplate)
			tmpl.POST("/:id/clone", middleware.RequireRole("admin", "super_admin"), h.CloneTemplate)
		}

		// ── Tier 6 Pro: Promotions ────────────────────────────────────────────
		promos := protected.Group("promotions")
		{
			promos.GET("", middleware.RequireRole("operator", "admin", "super_admin"), h.ListPromotions)
			promos.POST("", middleware.RequireRole("admin", "super_admin"), h.CreatePromotion)
			promos.PUT("/:id", middleware.RequireRole("admin", "super_admin"), h.UpdatePromotion)
			promos.DELETE("/:id", middleware.RequireRole("admin", "super_admin"), h.DeletePromotion)
			promos.POST("/validate", middleware.RequireRole("operator", "admin", "super_admin"), h.ValidatePromoCode)
			promos.POST("/apply", middleware.RequireRole("operator", "admin", "super_admin"), h.ApplyPromoCode)
		}

		// ── Tier 6 Pro: Bulk Operations ───────────────────────────────────────
		bulk := protected.Group("bulk")
		{
			bulk.POST("", middleware.RequireRole("admin", "super_admin"), h.BulkOperation)
			bulk.GET("/history", middleware.RequireRole("admin", "super_admin"), h.ListBulkOpHistory)
		}

		// ── Tier 7: Security Suite ────────────────────────────────────────────
		// RADIUS Simulator
		sim := protected.Group("radius/simulate")
		{
			sim.POST("", middleware.RequireRole("admin", "super_admin"), h.SimulateAuth)
			sim.POST("/batch", middleware.RequireRole("admin", "super_admin"), h.SimulateBatch)
		}

		// GeoIP Enforcement
		geoip := protected.Group("security/geoip")
		{
			geoip.GET("/lookup", middleware.RequireRole("operator", "admin", "super_admin"), h.GeoIPLookup)
			geoip.GET("/rules", middleware.RequireRole("operator", "admin", "super_admin"), h.ListGeoIPRules)
			geoip.POST("/rules", middleware.RequireRole("admin", "super_admin"), h.CreateGeoIPRule)
			geoip.DELETE("/rules/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteGeoIPRule)
		}

		// Honeypot
		hp := protected.Group("security/honeypot")
		{
			hp.GET("/status", middleware.RequireRole("operator", "admin", "super_admin"), h.HoneypotStatus)
			hp.GET("/logs", middleware.RequireRole("admin", "super_admin"), h.ListHoneypotLogs)
			hp.DELETE("/logs", middleware.RequireRole("admin", "super_admin"), h.ClearHoneypotLogs)
		}

		// Credential Stuffing + IP blocking
		sec := protected.Group("security")
		{
			sec.GET("/summary", middleware.RequireRole("operator", "admin", "super_admin"), h.SecuritySummary)
			sec.GET("/alerts", middleware.RequireRole("operator", "admin", "super_admin"), h.ListSecurityAlerts)
			sec.PUT("/alerts/:id/ack", middleware.RequireRole("operator", "admin", "super_admin"), h.AcknowledgeAlert)
			sec.PUT("/alerts/ack-all", middleware.RequireRole("admin", "super_admin"), h.AcknowledgeAllAlerts)
			sec.DELETE("/alerts/:id", middleware.RequireRole("admin", "super_admin"), h.DeleteAlert)
			sec.GET("/blocked-ips", middleware.RequireRole("admin", "super_admin"), h.GetBlockedIPs)
			sec.POST("/blocked-ips", middleware.RequireRole("admin", "super_admin"), h.BlockIP)
			sec.DELETE("/blocked-ips/:id", middleware.RequireRole("admin", "super_admin"), h.UnblockIP)
		}
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
