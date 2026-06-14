package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	strs "strings"
	"time"

	"github.com/freeradius-manager/backend/internal/models"
	"github.com/gin-gonic/gin"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

// ListNAS returns all NAS clients.
func (h *Handler) ListNAS(c *gin.Context) {
	offset, limit := paginationParams(c)

	rows, err := h.db.Query(`
		SELECT id, nasname, shortname, type, ports, secret, server, community, description, status, created_at, updated_at
		FROM nas
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch NAS devices")
		return
	}
	defer rows.Close()

	list := []models.NAS{}
	for rows.Next() {
		var n models.NAS
		rows.Scan(&n.ID, &n.NASName, &n.ShortName, &n.Type, &n.Ports, &n.Secret,
			&n.Server, &n.Community, &n.Description, &n.Status, &n.CreatedAt, &n.UpdatedAt)
		n.Secret = "***"
		list = append(list, n)
	}

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM nas`).Scan(&total)

	c.JSON(http.StatusOK, gin.H{"data": list, "total": total})
}

// GetNAS returns a single NAS client by ID.
func (h *Handler) GetNAS(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var n models.NAS
	err = h.db.QueryRow(`
		SELECT id, nasname, shortname, type, ports, secret, server, community, description, status, created_at, updated_at
		FROM nas WHERE id = $1`, id,
	).Scan(&n.ID, &n.NASName, &n.ShortName, &n.Type, &n.Ports, &n.Secret,
		&n.Server, &n.Community, &n.Description, &n.Status, &n.CreatedAt, &n.UpdatedAt)

	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "NAS not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database error")
		return
	}

	c.JSON(http.StatusOK, n)
}

