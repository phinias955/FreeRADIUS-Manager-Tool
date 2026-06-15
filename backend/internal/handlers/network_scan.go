package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/freeradius-manager/backend/internal/models"
	"github.com/freeradius-manager/backend/internal/scanner"
	"github.com/gin-gonic/gin"
)

// StartNetworkScan launches an async nmap scan of a subnet.
func (h *Handler) StartNetworkScan(c *gin.Context) {
	if !scanner.NmapAvailable() {
		respondError(c, http.StatusServiceUnavailable, "nmap is not installed on the server")
		return
	}

	var req models.NetworkScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := scanner.ValidateSubnet(req.Subnet); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.ScanType == "" {
		req.ScanType = "discovery"
	}

	var running int
	h.db.QueryRow(`SELECT COUNT(*) FROM network_scans WHERE status = 'running'`).Scan(&running)
	if running >= 2 {
		respondError(c, http.StatusConflict, "maximum concurrent scans reached; wait for current scans to finish")
		return
	}

	claims, _ := middleware.GetClaims(c)
	var startedBy *int
	if claims != nil {
		startedBy = &claims.UserID
	}

	var scanID int
	err := h.db.QueryRow(`
		INSERT INTO network_scans (subnet, scan_type, status, started_by)
		VALUES ($1, $2, 'running', $3)
		RETURNING id`, req.Subnet, req.ScanType, startedBy).Scan(&scanID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create scan job")
		return
	}

	go h.executeNetworkScan(scanID, req.Subnet, req.ScanType)

	c.JSON(http.StatusAccepted, gin.H{
		"id":        scanID,
		"subnet":    req.Subnet,
		"scan_type": req.ScanType,
		"status":    "running",
		"message":   "Network scan started",
	})
}

func (h *Handler) executeNetworkScan(scanID int, subnet, scanType string) {
	ctx, cancel := context.WithTimeout(context.Background(), scanner.ScanTimeout(scanType))
	defer cancel()

	hosts, err := scanner.RunScan(ctx, subnet, scanType)
	if err != nil {
		msg := err.Error()
		h.db.Exec(`
			UPDATE network_scans SET status = 'failed', error_message = $1, finished_at = NOW()
			WHERE id = $2`, msg, scanID)
		return
	}

	secret := os.Getenv("RADIUS_SECRET")
	if secret == "" {
		secret = "testing123"
	}
	port := getRadiusPort()

	for _, host := range hosts {
		isRadius := host.IsRadius
		if !isRadius && scanType != "ping" {
			result := sendTestRADIUS(host.IPAddress, secret, port)
			if result.Success {
				isRadius = true
			}
		}

		portsJSON, _ := json.Marshal(host.OpenPorts)
		h.db.Exec(`
			INSERT INTO network_scan_hosts
			(scan_id, ip_address, hostname, mac_address, vendor, device_type, os_guess,
			 open_ports, is_access_point, is_radius_capable, latency_ms)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			scanID,
			host.IPAddress,
			nullStr(host.Hostname),
			nullStr(host.MACAddress),
			nullStr(host.Vendor),
			host.DeviceType,
			nullStr(host.OSGuess),
			portsJSON,
			host.IsAP,
			isRadius,
			host.LatencyMs,
		)
	}

	h.db.Exec(`
		UPDATE network_scans SET status = 'completed', host_count = $1, finished_at = NOW()
		WHERE id = $2`, len(hosts), scanID)
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ListNetworkScans returns scan history.
func (h *Handler) ListNetworkScans(c *gin.Context) {
	offset, limit := paginationParams(c)

	rows, err := h.db.Query(`
		SELECT id, subnet, scan_type, status, host_count, error_message,
		       started_by, started_at, finished_at
		FROM network_scans
		ORDER BY started_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch scans")
		return
	}
	defer rows.Close()

	scans := []models.NetworkScan{}
	for rows.Next() {
		var s models.NetworkScan
		rows.Scan(&s.ID, &s.Subnet, &s.ScanType, &s.Status, &s.HostCount,
			&s.ErrorMessage, &s.StartedBy, &s.StartedAt, &s.FinishedAt)
		scans = append(scans, s)
	}

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM network_scans`).Scan(&total)

	c.JSON(http.StatusOK, gin.H{"data": scans, "total": total})
}

// GetNetworkScan returns a scan with discovered hosts.
func (h *Handler) GetNetworkScan(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid scan ID")
		return
	}

	var s models.NetworkScan
	err = h.db.QueryRow(`
		SELECT id, subnet, scan_type, status, host_count, error_message,
		       started_by, started_at, finished_at
		FROM network_scans WHERE id = $1`, id).
		Scan(&s.ID, &s.Subnet, &s.ScanType, &s.Status, &s.HostCount,
			&s.ErrorMessage, &s.StartedBy, &s.StartedAt, &s.FinishedAt)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "scan not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database error")
		return
	}

	rows, err := h.db.Query(`
		SELECT id, scan_id, ip_address::text, COALESCE(hostname,''), COALESCE(mac_address,''),
		       COALESCE(vendor,''), device_type, COALESCE(os_guess,''), open_ports,
		       is_access_point, is_radius_capable, latency_ms, created_at
		FROM network_scan_hosts
		WHERE scan_id = $1
		ORDER BY ip_address`, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch hosts")
		return
	}
	defer rows.Close()

	hosts := []models.NetworkScanHost{}
	summary := gin.H{
		"total": 0, "access_points": 0, "routers": 0, "radius_capable": 0, "other": 0,
	}

	for rows.Next() {
		var host models.NetworkScanHost
		var portsJSON []byte
		rows.Scan(&host.ID, &host.ScanID, &host.IPAddress, &host.Hostname, &host.MACAddress,
			&host.Vendor, &host.DeviceType, &host.OSGuess, &portsJSON,
			&host.IsAccessPoint, &host.IsRadiusCapable, &host.LatencyMs, &host.CreatedAt)
		_ = json.Unmarshal(portsJSON, &host.OpenPorts)
		hosts = append(hosts, host)

		summary["total"] = summary["total"].(int) + 1
		if host.IsAccessPoint {
			summary["access_points"] = summary["access_points"].(int) + 1
		}
		if host.DeviceType == "router" {
			summary["routers"] = summary["routers"].(int) + 1
		}
		if host.IsRadiusCapable {
			summary["radius_capable"] = summary["radius_capable"].(int) + 1
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"scan":    s,
		"hosts":   hosts,
		"summary": summary,
	})
}

// DeleteNetworkScan removes a scan and its hosts.
func (h *Handler) DeleteNetworkScan(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid scan ID")
		return
	}
	h.db.Exec(`DELETE FROM network_scans WHERE id = $1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "scan deleted"})
}

