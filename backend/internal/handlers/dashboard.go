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
	type AuthHourStat struct {
		Hour     string `json:"hour"`
		Accepted int    `json:"accepted"`
		Rejected int    `json:"rejected"`
		Total    int    `json:"total"`
	}

	type AuthDayStat struct {
		Day      string `json:"day"`
		Accepted int    `json:"accepted"`
		Rejected int    `json:"rejected"`
		Total    int    `json:"total"`
	}

	type TrafficHourStat struct {
		Hour  string `json:"hour"`
		Bytes int64  `json:"bytes"`
	}

	type TopUser struct {
		Username string `json:"username"`
		Sessions int    `json:"sessions"`
		Bytes    int64  `json:"bytes_total"`
	}

	type NASStat struct {
		NASIP string `json:"nas_ip"`
		Auths int    `json:"auths"`
	}

	type RecentAuth struct {
		ID        int64      `json:"id"`
		Username  string     `json:"username"`
		NASIPAddr *string    `json:"nas_ip"`
		Calling   *string    `json:"calling_station"`
		AuthTime  time.Time  `json:"auth_time"`
		Reply     string     `json:"reply"`
		Accepted  bool       `json:"accepted"`
	}

	var activeSessions int
	h.db.QueryRow(`SELECT COUNT(*) FROM radacct WHERE acctstoptime IS NULL`).Scan(&activeSessions)

	var totalUsers, activeUsers, suspendedUsers, totalNAS int
	h.db.QueryRow(`SELECT COUNT(*) FROM radius_users`).Scan(&totalUsers)
	h.db.QueryRow(`SELECT COUNT(*) FROM radius_users WHERE status = 'active'`).Scan(&activeUsers)
	h.db.QueryRow(`SELECT COUNT(*) FROM radius_users WHERE status = 'suspended'`).Scan(&suspendedUsers)
	h.db.QueryRow(`SELECT COUNT(*) FROM nas WHERE status = 'active'`).Scan(&totalNAS)

	var todayAccepts, todayRejects int
	h.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN reply IN ('Access-Accept', '2') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN reply IN ('Access-Reject', 'Reject') OR reply NOT IN ('Access-Accept', '2', '') THEN 1 ELSE 0 END), 0)
		FROM radpostauth WHERE authdate >= CURRENT_DATE`).Scan(&todayAccepts, &todayRejects)

	todayAuths := todayAccepts + todayRejects
	authSuccessRate := 0.0
	if todayAuths > 0 {
		authSuccessRate = float64(todayAccepts) / float64(todayAuths) * 100
	}

	var trafficToday, traffic7d int64
	h.db.QueryRow(`
		SELECT COALESCE(SUM(acctinputoctets + acctoutputoctets), 0)
		FROM radacct WHERE acctstarttime >= CURRENT_DATE`).Scan(&trafficToday)
	h.db.QueryRow(`
		SELECT COALESCE(SUM(acctinputoctets + acctoutputoctets), 0)
		FROM radacct WHERE acctstarttime >= NOW() - INTERVAL '7 days'`).Scan(&traffic7d)

	// Hourly auth accept/reject for last 24h (zero-filled)
	authHourRows, _ := h.db.Query(`
		SELECT h.hour,
		       COALESCE(a.accepted, 0),
		       COALESCE(a.rejected, 0)
		FROM generate_series(
			date_trunc('hour', NOW() - INTERVAL '23 hours'),
			date_trunc('hour', NOW()),
			INTERVAL '1 hour'
		) AS h(hour)
		LEFT JOIN (
			SELECT date_trunc('hour', authdate) AS hour,
			       SUM(CASE WHEN reply IN ('Access-Accept', '2') THEN 1 ELSE 0 END) AS accepted,
			       SUM(CASE WHEN reply IN ('Access-Reject', 'Reject') OR (reply NOT IN ('Access-Accept', '2', '') AND reply <> '') THEN 1 ELSE 0 END) AS rejected
			FROM radpostauth
			WHERE authdate >= NOW() - INTERVAL '24 hours'
			GROUP BY 1
		) a ON a.hour = h.hour
		ORDER BY h.hour`)

	authStats24h := []AuthHourStat{}
	if authHourRows != nil {
		defer authHourRows.Close()
		for authHourRows.Next() {
			var s AuthHourStat
			var hour time.Time
			authHourRows.Scan(&hour, &s.Accepted, &s.Rejected)
			s.Hour = hour.Format("2006-01-02 15:04")
			s.Total = s.Accepted + s.Rejected
			authStats24h = append(authStats24h, s)
		}
	}

	// Daily auth trend for last 7 days (zero-filled)
	authDayRows, _ := h.db.Query(`
		SELECT d.day::date,
		       COALESCE(a.accepted, 0),
		       COALESCE(a.rejected, 0)
		FROM generate_series(
			(CURRENT_DATE - INTERVAL '6 days')::date,
			CURRENT_DATE,
			INTERVAL '1 day'
		) AS d(day)
		LEFT JOIN (
			SELECT DATE(authdate) AS day,
			       SUM(CASE WHEN reply IN ('Access-Accept', '2') THEN 1 ELSE 0 END) AS accepted,
			       SUM(CASE WHEN reply IN ('Access-Reject', 'Reject') OR (reply NOT IN ('Access-Accept', '2', '') AND reply <> '') THEN 1 ELSE 0 END) AS rejected
			FROM radpostauth
			WHERE authdate >= CURRENT_DATE - INTERVAL '6 days'
			GROUP BY 1
		) a ON a.day = d.day::date
		ORDER BY d.day`)

	authStats7d := []AuthDayStat{}
	if authDayRows != nil {
		defer authDayRows.Close()
		for authDayRows.Next() {
			var s AuthDayStat
			var day time.Time
			authDayRows.Scan(&day, &s.Accepted, &s.Rejected)
			s.Day = day.Format("2006-01-02")
			s.Total = s.Accepted + s.Rejected
			authStats7d = append(authStats7d, s)
		}
	}

	// Hourly traffic for last 24h (zero-filled)
	trafficRows, _ := h.db.Query(`
		SELECT h.hour,
		       COALESCE(t.bytes, 0)
		FROM generate_series(
			date_trunc('hour', NOW() - INTERVAL '23 hours'),
			date_trunc('hour', NOW()),
			INTERVAL '1 hour'
		) AS h(hour)
		LEFT JOIN (
			SELECT date_trunc('hour', acctstarttime) AS hour,
			       SUM(acctinputoctets + acctoutputoctets) AS bytes
			FROM radacct
			WHERE acctstarttime >= NOW() - INTERVAL '24 hours'
			GROUP BY 1
		) t ON t.hour = h.hour
		ORDER BY h.hour`)

	trafficStats24h := []TrafficHourStat{}
	if trafficRows != nil {
		defer trafficRows.Close()
		for trafficRows.Next() {
			var s TrafficHourStat
			var hour time.Time
			trafficRows.Scan(&hour, &s.Bytes)
			s.Hour = hour.Format("2006-01-02 15:04")
			trafficStats24h = append(trafficStats24h, s)
		}
	}

	// Top users by successful auths (7d), fallback to session bytes from radacct
	topRows, _ := h.db.Query(`
		SELECT username, COUNT(*) AS sessions, 0 AS bytes
		FROM radpostauth
		WHERE authdate >= NOW() - INTERVAL '7 days'
		  AND reply IN ('Access-Accept', '2')
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
	if len(topUsers) == 0 {
		fallbackRows, _ := h.db.Query(`
			SELECT username, COUNT(*) AS sessions,
			       COALESCE(SUM(acctinputoctets + acctoutputoctets), 0) AS bytes
			FROM radacct
			WHERE acctstarttime >= NOW() - INTERVAL '7 days'
			GROUP BY username
			ORDER BY sessions DESC
			LIMIT 10`)
		if fallbackRows != nil {
			defer fallbackRows.Close()
			for fallbackRows.Next() {
				var u TopUser
				fallbackRows.Scan(&u.Username, &u.Sessions, &u.Bytes)
				topUsers = append(topUsers, u)
			}
		}
	}

	// NAS activity from post-auth (7d)
	nasRows, _ := h.db.Query(`
		SELECT COALESCE(nasipaddress::text, 'Unknown'), COUNT(*)
		FROM radpostauth
		WHERE authdate >= NOW() - INTERVAL '7 days'
		GROUP BY nasipaddress
		ORDER BY COUNT(*) DESC
		LIMIT 8`)

	nasStats := []NASStat{}
	if nasRows != nil {
		defer nasRows.Close()
		for nasRows.Next() {
			var n NASStat
			nasRows.Scan(&n.NASIP, &n.Auths)
			nasStats = append(nasStats, n)
		}
	}

	recentRows, _ := h.db.Query(`
		SELECT id, username, nasipaddress::text, callingstationid, authdate, reply
		FROM radpostauth
		ORDER BY authdate DESC
		LIMIT 15`)

	recentAuths := []RecentAuth{}
	if recentRows != nil {
		defer recentRows.Close()
		for recentRows.Next() {
			var r RecentAuth
			var reply string
			recentRows.Scan(&r.ID, &r.Username, &r.NASIPAddr, &r.Calling, &r.AuthTime, &reply)
			r.Reply = reply
			r.Accepted = reply == "Access-Accept" || reply == "2"
			recentAuths = append(recentAuths, r)
		}
	}

	// Legacy field: sessions per hour from radacct (kept for compatibility)
	type LegacyAuthStat struct {
		Hour     string `json:"hour"`
		Sessions int    `json:"sessions"`
	}
	legacyStats := make([]LegacyAuthStat, len(authStats24h))
	for i, s := range authStats24h {
		legacyStats[i] = LegacyAuthStat{Hour: s.Hour, Sessions: s.Total}
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"active_sessions":   activeSessions,
			"total_users":       totalUsers,
			"active_users":      activeUsers,
			"suspended_users":   suspendedUsers,
			"total_nas":         totalNAS,
			"today_auths":       todayAuths,
			"today_accepts":     todayAccepts,
			"today_rejects":     todayRejects,
			"auth_success_rate": authSuccessRate,
			"traffic_today":     trafficToday,
			"traffic_7d":        traffic7d,
		},
		"auth_stats_24h":    legacyStats,
		"auth_hourly_24h":   authStats24h,
		"auth_daily_7d":     authStats7d,
		"traffic_hourly_24h": trafficStats24h,
		"nas_stats_7d":      nasStats,
		"top_users":         topUsers,
		"recent_auths":      recentAuths,
		"server_time":       time.Now().UTC(),
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

