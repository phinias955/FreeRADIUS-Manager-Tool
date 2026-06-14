package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// UserPlan holds a subscription plan definition.
type UserPlan struct {
	ID                 int        `json:"id"`
	Name               string     `json:"name"`
	Description        *string    `json:"description"`
	Price              float64    `json:"price"`
	Currency           string     `json:"currency"`
	DataLimitMB        *int64     `json:"data_limit_mb"`
	ValidityDays       int        `json:"validity_days"`
	BandwidthProfileID *int       `json:"bandwidth_profile_id"`
	BandwidthName      *string    `json:"bandwidth_name"`
	MaxDevices         int        `json:"max_devices"`
	IsActive           bool       `json:"is_active"`
	UserCount          int        `json:"user_count"`
	CreatedAt          time.Time  `json:"created_at"`
}

// ListPlans returns all user plans.
func (h *Handler) ListPlans(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT up.id, up.name, up.description, up.price, up.currency,
		       up.data_limit_mb, up.validity_days, up.bandwidth_profile_id,
		       bp.name, up.max_devices, up.is_active, up.created_at,
		       COUNT(ru.id) as user_count
		FROM user_plans up
		LEFT JOIN bandwidth_profiles bp ON bp.id = up.bandwidth_profile_id
		LEFT JOIN radius_users ru ON ru.plan_id = up.id
		GROUP BY up.id, bp.name
		ORDER BY up.price ASC`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch plans")
		return
	}
	defer rows.Close()

	plans := []UserPlan{}
	for rows.Next() {
		var p UserPlan
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Currency,
			&p.DataLimitMB, &p.ValidityDays, &p.BandwidthProfileID,
			&p.BandwidthName, &p.MaxDevices, &p.IsActive, &p.CreatedAt, &p.UserCount)
		plans = append(plans, p)
	}
	c.JSON(http.StatusOK, gin.H{"data": plans})
}

// CreatePlan creates a new user plan.
func (h *Handler) CreatePlan(c *gin.Context) {
	var req struct {
		Name               string  `json:"name" binding:"required,min=2,max=100"`
		Description        string  `json:"description"`
		Price              float64 `json:"price"`
		Currency           string  `json:"currency"`
		DataLimitMB        *int64  `json:"data_limit_mb"`
		ValidityDays       int     `json:"validity_days"`
		BandwidthProfileID *int    `json:"bandwidth_profile_id"`
		MaxDevices         int     `json:"max_devices"`
		IsActive           *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.ValidityDays == 0 {
		req.ValidityDays = 30
	}
	if req.MaxDevices == 0 {
		req.MaxDevices = 1
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	var id int
	err := h.db.QueryRow(`
		INSERT INTO user_plans (name,description,price,currency,data_limit_mb,validity_days,
		    bandwidth_profile_id,max_devices,is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		req.Name, nullableString(req.Description), req.Price, req.Currency,
		req.DataLimitMB, req.ValidityDays, req.BandwidthProfileID, req.MaxDevices, isActive,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "plan name already exists")
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create plan")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "plan created"})
}

// UpdatePlan updates a plan's details.
func (h *Handler) UpdatePlan(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Name               string  `json:"name"`
		Description        string  `json:"description"`
		Price              float64 `json:"price"`
		Currency           string  `json:"currency"`
		DataLimitMB        *int64  `json:"data_limit_mb"`
		ValidityDays       int     `json:"validity_days"`
		BandwidthProfileID *int    `json:"bandwidth_profile_id"`
		MaxDevices         int     `json:"max_devices"`
		IsActive           *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	_, err = h.db.Exec(`
		UPDATE user_plans SET name=$1, description=$2, price=$3, currency=$4,
		    data_limit_mb=$5, validity_days=$6, bandwidth_profile_id=$7,
		    max_devices=$8, is_active=$9, updated_at=NOW()
		WHERE id=$10`,
		req.Name, nullableString(req.Description), req.Price, req.Currency,
		req.DataLimitMB, req.ValidityDays, req.BandwidthProfileID,
		req.MaxDevices, isActive, id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to update plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "plan updated"})
}

