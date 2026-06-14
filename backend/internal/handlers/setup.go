package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/freeradius-manager/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

// SetupStatus reports whether the first-run wizard has been completed.
func (h *Handler) SetupStatus(c *gin.Context) {
	var value string
	err := h.db.QueryRow(`SELECT value FROM system_settings WHERE key = 'setup_complete'`).Scan(&value)
	if err == sql.ErrNoRows || value != "true" {
		c.JSON(http.StatusOK, gin.H{"setup_required": true, "version": "1.0.0"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"setup_required": false, "version": "1.0.0"})
}

// SetupComplete processes the wizard form submission.
func (h *Handler) SetupComplete(c *gin.Context) {
	// Idempotency guard — cannot re-run setup once complete
	var existing string
	h.db.QueryRow(`SELECT value FROM system_settings WHERE key = 'setup_complete'`).Scan(&existing)
	if existing == "true" {
		respondError(c, http.StatusForbidden, "setup has already been completed")
		return
	}

	var req struct {
		Organization struct {
			Name     string `json:"name"      binding:"required,min=2,max=100"`
			Timezone string `json:"timezone"`
			LogoText string `json:"logo_text"`
		} `json:"organization" binding:"required"`
		Admin struct {
			Username string `json:"username"  binding:"required,min=3,max=50"`
			Email    string `json:"email"     binding:"required,email"`
			FullName string `json:"full_name" binding:"required"`
			Password string `json:"password"  binding:"required,min=8"`
		} `json:"admin" binding:"required"`
		RADIUS struct {
			DefaultSecret string `json:"default_secret"`
			MaxDevices    int    `json:"max_devices"`
		} `json:"radius"`
		Security struct {
			PasswordMinLength  int  `json:"password_min_length"`
			PasswordExpiryDays int  `json:"password_expiry_days"`
			SessionTimeout     int  `json:"session_timeout"`
			MFARequired        bool `json:"mfa_required"`
			BruteForceAttempts int  `json:"brute_force_attempts"`
		} `json:"security"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Apply safe defaults for optional numeric fields
	if req.Security.PasswordMinLength == 0 {
		req.Security.PasswordMinLength = 12
	}
	if req.Security.PasswordExpiryDays == 0 {
		req.Security.PasswordExpiryDays = 90
	}
	if req.Security.SessionTimeout == 0 {
		req.Security.SessionTimeout = 3600
	}
	if req.Security.BruteForceAttempts == 0 {
		req.Security.BruteForceAttempts = 5
	}
	if req.RADIUS.MaxDevices == 0 {
		req.RADIUS.MaxDevices = 20
	}
	if req.Organization.Timezone == "" {
		req.Organization.Timezone = "UTC"
	}

	// Hash the admin password
	hash, err := auth.HashPassword(req.Admin.Password)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to hash password")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "transaction failed")
		return
	}
	defer tx.Rollback()

	// Remove the seed superadmin and create the real admin account
	if _, err = tx.Exec(`DELETE FROM app_users`); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to reset users")
		return
	}
	if _, err = tx.Exec(`
		INSERT INTO app_users (username, password_hash, email, full_name, role, is_active)
		VALUES ($1, $2, $3, $4, 'super_admin', true)`,
		req.Admin.Username, hash, req.Admin.Email, req.Admin.FullName,
	); err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "username already exists")
			return
		}
		h.log.WithError(err).Error("setup: create admin user failed")
		respondError(c, http.StatusInternalServerError, "failed to create admin user")
		return
	}

	// Upsert all settings
	upsert := func(key, value, desc string) {
		tx.Exec(`
			INSERT INTO system_settings (key, value, description)
			VALUES ($1, $2, $3)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			key, value, desc,
		)
	}

	upsert("setup_complete", "true", "Initial setup wizard completed")
	upsert("org_name", req.Organization.Name, "Organisation display name")
	upsert("org_timezone", req.Organization.Timezone, "Organisation timezone")
	upsert("org_logo_text", req.Organization.LogoText, "Short logo/brand text")
	upsert("password_min_length", strconv.Itoa(req.Security.PasswordMinLength), "Minimum password length")
	upsert("password_expiry_days", strconv.Itoa(req.Security.PasswordExpiryDays), "Password expiry in days")
	upsert("session_timeout", strconv.Itoa(req.Security.SessionTimeout), "Session timeout in seconds")
	upsert("mfa_required", strconv.FormatBool(req.Security.MFARequired), "Require MFA for admin users")
	upsert("brute_force_attempts", strconv.Itoa(req.Security.BruteForceAttempts), "Failed attempts before lockout")
	upsert("max_device_limit", strconv.Itoa(req.RADIUS.MaxDevices), "Max devices per RADIUS user")
	if req.RADIUS.DefaultSecret != "" {
		upsert("radius_default_secret_hint", req.RADIUS.DefaultSecret[:min(3, len(req.RADIUS.DefaultSecret))]+"***", "RADIUS secret hint (partial)")
	}

	if err = tx.Commit(); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to save setup")
		return
	}

	h.log.Infof("Setup completed — admin account '%s' created", req.Admin.Username)
	c.JSON(http.StatusOK, gin.H{
		"message":  "Setup completed successfully",
		"username": req.Admin.Username,
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
