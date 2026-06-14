package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// APIKey represents an external integration key.
type APIKey struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	Permissions []string   `json:"permissions"`
	CreatedBy   *string    `json:"created_by"`
	LastUsed    *time.Time `json:"last_used"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ListAPIKeys returns all API keys (never returns the full key).
func (h *Handler) ListAPIKeys(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT ak.id, ak.name, ak.key_prefix, ak.permissions, au.username,
		       ak.last_used, ak.expires_at, ak.is_active, ak.created_at
		FROM api_keys ak
		LEFT JOIN app_users au ON au.id = ak.created_by
		ORDER BY ak.created_at DESC`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch API keys")
		return
	}
	defer rows.Close()

	keys := []APIKey{}
	for rows.Next() {
		var k APIKey
		rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.Permissions,
			&k.CreatedBy, &k.LastUsed, &k.ExpiresAt, &k.IsActive, &k.CreatedAt)
		keys = append(keys, k)
	}
	c.JSON(http.StatusOK, gin.H{"data": keys})
}

// CreateAPIKey generates a new API key. Returns the full key ONCE.
func (h *Handler) CreateAPIKey(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required,min=2,max=100"`
		Permissions []string `json:"permissions"`
		ExpiresAt   *string  `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Permissions) == 0 {
		req.Permissions = []string{"read"}
	}

	claims, _ := middleware.GetClaims(c)

	// Generate random 32-byte key
	raw := make([]byte, 32)
	rand.Read(raw)
	fullKey := "rmk_" + hex.EncodeToString(raw) // prefix + 64 hex chars

	hash := sha256.Sum256([]byte(fullKey))
	keyHash := hex.EncodeToString(hash[:])
	keyPrefix := fullKey[:12] // "rmk_" + first 8 hex chars

	// Build permissions array literal for PostgreSQL
	permsLiteral := "{" + joinStrings(req.Permissions, ",") + "}"

	var expiresAt interface{}
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		expiresAt = *req.ExpiresAt
	}

	var id int
	err := h.db.QueryRow(`
		INSERT INTO api_keys (name, key_hash, key_prefix, permissions, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		req.Name, keyHash, keyPrefix, permsLiteral, claims.UserID, expiresAt,
	).Scan(&id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create API key")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          id,
		"name":        req.Name,
		"key":         fullKey, // shown ONCE, never again
		"key_prefix":  keyPrefix,
		"permissions": req.Permissions,
		"message":     "API key created — copy it now, it will not be shown again",
	})
}

// RevokeAPIKey marks a key as inactive.
func (h *Handler) RevokeAPIKey(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`UPDATE api_keys SET is_active=false WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

// DeleteAPIKey permanently removes a key.
func (h *Handler) DeleteAPIKey(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`DELETE FROM api_keys WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "API key deleted"})
}

// ValidateAPIKey middleware authenticates requests using an API key.
// Checks Authorization: ApiKey <key> header.
func (h *Handler) ValidateAPIKey(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) < 8 || authHeader[:7] != "ApiKey " {
		c.Next()
		return
	}
	rawKey := authHeader[7:]
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	var keyID int
	var perms []string
	var expiresAt *time.Time
	err := h.db.QueryRow(`
		SELECT id, permissions, expires_at FROM api_keys
		WHERE key_hash=$1 AND is_active=true`, keyHash).
		Scan(&keyID, &perms, &expiresAt)

	if err == sql.ErrNoRows {
		respondError(c, http.StatusUnauthorized, "invalid API key")
		c.Abort()
		return
	}
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		respondError(c, http.StatusUnauthorized, "API key expired")
		c.Abort()
		return
	}

	// Update last_used asynchronously
	go h.db.Exec(`UPDATE api_keys SET last_used=NOW() WHERE id=$1`, keyID)

	c.Set("api_key_id", keyID)
	c.Set("api_key_perms", perms)
	c.Next()
}

// ── Internal helpers ─────────────────────────────────────────────────────────

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

// APIKeyStats returns usage statistics.
func (h *Handler) APIKeyStats(c *gin.Context) {
	var total, active, expired int
	h.db.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active),
		COUNT(*) FILTER (WHERE expires_at < NOW()) FROM api_keys`).
		Scan(&total, &active, &expired)
	c.JSON(http.StatusOK, gin.H{
		"total": total, "active": active, "expired": expired,
		"docs_url": fmt.Sprintf("%s/api/v1", c.Request.Host),
	})
}
