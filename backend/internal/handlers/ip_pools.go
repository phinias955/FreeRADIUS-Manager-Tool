package handlers

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// IPPool represents a managed IP address pool.
type IPPool struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Network     string    `json:"network"`
	Gateway     *string   `json:"gateway"`
	DNS1        *string   `json:"dns1"`
	DNS2        *string   `json:"dns2"`
	IsActive    bool      `json:"is_active"`
	TotalIPs    int       `json:"total_ips"`
	UsedIPs     int       `json:"used_ips"`
	FreeIPs     int       `json:"free_ips"`
	CreatedAt   time.Time `json:"created_at"`
}

// IPAssignment is a single allocated IP in a pool.
type IPAssignment struct {
	ID        int        `json:"id"`
	PoolID    int        `json:"pool_id"`
	PoolName  string     `json:"pool_name"`
	IPAddress string     `json:"ip_address"`
	Username  *string    `json:"username"`
	UserID    *int       `json:"user_id"`
	IsStatic  bool       `json:"is_static"`
	LeasedAt  *time.Time `json:"leased_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// ListIPPools returns all IP pools with usage stats.
func (h *Handler) ListIPPools(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT p.id, p.name, p.description, p.network, p.gateway, p.dns1, p.dns2, p.is_active, p.created_at,
		       COUNT(a.id) AS total,
		       COUNT(a.id) FILTER (WHERE a.username IS NOT NULL) AS used
		FROM ip_pools p
		LEFT JOIN ip_pool_assignments a ON a.pool_id = p.id
		GROUP BY p.id
		ORDER BY p.created_at DESC`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch pools")
		return
	}
	defer rows.Close()

	pools := []IPPool{}
	for rows.Next() {
		var p IPPool
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.Network, &p.Gateway, &p.DNS1, &p.DNS2, &p.IsActive, &p.CreatedAt, &p.TotalIPs, &p.UsedIPs)
		p.FreeIPs = p.TotalIPs - p.UsedIPs
		pools = append(pools, p)
	}
	c.JSON(http.StatusOK, gin.H{"data": pools})
}

// CreateIPPool creates a new pool and generates all IPs in the CIDR range.
func (h *Handler) CreateIPPool(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required,min=2,max=100"`
		Description string `json:"description"`
		Network     string `json:"network" binding:"required"`
		Gateway     string `json:"gateway"`
		DNS1        string `json:"dns1"`
		DNS2        string `json:"dns2"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.DNS1 == "" {
		req.DNS1 = "8.8.8.8"
	}
	if req.DNS2 == "" {
		req.DNS2 = "8.8.4.4"
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// Validate CIDR
	_, ipNet, err := net.ParseCIDR(req.Network)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid network CIDR (e.g. 10.10.0.0/24)")
		return
	}

	// Auto-derive gateway if not provided
	if req.Gateway == "" {
		req.Gateway = firstUsableIP(ipNet)
	}

	var poolID int
	err = h.db.QueryRow(`
		INSERT INTO ip_pools (name,description,network,gateway,dns1,dns2,is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		req.Name, nullableString(req.Description), req.Network,
		nullableString(req.Gateway), nullableString(req.DNS1), nullableString(req.DNS2), isActive,
	).Scan(&poolID)
	if err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "pool name already exists")
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create pool")
		return
	}

	// Populate IPs (skip network addr, broadcast, and gateway)
	ips := enumerateIPs(ipNet)
	tx, _ := h.db.Begin()
	count := 0
	for _, ip := range ips {
		if ip == req.Gateway {
			continue // skip gateway
		}
		tx.Exec(`INSERT INTO ip_pool_assignments (pool_id,ip_address) VALUES ($1,$2)
			ON CONFLICT (ip_address) DO NOTHING`, poolID, ip)
		count++
	}
	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"id":       poolID,
		"message":  fmt.Sprintf("pool created with %d addresses", count),
		"ip_count": count,
	})
}

