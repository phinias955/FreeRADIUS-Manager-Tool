package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/freeradius-manager/backend/internal/auth"
	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/freeradius-manager/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
)

// Login authenticates an app user and issues JWT tokens.
func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	// Fetch user
	var user models.AppUser
	err := h.db.QueryRow(`
		SELECT id, username, password_hash, email,
		       COALESCE(full_name, ''), role,
		       mfa_enabled, COALESCE(mfa_secret, ''), is_active, failed_attempts, locked_until
		FROM app_users WHERE username = $1`,
		req.Username,
	).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email,
		&user.FullName, &user.Role, &user.MFAEnabled, &user.MFASecret,
		&user.IsActive, &user.FailedAttempts, &user.LockedUntil,
	)

	if err == sql.ErrNoRows {
		respondError(c, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		h.log.WithError(err).Error("database error on login")
		respondError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	// Check if account is active
	if !user.IsActive {
		respondError(c, http.StatusForbidden, "account is disabled")
		return
	}

	// Check lockout
	maxAttempts := getBruteForceAttempts()
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		respondError(c, http.StatusTooManyRequests, "account temporarily locked due to too many failed attempts")
		return
	}

	// Verify password
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		newAttempts := user.FailedAttempts + 1
		var lockUntil interface{} = nil

		if newAttempts >= maxAttempts {
			lockDuration := time.Duration(getBruteForceLockout()) * time.Minute
			lockTime := time.Now().Add(lockDuration)
			lockUntil = lockTime
		}

		h.db.Exec(`
			UPDATE app_users SET failed_attempts = $1, locked_until = $2
			WHERE id = $3`,
			newAttempts, lockUntil, user.ID,
		)

		respondError(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// MFA verification
	if user.MFAEnabled {
		if req.MFACode == "" {
			c.JSON(http.StatusOK, gin.H{"mfa_required": true})
			return
		}
		if !totp.Validate(req.MFACode, user.MFASecret) {
			respondError(c, http.StatusUnauthorized, "invalid MFA code")
			return
		}
	}

	// Reset failed attempts on success
	h.db.Exec(`
		UPDATE app_users SET failed_attempts = 0, locked_until = NULL, last_login = NOW()
		WHERE id = $1`, user.ID,
	)

	// Issue tokens
	accessToken, err := auth.GenerateAccessToken(user.ID, user.Username, user.Role)
	if err != nil {
		h.log.WithError(err).Error("failed to generate access token")
		respondError(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	refreshToken, refreshHash, err := auth.GenerateRefreshToken()
	if err != nil {
		h.log.WithError(err).Error("failed to generate refresh token")
		respondError(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	expiry := time.Now().Add(auth.RefreshExpiry())
	h.db.Exec(`
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		user.ID, refreshHash, expiry,
	)

	user.PasswordHash = ""
	user.MFASecret = ""

	c.JSON(http.StatusOK, models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(15 * 60),
		User:         &user,
	})
}

// RefreshToken issues a new access token using a valid refresh token.
func (h *Handler) RefreshToken(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "refresh_token is required")
		return
	}

	hash := auth.HashRefreshToken(req.RefreshToken)

	var userID int
	var role, username string
	err := h.db.QueryRow(`
		SELECT rt.user_id, au.username, au.role
		FROM refresh_tokens rt
		JOIN app_users au ON au.id = rt.user_id
		WHERE rt.token_hash = $1
		  AND rt.revoked = FALSE
		  AND rt.expires_at > NOW()
		  AND au.is_active = TRUE`,
		hash,
	).Scan(&userID, &username, &role)

	if err == sql.ErrNoRows {
		respondError(c, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}
	if err != nil {
		h.log.WithError(err).Error("db error on refresh")
		respondError(c, http.StatusInternalServerError, "internal server error")
		return
	}

	// Rotate refresh token
	h.db.Exec(`UPDATE refresh_tokens SET revoked = TRUE WHERE token_hash = $1`, hash)

	newAccessToken, err := auth.GenerateAccessToken(userID, username, role)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	newRefreshToken, newHash, err := auth.GenerateRefreshToken()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	expiry := time.Now().Add(auth.RefreshExpiry())
	h.db.Exec(`
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, newHash, expiry,
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    15 * 60,
	})
}

// Logout revokes the refresh token.
func (h *Handler) Logout(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		hash := auth.HashRefreshToken(req.RefreshToken)
		h.db.Exec(`UPDATE refresh_tokens SET revoked = TRUE WHERE token_hash = $1`, hash)
	}
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// ChangePassword changes the current user's password.
func (h *Handler) ChangePassword(c *gin.Context) {
	claims, _ := middleware.GetClaims(c)

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := auth.ValidatePasswordComplexity(req.NewPassword); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var currentHash string
	h.db.QueryRow(`SELECT password_hash FROM app_users WHERE id = $1`, claims.UserID).Scan(&currentHash)

	if !auth.CheckPassword(currentHash, req.CurrentPassword) {
		respondError(c, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to hash password")
		return
	}

	h.db.Exec(`UPDATE app_users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		newHash, claims.UserID)

	// Revoke all refresh tokens (force re-login everywhere)
	h.db.Exec(`UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = $1`, claims.UserID)

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// MFASetup generates and returns a TOTP secret and QR code URI.
func (h *Handler) MFASetup(c *gin.Context) {
	claims, _ := middleware.GetClaims(c)

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "FreeRADIUS Manager",
		AccountName: claims.Username,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to generate MFA secret")
		return
	}

	h.db.Exec(`UPDATE app_users SET mfa_secret = $1 WHERE id = $2`, key.Secret(), claims.UserID)

	c.JSON(http.StatusOK, gin.H{
		"secret":   key.Secret(),
		"otpauth": key.URL(),
		"message": "Scan the QR code or enter the secret in your authenticator app, then verify with /auth/mfa/verify",
	})
}

// MFAVerify enables MFA after user confirms the TOTP code.
func (h *Handler) MFAVerify(c *gin.Context) {
	claims, _ := middleware.GetClaims(c)

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "code is required")
		return
	}

	var secret string
	h.db.QueryRow(`SELECT mfa_secret FROM app_users WHERE id = $1`, claims.UserID).Scan(&secret)

	if secret == "" {
		respondError(c, http.StatusBadRequest, "MFA setup not initiated, call /auth/mfa/setup first")
		return
	}

	if !totp.Validate(req.Code, secret) {
		respondError(c, http.StatusUnauthorized, "invalid MFA code")
		return
	}

	h.db.Exec(`UPDATE app_users SET mfa_enabled = TRUE WHERE id = $1`, claims.UserID)
	c.JSON(http.StatusOK, gin.H{"message": "MFA enabled successfully"})
}

func getBruteForceAttempts() int {
	if raw := os.Getenv("BRUTE_FORCE_ATTEMPTS"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			return v
		}
	}
	return 5
}

func getBruteForceLockout() int {
	if raw := os.Getenv("BRUTE_FORCE_LOCKOUT"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			return v
		}
	}
	return 15
}