// CreateNAS adds a new NAS client.
func (h *Handler) CreateNAS(c *gin.Context) {
	var req models.CreateNASRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Type == "" {
		req.Type = "other"
	}

	var newID int
	err := h.db.QueryRow(`
		INSERT INTO nas (nasname, shortname, type, ports, secret, server, community, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		req.NASName, req.ShortName, req.Type, req.Ports, req.Secret,
		req.Server, req.Community, req.Description,
	).Scan(&newID)

	if err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "NAS with this name/IP already exists")
			return
		}
		h.log.WithError(err).Error("create NAS failed")
		respondError(c, http.StatusInternalServerError, "failed to create NAS")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": newID, "message": "NAS device added successfully"})
}

// UpdateNAS updates an existing NAS client.
func (h *Handler) UpdateNAS(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var req models.UpdateNASRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	_, err = h.db.Exec(`
		UPDATE nas
		SET shortname   = COALESCE(NULLIF($1, ''), shortname),
		    type        = COALESCE(NULLIF($2, ''), type),
		    ports       = COALESCE($3, ports),
		    secret      = COALESCE(NULLIF($4, ''), secret),
		    server      = COALESCE(NULLIF($5, ''), server),
		    community   = COALESCE(NULLIF($6, ''), community),
		    description = COALESCE(NULLIF($7, ''), description),
		    status      = COALESCE(NULLIF($8, ''), status),
		    updated_at  = NOW()
		WHERE id = $9`,
		req.ShortName, req.Type, req.Ports, req.Secret, req.Server,
		req.Community, req.Description, req.Status, id,
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update NAS")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "NAS updated successfully"})
}

// DeleteNAS removes a NAS client.
func (h *Handler) DeleteNAS(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	result, err := h.db.Exec(`DELETE FROM nas WHERE id = $1`, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to delete NAS")
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		respondError(c, http.StatusNotFound, "NAS not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "NAS deleted successfully"})
}

// TestNAS sends a test RADIUS packet to verify the NAS is reachable.
func (h *Handler) TestNAS(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var nasName, secret string
	if err := h.db.QueryRow(`SELECT nasname, secret FROM nas WHERE id = $1`, id).Scan(&nasName, &secret); err != nil {
		if err == sql.ErrNoRows {
			respondError(c, http.StatusNotFound, "NAS not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "database error")
		return
	}

	result := sendTestRadius(c.Request.Context(), nasName, secret)
	c.JSON(http.StatusOK, result)
}

// TestRADIUS tests authentication against the FreeRADIUS server.
func (h *Handler) TestRADIUS(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		NASName  string `json:"nasname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	nasName := req.NASName
	if nasName == "" {
		nasName = os.Getenv("RADIUS_HOST")
		if nasName == "" {
			nasName = "127.0.0.1"
		}
	}

	secret := os.Getenv("RADIUS_SECRET")
	if secret == "" {
		secret = "testing123"
	}

	port := getRadiusPort()
	start := time.Now()
	addr := fmt.Sprintf("%s:%d", nasName, port)

	packet := radius.New(radius.CodeAccessRequest, []byte(secret))
	rfc2865.UserName_SetString(packet, req.Username)
	rfc2865.UserPassword_SetString(packet, req.Password)
	rfc2865.NASIPAddress_Set(packet, net.ParseIP("127.0.0.1"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response, err := radius.Exchange(ctx, packet, addr)
	latency := time.Since(start)

	if err != nil {
		c.JSON(http.StatusOK, models.NASTestResult{
			Success:   false,
			Message:   fmt.Sprintf("connection failed: %v", err),
			LatencyMs: float64(latency.Milliseconds()),
			NASName:   nasName,
			RadiusPort: port,
		})
		return
	}

	success := response.Code == radius.CodeAccessAccept
	msg := "Access-Accept"
	if !success {
		msg = fmt.Sprintf("Access-Reject (code %d)", response.Code)
	}

	c.JSON(http.StatusOK, models.NASTestResult{
		Success:   success,
		Message:   msg,
		LatencyMs: float64(latency.Milliseconds()),
		AuthTime:  float64(latency.Milliseconds()),
		NASName:   nasName,
		RadiusPort: port,
	})
}

// DiscoverNAS scans a subnet for RADIUS-capable devices.
func (h *Handler) DiscoverNAS(c *gin.Context) {
	var req models.DiscoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	_, ipNet, err := net.ParseCIDR(req.Subnet)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid subnet CIDR")
		return
	}

	secret := os.Getenv("RADIUS_SECRET")
	if secret == "" {
		secret = "testing123"
	}

	discovered := []gin.H{}
	count := 0
	for ip := cloneIP(ipNet.IP.Mask(ipNet.Mask)); ipNet.Contains(ip); incrementIP(ip) {
		if count >= 254 {
			break
		}
		count++
		ipStr := ip.String()

		result := sendTestRadius(c.Request.Context(), ipStr, secret)
		if result.Success {
			discovered = append(discovered, gin.H{
				"ip":        ipStr,
				"latency":   result.LatencyMs,
				"suggested": fmt.Sprintf("NAS-%s", strs.ReplaceAll(ipStr, ".", "-")),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"subnet":     req.Subnet,
		"discovered": discovered,
		"count":      len(discovered),
	})
}

func sendTestRadius(ctx context.Context, nasIP, secret string) models.NASTestResult {
	port := getRadiusPort()
	addr := fmt.Sprintf("%s:%d", nasIP, port)
	start := time.Now()

	packet := radius.New(radius.CodeAccessRequest, []byte(secret))
	rfc2865.UserName_SetString(packet, "connectivity-probe")
	rfc2865.UserPassword_SetString(packet, "probe-will-reject")
	rfc2865.NASIPAddress_Set(packet, net.ParseIP("127.0.0.1"))

	tctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	response, err := radius.Exchange(tctx, packet, addr)
	latency := time.Since(start)

	if err != nil {
		return models.NASTestResult{
			Success:    false,
			Message:    "unreachable: " + err.Error(),
			LatencyMs:  float64(latency.Milliseconds()),
			NASName:    nasIP,
			RadiusPort: port,
		}
	}

	return models.NASTestResult{
		Success:    true,
		Message:    fmt.Sprintf("RADIUS server responding (code %d)", response.Code),
		LatencyMs:  float64(latency.Milliseconds()),
		NASName:    nasIP,
		RadiusPort: port,
	}
}

func cloneIP(ip net.IP) net.IP {
	c := make(net.IP, len(ip))
	copy(c, ip)
	return c
}

func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

func getRadiusPort() int {
	port := os.Getenv("RADIUS_AUTH_PORT")
	if port == "" {
		return 1812
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return 1812
	}
	return p
}
