package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// PortalLogin authenticates a RADIUS user and returns a short-lived portal token.
func (h *Handler) PortalLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Look up user
	var userID int
	var storedHash, status string
	var phone, email *string
	err := h.db.QueryRow(`
		SELECT id, password, status, phone, email
		FROM radius_users WHERE username=$1`, req.Username).
		Scan(&userID, &storedHash, &status, &phone, &email)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)); err != nil {
		respondError(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if status == "suspended" {
		respondError(c, http.StatusForbidden, "account suspended — contact your administrator")
		return
	}

	// Generate portal session token (32-byte hex)
	raw := make([]byte, 32)
	rand.Read(raw)
	token := hex.EncodeToString(raw)
	expiresAt := time.Now().Add(24 * time.Hour)

	h.db.Exec(`
		INSERT INTO portal_sessions (token, username, expires_at)
		VALUES ($1,$2,$3)`, token, req.Username, expiresAt)

	// Clean up old expired sessions
	go h.db.Exec(`DELETE FROM portal_sessions WHERE expires_at < NOW()`)

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"username":   req.Username,
		"expires_at": expiresAt,
	})
}

// PortalDashboard returns all self-service data for the authenticated portal user.
func (h *Handler) PortalDashboard(c *gin.Context) {
	username := c.GetString("portal_username")
	if username == "" {
		respondError(c, http.StatusUnauthorized, "portal session required")
		return
	}

	// Basic user info
	type UserInfo struct {
		Username      string     `json:"username"`
		Email         *string    `json:"email"`
		Phone         *string    `json:"phone"`
		Status        string     `json:"status"`
		AccountExpiry *string    `json:"account_expiry"`
		PlanName      *string    `json:"plan_name"`
		PlanPrice     *float64   `json:"plan_price"`
		PlanCurrency  *string    `json:"plan_currency"`
		DataLimitMB   *int64     `json:"data_limit_mb"`
		ValidityDays  *int       `json:"validity_days"`
		CreatedAt     time.Time  `json:"created_at"`
	}

	var u UserInfo
	h.db.QueryRow(`
		SELECT ru.username, ru.email, ru.phone, ru.status,
		       ru.account_expiry::text, up.name, up.price, up.currency,
		       up.data_limit_mb, up.validity_days, ru.created_at
		FROM radius_users ru
		LEFT JOIN user_plans up ON up.id = ru.plan_id
		WHERE ru.username=$1`, username).
		Scan(&u.Username, &u.Email, &u.Phone, &u.Status,
			&u.AccountExpiry, &u.PlanName, &u.PlanPrice, &u.PlanCurrency,
			&u.DataLimitMB, &u.ValidityDays, &u.CreatedAt)

	// Data usage (last 30 days)
	type Usage struct {
		TotalMB      float64 `json:"total_mb"`
		UploadMB     float64 `json:"upload_mb"`
		DownloadMB   float64 `json:"download_mb"`
		SessionCount int     `json:"session_count"`
		UsedPct      float64 `json:"used_pct"`
	}
	var usage Usage
	h.db.QueryRow(`
		SELECT
		  COALESCE(SUM(acctinputoctets+acctoutputoctets),0)/(1024.0*1024.0),
		  COALESCE(SUM(acctinputoctets),0)/(1024.0*1024.0),
		  COALESCE(SUM(acctoutputoctets),0)/(1024.0*1024.0),
		  COUNT(*)
		FROM radacct
		WHERE username=$1 AND acctstarttime >= NOW()-INTERVAL '30 days'`, username).
		Scan(&usage.TotalMB, &usage.UploadMB, &usage.DownloadMB, &usage.SessionCount)

	if u.DataLimitMB != nil && *u.DataLimitMB > 0 {
		usage.UsedPct = (usage.TotalMB / float64(*u.DataLimitMB)) * 100
		if usage.UsedPct > 100 {
			usage.UsedPct = 100
		}
	}

	// Active sessions
	type Session struct {
		NASName   string  `json:"nasname"`
		FramedIP  *string `json:"framed_ip"`
		StartTime string  `json:"start_time"`
		Duration  int64   `json:"duration_seconds"`
		InputMB   float64 `json:"input_mb"`
		OutputMB  float64 `json:"output_mb"`
	}
	sessions := []Session{}
	rows, _ := h.db.Query(`
		SELECT COALESCE(nasipaddress,''),
		       framedipaddress,
		       TO_CHAR(acctstarttime,'YYYY-MM-DD HH24:MI:SS'),
		       EXTRACT(EPOCH FROM (NOW()-acctstarttime))::BIGINT,
		       acctinputoctets/1048576.0,
		       acctoutputoctets/1048576.0
		FROM radacct
		WHERE username=$1 AND acctstoptime IS NULL
		ORDER BY acctstarttime DESC LIMIT 5`, username)
	if rows != nil {
		for rows.Next() {
			var s Session
			rows.Scan(&s.NASName, &s.FramedIP, &s.StartTime, &s.Duration, &s.InputMB, &s.OutputMB)
			sessions = append(sessions, s)
		}
		rows.Close()
	}

	// Last 5 sessions
	type HistSession struct {
		NASName   string  `json:"nasname"`
		StartTime string  `json:"start_time"`
		StopTime  string  `json:"stop_time"`
		TotalMB   float64 `json:"total_mb"`
		Duration  int64   `json:"duration_seconds"`
	}
	history := []HistSession{}
	hrows, _ := h.db.Query(`
		SELECT COALESCE(nasipaddress,''),
		       TO_CHAR(acctstarttime,'YYYY-MM-DD HH24:MI'),
		       TO_CHAR(acctstoptime,'YYYY-MM-DD HH24:MI'),
		       (acctinputoctets+acctoutputoctets)/1048576.0,
		       COALESCE(acctsessiontime,0)
		FROM radacct
		WHERE username=$1 AND acctstoptime IS NOT NULL
		ORDER BY acctstarttime DESC LIMIT 10`, username)
	if hrows != nil {
		for hrows.Next() {
			var hs HistSession
			hrows.Scan(&hs.NASName, &hs.StartTime, &hs.StopTime, &hs.TotalMB, &hs.Duration)
			history = append(history, hs)
		}
		hrows.Close()
	}

	// Assigned IP
	var assignedIP *string
	h.db.QueryRow(`SELECT value FROM radreply WHERE username=$1 AND attribute='Framed-IP-Address'`, username).Scan(&assignedIP)

	c.JSON(http.StatusOK, gin.H{
		"user":           u,
		"usage":          usage,
		"active_sessions": sessions,
		"session_history": history,
		"assigned_ip":    assignedIP,
	})
}

// PortalLogout invalidates the portal session token.
func (h *Handler) PortalLogout(c *gin.Context) {
	token := c.GetHeader("X-Portal-Token")
	if token != "" {
		h.db.Exec(`DELETE FROM portal_sessions WHERE token=$1`, token)
	}
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// PortalAuthMiddleware validates the X-Portal-Token header.
func (h *Handler) PortalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Portal-Token")
		if token == "" {
			respondError(c, http.StatusUnauthorized, "portal token required")
			c.Abort()
			return
		}
		var username string
		var expiresAt time.Time
		err := h.db.QueryRow(`
			SELECT username, expires_at FROM portal_sessions
			WHERE token=$1`, token).Scan(&username, &expiresAt)
		if err != nil || expiresAt.Before(time.Now()) {
			h.db.Exec(`DELETE FROM portal_sessions WHERE token=$1`, token)
			respondError(c, http.StatusUnauthorized, "portal session expired or invalid")
			c.Abort()
			return
		}
		c.Set("portal_username", username)
		c.Next()
	}
}
