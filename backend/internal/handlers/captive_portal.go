package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Captive Portal
// ─────────────────────────────────────────────────────────────────────────────

type CaptivePortal struct {
	ID           int       `json:"id"`
	ZoneID       *int      `json:"zone_id"`
	ZoneName     *string   `json:"zone_name"`
	Name         string    `json:"name"`
	Title        string    `json:"title"`
	Subtitle     *string   `json:"subtitle"`
	LogoURL      *string   `json:"logo_url"`
	BgColor      string    `json:"bg_color"`
	PrimaryColor string    `json:"primary_color"`
	AuthType     string    `json:"auth_type"`
	RedirectURL  string    `json:"redirect_url"`
	TermsText    *string   `json:"terms_text"`
	FooterText   *string   `json:"footer_text"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListCaptivePortals returns all captive portal configs.
func (h *Handler) ListCaptivePortals(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT cp.id, cp.zone_id, hz.name, cp.name, cp.title, cp.subtitle,
		       cp.logo_url, cp.bg_color, cp.primary_color, cp.auth_type,
		       cp.redirect_url, cp.terms_text, cp.footer_text, cp.is_active, cp.created_at
		FROM captive_portals cp
		LEFT JOIN hotspot_zones hz ON hz.id = cp.zone_id
		ORDER BY cp.is_active DESC, cp.name`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch portals")
		return
	}
	defer rows.Close()

	portals := []CaptivePortal{}
	for rows.Next() {
		var p CaptivePortal
		rows.Scan(&p.ID, &p.ZoneID, &p.ZoneName, &p.Name, &p.Title, &p.Subtitle,
			&p.LogoURL, &p.BgColor, &p.PrimaryColor, &p.AuthType,
			&p.RedirectURL, &p.TermsText, &p.FooterText, &p.IsActive, &p.CreatedAt)
		portals = append(portals, p)
	}
	c.JSON(http.StatusOK, gin.H{"data": portals})
}

