package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HotspotZone represents a physical or logical zone with NAS devices.
type HotspotZone struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Location    *string   `json:"location"`
	MaxClients  int       `json:"max_clients"`
	IsActive    bool      `json:"is_active"`
	NASCount    int       `json:"nas_count"`
	ActiveUsers int       `json:"active_users"`
	CreatedAt   time.Time `json:"created_at"`
}

// ListZones returns all hotspot zones with live stats.
func (h *Handler) ListZones(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT z.id, z.name, z.description, z.location, z.max_clients, z.is_active, z.created_at,
		       COUNT(DISTINCT n.id) AS nas_count,
		       COUNT(DISTINCT ra.username) AS active_users
		FROM hotspot_zones z
		LEFT JOIN nas n ON n.zone_id = z.id AND n.status='active'
		LEFT JOIN radacct ra ON ra.nasipaddress::text = n.nasname AND ra.acctstoptime IS NULL
		GROUP BY z.id
		ORDER BY z.is_active DESC, z.name`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch zones")
		return
	}
	defer rows.Close()

	zones := []HotspotZone{}
	for rows.Next() {
		var z HotspotZone
		rows.Scan(&z.ID, &z.Name, &z.Description, &z.Location, &z.MaxClients,
			&z.IsActive, &z.CreatedAt, &z.NASCount, &z.ActiveUsers)
		zones = append(zones, z)
	}
	c.JSON(http.StatusOK, gin.H{"data": zones})
}

// CreateZone adds a new hotspot zone.
func (h *Handler) CreateZone(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required,min=2,max=100"`
		Description string `json:"description"`
		Location    string `json:"location"`
		MaxClients  int    `json:"max_clients"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	var id int
	err := h.db.QueryRow(`
		INSERT INTO hotspot_zones (name,description,location,max_clients,is_active)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		req.Name, nullableString(req.Description), nullableString(req.Location),
		req.MaxClients, isActive).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "zone name already exists")
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create zone")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "zone created"})
}

// UpdateZone updates zone details.
func (h *Handler) UpdateZone(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Location    string `json:"location"`
		MaxClients  int    `json:"max_clients"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	h.db.Exec(`UPDATE hotspot_zones SET
		name=COALESCE(NULLIF($1,''),name),
		description=COALESCE(NULLIF($2,''),description),
		location=COALESCE(NULLIF($3,''),location),
		max_clients=CASE WHEN $4>0 THEN $4 ELSE max_clients END,
		is_active=$5
		WHERE id=$6`, req.Name, req.Description, req.Location, req.MaxClients, isActive, id)
	c.JSON(http.StatusOK, gin.H{"message": "zone updated"})
}

// DeleteZone removes a zone (unlinking its NAS devices).
func (h *Handler) DeleteZone(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`UPDATE nas SET zone_id=NULL WHERE zone_id=$1`, id)
	h.db.Exec(`DELETE FROM hotspot_zones WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "zone deleted"})
}

// AssignNASToZone assigns a NAS device to a zone.
func (h *Handler) AssignNASToZone(c *gin.Context) {
	var req struct {
		NASID  int  `json:"nas_id" binding:"required"`
		ZoneID *int `json:"zone_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	h.db.Exec(`UPDATE nas SET zone_id=$1 WHERE id=$2`, req.ZoneID, req.NASID)
	c.JSON(http.StatusOK, gin.H{"message": "NAS zone updated"})
}

// ZoneStats returns detailed statistics for a specific zone.
func (h *Handler) ZoneStats(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	type NASInZone struct {
		ID          int     `json:"id"`
		NASName     string  `json:"nasname"`
		ShortName   string  `json:"shortname"`
		PingStatus  string  `json:"ping_status"`
		LatencyMs   float64 `json:"ping_latency_ms"`
		ActiveUsers int     `json:"active_users"`
	}
	devices := []NASInZone{}
	rows, _ := h.db.Query(`
		SELECT n.id, n.nasname, COALESCE(n.shortname,''),
		       COALESCE(n.ping_status,'unknown'), COALESCE(n.ping_latency_ms,0),
		       COUNT(ra.username) AS active_users
		FROM nas n
		LEFT JOIN radacct ra ON ra.nasipaddress::text=n.nasname AND ra.acctstoptime IS NULL
		WHERE n.zone_id=$1
		GROUP BY n.id
		ORDER BY n.shortname`, id)
	if rows != nil {
		for rows.Next() {
			var d NASInZone
			rows.Scan(&d.ID, &d.NASName, &d.ShortName, &d.PingStatus, &d.LatencyMs, &d.ActiveUsers)
			devices = append(devices, d)
		}
		rows.Close()
	}

	// Traffic stats for this zone (last 7 days)
	type DayStat struct {
		Day       string  `json:"day"`
		TotalMB   float64 `json:"total_mb"`
		Sessions  int     `json:"sessions"`
	}
	dayStats := []DayStat{}
	drows, _ := h.db.Query(`
		SELECT DATE(ra.acctstarttime) AS day,
		       SUM(ra.acctinputoctets+ra.acctoutputoctets)/1048576.0,
		       COUNT(*)
		FROM radacct ra
		JOIN nas n ON n.nasname=ra.nasipaddress::text
		WHERE n.zone_id=$1 AND ra.acctstarttime >= NOW()-INTERVAL '7 days'
		GROUP BY day ORDER BY day`, id)
	if drows != nil {
		for drows.Next() {
			var ds DayStat
			drows.Scan(&ds.Day, &ds.TotalMB, &ds.Sessions)
			dayStats = append(dayStats, ds)
		}
		drows.Close()
	}

	c.JSON(http.StatusOK, gin.H{"devices": devices, "daily_stats": dayStats})
}
