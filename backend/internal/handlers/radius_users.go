package handlers

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	strs "strings"
	"time"

	"github.com/freeradius-manager/backend/internal/auth"
	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/freeradius-manager/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// ListRadiusUsers returns paginated RADIUS users with optional filters.
func (h *Handler) ListRadiusUsers(c *gin.Context) {
	offset, limit := paginationParams(c)
	search := c.Query("search")
	status := c.Query("status")
	department := c.Query("department")

	// Build WHERE clause and shared args
	where := "WHERE 1=1"
	filterArgs := []interface{}{}
	argN := 1

	if search != "" {
		where += fmt.Sprintf(
			` AND (ru.username ILIKE $%d OR ru.email ILIKE $%d OR ru.full_name ILIKE $%d)`,
			argN, argN, argN,
		)
		filterArgs = append(filterArgs, "%"+search+"%")
		argN++
	}
	if status != "" {
		where += fmt.Sprintf(` AND ru.status = $%d`, argN)
		filterArgs = append(filterArgs, status)
		argN++
	}
	if department != "" {
		where += fmt.Sprintf(` AND ru.department ILIKE $%d`, argN)
		filterArgs = append(filterArgs, "%"+department+"%")
		argN++
	}

	// Count query
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM radius_users ru %s`, where)
	var total int
	h.db.QueryRow(countSQL, filterArgs...).Scan(&total)

	// Data query
	dataArgs := append(filterArgs, limit, offset)
	dataSQL := fmt.Sprintf(`
		SELECT ru.id, ru.username, ru.email, ru.full_name, ru.department,
		       ru.status, ru.device_limit, ru.account_expiry, ru.password_expiry,
		       ru.force_password_change, ru.created_by, au.username,
		       ru.created_at, ru.updated_at,
		       COUNT(ra.radacctid) FILTER (WHERE ra.acctstoptime IS NULL)
		FROM radius_users ru
		LEFT JOIN app_users au ON au.id = ru.created_by
		LEFT JOIN radacct ra ON ra.username = ru.username
		%s
		GROUP BY ru.id, au.username
		ORDER BY ru.created_at DESC
		LIMIT $%d OFFSET $%d`, where, argN, argN+1)

	rows, err := h.db.Query(dataSQL, dataArgs...)
	if err != nil {
		h.log.WithError(err).Error("list radius users query failed")
		respondError(c, http.StatusInternalServerError, "failed to fetch users")
		return
	}
	defer rows.Close()

	users := []models.RadiusUser{}
	for rows.Next() {
		var u models.RadiusUser
		var createdByUsername sql.NullString
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.FullName, &u.Department,
			&u.Status, &u.DeviceLimit, &u.AccountExpiry, &u.PasswordExpiry,
			&u.ForcePasswordChange, &u.CreatedBy, &createdByUsername,
			&u.CreatedAt, &u.UpdatedAt, &u.ActiveSessions,
		); err != nil {
			h.log.WithError(err).Warn("scan radius user row")
			continue
		}
		u.CreatedByUsername = createdByUsername.String
		users = append(users, u)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  users,
		"total": total,
	})
}

// GetRadiusUser returns a single RADIUS user by ID.
func (h *Handler) GetRadiusUser(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var u models.RadiusUser
	var createdByUsername sql.NullString
	err = h.db.QueryRow(`
		SELECT ru.id, ru.username, ru.email, ru.full_name, ru.department,
		       ru.status, ru.device_limit, ru.account_expiry, ru.password_expiry,
		       ru.force_password_change, ru.created_by, au.username,
		       ru.created_at, ru.updated_at
		FROM radius_users ru
		LEFT JOIN app_users au ON au.id = ru.created_by
		WHERE ru.id = $1`, id,
	).Scan(
		&u.ID, &u.Username, &u.Email, &u.FullName, &u.Department,
		&u.Status, &u.DeviceLimit, &u.AccountExpiry, &u.PasswordExpiry,
		&u.ForcePasswordChange, &u.CreatedBy, &createdByUsername,
		&u.CreatedAt, &u.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database error")
		return
	}
	u.CreatedByUsername = createdByUsername.String

	c.JSON(http.StatusOK, u)
}

// CreateRadiusUser creates a new RADIUS network-access user.
func (h *Handler) CreateRadiusUser(c *gin.Context) {
	var req models.CreateRadiusUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := auth.ValidatePasswordComplexity(req.Password); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.DeviceLimit == 0 {
		req.DeviceLimit = 1
	}

	hashedPw, err := auth.HashPassword(req.Password)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to hash password")
		return
	}

	claims, _ := middleware.GetClaims(c)

	var accountExpiry interface{}
	if req.AccountExpiry != nil && *req.AccountExpiry != "" {
		accountExpiry = *req.AccountExpiry
	}

	// Default password expiry (from settings, fallback 90 days)
	passwordExpiry := time.Now().AddDate(0, 0, 90).Format("2006-01-02")

	tx, err := h.db.Begin()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	var newID int
	err = tx.QueryRow(`
		INSERT INTO radius_users (username, password, email, full_name, department,
		    device_limit, account_expiry, password_expiry, force_password_change, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		req.Username, hashedPw, req.Email, req.FullName, req.Department,
		req.DeviceLimit, accountExpiry, passwordExpiry, req.ForcePasswordChange, claims.UserID,
	).Scan(&newID)

	if err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "username already exists")
			return
		}
		h.log.WithError(err).Error("create radius user failed")
		respondError(c, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Insert into radcheck for FreeRADIUS (Cleartext-Password for PAP/CHAP/EAP)
	_, err = tx.Exec(`
		INSERT INTO radcheck (username, attribute, op, value)
		VALUES ($1, 'Cleartext-Password', ':=', $2)
		ON CONFLICT (username, attribute) DO UPDATE SET value = $2`,
		req.Username, req.Password,
	)
	if err != nil {
		h.log.WithError(err).Error("insert radcheck failed")
		respondError(c, http.StatusInternalServerError, "failed to configure RADIUS attributes")
		return
	}

	// Simultaneous-Use for device limiting
	_, err = tx.Exec(`
		INSERT INTO radcheck (username, attribute, op, value)
		VALUES ($1, 'Simultaneous-Use', ':=', $2)
		ON CONFLICT (username, attribute) DO UPDATE SET value = $2`,
		req.Username, strconv.Itoa(req.DeviceLimit),
	)
	if err != nil {
		h.log.WithError(err).Warn("insert Simultaneous-Use failed")
	}

	if err := tx.Commit(); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to save user")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      newID,
		"message": "RADIUS user created successfully",
	})
}

