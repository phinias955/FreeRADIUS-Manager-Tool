package handlers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/freeradius-manager/backend/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ─────────────────────────────────────────────────────────────────────────────
// Honeypot Listener  (runs on UDP port 11812)
// ─────────────────────────────────────────────────────────────────────────────

var honeypotRunning bool
var honeypotMu sync.Mutex

// StartHoneypot starts a fake RADIUS listener on port 11812 that logs all packets.
func StartHoneypot(db *database.DB, log *logrus.Logger) {
	honeypotMu.Lock()
	if honeypotRunning {
		honeypotMu.Unlock()
		return
	}
	honeypotRunning = true
	honeypotMu.Unlock()

	var enabled string
	db.QueryRow(`SELECT value FROM system_settings WHERE key='honeypot_enabled'`).Scan(&enabled)
	if enabled == "false" {
		log.Info("Honeypot: disabled via settings")
		return
	}

	port := os.Getenv("HONEYPOT_PORT")
	if port == "" {
		port = "11812"
	}

	go func() {
		addr := fmt.Sprintf("0.0.0.0:%s", port)
		conn, err := net.ListenPacket("udp", addr)
		if err != nil {
			log.WithField("error", err).Error("Honeypot: failed to listen on " + addr)
			honeypotMu.Lock()
			honeypotRunning = false
			honeypotMu.Unlock()
			return
		}
		defer conn.Close()
		log.Info("Honeypot RADIUS listener started on UDP " + addr)

		buf := make([]byte, 4096)
		for {
			n, srcAddr, err := conn.ReadFrom(buf)
			if err != nil {
				continue
			}
			pkt := buf[:n]
			if len(pkt) < 20 {
				continue
			}

			srcIP, _, _ := net.SplitHostPort(srcAddr.String())
			username, nasIP := extractHoneypotInfo(pkt)

			attrMap := map[string]string{}
			if len(pkt) > 20 {
				decodeReplyAttrs(pkt[20:], attrMap)
			}
			attrJSON, _ := json.Marshal(attrMap)

			db.Exec(`INSERT INTO honeypot_logs (source_ip, username, nas_ip, packet_type, attributes)
				VALUES ($1,$2,$3,'Access-Request',$4)`,
				srcIP, nullableString(username), nullableString(nasIP), string(attrJSON))

			// Alert on first probe from a new IP
			var existingCount int
			db.QueryRow(`SELECT COUNT(*) FROM honeypot_logs WHERE source_ip=$1`, srcIP).Scan(&existingCount)
			if existingCount <= 1 {
				db.Exec(`INSERT INTO security_alerts (alert_type,severity,ip_address,username,details)
					VALUES ('honeypot_probe','high',$1,$2,$3::jsonb)`,
					srcIP, nullableString(username),
					fmt.Sprintf(`{"source":"%s","username":"%s","honeypot_port":"%s"}`, srcIP, username, port))
			}

			// Respond with Access-Reject to keep scanners busy
			reject := buildRejectPacket(pkt[1], pkt[4:20])
			conn.WriteTo(reject, srcAddr)
		}
	}()
}

func extractHoneypotInfo(pkt []byte) (username, nasIP string) {
	if len(pkt) <= 20 {
		return
	}
	attrs := pkt[20:]
	for i := 0; i+2 <= len(attrs); {
		t := attrs[i]
		l := int(attrs[i+1])
		if l < 2 || i+l > len(attrs) {
			break
		}
		val := attrs[i+2 : i+l]
		switch t {
		case 1:
			username = string(val)
		case 4:
			if len(val) == 4 {
				nasIP = fmt.Sprintf("%d.%d.%d.%d", val[0], val[1], val[2], val[3])
			}
		}
		i += l
	}
	return
}

func buildRejectPacket(id byte, reqAuth []byte) []byte {
	pkt := []byte{3, id, 0, 20}
	pkt = append(pkt, reqAuth...)
	h := md5.New()
	h.Write(pkt)
	auth := h.Sum(nil)
	copy(pkt[4:], auth)
	return pkt
}

func computeMD5(data []byte) []byte {
	h := md5.Sum(data)
	return h[:]
}

// ─────────────────────────────────────────────────────────────────────────────
// Honeypot HTTP API handlers
// ─────────────────────────────────────────────────────────────────────────────

// ListHoneypotLogs returns recent honeypot activity.
func (h *Handler) ListHoneypotLogs(c *gin.Context) {
	offset, limit := paginationParams(c)
	srcIP := c.Query("ip")

	rows, err := h.db.Query(`
		SELECT id, source_ip, username, nas_ip, packet_type, attributes, created_at
		FROM honeypot_logs
		WHERE ($1='' OR source_ip=$1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		srcIP, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch honeypot logs")
		return
	}
	defer rows.Close()

	type HoneypotLog struct {
		ID         int             `json:"id"`
		SourceIP   string          `json:"source_ip"`
		Username   *string         `json:"username"`
		NasIP      *string         `json:"nas_ip"`
		PacketType string          `json:"packet_type"`
		Attributes json.RawMessage `json:"attributes"`
		CreatedAt  time.Time       `json:"created_at"`
	}
	logs := []HoneypotLog{}
	for rows.Next() {
		var l HoneypotLog
		var attrsStr string
		rows.Scan(&l.ID, &l.SourceIP, &l.Username, &l.NasIP, &l.PacketType, &attrsStr, &l.CreatedAt)
		if attrsStr == "" {
			attrsStr = "{}"
		}
		l.Attributes = json.RawMessage(attrsStr)
		logs = append(logs, l)
	}

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM honeypot_logs WHERE ($1='' OR source_ip=$1)`, srcIP).Scan(&total)

	type TopIP struct {
		IP    string `json:"ip"`
		Count int    `json:"count"`
	}
	topIPs := []TopIP{}
	tipRows, _ := h.db.Query(`SELECT source_ip, COUNT(*) FROM honeypot_logs
		GROUP BY source_ip ORDER BY COUNT(*) DESC LIMIT 10`)
	if tipRows != nil {
		defer tipRows.Close()
		for tipRows.Next() {
			var t TopIP
			tipRows.Scan(&t.IP, &t.Count)
			topIPs = append(topIPs, t)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total, "top_ips": topIPs})
}

// ClearHoneypotLogs deletes old honeypot entries.
func (h *Handler) ClearHoneypotLogs(c *gin.Context) {
	var req struct {
		OlderThanDays int `json:"older_than_days"`
	}
	c.ShouldBindJSON(&req)
	if req.OlderThanDays == 0 {
		req.OlderThanDays = 30
	}
	result, _ := h.db.Exec(`DELETE FROM honeypot_logs WHERE created_at < NOW()-($1 || ' days')::interval`,
		fmt.Sprintf("%d", req.OlderThanDays))
	n, _ := result.RowsAffected()
	c.JSON(http.StatusOK, gin.H{"deleted": n})
}

// HoneypotStatus returns whether the honeypot is running.
func (h *Handler) HoneypotStatus(c *gin.Context) {
	var enabled string
	h.db.QueryRow(`SELECT value FROM system_settings WHERE key='honeypot_enabled'`).Scan(&enabled)

	var total, todayCount int
	h.db.QueryRow(`SELECT COUNT(*) FROM honeypot_logs`).Scan(&total)
	h.db.QueryRow(`SELECT COUNT(*) FROM honeypot_logs WHERE created_at >= CURRENT_DATE`).Scan(&todayCount)

	c.JSON(http.StatusOK, gin.H{
		"running":      honeypotRunning,
		"enabled":      enabled != "false",
		"total_probes": total,
		"today_probes": todayCount,
	})
}
