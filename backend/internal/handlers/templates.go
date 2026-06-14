package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// RADIUS Attribute Templates
// ─────────────────────────────────────────────────────────────────────────────

type AttributeEntry struct {
	Attribute string `json:"attribute"`
	Op        string `json:"op"`
	Value     string `json:"value"`
	Table     string `json:"table"` // "check" or "reply"
}

type RadiusTemplate struct {
	ID          int              `json:"id"`
	Name        string           `json:"name"`
	Description *string          `json:"description"`
	Attributes  []AttributeEntry `json:"attributes"`
	IsActive    bool             `json:"is_active"`
	CreatedAt   time.Time        `json:"created_at"`
}

// ListTemplates returns all RADIUS attribute templates.
func (h *Handler) ListTemplates(c *gin.Context) {
	rows, err := h.db.Query(`SELECT id, name, description, attributes, is_active, created_at
		FROM radius_templates ORDER BY is_active DESC, name`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch templates")
		return
	}
	defer rows.Close()

	templates := []RadiusTemplate{}
	for rows.Next() {
		var t RadiusTemplate
		var attrsRaw []byte
		rows.Scan(&t.ID, &t.Name, &t.Description, &attrsRaw, &t.IsActive, &t.CreatedAt)
		json.Unmarshal(attrsRaw, &t.Attributes)
		if t.Attributes == nil {
			t.Attributes = []AttributeEntry{}
		}
		templates = append(templates, t)
	}
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// CreateTemplate creates a new RADIUS attribute template.
func (h *Handler) CreateTemplate(c *gin.Context) {
	var req struct {
		Name        string           `json:"name" binding:"required,min=2,max=100"`
		Description string           `json:"description"`
		Attributes  []AttributeEntry `json:"attributes"`
		IsActive    *bool            `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if req.Attributes == nil {
		req.Attributes = []AttributeEntry{}
	}
	attrsJSON, _ := json.Marshal(req.Attributes)

	var id int
	err := h.db.QueryRow(`INSERT INTO radius_templates (name,description,attributes,is_active)
		VALUES ($1,$2,$3,$4) RETURNING id`,
		req.Name, nullableString(req.Description), string(attrsJSON), isActive).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "template name already exists")
			return
		}
		respondError(c, http.StatusInternalServerError, "failed to create template")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "template created"})
}

// UpdateTemplate updates template details and attributes.
func (h *Handler) UpdateTemplate(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Name        string           `json:"name"`
		Description string           `json:"description"`
		Attributes  []AttributeEntry `json:"attributes"`
		IsActive    *bool            `json:"is_active"`
	}
	c.ShouldBindJSON(&req)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	attrsJSON, _ := json.Marshal(req.Attributes)

	h.db.Exec(`UPDATE radius_templates SET
		name=COALESCE(NULLIF($1,''),name),
		description=COALESCE(NULLIF($2,''),description),
		attributes=CASE WHEN $3='null' OR $3='[]' THEN attributes ELSE $3::jsonb END,
		is_active=$4
		WHERE id=$5`,
		req.Name, req.Description, string(attrsJSON), isActive, id)
	c.JSON(http.StatusOK, gin.H{"message": "template updated"})
}

// DeleteTemplate removes a template.
func (h *Handler) DeleteTemplate(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`DELETE FROM radius_templates WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "template deleted"})
}

// ApplyTemplate applies a template's attributes to one or more RADIUS users.
func (h *Handler) ApplyTemplate(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Usernames []string `json:"usernames" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Load template
	var attrsRaw []byte
	err = h.db.QueryRow(`SELECT attributes FROM radius_templates WHERE id=$1 AND is_active=true`, id).Scan(&attrsRaw)
	if err != nil {
		respondError(c, http.StatusNotFound, "template not found or inactive")
		return
	}
	var attrs []AttributeEntry
	json.Unmarshal(attrsRaw, &attrs)

	applied, failed := 0, 0
	for _, username := range req.Usernames {
		// Verify user exists
		var exists bool
		h.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM radius_users WHERE username=$1)`, username).Scan(&exists)
		if !exists {
			failed++
			continue
		}

		// Apply each attribute
		for _, attr := range attrs {
			if attr.Table == "reply" || attr.Table == "" {
				h.db.Exec(`INSERT INTO radreply (username,attribute,op,value) VALUES ($1,$2,$3,$4)
					ON CONFLICT (username,attribute) DO UPDATE SET value=$4, op=$3`,
					username, attr.Attribute, attr.Op, attr.Value)
			} else {
				h.db.Exec(`INSERT INTO radcheck (username,attribute,op,value) VALUES ($1,$2,$3,$4)
					ON CONFLICT (username,attribute) DO UPDATE SET value=$4, op=$3`,
					username, attr.Attribute, attr.Op, attr.Value)
			}
		}
		applied++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "template applied",
		"applied": applied,
		"failed":  failed,
	})
}

// CloneTemplate duplicates an existing template.
func (h *Handler) CloneTemplate(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var newID int
	err = h.db.QueryRow(`INSERT INTO radius_templates (name,description,attributes,is_active)
		SELECT $1, description, attributes, is_active FROM radius_templates WHERE id=$2
		RETURNING id`, req.Name, id).Scan(&newID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to clone template")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": newID, "message": "template cloned"})
}