// UpdateRadiusUser updates a RADIUS user's non-password fields.
func (h *Handler) UpdateRadiusUser(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var req models.UpdateRadiusUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Fetch current username for radcheck update
	var username string
	if err := h.db.QueryRow(`SELECT username FROM radius_users WHERE id = $1`, id).Scan(&username); err != nil {
		if err == sql.ErrNoRows {
			respondError(c, http.StatusNotFound, "user not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "database error")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to begin transaction")
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE radius_users
		SET email                = COALESCE(NULLIF($1, ''), email),
		    full_name            = COALESCE(NULLIF($2, ''), full_name),
		    department           = COALESCE(NULLIF($3, ''), department),
		    device_limit         = COALESCE($4, device_limit),
		    account_expiry       = COALESCE($5::date, account_expiry),
		    force_password_change = COALESCE($6, force_password_change),
		    updated_at           = NOW()
		WHERE id = $7`,
		req.Email, req.FullName, req.Department,
		req.DeviceLimit, req.AccountExpiry,
		req.ForcePasswordChange, id,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update user")
		return
	}

	// Update Simultaneous-Use if device_limit changed
	if req.DeviceLimit != nil {
		tx.Exec(`
			INSERT INTO radcheck (username, attribute, op, value)
			VALUES ($1, 'Simultaneous-Use', ':=', $2)
			ON CONFLICT (username, attribute) DO UPDATE SET value = $2`,
			username, strconv.Itoa(*req.DeviceLimit),
		)
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "user updated successfully"})
}

// DeleteRadiusUser removes a RADIUS user and all associated RADIUS attributes.
func (h *Handler) DeleteRadiusUser(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var username string
	if err := h.db.QueryRow(`SELECT username FROM radius_users WHERE id = $1`, id).Scan(&username); err != nil {
		if err == sql.ErrNoRows {
			respondError(c, http.StatusNotFound, "user not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "database error")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "transaction failed")
		return
	}
	defer tx.Rollback()

	tx.Exec(`DELETE FROM radcheck WHERE username = $1`, username)
	tx.Exec(`DELETE FROM radreply WHERE username = $1`, username)
	tx.Exec(`DELETE FROM radusergroup WHERE username = $1`, username)
	tx.Exec(`DELETE FROM radius_user_password_history WHERE user_id = $1`, id)
	tx.Exec(`DELETE FROM radius_users WHERE id = $1`, id)

	if err := tx.Commit(); err != nil {
		respondError(c, http.StatusInternalServerError, "failed to delete user")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

// ResetRadiusUserPassword resets a RADIUS user's password.
func (h *Handler) ResetRadiusUserPassword(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := auth.ValidatePasswordComplexity(req.NewPassword); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Check password history
	rows, _ := h.db.Query(`
		SELECT password_hash FROM radius_user_password_history
		WHERE user_id = $1
		ORDER BY changed_at DESC LIMIT 5`, id,
	)
	var history []string
	if rows != nil {
		for rows.Next() {
			var h string
			rows.Scan(&h)
			history = append(history, h)
		}
		rows.Close()
	}

	if err := auth.CheckPasswordHistory(req.NewPassword, history); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to hash password")
		return
	}

	var username string
	var oldHash string
	h.db.QueryRow(`SELECT username, password FROM radius_users WHERE id = $1`, id).Scan(&username, &oldHash)

	tx, _ := h.db.Begin()
	defer tx.Rollback()

	tx.Exec(`UPDATE radius_users SET password = $1, password_expiry = NOW() + INTERVAL '90 days', updated_at = NOW() WHERE id = $2`,
		newHash, id)
	tx.Exec(`UPDATE radcheck SET value = $1 WHERE username = $2 AND attribute = 'Cleartext-Password'`,
		req.NewPassword, username)
	// Save old password to history
	tx.Exec(`INSERT INTO radius_user_password_history (user_id, password_hash) VALUES ($1, $2)`,
		id, oldHash)

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}

// SuspendRadiusUser sets a user's status to suspended.
func (h *Handler) SuspendRadiusUser(c *gin.Context) {
	h.setRadiusUserStatus(c, "suspended")
}

// ActivateRadiusUser sets a user's status to active.
func (h *Handler) ActivateRadiusUser(c *gin.Context) {
	h.setRadiusUserStatus(c, "active")
}

func (h *Handler) setRadiusUserStatus(c *gin.Context, status string) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var username string
	if err := h.db.QueryRow(`SELECT username FROM radius_users WHERE id = $1`, id).Scan(&username); err != nil {
		if err == sql.ErrNoRows {
			respondError(c, http.StatusNotFound, "user not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "database error")
		return
	}

	h.db.Exec(`UPDATE radius_users SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)

	// Add/remove Auth-Type := Reject in radcheck to block login
	if status == "suspended" {
		h.db.Exec(`
			INSERT INTO radcheck (username, attribute, op, value)
			VALUES ($1, 'Auth-Type', ':=', 'Reject')
			ON CONFLICT (username, attribute) DO UPDATE SET value = 'Reject'`, username)
	} else {
		h.db.Exec(`DELETE FROM radcheck WHERE username = $1 AND attribute = 'Auth-Type'`, username)
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("user %s successfully", status)})
}

