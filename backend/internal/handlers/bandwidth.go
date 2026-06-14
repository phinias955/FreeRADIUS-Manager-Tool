package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// BandwidthProfile holds speed plan data.
type BandwidthProfile struct {
	ID                int        `json:"id"`
	Name              string     `json:"name"`
	Description       *string    `json:"description"`
	UploadKbps        int        `json:"upload_kbps"`
	DownloadKbps      int        `json:"download_kbps"`
	BurstUploadKbps   int        `json:"burst_upload_kbps"`
	BurstDownloadKbps int        `json:"burst_download_kbps"`
	MikrotikRateLimit *string    `json:"mikrotik_rate_limit"`
	IsActive          bool       `json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
}

// BandwidthProfileRequest is the create/update payload.
type BandwidthProfileRequest struct {
	Name              string `json:"name" binding:"required,min=2,max=100"`
	Description       string `json:"description"`
	UploadKbps        int    `json:"upload_kbps" binding:"required,min=1"`
	DownloadKbps      int    `json:"download_kbps" binding:"required,min=1"`
	BurstUploadKbps   int    `json:"burst_upload_kbps"`
	BurstDownloadKbps int    `json:"burst_download_kbps"`
	MikrotikRateLimit string `json:"mikrotik_rate_limit"`
	IsActive          *bool  `json:"is_active"`
}

func kbpsToMikrotik(upload, download int) string {
	unit := func(kbps int) string {
		if kbps >= 1000 && kbps%1000 == 0 {
			return fmt.Sprintf("%dM", kbps/1000)
		}
		return fmt.Sprintf("%dk", kbps)
	}
	return fmt.Sprintf("%s/%s", unit(upload), unit(download))
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ListBandwidthProfiles returns all bandwidth speed plans.
func (h *Handler) ListBandwidthProfiles(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, description, upload_kbps, download_kbps,
		       burst_upload_kbps, burst_download_kbps, mikrotik_rate_limit, is_active, created_at
		FROM bandwidth_profiles
		ORDER BY download_kbps ASC`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch profiles")
		return
	}
	defer rows.Close()

	profiles := []BandwidthProfile{}
	for rows.Next() {
		var p BandwidthProfile
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.UploadKbps, &p.DownloadKbps,
			&p.BurstUploadKbps, &p.BurstDownloadKbps, &p.MikrotikRateLimit, &p.IsActive, &p.CreatedAt)
		profiles = append(profiles, p)
	}
	c.JSON(http.StatusOK, gin.H{"data": profiles})
}

// CreateBandwidthProfile creates a new speed plan.
func (h *Handler) CreateBandwidthProfile(c *gin.Context) {
	var req BandwidthProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.MikrotikRateLimit == "" {
		req.MikrotikRateLimit = kbpsToMikrotik(req.UploadKbps, req.DownloadKbps)
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	var id int
	err := h.db.QueryRow(`
		INSERT INTO bandwidth_profiles (name, description, upload_kbps, download_kbps,
		    burst_upload_kbps, burst_download_kbps, mikrotik_rate_limit, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		req.Name, nullableString(req.Description), req.UploadKbps, req.DownloadKbps,
		req.BurstUploadKbps, req.BurstDownloadKbps, req.MikrotikRateLimit, isActive,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "profile name already exists")
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create profile")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "bandwidth profile created"})
}

// UpdateBandwidthProfile edits a speed plan and re-applies it to all assigned users.
func (h *Handler) UpdateBandwidthProfile(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req BandwidthProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.MikrotikRateLimit == "" {
		req.MikrotikRateLimit = kbpsToMikrotik(req.UploadKbps, req.DownloadKbps)
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	_, err = h.db.Exec(`
		UPDATE bandwidth_profiles
		SET name = $1, description = $2, upload_kbps = $3, download_kbps = $4,
		    burst_upload_kbps = $5, burst_download_kbps = $6,
		    mikrotik_rate_limit = $7, is_active = $8, updated_at = NOW()
		WHERE id = $9`,
		req.Name, nullableString(req.Description), req.UploadKbps, req.DownloadKbps,
		req.BurstUploadKbps, req.BurstDownloadKbps, req.MikrotikRateLimit, isActive, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update profile")
		return
	}
	go h.reapplyBandwidthProfile(id)
	c.JSON(http.StatusOK, gin.H{"message": "profile updated and will be re-applied to all users"})
}

// DeleteBandwidthProfile removes a speed plan (unlinking it from users first).
func (h *Handler) DeleteBandwidthProfile(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`UPDATE radius_users SET bandwidth_profile_id = NULL WHERE bandwidth_profile_id = $1`, id)
	h.db.Exec(`DELETE FROM bandwidth_profiles WHERE id = $1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "profile deleted"})
}

