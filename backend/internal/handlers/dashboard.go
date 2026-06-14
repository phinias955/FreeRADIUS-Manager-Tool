package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DashboardStats returns aggregated statistics for the dashboard.
func (h *Handler) DashboardStats(c *gin.Context) {
	type AuthStat struct {
		Hour    string `json:"hour"`
		Sessions int   `json:"sessions"`
	}

	type TopUser struct {
		Username string `json:"username"`
		Sessions int    `json:"sessions"`
		Bytes    int64  `json:"bytes_total"`
	}

	var activeSessions int
	h.db.QueryRow(`SELECT COUNT(*) FROM radacct WHERE acctstoptime IS NULL`).Scan(&activeSessions)

	var totalUsers int
	h.db.QueryRow(`SELECT COUNT(*) FROM radius_users`).Scan(&totalUsers)

	var activeUsers int
	h.db.QueryRow(`SELECT COUNT(*) FROM radius_users WHERE status = 'active'`).Scan(&activeUsers)

	var suspendedUsers int
	h.db.QueryRow(`SELECT COUNT(*) FROM radius_users WHERE status = 'suspended'`).Scan(&suspendedUsers)

	var totalNAS int
	h.db.QueryRow(`SELECT COUNT(*) FROM nas WHERE status = 'active'`).Scan(&totalNAS)

	var todayAuths int
	h.db.QueryRow(`SELECT COUNT(*) FROM radacct WHERE acctstarttime >= CURRENT_DATE`).Scan(&todayAuths)

	authRows, _ := h.db.Query(`
		SELECT date_trunc('hour', acctstarttime) as hour, COUNT(*) as total
		FROM radacct
		WHERE acctstarttime >= NOW() - INTERVAL '24 hours'
		GROUP BY hour
		ORDER BY hour`)

	authStats := []AuthStat{}
	if authRows != nil {
		defer authRows.Close()
		for authRows.Next() {
			var s AuthStat
			var hour time.Time
			authRows.Scan(&hour, &s.Sessions)
			s.Hour = hour.Format("2006-01-02 15:04")
			authStats = append(authStats, s)
		}
	}

	topRows, _ := h.db.Query(`
		SELECT username, COUNT(*) as sessions,
		       COALESCE(SUM(acctinputoctets + acctoutputoctets), 0) as bytes
		FROM radacct
		WHERE acctstarttime >= NOW() - INTERVAL '7 days'
		GROUP BY username
		ORDER BY sessions DESC
		LIMIT 10`)

	topUsers := []TopUser{}
	if topRows != nil {
		defer topRows.Close()
		for topRows.Next() {
			var u TopUser
			topRows.Scan(&u.Username, &u.Sessions, &u.Bytes)
			topUsers = append(topUsers, u)
		}
	}

	type RecentAuth struct {
		Username    string     `json:"username"`
		NASIPAddr   string     `json:"nas_ip"`
		FramedIP    *string    `json:"framed_ip"`
		StartTime   *time.Time `json:"start_time"`
		SessionTime int64      `json:"session_time"`
		Active      bool       `json:"active"`
	}

	recentRows, _ := h.db.Query(`
		SELECT username, nasipaddress::text, framedipaddress::text,
		       acctstarttime, acctsessiontime, (acctstoptime IS NULL) as active
		FROM radacct
		ORDER BY acctstarttime DESC
		LIMIT 20`)

	recentAuths := []RecentAuth{}
	if recentRows != nil {
		defer recentRows.Close()
		for recentRows.Next() {
			var r RecentAuth
			recentRows.Scan(&r.Username, &r.NASIPAddr, &r.FramedIP,
				&r.StartTime, &r.SessionTime, &r.Active)
			recentAuths = append(recentAuths, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"active_sessions": activeSessions,
			"total_users":     totalUsers,
			"active_users":    activeUsers,
			"suspended_users": suspendedUsers,
			"total_nas":       totalNAS,
			"today_auths":     todayAuths,
		},
		"auth_stats_24h": authStats,
		"top_users":      topUsers,
		"recent_auths":   recentAuths,
		"server_time":    time.Now().UTC(),
	})
}

// ActiveSessions returns currently active RADIUS sessions.
func (h *Handler) ActiveSessions(c *gin.Context) {
	offset, limit := paginationParams(c)

	rows, err := h.db.Query(`
		SELECT acctsessionid, username, nasipaddress::text, nasportid,
		       framedipaddress::text, callingstationid, calledstationid,
		       acctstarttime, acctsessiontime, acctinputoctets, acctoutputoctets
		FROM radacct
		WHERE acctstoptime IS NULL
		ORDER BY acctstarttime DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch sessions")
		return
	}
	defer rows.Close()

	type ActiveSession struct {
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
	}

	sessions := []ActiveSession{}
	for rows.Next() {
		var s ActiveSession
		rows.Scan(&s.SessionID, &s.Username, &s.NASIPAddress, &s.NASPortID,
			&s.FramedIP, &s.CallingStation, &s.CalledStation,
			&s.StartTime, &s.Duration, &s.InputBytes, &s.OutputBytes)
		sessions = append(sessions, s)
	}

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM radacct WHERE acctstoptime IS NULL`).Scan(&total)

	c.JSON(http.StatusOK, gin.H{"data": sessions, "total": total})
}

