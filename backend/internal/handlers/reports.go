package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// UsageReport returns per-user data usage over a time period.
func (h *Handler) UsageReport(c *gin.Context) {
	period := c.DefaultQuery("period", "7d")
	username := c.Query("username")

	interval := "7 days"
	switch period {
	case "1d":
		interval = "1 day"
	case "30d":
		interval = "30 days"
	case "90d":
		interval = "90 days"
	}

	type UserUsage struct {
		Username    string  `json:"username"`
		Sessions    int     `json:"sessions"`
		InputBytes  int64   `json:"input_bytes"`
		OutputBytes int64   `json:"output_bytes"`
		TotalBytes  int64   `json:"total_bytes"`
		TotalGB     float64 `json:"total_gb"`
		AvgDuration float64 `json:"avg_duration_seconds"`
	}

	where := ""
	args := []interface{}{fmt.Sprintf("%s", interval)}
	argN := 2
	if username != "" {
		where = fmt.Sprintf(" AND username ILIKE $%d", argN)
		args = append(args, "%"+username+"%")
	}

	rows, err := h.db.Query(fmt.Sprintf(`
		SELECT username,
		       COUNT(*) as sessions,
		       COALESCE(SUM(acctinputoctets), 0) as input_bytes,
		       COALESCE(SUM(acctoutputoctets), 0) as output_bytes,
		       COALESCE(SUM(acctinputoctets + acctoutputoctets), 0) as total_bytes,
		       COALESCE(AVG(acctsessiontime), 0) as avg_duration
		FROM radacct
		WHERE acctstarttime >= NOW() - INTERVAL $1 %s
		GROUP BY username
		ORDER BY total_bytes DESC
		LIMIT 100`, where), args...)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch usage report")
		return
	}
	defer rows.Close()

	users := []UserUsage{}
	for rows.Next() {
		var u UserUsage
		rows.Scan(&u.Username, &u.Sessions, &u.InputBytes, &u.OutputBytes, &u.TotalBytes, &u.AvgDuration)
		u.TotalGB = float64(u.TotalBytes) / (1024 * 1024 * 1024)
		users = append(users, u)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   users,
		"period": period,
	})
}

// DailyUsageReport returns day-by-day aggregate data for charts.
func (h *Handler) DailyUsageReport(c *gin.Context) {
	period := c.DefaultQuery("period", "30d")

	interval := "30 days"
	switch period {
	case "7d":
		interval = "7 days"
	case "90d":
		interval = "90 days"
	}

	type DayEntry struct {
		Day     string  `json:"day"`
		Sessions int    `json:"sessions"`
		TotalGB float64 `json:"total_gb"`
		Users   int     `json:"unique_users"`
	}

	rows, err := h.db.Query(fmt.Sprintf(`
		SELECT DATE(acctstarttime) as day,
		       COUNT(*) as sessions,
		       COALESCE(SUM(acctinputoctets + acctoutputoctets), 0) / 1073741824.0 as total_gb,
		       COUNT(DISTINCT username) as unique_users
		FROM radacct
		WHERE acctstarttime >= NOW() - INTERVAL '%s'
		GROUP BY day
		ORDER BY day ASC`, interval))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch daily report")
		return
	}
	defer rows.Close()

	days := []DayEntry{}
	for rows.Next() {
		var d DayEntry
		var dayTime time.Time
		rows.Scan(&dayTime, &d.Sessions, &d.TotalGB, &d.Users)
		d.Day = dayTime.Format("2006-01-02")
		days = append(days, d)
	}

	c.JSON(http.StatusOK, gin.H{"data": days, "period": period})
}

