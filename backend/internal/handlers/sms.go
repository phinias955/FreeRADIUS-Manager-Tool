package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// SendSMS sends a text message via the configured SMS gateway.
func (h *Handler) SendSMS(c *gin.Context) {
	var req struct {
		To       string `json:"to" binding:"required"`
		Message  string `json:"message" binding:"required"`
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	claims, _ := middleware.GetClaims(c)
	_ = claims

	ref, err := sendSMSGateway(req.To, req.Message)
	status := "sent"
	if err != nil {
		status = "failed"
	}

	h.db.Exec(`INSERT INTO sms_logs (recipient,message,status,gateway_ref,username)
		VALUES ($1,$2,$3,$4,$5)`, req.To, req.Message, status, ref, nullableString(req.Username))

	if err != nil {
		respondError(c, http.StatusBadRequest, "SMS gateway error: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "SMS sent",
		"to":          req.To,
		"gateway_ref": ref,
	})
}

// ListSMSLogs returns SMS send history.
func (h *Handler) ListSMSLogs(c *gin.Context) {
	offset, limit := paginationParams(c)
	rows, err := h.db.Query(`
		SELECT id, recipient, LEFT(message,80), status, gateway_ref, username, created_at
		FROM sms_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch SMS logs")
		return
	}
	defer rows.Close()

	type SMSLog struct {
		ID         int        `json:"id"`
		Recipient  string     `json:"recipient"`
		Message    string     `json:"message"`
		Status     string     `json:"status"`
		GatewayRef *string    `json:"gateway_ref"`
		Username   *string    `json:"username"`
		CreatedAt  time.Time  `json:"created_at"`
	}
	logs := []SMSLog{}
	for rows.Next() {
		var l SMSLog
		rows.Scan(&l.ID, &l.Recipient, &l.Message, &l.Status, &l.GatewayRef, &l.Username, &l.CreatedAt)
		logs = append(logs, l)
	}
	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM sms_logs`).Scan(&total)
	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
}

// NotifyUserExpiry sends SMS to users expiring within `days` days (if phone set).
func (h *Handler) NotifyUserExpiry(c *gin.Context) {
	var req struct {
		Days    int    `json:"days"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Days == 0 {
		req.Days = 3
	}
	if req.Message == "" {
		req.Message = "Dear {username}, your account expires on {expiry}. Please renew to continue service."
	}

	rows, _ := h.db.Query(`
		SELECT username, COALESCE(phone,''), COALESCE(account_expiry::text,'')
		FROM radius_users
		WHERE status='active'
		  AND phone IS NOT NULL AND phone != ''
		  AND account_expiry BETWEEN CURRENT_DATE AND CURRENT_DATE+$1`, req.Days)

	sent, failed := 0, 0
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var username, phone, expiry string
			rows.Scan(&username, &phone, &expiry)

			msg := strings.ReplaceAll(req.Message, "{username}", username)
			msg = strings.ReplaceAll(msg, "{expiry}", expiry)

			ref, err := sendSMSGateway(phone, msg)
			status := "sent"
			if err != nil {
				status = "failed"
				failed++
			} else {
				sent++
			}
			h.db.Exec(`INSERT INTO sms_logs (recipient,message,status,gateway_ref,username)
				VALUES ($1,$2,$3,$4,$5)`, phone, msg, status, ref, username)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Notifications: %d sent, %d failed", sent, failed),
		"sent":    sent,
		"failed":  failed,
	})
}

// SMSConfig returns current SMS gateway configuration status.
func (h *Handler) SMSConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"gateway":    os.Getenv("SMS_GATEWAY"),
		"configured": os.Getenv("SMS_GATEWAY") != "",
		"sender_id":  os.Getenv("SMS_SENDER_ID"),
	})
}

// sendSMSGateway sends a message via HTTP SMS gateway.
// Supports generic HTTP POST (Bulksms, Africa's Talking, custom gateways).
func sendSMSGateway(to, message string) (string, error) {
	gatewayURL := os.Getenv("SMS_GATEWAY")
	if gatewayURL == "" {
		return "", fmt.Errorf("SMS_GATEWAY not configured in .env")
	}

	apiKey := os.Getenv("SMS_API_KEY")
	senderID := os.Getenv("SMS_SENDER_ID")
	if senderID == "" {
		senderID = "RADIUS"
	}

	// Build request body — supports {to}, {message}, {api_key}, {sender} placeholders
	bodyTemplate := os.Getenv("SMS_BODY_TEMPLATE")
	if bodyTemplate == "" {
		// Default: Africa's Talking / generic JSON
		bodyTemplate = `{"to":"{to}","message":"{message}","from":"{sender}"}`
	}

	body := strings.ReplaceAll(bodyTemplate, "{to}", to)
	body = strings.ReplaceAll(body, "{message}", message)
	body = strings.ReplaceAll(body, "{api_key}", apiKey)
	body = strings.ReplaceAll(body, "{sender}", senderID)

	contentType := os.Getenv("SMS_CONTENT_TYPE")
	if contentType == "" {
		contentType = "application/json"
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(gatewayURL, contentType, bytes.NewBufferString(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("gateway returned HTTP %d", resp.StatusCode)
	}

	return fmt.Sprintf("HTTP-%d", resp.StatusCode), nil
}