// DeletePlan removes a plan (unlinking users first).
func (h *Handler) DeletePlan(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`UPDATE radius_users SET plan_id=NULL WHERE plan_id=$1`, id)
	h.db.Exec(`UPDATE invoices SET plan_id=NULL WHERE plan_id=$1`, id)
	h.db.Exec(`DELETE FROM user_plans WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "plan deleted"})
}

// AssignPlan assigns a plan to a RADIUS user and applies plan attributes.
func (h *Handler) AssignPlan(c *gin.Context) {
	userID, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid user ID")
		return
	}
	var body struct {
		PlanID *int `json:"plan_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var username string
	if err := h.db.QueryRow(`SELECT username FROM radius_users WHERE id=$1`, userID).Scan(&username); err != nil {
		respondError(c, http.StatusNotFound, "user not found")
		return
	}

	claims, _ := middleware.GetClaims(c)
	h.db.Exec(`UPDATE radius_users SET plan_id=$1, updated_at=NOW() WHERE id=$2`, body.PlanID, userID)

	if body.PlanID == nil {
		c.JSON(http.StatusOK, gin.H{"message": "plan removed from user"})
		return
	}

	var plan UserPlan
	err = h.db.QueryRow(`
		SELECT id, name, data_limit_mb, validity_days, bandwidth_profile_id, max_devices
		FROM user_plans WHERE id=$1`, *body.PlanID).
		Scan(&plan.ID, &plan.Name, &plan.DataLimitMB, &plan.ValidityDays, &plan.BandwidthProfileID, &plan.MaxDevices)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "plan not found")
		return
	}

	// Set account expiry from plan validity
	if plan.ValidityDays > 0 {
		expiry := time.Now().AddDate(0, 0, plan.ValidityDays).Format("2006-01-02")
		h.db.Exec(`UPDATE radius_users SET account_expiry=$1 WHERE id=$2`, expiry, userID)
	}

	// Set device limit
	h.db.Exec(`UPDATE radius_users SET device_limit=$1 WHERE id=$2`, plan.MaxDevices, userID)
	h.db.Exec(`INSERT INTO radcheck (username,attribute,op,value) VALUES ($1,'Simultaneous-Use',':=',$2)
		ON CONFLICT (username,attribute) DO UPDATE SET value=$2`, username, fmt.Sprintf("%d", plan.MaxDevices))

	// Set data limit
	if plan.DataLimitMB != nil {
		h.db.Exec(`INSERT INTO radcheck (username,attribute,op,value) VALUES ($1,'Max-Octets',':=',$2)
			ON CONFLICT (username,attribute) DO UPDATE SET value=$2`,
			username, fmt.Sprintf("%d", *plan.DataLimitMB*1024*1024))
	} else {
		h.db.Exec(`DELETE FROM radcheck WHERE username=$1 AND attribute='Max-Octets'`, username)
	}

	// Apply bandwidth profile if linked
	if plan.BandwidthProfileID != nil {
		go h.ApplyBandwidthProfileDirect(userID, *plan.BandwidthProfileID)
	}

	// Auto-generate invoice for paid plans
	if plan.ValidityDays > 0 {
		invoiceNum := fmt.Sprintf("INV-%s-%04d", time.Now().Format("200601"), userID)
		h.db.Exec(`INSERT INTO invoices (invoice_number,user_id,username,plan_id,plan_name,amount,currency,due_date,created_by)
			VALUES ($1,$2,$3,$4,$5,0,'USD',CURRENT_DATE+$6,$7)
			ON CONFLICT (invoice_number) DO NOTHING`,
			invoiceNum, userID, username, body.PlanID, plan.Name, plan.ValidityDays, claims.UserID)
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("plan '%s' assigned to %s", plan.Name, username)})
}

// ApplyBandwidthProfileDirect applies a profile directly (called internally).
func (h *Handler) ApplyBandwidthProfileDirect(userID, profileID int) {
	var username string
	if err := h.db.QueryRow(`SELECT username FROM radius_users WHERE id=$1`, userID).Scan(&username); err != nil {
		return
	}
	var p BandwidthProfile
	if err := h.db.QueryRow(`SELECT id,name,upload_kbps,download_kbps,mikrotik_rate_limit FROM bandwidth_profiles WHERE id=$1`, profileID).
		Scan(&p.ID, &p.Name, &p.UploadKbps, &p.DownloadKbps, &p.MikrotikRateLimit); err != nil {
		return
	}
	rl := kbpsToMikrotik(p.UploadKbps, p.DownloadKbps)
	if p.MikrotikRateLimit != nil && *p.MikrotikRateLimit != "" {
		rl = *p.MikrotikRateLimit
	}
	h.db.Exec(`DELETE FROM radreply WHERE username=$1 AND attribute IN ('Mikrotik-Rate-Limit','WISPr-Bandwidth-Max-Up','WISPr-Bandwidth-Max-Down')`, username)
	h.db.Exec(`INSERT INTO radreply (username,attribute,op,value) VALUES ($1,'Mikrotik-Rate-Limit','=',$2)`, username, rl)
	h.db.Exec(`INSERT INTO radreply (username,attribute,op,value) VALUES ($1,'WISPr-Bandwidth-Max-Up','=',$2)`, username, fmt.Sprintf("%d", p.UploadKbps*1000))
	h.db.Exec(`INSERT INTO radreply (username,attribute,op,value) VALUES ($1,'WISPr-Bandwidth-Max-Down','=',$2)`, username, fmt.Sprintf("%d", p.DownloadKbps*1000))
}