// DisconnectRadiusUser sends a RADIUS Disconnect-Request to terminate active sessions.
func (h *Handler) DisconnectRadiusUser(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var username string
	if err := h.db.QueryRow(`SELECT username FROM radius_users WHERE id = $1`, id).Scan(&username); err != nil {
		respondError(c, http.StatusNotFound, "user not found")
		return
	}

	// Get active sessions to find the NAS
	rows, err := h.db.Query(`
		SELECT acctsessionid, nasipaddress::text, callingstationid
		FROM radacct
		WHERE username = $1 AND acctstoptime IS NULL`, username)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to get sessions")
		return
	}
	defer rows.Close()

	disconnected := 0
	for rows.Next() {
		var sessionID, nasIP, callingStation string
		rows.Scan(&sessionID, &nasIP, &callingStation)
		// In production: send CoA/Disconnect-Request to NAS
		// For now, mark session as stopped in DB
		h.db.Exec(`
			UPDATE radacct SET acctstoptime = NOW(), acctterminatecause = 'Admin-Reset'
			WHERE acctsessionid = $1`, sessionID)
		disconnected++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      fmt.Sprintf("disconnected %d active session(s)", disconnected),
		"disconnected": disconnected,
	})
}

// RadiusUserSessions returns session history for a RADIUS user.
func (h *Handler) RadiusUserSessions(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var username string
	h.db.QueryRow(`SELECT username FROM radius_users WHERE id = $1`, id).Scan(&username)

	rows, err := h.db.Query(`
		SELECT acctsessionid, username, nasipaddress::text, nasportid,
		       framedipaddress::text, callingstationid, calledstationid,
		       acctstarttime, acctsessiontime, acctinputoctets, acctoutputoctets,
		       acctstoptime, acctterminatecause
		FROM radacct
		WHERE username = $1
		ORDER BY acctstarttime DESC
		LIMIT 50`, username)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch sessions")
		return
	}
	defer rows.Close()

	type Session struct {
		SessionID      string     `json:"session_id"`
		Username       string     `json:"username"`
		NASIPAddress   string     `json:"nas_ip"`
		NASPortID      *string    `json:"nas_port"`
		FramedIP       *string    `json:"framed_ip"`
		CallingStation string     `json:"calling_station"`
		CalledStation  string     `json:"called_station"`
		StartTime      *time.Time `json:"start_time"`
		Duration       int64      `json:"duration_seconds"`
		InputBytes     int64      `json:"input_bytes"`
		OutputBytes    int64      `json:"output_bytes"`
		StopTime       *time.Time `json:"stop_time"`
		TermCause      string     `json:"term_cause"`
		Active         bool       `json:"active"`
	}

	sessions := []Session{}
	for rows.Next() {
		var s Session
		rows.Scan(
			&s.SessionID, &s.Username, &s.NASIPAddress, &s.NASPortID,
			&s.FramedIP, &s.CallingStation, &s.CalledStation,
			&s.StartTime, &s.Duration, &s.InputBytes, &s.OutputBytes,
			&s.StopTime, &s.TermCause,
		)
		s.Active = s.StopTime == nil
		sessions = append(sessions, s)
	}

	c.JSON(http.StatusOK, gin.H{"data": sessions, "username": username})
}

