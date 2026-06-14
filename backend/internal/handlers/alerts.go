package handlers

import (
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/freeradius-manager/backend/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AlertRule holds an alert notification rule.
type AlertRule struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	EventType    string     `json:"event_type"`
	NotifyEmail  bool       `json:"notify_email"`
	EmailAddress *string    `json:"email_address"`
	IsActive     bool       `json:"is_active"`
	LastTriggered *time.Time `json:"last_triggered"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ListAlertRules returns all alert rules.
func (h *Handler) ListAlertRules(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, event_type, notify_email, email_address, is_active, last_triggered, created_at
		FROM alert_rules ORDER BY created_at ASC`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch alert rules")
		return
	}
	defer rows.Close()

	rules := []AlertRule{}
	for rows.Next() {
		var r AlertRule
		rows.Scan(&r.ID, &r.Name, &r.EventType, &r.NotifyEmail, &r.EmailAddress, &r.IsActive, &r.LastTriggered, &r.CreatedAt)
		rules = append(rules, r)
	}
	c.JSON(http.StatusOK, gin.H{"data": rules})
}

// CreateAlertRule creates a new alert rule.
func (h *Handler) CreateAlertRule(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		EventType    string `json:"event_type" binding:"required"`
		NotifyEmail  bool   `json:"notify_email"`
		EmailAddress string `json:"email_address"`
		IsActive     *bool  `json:"is_active"`
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
		INSERT INTO alert_rules (name,event_type,notify_email,email_address,is_active)
		VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		req.Name, req.EventType, req.NotifyEmail, nullableString(req.EmailAddress), isActive,
	).Scan(&id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create alert rule")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "alert rule created"})
}

// UpdateAlertRule updates an existing rule.
func (h *Handler) UpdateAlertRule(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Name         string `json:"name"`
		NotifyEmail  bool   `json:"notify_email"`
		EmailAddress string `json:"email_address"`
		IsActive     *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	h.db.Exec(`UPDATE alert_rules SET name=COALESCE(NULLIF($1,''),name),
		notify_email=$2, email_address=$3, is_active=$4 WHERE id=$5`,
		req.Name, req.NotifyEmail, nullableString(req.EmailAddress), isActive, id)
	c.JSON(http.StatusOK, gin.H{"message": "alert rule updated"})
}

// DeleteAlertRule removes a rule.
func (h *Handler) DeleteAlertRule(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`DELETE FROM alert_rules WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "alert rule deleted"})
}

// SendTestEmail sends a test email to verify SMTP config.
func (h *Handler) SendTestEmail(c *gin.Context) {
	var req struct {
		To string `json:"to" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := sendEmail(req.To, "RADIUS Manager Pro — Test Email",
		"This is a test email from RADIUS Manager Pro.\n\nSMTP is configured correctly.")
	if err != nil {
		respondError(c, http.StatusBadRequest, "failed to send email: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "test email sent to " + req.To})
}

// sendEmail sends a plain-text email using the SMTP settings from environment.
func sendEmail(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return fmt.Errorf("SMTP_HOST not configured")
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = "noreply@radius-manager.local"
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body)

	addr := host + ":" + port
	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

// StartAlertWorker launches a background goroutine that checks alert conditions
// every 5 minutes and sends email notifications.
func StartAlertWorker(db *database.DB, log *logrus.Logger) {
	go func() {
		log.Info("Alert worker started — checking every 5 minutes")
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		// Initial check after 1 minute
		time.Sleep(1 * time.Minute)
		checkAlerts(db, log)

		for range ticker.C {
			checkAlerts(db, log)
		}
	}()
}

func checkAlerts(db *database.DB, log *logrus.Logger) {
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		return // No SMTP configured, skip email
	}

	adminEmail := os.Getenv("SMTP_FROM")

	// Get active rules
	rows, err := db.Query(`SELECT id, event_type, email_address, name FROM alert_rules WHERE is_active=true`)
	if err != nil {
		return
	}
	defer rows.Close()

	type rule struct {
		id    int
		event string
		email string
		name  string
	}
	rules := []rule{}
	for rows.Next() {
		var r rule
		var email *string
		rows.Scan(&r.id, &r.event, &email, &r.name)
		if email != nil && *email != "" {
			r.email = *email
		} else {
			r.email = adminEmail
		}
		rules = append(rules, r)
	}

	for _, r := range rules {
		var triggered bool
		var details string

		switch r.event {
		case "account_expiry_3d":
			var count int
			db.QueryRow(`SELECT COUNT(*) FROM radius_users
				WHERE status='active' AND account_expiry BETWEEN CURRENT_DATE AND CURRENT_DATE+3`).Scan(&count)
			if count > 0 {
				triggered = true
				details = fmt.Sprintf("%d user account(s) expire within 3 days.", count)
			}
		case "account_expiry_7d":
			var count int
			db.QueryRow(`SELECT COUNT(*) FROM radius_users
				WHERE status='active' AND account_expiry BETWEEN CURRENT_DATE AND CURRENT_DATE+7`).Scan(&count)
			if count > 0 {
				triggered = true
				details = fmt.Sprintf("%d user account(s) expire within 7 days.", count)
			}
		case "nas_down":
			var count int
			db.QueryRow(`SELECT COUNT(*) FROM nas WHERE ping_status='down'`).Scan(&count)
			if count > 0 {
				triggered = true
				details = fmt.Sprintf("%d NAS device(s) are currently unreachable.", count)
			}
		case "data_limit_100pct":
			// Find users with Max-Octets and usage >= limit
			var userList []string
			uRows, _ := db.Query(`
				SELECT DISTINCT rc.username
				FROM radcheck rc
				JOIN radacct ra ON ra.username = rc.username
				WHERE rc.attribute = 'Max-Octets'
				AND (SELECT COALESCE(SUM(acctinputoctets+acctoutputoctets),0)
				     FROM radacct WHERE username=rc.username AND acctstarttime>=CURRENT_DATE-30)
				     >= rc.value::bigint`)
			if uRows != nil {
				for uRows.Next() {
					var u string
					uRows.Scan(&u)
					userList = append(userList, u)
				}
				uRows.Close()
			}
			if len(userList) > 0 {
				triggered = true
				details = fmt.Sprintf("Data limit reached for users: %s", strings.Join(userList, ", "))
			}
		}

		if triggered && r.email != "" {
			err := sendEmail(r.email,
				fmt.Sprintf("[RADIUS Manager Alert] %s", r.name),
				fmt.Sprintf("Alert: %s\n\n%s\n\nTime: %s",
					r.name, details, time.Now().Format(time.RFC1123)))
			if err != nil {
				log.WithError(err).Warnf("failed to send alert email for rule %s", r.name)
			} else {
				db.Exec(`UPDATE alert_rules SET last_triggered=NOW() WHERE id=$1`, r.id)
				log.Infof("Alert sent: %s → %s", r.name, r.email)
			}
		}
	}
}
