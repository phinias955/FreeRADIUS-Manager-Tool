package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// LiveStats streams real-time system statistics using Server-Sent Events.
func (h *Handler) LiveStats(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable nginx buffering

	ctx := c.Request.Context()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send first event immediately
	sendStats(c, h)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendStats(c, h)
		}
	}
}

type LiveStatsPayload struct {
	ActiveSessions  int     `json:"active_sessions"`
	AuthLast5m      int     `json:"auth_last_5m"`
	RejectLast5m    int     `json:"reject_last_5m"`
	TotalUsers      int     `json:"total_users"`
	ActiveUsers     int     `json:"active_users"`
	ExpiredUsers    int     `json:"expired_users"`
	NASUp           int     `json:"nas_up"`
	NASDown         int     `json:"nas_down"`
	BandwidthInMbps float64 `json:"bandwidth_in_mbps"`
	BandwidthOutMbps float64 `json:"bandwidth_out_mbps"`
	TopUser         string  `json:"top_user"`
	Timestamp       int64   `json:"timestamp"`
}

func sendStats(c *gin.Context, h *Handler) {
	var stats LiveStatsPayload
	stats.Timestamp = time.Now().Unix()

	h.db.QueryRow(`SELECT COUNT(*) FROM radacct WHERE acctstoptime IS NULL`).Scan(&stats.ActiveSessions)

	h.db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE reply='Access-Accept'),
		COUNT(*) FILTER (WHERE reply='Access-Reject')
		FROM radpostauth WHERE authdate >= NOW()-INTERVAL '5 minutes'`).
		Scan(&stats.AuthLast5m, &stats.RejectLast5m)

	h.db.QueryRow(`SELECT COUNT(*),
		COUNT(*) FILTER (WHERE status='active'),
		COUNT(*) FILTER (WHERE status='expired')
		FROM radius_users`).Scan(&stats.TotalUsers, &stats.ActiveUsers, &stats.ExpiredUsers)

	h.db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE ping_status='up'),
		COUNT(*) FILTER (WHERE ping_status='down')
		FROM nas WHERE status='active'`).Scan(&stats.NASUp, &stats.NASDown)

	// Estimate bandwidth from recent radacct updates
	h.db.QueryRow(`SELECT
		COALESCE(SUM(acctinputoctets),0)*8.0/300.0/1048576.0,
		COALESCE(SUM(acctoutputoctets),0)*8.0/300.0/1048576.0
		FROM radacct
		WHERE acctstoptime IS NULL AND acctupdatetime >= NOW()-INTERVAL '5 minutes'`).
		Scan(&stats.BandwidthInMbps, &stats.BandwidthOutMbps)

	// Top user by data today
	h.db.QueryRow(`SELECT COALESCE(username,'') FROM radacct
		WHERE acctstarttime >= CURRENT_DATE
		GROUP BY username
		ORDER BY SUM(acctinputoctets+acctoutputoctets) DESC LIMIT 1`).Scan(&stats.TopUser)

	data, _ := json.Marshal(stats)
	fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	c.Writer.Flush()
}

// GetCurrentStats returns current stats as a regular JSON response (non-SSE).
func (h *Handler) GetCurrentStats(c *gin.Context) {
	var stats LiveStatsPayload
	stats.Timestamp = time.Now().Unix()

	h.db.QueryRow(`SELECT COUNT(*) FROM radacct WHERE acctstoptime IS NULL`).Scan(&stats.ActiveSessions)
	h.db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE reply='Access-Accept'),
		COUNT(*) FILTER (WHERE reply='Access-Reject')
		FROM radpostauth WHERE authdate >= NOW()-INTERVAL '5 minutes'`).
		Scan(&stats.AuthLast5m, &stats.RejectLast5m)
	h.db.QueryRow(`SELECT COUNT(*),
		COUNT(*) FILTER (WHERE status='active'),
		COUNT(*) FILTER (WHERE status='expired')
		FROM radius_users`).Scan(&stats.TotalUsers, &stats.ActiveUsers, &stats.ExpiredUsers)
	h.db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE ping_status='up'),
		COUNT(*) FILTER (WHERE ping_status='down')
		FROM nas WHERE status='active'`).Scan(&stats.NASUp, &stats.NASDown)
	h.db.QueryRow(`SELECT COALESCE(username,'') FROM radacct
		WHERE acctstarttime >= CURRENT_DATE
		GROUP BY username
		ORDER BY SUM(acctinputoctets+acctoutputoctets) DESC LIMIT 1`).Scan(&stats.TopUser)

	c.JSON(http.StatusOK, stats)
}
