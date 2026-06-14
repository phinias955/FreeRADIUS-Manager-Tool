package handlers

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	strs "strings"
	"time"

	"github.com/freeradius-manager/backend/internal/models"
	"github.com/gin-gonic/gin"
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

// TestNAS sends a test RADIUS Access-Request to the stored NAS device.
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

	result := sendTestRADIUS(nasName, secret, getRadiusPort())
	c.JSON(http.StatusOK, result)
}

// TestRADIUS tests authentication of a given username/password against FreeRADIUS.
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

	packet := buildAccessRequest(req.Username, req.Password, secret)
	response, err := sendRADIUSPacket(c.Request.Context(), addr, packet, 5*time.Second)
	latency := time.Since(start)

	if err != nil {
		c.JSON(http.StatusOK, models.NASTestResult{
			Success:    false,
			Message:    fmt.Sprintf("connection failed: %v", err),
			LatencyMs:  float64(latency.Milliseconds()),
			NASName:    nasName,
			RadiusPort: port,
		})
		return
	}

	// Code 2 = Access-Accept, Code 3 = Access-Reject
	success := len(response) > 0 && response[0] == 2
	msg := "Access-Accept"
	if !success {
		if len(response) > 0 {
			msg = fmt.Sprintf("Access-Reject (code %d)", response[0])
		} else {
			msg = "No response"
		}
	}

	c.JSON(http.StatusOK, models.NASTestResult{
		Success:    success,
		Message:    msg,
		LatencyMs:  float64(latency.Milliseconds()),
		AuthTime:   float64(latency.Milliseconds()),
		NASName:    nasName,
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

	port := getRadiusPort()
	discovered := []gin.H{}
	count := 0

	for ip := cloneIP(ipNet.IP.Mask(ipNet.Mask)); ipNet.Contains(ip); incrementIP(ip) {
		if count >= 254 {
			break
		}
		count++
		ipStr := ip.String()
		result := sendTestRADIUS(ipStr, secret, port)
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

// ─── RADIUS packet helpers (pure Go, no external library) ────────────────────

// buildAccessRequest constructs a minimal RADIUS Access-Request packet.
func buildAccessRequest(username, password, secret string) []byte {
	// Random authenticator (16 bytes)
	authenticator := make([]byte, 16)
	rand.Read(authenticator)

	// Random identifier
	idByte := make([]byte, 1)
	rand.Read(idByte)
	id := idByte[0]

	// Encode User-Password: RFC 2865 §5.2
	encPw := encodePassword(password, secret, authenticator)

	// Build attributes
	attrs := []byte{}

	// User-Name attribute (type=1)
	attrs = appendAttr(attrs, 1, []byte(username))
	// User-Password attribute (type=2)
	attrs = appendAttr(attrs, 2, encPw)
	// NAS-IP-Address attribute (type=4) — use 127.0.0.1
	nasIP := net.ParseIP("127.0.0.1").To4()
	attrs = appendAttr(attrs, 4, nasIP)

	// Total length = 20 (header) + len(attrs)
	totalLen := uint16(20 + len(attrs))
	packet := make([]byte, 20)
	packet[0] = 1 // Code: Access-Request
	packet[1] = id
	binary.BigEndian.PutUint16(packet[2:4], totalLen)
	copy(packet[4:20], authenticator)
	packet = append(packet, attrs...)

	return packet
}

// encodePassword encodes the User-Password per RFC 2865 §5.2.
func encodePassword(password, secret string, authenticator []byte) []byte {
	pw := []byte(password)
	// Pad to multiple of 16 bytes
	for len(pw)%16 != 0 {
		pw = append(pw, 0)
	}
	if len(pw) == 0 {
		pw = make([]byte, 16)
	}

	result := make([]byte, len(pw))
	prev := authenticator
	for i := 0; i < len(pw); i += 16 {
		h := md5.New()
		h.Write([]byte(secret))
		h.Write(prev)
		b := h.Sum(nil)
		for j := 0; j < 16; j++ {
			result[i+j] = pw[i+j] ^ b[j]
		}
		prev = result[i : i+16]
	}
	return result
}

// appendAttr appends a RADIUS TLV attribute (type, length, value).
func appendAttr(buf []byte, attrType byte, value []byte) []byte {
	buf = append(buf, attrType)
	buf = append(buf, byte(2+len(value)))
	buf = append(buf, value...)
	return buf
}

// sendRADIUSPacket sends a RADIUS packet over UDP and waits for a response.
func sendRADIUSPacket(ctx context.Context, addr string, packet []byte, timeout time.Duration) ([]byte, error) {
	conn, err := net.DialTimeout("udp", addr, timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Set deadline from context or timeout
	deadline := time.Now().Add(timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	conn.SetDeadline(deadline)

	if _, err := conn.Write(packet); err != nil {
		return nil, err
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

// sendTestRADIUS sends a probe packet and returns a connectivity result.
// A response (even Access-Reject) means the server is reachable.
func sendTestRADIUS(nasIP, secret string, port int) models.NASTestResult {
	addr := fmt.Sprintf("%s:%d", nasIP, port)
	start := time.Now()

	packet := buildAccessRequest("connectivity-probe", "probe-will-reject", secret)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	response, err := sendRADIUSPacket(ctx, addr, packet, 2*time.Second)
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

	code := "unknown"
	if len(response) > 0 {
		code = strconv.Itoa(int(response[0]))
	}

	return models.NASTestResult{
		Success:    true,
		Message:    fmt.Sprintf("RADIUS server responding (code %s)", code),
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