// AuthSuccessReport returns auth success vs failure counts per day.
func (h *Handler) AuthSuccessReport(c *gin.Context) {
	period := c.DefaultQuery("period", "7d")

	interval := "7 days"
	switch period {
	case "1d":
		interval = "1 day"
	case "30d":
		interval = "30 days"
	}

	type AuthDay struct {
		Day      string `json:"day"`
		Accepted int    `json:"accepted"`
		Rejected int    `json:"rejected"`
	}

	rows, err := h.db.Query(fmt.Sprintf(`
		SELECT DATE(authdate) as day,
		       SUM(CASE WHEN reply = 'Access-Accept' THEN 1 ELSE 0 END) as accepted,
		       SUM(CASE WHEN reply = 'Access-Reject'  THEN 1 ELSE 0 END) as rejected
		FROM radpostauth
		WHERE authdate >= NOW() - INTERVAL '%s'
		GROUP BY day
		ORDER BY day ASC`, interval))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch auth report")
		return
	}
	defer rows.Close()

	days := []AuthDay{}
	for rows.Next() {
		var d AuthDay
		var dayTime time.Time
		rows.Scan(&dayTime, &d.Accepted, &d.Rejected)
		d.Day = dayTime.Format("2006-01-02")
		days = append(days, d)
	}

	c.JSON(http.StatusOK, gin.H{"data": days, "period": period})
}

// NASUsageReport returns per-NAS session and traffic stats.
func (h *Handler) NASUsageReport(c *gin.Context) {
	period := c.DefaultQuery("period", "7d")
	interval := "7 days"
	switch period {
	case "1d":
		interval = "1 day"
	case "30d":
		interval = "30 days"
	}

	type NASStats struct {
		NASIP      string  `json:"nas_ip"`
		Sessions   int     `json:"sessions"`
		TotalBytes int64   `json:"total_bytes"`
		TotalGB    float64 `json:"total_gb"`
		UniqueUsers int    `json:"unique_users"`
	}

	rows, err := h.db.Query(fmt.Sprintf(`
		SELECT nasipaddress::text,
		       COUNT(*) as sessions,
		       COALESCE(SUM(acctinputoctets + acctoutputoctets), 0) as total_bytes,
		       COUNT(DISTINCT username) as unique_users
		FROM radacct
		WHERE acctstarttime >= NOW() - INTERVAL '%s'
		GROUP BY nasipaddress
		ORDER BY sessions DESC`, interval))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch NAS report")
		return
	}
	defer rows.Close()

	stats := []NASStats{}
	for rows.Next() {
		var s NASStats
		rows.Scan(&s.NASIP, &s.Sessions, &s.TotalBytes, &s.UniqueUsers)
		s.TotalGB = float64(s.TotalBytes) / (1024 * 1024 * 1024)
		stats = append(stats, s)
	}

	c.JSON(http.StatusOK, gin.H{"data": stats, "period": period})
}

// ExportUsageReport exports usage data as CSV.
func (h *Handler) ExportUsageReport(c *gin.Context) {
	period := c.DefaultQuery("period", "30d")
	interval := "30 days"
	switch period {
	case "7d":
		interval = "7 days"
	case "90d":
		interval = "90 days"
	}

	rows, err := h.db.Query(fmt.Sprintf(`
		SELECT username,
		       COUNT(*) as sessions,
		       COALESCE(SUM(acctinputoctets), 0),
		       COALESCE(SUM(acctoutputoctets), 0),
		       COALESCE(SUM(acctinputoctets + acctoutputoctets), 0),
		       COALESCE(AVG(acctsessiontime), 0)
		FROM radacct
		WHERE acctstarttime >= NOW() - INTERVAL '%s'
		GROUP BY username
		ORDER BY SUM(acctinputoctets + acctoutputoctets) DESC`, interval))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "export failed")
		return
	}
	defer rows.Close()

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="usage_report_%s_%s.csv"`, period, time.Now().Format("20060102")))

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"username", "sessions", "input_bytes", "output_bytes", "total_bytes", "total_gb", "avg_duration_sec"})

	for rows.Next() {
		var username string
		var sessions int
		var inputBytes, outputBytes, totalBytes int64
		var avgDuration float64
		rows.Scan(&username, &sessions, &inputBytes, &outputBytes, &totalBytes, &avgDuration)
		totalGB := float64(totalBytes) / (1024 * 1024 * 1024)
		w.Write([]string{
			username,
			fmt.Sprintf("%d", sessions),
			fmt.Sprintf("%d", inputBytes),
			fmt.Sprintf("%d", outputBytes),
			fmt.Sprintf("%d", totalBytes),
			fmt.Sprintf("%.3f", totalGB),
			fmt.Sprintf("%.0f", avgDuration),
		})
	}
	w.Flush()
}
