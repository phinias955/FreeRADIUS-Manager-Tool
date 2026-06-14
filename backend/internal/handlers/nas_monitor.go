package handlers

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/freeradius-manager/backend/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NASPingResult holds the result of a ping check.
type NASPingResult struct {
	NASID     int       `json:"id"`
	NASName   string    `json:"nasname"`
	ShortName string    `json:"shortname"`
	Status    string    `json:"ping_status"`
	LatencyMs float64   `json:"ping_latency_ms"`
	LastPing  time.Time `json:"last_ping"`
}

// GetNASStatus returns all NAS devices with current ping status.
func (h *Handler) GetNASStatus(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, nasname, COALESCE(shortname,''), COALESCE(ping_status,'unknown'),
		       COALESCE(ping_latency_ms,0), COALESCE(last_ping, NOW()-INTERVAL '999 days')
		FROM nas WHERE status='active'
		ORDER BY shortname`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch status")
		return
	}
	defer rows.Close()

	results := []NASPingResult{}
	for rows.Next() {
		var r NASPingResult
		rows.Scan(&r.NASID, &r.NASName, &r.ShortName, &r.Status, &r.LatencyMs, &r.LastPing)
		results = append(results, r)
	}
	c.JSON(http.StatusOK, gin.H{"data": results})
}

// PingNASNow forces an immediate ping of a single NAS device.
func (h *Handler) PingNASNow(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var nasname string
	if err := h.db.QueryRow(`SELECT nasname FROM nas WHERE id=$1`, id).Scan(&nasname); err != nil {
		respondError(c, http.StatusNotFound, "NAS not found")
		return
	}
	up, latency := pingHost(nasname)
	status := "up"
	if !up {
		status = "down"
	}
	h.db.Exec(`UPDATE nas SET ping_status=$1, ping_latency_ms=$2, last_ping=NOW() WHERE id=$3`,
		status, latency, id)
	c.JSON(http.StatusOK, gin.H{
		"nasname":   nasname,
		"status":    status,
		"latency_ms": latency,
	})
}

// pingHost checks if a host is reachable by trying TCP on common management ports.
// Returns (up, latency_ms).
func pingHost(host string) (bool, float64) {
	// Strip CIDR notation if present
	ip := host
	if strings.Contains(host, "/") {
		parts := strings.SplitN(host, "/", 2)
		ip = parts[0]
	}

	ports := []int{22, 80, 443, 8291, 23, 8080, 8443}
	start := time.Now()

	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 1500*time.Millisecond)
		if err == nil {
			conn.Close()
			return true, float64(time.Since(start).Milliseconds())
		}
		// "connection refused" = host is up but port closed
		if strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "refused") {
			return true, float64(time.Since(start).Milliseconds())
		}
	}
	return false, float64(time.Since(start).Milliseconds())
}

// StartNASMonitor launches a background goroutine that pings all active NAS
// devices every 60 seconds and updates their ping_status in the database.
func StartNASMonitor(db *database.DB, log *logrus.Logger) {
	go func() {
		log.Info("NAS monitor started — pinging devices every 60s")
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		// Run immediately on startup
		pingAllNAS(db, log)

		for range ticker.C {
			pingAllNAS(db, log)
		}
	}()
}

func pingAllNAS(db *database.DB, log *logrus.Logger) {
	rows, err := db.Query(`SELECT id, nasname FROM nas WHERE status='active'`)
	if err != nil {
		return
	}
	defer rows.Close()

	type nasEntry struct {
		id      int
		nasname string
	}
	devices := []nasEntry{}
	for rows.Next() {
		var e nasEntry
		rows.Scan(&e.id, &e.nasname)
		devices = append(devices, e)
	}

	// Ping concurrently (max 20 at once)
	sem := make(chan struct{}, 20)
	var wg sync.WaitGroup

	for _, d := range devices {
		wg.Add(1)
		sem <- struct{}{}
		go func(dev nasEntry) {
			defer wg.Done()
			defer func() { <-sem }()

			up, latency := pingHost(dev.nasname)
			status := "up"
			if !up {
				status = "down"
			}
			db.Exec(`UPDATE nas SET ping_status=$1, ping_latency_ms=$2, last_ping=NOW() WHERE id=$3`,
				status, latency, dev.id)
		}(d)
	}
	wg.Wait()
	log.Debugf("NAS monitor: pinged %d devices", len(devices))
}