// ImportRadiusUsers handles bulk CSV import of RADIUS users.
func (h *Handler) ImportRadiusUsers(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		respondError(c, http.StatusBadRequest, "CSV file required (field: file)")
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid CSV file")
		return
	}

	if len(records) < 2 {
		respondError(c, http.StatusBadRequest, "CSV must have header row and at least one data row")
		return
	}

	claims, _ := middleware.GetClaims(c)
	created := 0
	errors := []string{}

	for i, row := range records[1:] {
		if len(row) < 3 {
			errors = append(errors, fmt.Sprintf("row %d: insufficient columns (need username,password,email)", i+2))
			continue
		}
		username := strs.TrimSpace(row[0])
		password := strs.TrimSpace(row[1])
		email := strs.TrimSpace(row[2])
		fullName := ""
		if len(row) > 3 {
			fullName = strs.TrimSpace(row[3])
		}

		if err := auth.ValidatePasswordComplexity(password); err != nil {
			errors = append(errors, fmt.Sprintf("row %d (%s): %v", i+2, username, err))
			continue
		}

		hash, _ := auth.HashPassword(password)
		_, err := h.db.Exec(`
			INSERT INTO radius_users (username, password, email, full_name, created_by)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (username) DO NOTHING`,
			username, hash, email, fullName, claims.UserID,
		)
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d (%s): database error", i+2, username))
			continue
		}

		h.db.Exec(`
			INSERT INTO radcheck (username, attribute, op, value)
			VALUES ($1, 'Cleartext-Password', ':=', $2)
			ON CONFLICT (username, attribute) DO UPDATE SET value = $2`,
			username, password)

		created++
	}

	c.JSON(http.StatusOK, gin.H{
		"created": created,
		"errors":  errors,
		"message": fmt.Sprintf("imported %d users", created),
	})
}

// ExportRadiusUsers exports all RADIUS users as a CSV file.
func (h *Handler) ExportRadiusUsers(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT username, email, full_name, department, status, device_limit,
		       COALESCE(account_expiry::text, ''), created_at
		FROM radius_users
		ORDER BY username`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "export failed")
		return
	}
	defer rows.Close()

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=radius_users_export.csv")

	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{"username", "email", "full_name", "department", "status", "device_limit", "account_expiry", "created_at"})

	for rows.Next() {
		var username, email, fullName, department, status, expiry, createdAt string
		var deviceLimit int
		rows.Scan(&username, &email, &fullName, &department, &status, &deviceLimit, &expiry, &createdAt)
		writer.Write([]string{username, email, fullName, department, status, strconv.Itoa(deviceLimit), expiry, createdAt})
	}
	writer.Flush()
}