// ApplyBandwidthProfile assigns a speed plan to a RADIUS user and sets radreply attributes.
func (h *Handler) ApplyBandwidthProfile(c *gin.Context) {
	userID, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid user ID")
		return
	}
	var body struct {
		ProfileID *int `json:"profile_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	var username string
	if err := h.db.QueryRow(`SELECT username FROM radius_users WHERE id = $1`, userID).Scan(&username); err != nil {
		respondError(c, http.StatusNotFound, "user not found")
		return
	}

	// Clear existing rate-limit reply attributes
	h.db.Exec(`DELETE FROM radreply WHERE username = $1
		AND attribute IN ('Mikrotik-Rate-Limit','WISPr-Bandwidth-Max-Up','WISPr-Bandwidth-Max-Down')`, username)
	h.db.Exec(`UPDATE radius_users SET bandwidth_profile_id = $1, updated_at = NOW() WHERE id = $2`, body.ProfileID, userID)

	if body.ProfileID == nil {
		c.JSON(http.StatusOK, gin.H{"message": "bandwidth profile removed from user"})
		return
	}

	var p BandwidthProfile
	err = h.db.QueryRow(`
		SELECT id, name, upload_kbps, download_kbps, mikrotik_rate_limit
		FROM bandwidth_profiles WHERE id = $1`, *body.ProfileID).
		Scan(&p.ID, &p.Name, &p.UploadKbps, &p.DownloadKbps, &p.MikrotikRateLimit)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "bandwidth profile not found")
		return
	}

	rateLimit := kbpsToMikrotik(p.UploadKbps, p.DownloadKbps)
	if p.MikrotikRateLimit != nil && *p.MikrotikRateLimit != "" {
		rateLimit = *p.MikrotikRateLimit
	}

	h.db.Exec(`INSERT INTO radreply (username, attribute, op, value) VALUES ($1, 'Mikrotik-Rate-Limit', '=', $2)`, username, rateLimit)
	h.db.Exec(`INSERT INTO radreply (username, attribute, op, value) VALUES ($1, 'WISPr-Bandwidth-Max-Up', '=', $2)`, username, fmt.Sprintf("%d", p.UploadKbps*1000))
	h.db.Exec(`INSERT INTO radreply (username, attribute, op, value) VALUES ($1, 'WISPr-Bandwidth-Max-Down', '=', $2)`, username, fmt.Sprintf("%d", p.DownloadKbps*1000))

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("applied '%s' to user %s", p.Name, username)})
}

// reapplyBandwidthProfile re-syncs radreply for every user assigned to a profile.
func (h *Handler) reapplyBandwidthProfile(profileID int) {
	var p BandwidthProfile
	if err := h.db.QueryRow(`
		SELECT id, upload_kbps, download_kbps, mikrotik_rate_limit
		FROM bandwidth_profiles WHERE id = $1`, profileID).
		Scan(&p.ID, &p.UploadKbps, &p.DownloadKbps, &p.MikrotikRateLimit); err != nil {
		return
	}

	rows, err := h.db.Query(`SELECT username FROM radius_users WHERE bandwidth_profile_id = $1`, profileID)
	if err != nil {
		return
	}
	defer rows.Close()

	rateLimit := kbpsToMikrotik(p.UploadKbps, p.DownloadKbps)
	if p.MikrotikRateLimit != nil && *p.MikrotikRateLimit != "" {
		rateLimit = *p.MikrotikRateLimit
	}

	for rows.Next() {
		var username string
		rows.Scan(&username)
		h.db.Exec(`DELETE FROM radreply WHERE username = $1 AND attribute IN ('Mikrotik-Rate-Limit','WISPr-Bandwidth-Max-Up','WISPr-Bandwidth-Max-Down')`, username)
		h.db.Exec(`INSERT INTO radreply (username, attribute, op, value) VALUES ($1, 'Mikrotik-Rate-Limit', '=', $2)`, username, rateLimit)
		h.db.Exec(`INSERT INTO radreply (username, attribute, op, value) VALUES ($1, 'WISPr-Bandwidth-Max-Up', '=', $2)`, username, fmt.Sprintf("%d", p.UploadKbps*1000))
		h.db.Exec(`INSERT INTO radreply (username, attribute, op, value) VALUES ($1, 'WISPr-Bandwidth-Max-Down', '=', $2)`, username, fmt.Sprintf("%d", p.DownloadKbps*1000))
	}
}
