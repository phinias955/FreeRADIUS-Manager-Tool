package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Security Alerts
// ─────────────────────────────────────────────────────────────────────────────

type SecurityAlert struct {
	ID              int             `json:"id"`
	AlertType       string          `json:"alert_type"`
	Severity        string          `json:"severity"`
	IPAddress       *string         `json:"ip_address"`
	Username        *string         `json:"username"`
	CountryCode     *string         `json:"country_code"`
	Details         []byte          `json:"details"`
	IsAcknowledged  bool            `json:"is_acknowledged"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ListSecurityAlerts returns security alerts with optional filters.
func (h *Handler) ListSecurityAlerts(c *gin.Context) {
	severity := c.Query("severity")
	alertType := c.Query("type")
	onlyNew := c.Query("unread") == "true"
	offset, limit := paginationParams(c)

	rows, err := h.db.Query(`
		SELECT id, alert_type, severity, ip_address, username, country_code,
		       details, is_acknowledged, created_at
		FROM security_alerts
		WHERE ($1='' OR severity=$1)
		  AND ($2='' OR alert_type=$2)
		  AND ($3=FALSE OR is_acknowledged=FALSE)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5`,
		severity, alertType, onlyNew, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch alerts")
		return
	}
	defer rows.Close()

	type Alert struct {
		ID             int       `json:"id"`
		AlertType      string    `json:"alert_type"`
		Severity       string    `json:"severity"`
		IPAddress      *string   `json:"ip_address"`
		Username       *string   `json:"username"`
		CountryCode    *string   `json:"country_code"`
		Details        string    `json:"details"`
		IsAcknowledged bool      `json:"is_acknowledged"`
		CreatedAt      time.Time `json:"created_at"`
	}

	alerts := []Alert{}
	for rows.Next() {
		var a Alert
		rows.Scan(&a.ID, &a.AlertType, &a.Severity, &a.IPAddress, &a.Username,
			&a.CountryCode, &a.Details, &a.IsAcknowledged, &a.CreatedAt)
		alerts = append(alerts, a)
	}

	// Summary counts
	type Summary struct {
		Critical int `json:"critical"`
		High     int `json:"high"`
		Medium   int `json:"medium"`
		Low      int `json:"low"`
		Unread   int `json:"unread"`
	}
	var s Summary
	h.db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE severity='critical'),
		COUNT(*) FILTER (WHERE severity='high'),
		COUNT(*) FILTER (WHERE severity='medium'),
		COUNT(*) FILTER (WHERE severity='low'),
		COUNT(*) FILTER (WHERE is_acknowledged=FALSE)
		FROM security_alerts`).Scan(&s.Critical, &s.High, &s.Medium, &s.Low, &s.Unread)

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM security_alerts
		WHERE ($1='' OR severity=$1) AND ($2='' OR alert_type=$2)`, severity, alertType).Scan(&total)

	c.JSON(http.StatusOK, gin.H{"data": alerts, "total": total, "summary": s})
}

// AcknowledgeAlert marks an alert as acknowledged.
func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`UPDATE security_alerts SET is_acknowledged=TRUE WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "alert acknowledged"})
}

// AcknowledgeAllAlerts marks all alerts as acknowledged.
func (h *Handler) AcknowledgeAllAlerts(c *gin.Context) {
	result, _ := h.db.Exec(`UPDATE security_alerts SET is_acknowledged=TRUE WHERE is_acknowledged=FALSE`)
	n, _ := result.RowsAffected()
	c.JSON(http.StatusOK, gin.H{"message": "all alerts acknowledged", "count": n})
}

// DeleteAlert removes a single alert.
func (h *Handler) DeleteAlert(c *gin.Context) {
	id, _ := mustInt(c, "id")
	h.db.Exec(`DELETE FROM security_alerts WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "alert deleted"})
}

// SecuritySummary returns a high-level security health snapshot.
func (h *Handler) SecuritySummary(c *gin.Context) {
	type Stat struct {
		Label string `json:"label"`
		Value int    `json:"value"`
		Color string `json:"color"`
	}

	var critAlerts, blockedIPs, honeypotToday, failedLast1h, csPatterns int
	h.db.QueryRow(`SELECT COUNT(*) FROM security_alerts WHERE severity IN ('critical','high') AND is_acknowledged=FALSE`).Scan(&critAlerts)
	h.db.QueryRow(`SELECT COUNT(*) FROM cred_stuffing_blocks WHERE blocked_until > NOW()`).Scan(&blockedIPs)
	h.db.QueryRow(`SELECT COUNT(*) FROM honeypot_logs WHERE created_at >= CURRENT_DATE`).Scan(&honeypotToday)
	h.db.QueryRow(`SELECT COUNT(*) FROM radpostauth WHERE reply='Access-Reject' AND authdate >= NOW()-INTERVAL '1 hour'`).Scan(&failedLast1h)
	h.db.QueryRow(`SELECT COUNT(*) FROM security_alerts WHERE alert_type='credential_stuffing_pattern' AND created_at >= NOW()-INTERVAL '24 hours'`).Scan(&csPatterns)

	stats := []Stat{
		{"Unread High/Critical Alerts", critAlerts, "red"},
		{"Blocked IPs", blockedIPs, "orange"},
		{"Honeypot Probes Today", honeypotToday, "purple"},
		{"Failed Auths (1h)", failedLast1h, "yellow"},
		{"CS Patterns (24h)", csPatterns, "red"},
	}

	// Trend: auth failures per hour (last 24h)
	type HourStat struct {
		Hour  string `json:"hour"`
		Fails int    `json:"fails"`
	}
	trend := []HourStat{}
	trows, _ := h.db.Query(`SELECT TO_CHAR(DATE_TRUNC('hour',authdate),'HH24:MI') AS hour,
		COUNT(*) FROM radpostauth
		WHERE reply='Access-Reject' AND authdate >= NOW()-INTERVAL '24 hours'
		GROUP BY hour ORDER BY hour`)
	if trows != nil {
		defer trows.Close()
		for trows.Next() {
			var t HourStat
			trows.Scan(&t.Hour, &t.Fails)
			trend = append(trend, t)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":     stats,
		"fail_trend": trend,
	})
}