// DeleteIPPool removes a pool and all its assignments.
func (h *Handler) DeleteIPPool(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	// Release RADIUS assignments first
	rows, _ := h.db.Query(`SELECT username FROM ip_pool_assignments WHERE pool_id=$1 AND username IS NOT NULL`, id)
	if rows != nil {
		for rows.Next() {
			var u string
			rows.Scan(&u)
			h.db.Exec(`DELETE FROM radreply WHERE username=$1 AND attribute='Framed-IP-Address'`, u)
		}
		rows.Close()
	}
	h.db.Exec(`DELETE FROM ip_pools WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "pool deleted"})
}

// ListPoolIPs returns all IP assignments in a specific pool.
func (h *Handler) ListPoolIPs(c *gin.Context) {
	poolID, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid pool ID")
		return
	}
	rows, err := h.db.Query(`
		SELECT a.id, a.pool_id, p.name, a.ip_address, a.username, a.user_id, a.is_static, a.leased_at, a.created_at
		FROM ip_pool_assignments a
		JOIN ip_pools p ON p.id = a.pool_id
		WHERE a.pool_id=$1
		ORDER BY inet(a.ip_address)`, poolID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch IPs")
		return
	}
	defer rows.Close()

	list := []IPAssignment{}
	for rows.Next() {
		var a IPAssignment
		rows.Scan(&a.ID, &a.PoolID, &a.PoolName, &a.IPAddress, &a.Username, &a.UserID, &a.IsStatic, &a.LeasedAt, &a.CreatedAt)
		list = append(list, a)
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// AssignIP assigns a specific IP to a user (or auto-picks a free one).
func (h *Handler) AssignIP(c *gin.Context) {
	var req struct {
		Username  string  `json:"username" binding:"required"`
		PoolID    int     `json:"pool_id" binding:"required"`
		IPAddress *string `json:"ip_address"` // optional: auto-assign if not provided
		IsStatic  bool    `json:"is_static"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Get user ID
	var userID int
	h.db.QueryRow(`SELECT id FROM radius_users WHERE username=$1`, req.Username).Scan(&userID)

	var ip string
	if req.IPAddress != nil && *req.IPAddress != "" {
		ip = *req.IPAddress
		// Verify it belongs to the pool
		var exists bool
		h.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM ip_pool_assignments WHERE pool_id=$1 AND ip_address=$2)`,
			req.PoolID, ip).Scan(&exists)
		if !exists {
			respondError(c, http.StatusBadRequest, "IP does not belong to this pool")
			return
		}
	} else {
		// Auto-pick first free IP
		err := h.db.QueryRow(`SELECT ip_address FROM ip_pool_assignments
			WHERE pool_id=$1 AND username IS NULL ORDER BY inet(ip_address) LIMIT 1`, req.PoolID).Scan(&ip)
		if err != nil {
			respondError(c, http.StatusConflict, "no free IPs available in pool")
			return
		}
	}

	// Release old assignment if user had one
	var oldIP string
	if err := h.db.QueryRow(`SELECT ip_address FROM ip_pool_assignments WHERE username=$1`, req.Username).Scan(&oldIP); err == nil {
		h.db.Exec(`UPDATE ip_pool_assignments SET username=NULL, user_id=NULL, leased_at=NULL WHERE ip_address=$1`, oldIP)
		h.db.Exec(`DELETE FROM radreply WHERE username=$1 AND attribute='Framed-IP-Address'`, req.Username)
	}

	// Assign
	h.db.Exec(`UPDATE ip_pool_assignments
		SET username=$1, user_id=$2, is_static=$3, leased_at=NOW()
		WHERE ip_address=$4`, req.Username, userID, req.IsStatic, ip)

	// Set RADIUS Framed-IP-Address
	h.db.Exec(`INSERT INTO radreply (username,attribute,op,value) VALUES ($1,'Framed-IP-Address','=',$2)
		ON CONFLICT (username,attribute) DO UPDATE SET value=$2`, req.Username, ip)

	c.JSON(http.StatusOK, gin.H{
		"message":    fmt.Sprintf("IP %s assigned to %s", ip, req.Username),
		"ip_address": ip,
		"username":   req.Username,
	})
}

// ReleaseIP releases an IP assignment from a user.
func (h *Handler) ReleaseIP(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	h.db.Exec(`UPDATE ip_pool_assignments SET username=NULL, user_id=NULL, leased_at=NULL WHERE username=$1`, req.Username)
	h.db.Exec(`DELETE FROM radreply WHERE username=$1 AND attribute='Framed-IP-Address'`, req.Username)
	c.JSON(http.StatusOK, gin.H{"message": "IP released"})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func enumerateIPs(ipNet *net.IPNet) []string {
	var ips []string
	// Convert to 32-bit uint for iteration
	start := ip2uint(ipNet.IP.To4()) + 1     // skip network address
	end := ip2uint(broadcastIP(ipNet)) - 1   // skip broadcast
	if end < start {
		return ips
	}
	// Cap at 1022 IPs to avoid huge pools
	if end-start > 1022 {
		end = start + 1022
	}
	for i := start; i <= end; i++ {
		ips = append(ips, uint2ip(i))
	}
	return ips
}

func firstUsableIP(ipNet *net.IPNet) string {
	start := ip2uint(ipNet.IP.To4()) + 1
	return uint2ip(start)
}

func broadcastIP(ipNet *net.IPNet) net.IP {
	ip := ipNet.IP.To4()
	mask := ipNet.Mask
	bcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		bcast[i] = ip[i] | ^mask[i]
	}
	return bcast
}

func ip2uint(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip.To4())
}

func uint2ip(n uint32) string {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, n)
	return net.IP(b).String()
}
