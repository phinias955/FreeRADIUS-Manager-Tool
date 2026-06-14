package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/freeradius-manager/backend/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ─────────────────────────────────────────────────────────────────────────────
// GeoIP Enforcement
// ─────────────────────────────────────────────────────────────────────────────

// GeoIPLookup looks up an IP address country via ip-api.com (cached in DB).
func (h *Handler) GeoIPLookup(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		respondError(c, http.StatusBadRequest, "ip query parameter required")
		return
	}
	info, err := lookupIP(h, ip)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, info)
}

// GeoIPInfo holds the result of a GeoIP lookup.
type GeoIPInfo struct {
	IP          string `json:"ip"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	City        string `json:"city"`
	ISP         string `json:"isp"`
	IsVPN       bool   `json:"is_vpn"`
	Cached      bool   `json:"cached"`
}

func lookupIP(h *Handler, ip string) (*GeoIPInfo, error) {
	var info GeoIPInfo
	info.IP = ip

	// Check cache (valid for 24 hours)
	err := h.db.QueryRow(`SELECT country_code, country_name, city, isp, is_vpn
		FROM geoip_cache WHERE ip_address=$1 AND looked_up_at > NOW()-INTERVAL '24 hours'`, ip).
		Scan(&info.CountryCode, &info.CountryName, &info.City, &info.ISP, &info.IsVPN)
	if err == nil {
		info.Cached = true
		return &info, nil
	}

	// Call ip-api.com (free, no key required)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(
		fmt.Sprintf("http://ip-api.com/json/%s?fields=countryCode,country,city,isp,proxy", ip))
	if err != nil {
		return nil, fmt.Errorf("GeoIP lookup failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var data struct {
		CountryCode string `json:"countryCode"`
		Country     string `json:"country"`
		City        string `json:"city"`
		ISP         string `json:"isp"`
		Proxy       bool   `json:"proxy"`
	}
	json.Unmarshal(body, &data)

	info.CountryCode = data.CountryCode
	info.CountryName = data.Country
	info.City = data.City
	info.ISP = data.ISP
	info.IsVPN = data.Proxy

	h.db.Exec(`INSERT INTO geoip_cache (ip_address,country_code,country_name,city,isp,is_vpn)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (ip_address) DO UPDATE SET
		  country_code=$2, country_name=$3, city=$4, isp=$5, is_vpn=$6, looked_up_at=NOW()`,
		ip, info.CountryCode, info.CountryName, info.City, info.ISP, info.IsVPN)

	return &info, nil
}

// ListGeoIPRules returns all country block/allow/flag rules.
func (h *Handler) ListGeoIPRules(c *gin.Context) {
	rows, _ := h.db.Query(`SELECT id,country_code,country_name,action,is_active,created_at
		FROM geoip_rules ORDER BY action, country_name`)
	type Rule struct {
		ID          int       `json:"id"`
		CountryCode string    `json:"country_code"`
		CountryName string    `json:"country_name"`
		Action      string    `json:"action"`
		IsActive    bool      `json:"is_active"`
		CreatedAt   time.Time `json:"created_at"`
	}
	rules := []Rule{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var r Rule
			rows.Scan(&r.ID, &r.CountryCode, &r.CountryName, &r.Action, &r.IsActive, &r.CreatedAt)
			rules = append(rules, r)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": rules})
}

// CreateGeoIPRule adds or updates a country rule.
func (h *Handler) CreateGeoIPRule(c *gin.Context) {
	var req struct {
		CountryCode string `json:"country_code" binding:"required"`
		CountryName string `json:"country_name" binding:"required"`
		Action      string `json:"action"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	req.CountryCode = strings.ToUpper(req.CountryCode)
	if req.Action == "" {
		req.Action = "block"
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	var id int
	err := h.db.QueryRow(`INSERT INTO geoip_rules (country_code,country_name,action,is_active)
		VALUES ($1,$2,$3,$4) ON CONFLICT (country_code) DO UPDATE SET
		action=$3, country_name=$2, is_active=$4 RETURNING id`,
		req.CountryCode, req.CountryName, req.Action, isActive).Scan(&id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to save rule")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "GeoIP rule saved"})
}

// DeleteGeoIPRule removes a country rule.
func (h *Handler) DeleteGeoIPRule(c *gin.Context) {
	id, _ := mustInt(c, "id")
	h.db.Exec(`DELETE FROM geoip_rules WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "rule deleted"})
}

// ─────────────────────────────────────────────────────────────────────────────
// Credential Stuffing Detection  (realtime sliding window, in-memory)
// ─────────────────────────────────────────────────────────────────────────────

type failEntry struct {
	timestamps []time.Time
	mu         sync.Mutex
}

var (
	failWindows  = sync.Map{} // IP → *failEntry
	csWorkerOnce sync.Once
)

// StartCredStuffingDetector starts background analysis + cleanup goroutines.
func StartCredStuffingDetector(db *database.DB, log *logrus.Logger) {
	csWorkerOnce.Do(func() {
		go credStuffingAnalysisWorker(db, log)
		go credStuffingCleanupWorker(db, log)
		log.Info("Credential stuffing detector started")
	})
}

// CheckCredStuffing is called on every failed auth to update the sliding window.
// Returns true if the IP should be blocked.
func CheckCredStuffing(h *Handler, ip, username string) bool {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return false
	}

	maxFails := 10
	windowSecs := 300
	blockMins := 60
	var mfStr, wsStr, bmStr string
	h.db.QueryRow(`SELECT value FROM system_settings WHERE key='cs_max_fails'`).Scan(&mfStr)
	h.db.QueryRow(`SELECT value FROM system_settings WHERE key='cs_window_secs'`).Scan(&wsStr)
	h.db.QueryRow(`SELECT value FROM system_settings WHERE key='cs_block_mins'`).Scan(&bmStr)
	if mfStr != "" {
		fmt.Sscanf(mfStr, "%d", &maxFails)
	}
	if wsStr != "" {
		fmt.Sscanf(wsStr, "%d", &windowSecs)
	}
	if bmStr != "" {
		fmt.Sscanf(bmStr, "%d", &blockMins)
	}

	// Check if already DB-blocked
	var blockedUntil *time.Time
	h.db.QueryRow(`SELECT blocked_until FROM cred_stuffing_blocks WHERE ip_address=$1`, ip).Scan(&blockedUntil)
	if blockedUntil != nil && blockedUntil.After(time.Now()) {
		return true
	}

	now := time.Now()
	cutoff := now.Add(-time.Duration(windowSecs) * time.Second)

	raw, _ := failWindows.LoadOrStore(ip, &failEntry{})
	entry := raw.(*failEntry)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	valid := entry.timestamps[:0]
	for _, ts := range entry.timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	valid = append(valid, now)
	entry.timestamps = valid

	if len(valid) >= maxFails {
		blockedUntilTime := now.Add(time.Duration(blockMins) * time.Minute)
		h.db.Exec(`INSERT INTO cred_stuffing_blocks (ip_address,fail_count,blocked_until,reason)
			VALUES ($1,$2,$3,'sliding-window rate limit')
			ON CONFLICT (ip_address) DO UPDATE SET
			fail_count=cred_stuffing_blocks.fail_count+1, blocked_until=$3, updated_at=NOW()`,
			ip, len(valid), blockedUntilTime)
		h.db.Exec(`INSERT INTO security_alerts (alert_type,severity,ip_address,username,details)
			VALUES ('credential_stuffing','critical',$1,$2,$3::jsonb)`,
			ip, nullableString(username),
			fmt.Sprintf(`{"fail_count":%d,"window_secs":%d,"blocked_until":"%s"}`,
				len(valid), windowSecs, blockedUntilTime.Format(time.RFC3339)))
		return true
	}
	return false
}

// GetBlockedIPs returns the current block list.
func (h *Handler) GetBlockedIPs(c *gin.Context) {
	rows, _ := h.db.Query(`SELECT id,ip_address,fail_count,blocked_until,reason,auto_blocked,created_at
		FROM cred_stuffing_blocks
		WHERE blocked_until IS NULL OR blocked_until > NOW()
		ORDER BY fail_count DESC LIMIT 200`)
	type Block struct {
		ID           int        `json:"id"`
		IPAddress    string     `json:"ip_address"`
		FailCount    int        `json:"fail_count"`
		BlockedUntil *time.Time `json:"blocked_until"`
		Reason       *string    `json:"reason"`
		AutoBlocked  bool       `json:"auto_blocked"`
		CreatedAt    time.Time  `json:"created_at"`
	}
	blocks := []Block{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var b Block
			rows.Scan(&b.ID, &b.IPAddress, &b.FailCount, &b.BlockedUntil, &b.Reason, &b.AutoBlocked, &b.CreatedAt)
			blocks = append(blocks, b)
		}
	}
	var totalBlocked int
	h.db.QueryRow(`SELECT COUNT(*) FROM cred_stuffing_blocks WHERE blocked_until > NOW()`).Scan(&totalBlocked)
	c.JSON(http.StatusOK, gin.H{"data": blocks, "active_blocks": totalBlocked})
}

// BlockIP manually blocks an IP.
func (h *Handler) BlockIP(c *gin.Context) {
	var req struct {
		IP       string `json:"ip" binding:"required"`
		Duration int    `json:"duration_hours"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Duration == 0 {
		req.Duration = 24
	}
	blockedUntil := time.Now().Add(time.Duration(req.Duration) * time.Hour)
	reason := req.Reason
	if reason == "" {
		reason = "manually blocked"
	}
	h.db.Exec(`INSERT INTO cred_stuffing_blocks (ip_address,fail_count,blocked_until,reason,auto_blocked)
		VALUES ($1,0,$2,$3,false)
		ON CONFLICT (ip_address) DO UPDATE SET blocked_until=$2, reason=$3, updated_at=NOW()`,
		req.IP, blockedUntil, reason)
	c.JSON(http.StatusOK, gin.H{"message": "IP blocked until " + blockedUntil.Format(time.RFC822)})
}

// UnblockIP removes an IP from the block list.
func (h *Handler) UnblockIP(c *gin.Context) {
	id, _ := mustInt(c, "id")
	h.db.Exec(`DELETE FROM cred_stuffing_blocks WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "IP unblocked"})
}

// credStuffingAnalysisWorker scans radpostauth every 5 minutes for patterns.
func credStuffingAnalysisWorker(db *database.DB, log *logrus.Logger) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		// Detect: single NAS IP probing many different usernames in 10 minutes
		rows, err := db.Query(`
			SELECT nasipaddress::text, COUNT(DISTINCT username) AS unique_users,
			       COUNT(*) AS fails
			FROM radpostauth
			WHERE authdate >= NOW()-INTERVAL '10 minutes'
			  AND reply = 'Access-Reject'
			GROUP BY nasipaddress
			HAVING COUNT(DISTINCT username) >= 5`)
		if err != nil || rows == nil {
			continue
		}
		for rows.Next() {
			var ip string
			var uniqueUsers, fails int
			rows.Scan(&ip, &uniqueUsers, &fails)

			var exists bool
			db.QueryRow(`SELECT EXISTS(SELECT 1 FROM security_alerts WHERE ip_address=$1
				AND alert_type='credential_stuffing_pattern' AND created_at > NOW()-INTERVAL '1 hour')`, ip).Scan(&exists)
			if !exists {
				db.Exec(`INSERT INTO security_alerts (alert_type,severity,ip_address,details)
					VALUES ('credential_stuffing_pattern','high',$1,$2::jsonb)`,
					ip, fmt.Sprintf(`{"unique_usernames":%d,"fail_count":%d,"window":"10min"}`, uniqueUsers, fails))
				log.WithField("ip", ip).WithField("unique_users", uniqueUsers).
					Warn("Credential stuffing pattern detected")
			}
		}
		rows.Close()
	}
}

// credStuffingCleanupWorker removes expired blocks every hour.
func credStuffingCleanupWorker(db *database.DB, log *logrus.Logger) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		result, _ := db.Exec(`DELETE FROM cred_stuffing_blocks WHERE blocked_until < NOW()`)
		n, _ := result.RowsAffected()
		if n > 0 {
			log.WithField("count", n).Info("Credential stuffing: expired blocks removed")
		}
		db.Exec(`DELETE FROM geoip_cache WHERE looked_up_at < NOW()-INTERVAL '48 hours'`)
	}
}