// NetworkScannerStatus returns whether nmap is available.
func (h *Handler) NetworkScannerStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"nmap_available": scanner.NmapAvailable(),
		"max_subnet":     "/24",
		"scan_types": []gin.H{
			{"id": "ping", "label": "Ping Scan", "description": "Fast host discovery (ICMP/ARP)"},
			{"id": "discovery", "label": "Discovery", "description": "Host discovery with common port probes"},
			{"id": "standard", "label": "Standard", "description": "Top 100 TCP ports on live hosts"},
			{"id": "ap", "label": "AP & Network Gear", "description": "Scan for APs, routers, MikroTik, RADIUS ports"},
			{"id": "full", "label": "Full Network", "description": "Deep scan of network device ports"},
		},
	})
}

// ImportScanHostAsNAS creates a NAS entry from a discovered host.
func (h *Handler) ImportScanHostAsNAS(c *gin.Context) {
	hostID, err := mustInt(c, "hostId")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid host ID")
		return
	}

	var ip, hostname, deviceType string
	var isRadius bool
	err = h.db.QueryRow(`
		SELECT ip_address::text, COALESCE(hostname,''), device_type, is_radius_capable
		FROM network_scan_hosts WHERE id = $1`, hostID).
		Scan(&ip, &hostname, &deviceType, &isRadius)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "host not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database error")
		return
	}

	shortName := hostname
	if shortName == "" {
		shortName = "NAS-" + ip
	}
	if len(shortName) > 32 {
		shortName = shortName[:32]
	}

	nasType := "other"
	if deviceType == "access_point" {
		nasType = "wireless"
	} else if deviceType == "router" {
		nasType = "mikrotik"
	}

	secret := os.Getenv("RADIUS_SECRET")
	if secret == "" {
		secret = "testing123"
	}

	var nasID int
	err = h.db.QueryRow(`
		INSERT INTO nas (nasname, shortname, type, secret, description, status)
		VALUES ($1, $2, $3, $4, $5, 'active')
		ON CONFLICT (nasname) DO UPDATE SET shortname = EXCLUDED.shortname, updated_at = NOW()
		RETURNING id`,
		ip, shortName, nasType, secret,
		"Imported from network scan — RADIUS "+strconv.FormatBool(isRadius),
	).Scan(&nasID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create NAS device")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "NAS device created",
		"nas_id":  nasID,
		"nasname": ip,
	})
}
