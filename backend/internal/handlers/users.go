package handlers

import (
	"database/sql"
	"net/http"

	"github.com/freeradius-manager/backend/internal/auth"
	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/freeradius-manager/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// ListAdminUsers returns all admin/operator users.
func (h *Handler) ListAdminUsers(c *gin.Context) {
	offset, limit := paginationParams(c)

	rows, err := h.db.Query(`
		SELECT id, username, email, COALESCE(full_name, ''), role,
		       mfa_enabled, is_active, last_login, created_at, updated_at
		FROM app_users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		h.log.WithError(err).Error("failed to list admin users")
		respondError(c, http.StatusInternalServerError, "failed to fetch users")
		return
	}
	defer rows.Close()

	users := []models.AppUser{}
	for rows.Next() {
		var u models.AppUser
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.FullName, &u.Role,
			&u.MFAEnabled, &u.IsActive, &u.LastLogin, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			continue
		}
		users = append(users, u)
	}

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM app_users`).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"data":  users,
		"total": total,
	})
}

// CreateAdminUser creates a new admin or operator user.
func (h *Handler) CreateAdminUser(c *gin.Context) {
	var req models.CreateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Super admin can create any role; validate caller's role
	actorClaims, _ := middleware.GetClaims(c)
	if !auth.CanManageRole(actorClaims.Role, req.Role) {
		respondError(c, http.StatusForbidden, "you cannot create a user with equal or higher privileges")
		return
	}

	if err := auth.ValidatePasswordComplexity(req.Password); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to process password")
		return
	}

	var newID int
	err = h.db.QueryRow(`
		INSERT INTO app_users (username, password_hash, email, full_name, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		req.Username, hash, req.Email, req.FullName, req.Role,
	).Scan(&newID)

	if err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "username already exists")
			return
		}
		h.log.WithError(err).Error("failed to create admin user")
		respondError(c, http.StatusInternalServerError, "failed to create user")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      newID,
		"message": "user created successfully",
	})
}

// UpdateAdminUser updates an existing admin/operator user.
func (h *Handler) UpdateAdminUser(c *gin.Context) {
	targetID, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	actorClaims, _ := middleware.GetClaims(c)

	// Fetch target user's current role
	var targetRole string
	if err := h.db.QueryRow(`SELECT role FROM app_users WHERE id = $1`, targetID).Scan(&targetRole); err != nil {
		if err == sql.ErrNoRows {
			respondError(c, http.StatusNotFound, "user not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "database error")
		return
	}

	if !auth.CanManageRole(actorClaims.Role, targetRole) {
		respondError(c, http.StatusForbidden, "you cannot modify a user with equal or higher privileges")
		return
	}

	var req models.UpdateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	_, err = h.db.Exec(`
		UPDATE app_users
		SET email = COALESCE(NULLIF($1, ''), email),
		    full_name = COALESCE(NULLIF($2, ''), full_name),
		    role = COALESCE(NULLIF($3, ''), role),
		    is_active = COALESCE($4, is_active),
		    updated_at = NOW()
		WHERE id = $5`,
		req.Email, req.FullName, req.Role, req.IsActive, targetID,
	)
	if err != nil {
		h.log.WithError(err).Error("failed to update admin user")
		respondError(c, http.StatusInternalServerError, "failed to update user")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user updated successfully"})
}

// DeleteAdminUser deletes an admin/operator user.
func (h *Handler) DeleteAdminUser(c *gin.Context) {
	targetID, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid user ID")
		return
	}

	actorClaims, _ := middleware.GetClaims(c)
	if targetID == actorClaims.UserID {
		respondError(c, http.StatusBadRequest, "you cannot delete your own account")
		return
	}

	var targetRole string
	h.db.QueryRow(`SELECT role FROM app_users WHERE id = $1`, targetID).Scan(&targetRole)

	if !auth.CanManageRole(actorClaims.Role, targetRole) {
		respondError(c, http.StatusForbidden, "you cannot delete a user with equal or higher privileges")
		return
	}

	result, err := h.db.Exec(`DELETE FROM app_users WHERE id = $1`, targetID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to delete user")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		respondError(c, http.StatusNotFound, "user not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

// isUniqueViolation checks for PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "23505") || contains(err.Error(), "unique"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