// CreateCaptivePortal creates a new portal configuration.
func (h *Handler) CreateCaptivePortal(c *gin.Context) {
	var req struct {
		ZoneID       *int   `json:"zone_id"`
		Name         string `json:"name" binding:"required,min=2"`
		Title        string `json:"title"`
		Subtitle     string `json:"subtitle"`
		LogoURL      string `json:"logo_url"`
		BgColor      string `json:"bg_color"`
		PrimaryColor string `json:"primary_color"`
		AuthType     string `json:"auth_type"`
		RedirectURL  string `json:"redirect_url"`
		TermsText    string `json:"terms_text"`
		FooterText   string `json:"footer_text"`
		IsActive     *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.BgColor == "" {
		req.BgColor = "#f0f4ff"
	}
	if req.PrimaryColor == "" {
		req.PrimaryColor = "#3b82f6"
	}
	if req.AuthType == "" {
		req.AuthType = "userpass"
	}
	if req.Title == "" {
		req.Title = "WiFi Login"
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var id int
	h.db.QueryRow(`INSERT INTO captive_portals
		(zone_id,name,title,subtitle,logo_url,bg_color,primary_color,auth_type,redirect_url,terms_text,footer_text,is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
		req.ZoneID, req.Name, req.Title,
		nullableString(req.Subtitle), nullableString(req.LogoURL),
		req.BgColor, req.PrimaryColor, req.AuthType,
		req.RedirectURL, nullableString(req.TermsText),
		nullableString(req.FooterText), isActive).Scan(&id)

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "captive portal created"})
}

// UpdateCaptivePortal updates portal settings.
func (h *Handler) UpdateCaptivePortal(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Name         string `json:"name"`
		Title        string `json:"title"`
		Subtitle     string `json:"subtitle"`
		LogoURL      string `json:"logo_url"`
		BgColor      string `json:"bg_color"`
		PrimaryColor string `json:"primary_color"`
		AuthType     string `json:"auth_type"`
		RedirectURL  string `json:"redirect_url"`
		TermsText    string `json:"terms_text"`
		FooterText   string `json:"footer_text"`
		IsActive     *bool  `json:"is_active"`
	}
	c.ShouldBindJSON(&req)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	h.db.Exec(`UPDATE captive_portals SET
		name=COALESCE(NULLIF($1,''),name),
		title=COALESCE(NULLIF($2,''),title),
		subtitle=COALESCE(NULLIF($3,''),subtitle),
		logo_url=COALESCE(NULLIF($4,''),logo_url),
		bg_color=COALESCE(NULLIF($5,''),bg_color),
		primary_color=COALESCE(NULLIF($6,''),primary_color),
		auth_type=COALESCE(NULLIF($7,''),auth_type),
		redirect_url=COALESCE(NULLIF($8,''),redirect_url),
		terms_text=COALESCE(NULLIF($9,''),terms_text),
		footer_text=COALESCE(NULLIF($10,''),footer_text),
		is_active=$11
		WHERE id=$12`,
		req.Name, req.Title, req.Subtitle, req.LogoURL,
		req.BgColor, req.PrimaryColor, req.AuthType,
		req.RedirectURL, req.TermsText, req.FooterText, isActive, id)
	c.JSON(http.StatusOK, gin.H{"message": "portal updated"})
}

// DeleteCaptivePortal removes a portal config.
func (h *Handler) DeleteCaptivePortal(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`DELETE FROM captive_portals WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "portal deleted"})
}

// ServeCaptivePortal generates and returns the live HTML captive portal page.
// This is a PUBLIC endpoint — no auth required.
func (h *Handler) ServeCaptivePortal(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		c.String(http.StatusBadRequest, "invalid portal ID")
		return
	}

	var p CaptivePortal
	err = h.db.QueryRow(`SELECT id,zone_id,NULL,name,title,subtitle,logo_url,bg_color,primary_color,
		auth_type,redirect_url,terms_text,footer_text,is_active,created_at
		FROM captive_portals WHERE id=$1 AND is_active=true`, id).
		Scan(&p.ID, &p.ZoneID, &p.ZoneName, &p.Name, &p.Title, &p.Subtitle,
			&p.LogoURL, &p.BgColor, &p.PrimaryColor, &p.AuthType,
			&p.RedirectURL, &p.TermsText, &p.FooterText, &p.IsActive, &p.CreatedAt)
	if err != nil {
		c.String(http.StatusNotFound, "portal not found or inactive")
		return
	}

	html := buildCaptivePortalHTML(p)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func buildCaptivePortalHTML(p CaptivePortal) string {
	subtitle := ""
	if p.Subtitle != nil {
		subtitle = *p.Subtitle
	}
	logoHTML := ""
	if p.LogoURL != nil && *p.LogoURL != "" {
		logoHTML = fmt.Sprintf(`<img src="%s" alt="logo" style="max-height:80px;margin-bottom:16px;border-radius:8px;">`, *p.LogoURL)
	}
	termsHTML := ""
	if p.TermsText != nil && *p.TermsText != "" {
		termsHTML = fmt.Sprintf(`<p style="font-size:11px;color:#94a3b8;text-align:center;margin-top:16px;">%s</p>`, *p.TermsText)
	}
	footerText := "Powered by RADIUS Manager"
	if p.FooterText != nil && *p.FooterText != "" {
		footerText = *p.FooterText
	}

	voucherField := ""
	userpassFields := ""
	switch p.AuthType {
	case "voucher":
		voucherField = `<input name="voucher" type="text" placeholder="Enter voucher code" required
			style="width:100%;padding:12px 16px;border:1.5px solid #e2e8f0;border-radius:10px;font-size:15px;margin-bottom:12px;box-sizing:border-box;">`
	case "userpass":
		userpassFields = fmt.Sprintf(`
			<input name="username" type="text" placeholder="Username" required
				style="width:100%%;padding:12px 16px;border:1.5px solid #e2e8f0;border-radius:10px;font-size:15px;margin-bottom:10px;box-sizing:border-box;">
			<input name="password" type="password" placeholder="Password" required
				style="width:100%%;padding:12px 16px;border:1.5px solid #e2e8f0;border-radius:10px;font-size:15px;margin-bottom:12px;box-sizing:border-box;">`)
	case "both":
		userpassFields = fmt.Sprintf(`
			<div id="tab-bar" style="display:flex;gap:8px;margin-bottom:16px;">
				<button type="button" onclick="setTab('user')" id="tab-user"
					style="flex:1;padding:8px;border-radius:8px;border:2px solid %s;background:%s;color:#fff;cursor:pointer;font-size:13px;">Username</button>
				<button type="button" onclick="setTab('voucher')" id="tab-voucher"
					style="flex:1;padding:8px;border-radius:8px;border:2px solid #e2e8f0;background:#fff;color:#64748b;cursor:pointer;font-size:13px;">Voucher</button>
			</div>
			<div id="fields-user">
				<input name="username" type="text" placeholder="Username" style="width:100%%;padding:12px 16px;border:1.5px solid #e2e8f0;border-radius:10px;font-size:15px;margin-bottom:10px;box-sizing:border-box;">
				<input name="password" type="password" placeholder="Password" style="width:100%%;padding:12px 16px;border:1.5px solid #e2e8f0;border-radius:10px;font-size:15px;margin-bottom:12px;box-sizing:border-box;">
			</div>
			<div id="fields-voucher" style="display:none;">
				<input name="voucher" type="text" placeholder="Voucher code" style="width:100%%;padding:12px 16px;border:1.5px solid #e2e8f0;border-radius:10px;font-size:15px;margin-bottom:12px;box-sizing:border-box;">
			</div>`, p.PrimaryColor, p.PrimaryColor)
	}

	tabScript := ""
	if p.AuthType == "both" {
		tabScript = fmt.Sprintf(`<script>
		function setTab(t) {
			document.getElementById('fields-user').style.display = t==='user'?'':'none';
			document.getElementById('fields-voucher').style.display = t==='voucher'?'':'none';
			document.getElementById('tab-user').style.background = t==='user'?'%s':'#fff';
			document.getElementById('tab-user').style.color = t==='user'?'#fff':'#64748b';
			document.getElementById('tab-voucher').style.background = t==='voucher'?'%s':'#fff';
			document.getElementById('tab-voucher').style.color = t==='voucher'?'#fff':'#64748b';
		}
		</script>`, p.PrimaryColor, p.PrimaryColor)
	}

	return strings.ReplaceAll(fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { min-height: 100vh; display: flex; flex-direction: column; align-items: center;
         justify-content: center; background: %s; font-family: -apple-system,sans-serif; padding: 20px; }
  .card { background: #fff; border-radius: 20px; padding: 40px 36px; width: 100%%;
          max-width: 380px; box-shadow: 0 20px 60px rgba(0,0,0,0.1); }
  button[type=submit] { width: 100%%; padding: 14px; background: %s; color: #fff;
                        border: none; border-radius: 10px; font-size: 16px; font-weight: 600;
                        cursor: pointer; }
  button[type=submit]:hover { opacity: 0.9; }
  .footer { margin-top: 24px; font-size: 12px; color: #94a3b8; }
</style>
</head>
<body>
<div class="card">
  <div style="text-align:center;margin-bottom:24px;">
    %s
    <h1 style="font-size:22px;font-weight:700;color:#0f172a;">%s</h1>
    %s
  </div>
  <form method="POST" action="/api/v1/captive/authenticate">
    <input type="hidden" name="redirect" value="%s">
    %s%s
    <button type="submit">Connect to Internet</button>
  </form>
  %s
</div>
<p class="footer">%s</p>
%s
</body>
</html>`,
		p.Title, p.BgColor, p.PrimaryColor,
		logoHTML, p.Title,
		func() string {
			if subtitle != "" {
				return fmt.Sprintf(`<p style="color:#64748b;font-size:14px;margin-top:4px;">%s</p>`, subtitle)
			}
			return ""
		}(),
		p.RedirectURL,
		userpassFields, voucherField,
		termsHTML, footerText, tabScript), "%%", "%")
}