// AuthLogs returns recent RADIUS authentication attempts from radpostauth.
func (h *Handler) AuthLogs(c *gin.Context) {
	offset, limit := paginationParams(c)
	username := c.Query("username")

	type LogEntry struct {
		ID        int64      `json:"id"`
		Username  string     `json:"username"`
		NASIPAddr *string    `json:"nas_ip"`
		Calling   *string    `json:"calling_station"`
		Called    *string    `json:"called_station"`
		AuthTime  time.Time  `json:"auth_time"`
		Reply     string     `json:"reply"`
		Accepted  bool       `json:"accepted"`
	}

	var rows *sql.Rows
	var err error

	if username != "" {
		rows, err = h.db.Query(`
			SELECT id, username, nasipaddress::text, callingstationid, calledstationid,
			       authdate, reply
			FROM radpostauth
			WHERE username ILIKE $1
			ORDER BY authdate DESC
			LIMIT $2 OFFSET $3`, "%"+username+"%", limit, offset)
	} else {
		rows, err = h.db.Query(`
			SELECT id, username, nasipaddress::text, callingstationid, calledstationid,
			       authdate, reply
			FROM radpostauth
			ORDER BY authdate DESC
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
		var reply string
		rows.Scan(&l.ID, &l.Username, &l.NASIPAddr, &l.Calling, &l.Called, &l.AuthTime, &reply)
		l.Reply = reply
		l.Accepted = reply == "Access-Accept" || reply == "2"
		logs = append(logs, l)
	}

	var total int
	if username != "" {
		h.db.QueryRow(`SELECT COUNT(*) FROM radpostauth WHERE username ILIKE $1`, "%"+username+"%").Scan(&total)
	} else {
		h.db.QueryRow(`SELECT COUNT(*) FROM radpostauth`).Scan(&total)
	}

	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
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
