package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Organizations / Resellers
// ─────────────────────────────────────────────────────────────────────────────

type Organization struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Email     *string    `json:"email"`
	Phone     *string    `json:"phone"`
	Address   *string    `json:"address"`
	LogoURL   *string    `json:"logo_url"`
	UserLimit int        `json:"user_limit"`
	IsActive  bool       `json:"is_active"`
	ParentID  *int       `json:"parent_id"`
	UserCount int        `json:"user_count"`
	NASCount  int        `json:"nas_count"`
	CreatedAt time.Time  `json:"created_at"`
}

// ListOrganizations returns all orgs with live stats.
func (h *Handler) ListOrganizations(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT o.id, o.name, o.slug, o.email, o.phone, o.address, o.logo_url,
		       o.user_limit, o.is_active, o.parent_id, o.created_at,
		       COUNT(DISTINCT ru.id)  AS user_count,
		       COUNT(DISTINCT n.id)   AS nas_count
		FROM organizations o
		LEFT JOIN radius_users ru ON ru.org_id = o.id
		LEFT JOIN nas n ON n.org_id = o.id
		GROUP BY o.id
		ORDER BY o.is_active DESC, o.name`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch organizations")
		return
	}
	defer rows.Close()

	orgs := []Organization{}
	for rows.Next() {
		var org Organization
		rows.Scan(&org.ID, &org.Name, &org.Slug, &org.Email, &org.Phone, &org.Address,
			&org.LogoURL, &org.UserLimit, &org.IsActive, &org.ParentID, &org.CreatedAt,
			&org.UserCount, &org.NASCount)
		orgs = append(orgs, org)
	}
	c.JSON(http.StatusOK, gin.H{"data": orgs})
}

// CreateOrganization adds a new organization.
func (h *Handler) CreateOrganization(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required,min=2,max=100"`
		Slug      string `json:"slug" binding:"required,min=2,max=50"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		Address   string `json:"address"`
		UserLimit int    `json:"user_limit"`
		IsActive  *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Auto-slugify
	req.Slug = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.Slug), " ", "-"))
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var id int
	err := h.db.QueryRow(`
		INSERT INTO organizations (name,slug,email,phone,address,user_limit,is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		req.Name, req.Slug,
		nullableString(req.Email), nullableString(req.Phone),
		nullableString(req.Address), req.UserLimit, isActive).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "organization name or slug already exists")
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create organization")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "organization created"})
}

// UpdateOrganization updates an org's details.
func (h *Handler) UpdateOrganization(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Name      string `json:"name"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
		Address   string `json:"address"`
		LogoURL   string `json:"logo_url"`
		UserLimit int    `json:"user_limit"`
		IsActive  *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	h.db.Exec(`UPDATE organizations SET
		name=COALESCE(NULLIF($1,''),name),
		email=COALESCE(NULLIF($2,''),email),
		phone=COALESCE(NULLIF($3,''),phone),
		address=COALESCE(NULLIF($4,''),address),
		logo_url=COALESCE(NULLIF($5,''),logo_url),
		user_limit=CASE WHEN $6>0 THEN $6 ELSE user_limit END,
		is_active=$7
		WHERE id=$8`,
		req.Name, req.Email, req.Phone, req.Address, req.LogoURL, req.UserLimit, isActive, id)
	c.JSON(http.StatusOK, gin.H{"message": "organization updated"})
}

// DeleteOrganization removes an org (unlinking resources first).
func (h *Handler) DeleteOrganization(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`UPDATE radius_users SET org_id=NULL WHERE org_id=$1`, id)
	h.db.Exec(`UPDATE nas SET org_id=NULL WHERE org_id=$1`, id)
	h.db.Exec(`UPDATE app_users SET org_id=NULL WHERE org_id=$1`, id)
	h.db.Exec(`DELETE FROM organizations WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "organization deleted"})
}

// OrgStats returns detailed statistics for a specific org.
func (h *Handler) OrgStats(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	type Stat struct {
		TotalUsers    int     `json:"total_users"`
		ActiveUsers   int     `json:"active_users"`
		ExpiredUsers  int     `json:"expired_users"`
		TotalNAS      int     `json:"total_nas"`
		ActiveSessions int    `json:"active_sessions"`
		TotalRevenue  float64 `json:"total_revenue"`
		MonthRevenue  float64 `json:"month_revenue"`
	}
	var s Stat
	h.db.QueryRow(`SELECT
		COUNT(*),
		COUNT(*) FILTER (WHERE status='active'),
		COUNT(*) FILTER (WHERE status='expired')
		FROM radius_users WHERE org_id=$1`, id).
		Scan(&s.TotalUsers, &s.ActiveUsers, &s.ExpiredUsers)

	h.db.QueryRow(`SELECT COUNT(*) FROM nas WHERE org_id=$1`, id).Scan(&s.TotalNAS)

	h.db.QueryRow(`SELECT COUNT(*) FROM radacct ra
		JOIN radius_users u ON u.username=ra.username
		WHERE u.org_id=$1 AND ra.acctstoptime IS NULL`, id).Scan(&s.ActiveSessions)

	h.db.QueryRow(`SELECT
		COALESCE(SUM(amount) FILTER (WHERE status='paid'),0),
		COALESCE(SUM(amount) FILTER (WHERE status='paid' AND created_at >= DATE_TRUNC('month',NOW())),0)
		FROM invoices i
		JOIN radius_users u ON u.username=i.username
		WHERE u.org_id=$1`, id).Scan(&s.TotalRevenue, &s.MonthRevenue)

	c.JSON(http.StatusOK, s)
}

// AssignUserToOrg links a RADIUS user to an organization.
func (h *Handler) AssignUserToOrg(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		OrgID    *int   `json:"org_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	h.db.Exec(`UPDATE radius_users SET org_id=$1 WHERE username=$2`, req.OrgID, req.Username)
	c.JSON(http.StatusOK, gin.H{"message": "user organization updated"})
}