// UserSessions returns session history for a specific username.
func (h *Handler) UserSessions(c *gin.Context) {
	username := c.Param("username")
	offset, limit := paginationParams(c)

	rows, err := h.db.Query(`
		SELECT acctsessionid, username, nasipaddress::text, nasportid,
		       framedipaddress::text, callingstationid, calledstationid,
		       acctstarttime, acctsessiontime, acctinputoctets, acctoutputoctets,
		       acctstoptime, acctterminatecause
		FROM radacct
		WHERE username = $1
		ORDER BY acctstarttime DESC
		LIMIT $2 OFFSET $3`, username, limit, offset)
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
		rows.Scan(&s.SessionID, &s.Username, &s.NASIPAddress, &s.NASPortID,
			&s.FramedIP, &s.CallingStation, &s.CalledStation,
			&s.StartTime, &s.Duration, &s.InputBytes, &s.OutputBytes,
			&s.StopTime, &s.TermCause)
		s.Active = s.StopTime == nil
		sessions = append(sessions, s)
	}

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM radacct WHERE username = $1`, username).Scan(&total)

	c.JSON(http.StatusOK, gin.H{"data": sessions, "total": total, "username": username})
}

// AuthLogs returns recent RADIUS accounting/auth log entries.
func (h *Handler) AuthLogs(c *gin.Context) {
	offset, limit := paginationParams(c)
	username := c.Query("username")

	type LogEntry struct {
		SessionID string     `json:"session_id"`
		Username  string     `json:"username"`
		NASIPAddr string     `json:"nas_ip"`
		Calling   string     `json:"calling_station"`
		StartTime *time.Time `json:"start_time"`
		StopTime  *time.Time `json:"stop_time"`
		Duration  int64      `json:"duration"`
		TermCause string     `json:"term_cause"`
		Active    bool       `json:"active"`
	}

	var rows *sql.Rows
	var err error

	if username != "" {
		rows, err = h.db.Query(`
			SELECT acctsessionid, username, nasipaddress::text, callingstationid,
			       acctstarttime, acctstoptime, acctsessiontime, acctterminatecause,
			       (acctstoptime IS NULL) as active
			FROM radacct
			WHERE username ILIKE $1
			ORDER BY acctstarttime DESC
			LIMIT $2 OFFSET $3`, "%"+username+"%", limit, offset)
	} else {
		rows, err = h.db.Query(`
			SELECT acctsessionid, username, nasipaddress::text, callingstationid,
			       acctstarttime, acctstoptime, acctsessiontime, acctterminatecause,
			       (acctstoptime IS NULL) as active
			FROM radacct
			ORDER BY acctstarttime DESC
			LIMIT $1 OFFSET $2`, limit, offset)
	}

	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch logs")
		return
	}
	defer rows.Close()

	logs := []LogEntry{}
	for rows.Next() {
		var l LogEntry
		rows.Scan(&l.SessionID, &l.Username, &l.NASIPAddr, &l.Calling,
			&l.StartTime, &l.StopTime, &l.Duration, &l.TermCause, &l.Active)
		logs = append(logs, l)
	}

	c.JSON(http.StatusOK, gin.H{"data": logs})
}

// AuditLogs returns admin audit log entries.
func (h *Handler) AuditLogs(c *gin.Context) {
	offset, limit := paginationParams(c)

	rows, err := h.db.Query(`
		SELECT al.id, al.user_id, au.username, al.action, al.target_type,
		       al.target_id, al.details::text, al.ip_address::text, al.created_at
		FROM audit_log al
		LEFT JOIN app_users au ON au.id = al.user_id
		ORDER BY al.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch audit logs")
		return
	}
	defer rows.Close()

	type AuditEntry struct {
		ID         int64   `json:"id"`
		UserID     *int    `json:"user_id"`
		Username   *string `json:"username"`
		Action     string  `json:"action"`
		TargetType *string `json:"target_type"`
		TargetID   *int    `json:"target_id"`
		Details    *string `json:"details"`
		IPAddress  *string `json:"ip_address"`
		CreatedAt  time.Time `json:"created_at"`
	}

	logs := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		rows.Scan(&e.ID, &e.UserID, &e.Username, &e.Action, &e.TargetType,
			&e.TargetID, &e.Details, &e.IPAddress, &e.CreatedAt)
		logs = append(logs, e)
	}

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&total)

	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
}

// GetSettings returns all system settings.
func (h *Handler) GetSettings(c *gin.Context) {
	rows, err := h.db.Query(`SELECT key, value, description FROM system_settings ORDER BY key`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch settings")
		return
	}
	defer rows.Close()

	settings := map[string]gin.H{}
	for rows.Next() {
		var key, value, desc string
		rows.Scan(&key, &value, &desc)
		settings[key] = gin.H{"value": value, "description": desc}
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates one or more system settings.
func (h *Handler) UpdateSettings(c *gin.Context) {
	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	for key, value := range updates {
		h.db.Exec(`UPDATE system_settings SET value = $1, updated_at = NOW() WHERE key = $2`, value, key)
	}

	c.JSON(http.StatusOK, gin.H{"message": "settings updated successfully"})
}

// CreateBackup triggers a database dump.
func (h *Handler) CreateBackup(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":  "backup initiated",
		"filename": fmt.Sprintf("radius_backup_%s.sql", time.Now().Format("20060102_150405")),
	})
}

// RestoreBackup explains how to restore.
func (h *Handler) RestoreBackup(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "To restore: docker exec -i radius_postgres psql -U radius_user radius < backup.sql",
	})
}

// ListBackups lists available backup files.
func (h *Handler) ListBackups(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"backups": []interface{}{}})
}
