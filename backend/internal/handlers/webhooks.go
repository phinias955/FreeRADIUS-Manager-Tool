package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Webhooks
// ─────────────────────────────────────────────────────────────────────────────

type Webhook struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	Secret        *string    `json:"secret,omitempty"`
	Events        []string   `json:"events"`
	IsActive      bool       `json:"is_active"`
	LastTriggered *time.Time `json:"last_triggered"`
	FailCount     int        `json:"fail_count"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ListWebhooks returns all webhook configurations.
func (h *Handler) ListWebhooks(c *gin.Context) {
	rows, err := h.db.Query(`SELECT id,name,url,events,is_active,last_triggered,fail_count,created_at
		FROM webhooks ORDER BY is_active DESC, name`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch webhooks")
		return
	}
	defer rows.Close()

	hooks := []Webhook{}
	for rows.Next() {
		var wh Webhook
		var eventsArr []byte
		rows.Scan(&wh.ID, &wh.Name, &wh.URL, &eventsArr,
			&wh.IsActive, &wh.LastTriggered, &wh.FailCount, &wh.CreatedAt)
		json.Unmarshal(eventsArr, &wh.Events)
		if wh.Events == nil {
			wh.Events = []string{}
		}
		hooks = append(hooks, wh)
	}
	c.JSON(http.StatusOK, gin.H{"data": hooks})
}

// CreateWebhook registers a new webhook.
func (h *Handler) CreateWebhook(c *gin.Context) {
	var req struct {
		Name     string   `json:"name" binding:"required,min=2"`
		URL      string   `json:"url" binding:"required,url"`
		Secret   string   `json:"secret"`
		Events   []string `json:"events"`
		IsActive *bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	eventsJSON, _ := json.Marshal(req.Events)

	var id int
	h.db.QueryRow(`INSERT INTO webhooks (name,url,secret,events,is_active) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		req.Name, req.URL, nullableString(req.Secret), string(eventsJSON), isActive).Scan(&id)

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "webhook created"})
}

// UpdateWebhook updates a webhook's configuration.
func (h *Handler) UpdateWebhook(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Name     string   `json:"name"`
		URL      string   `json:"url"`
		Secret   string   `json:"secret"`
		Events   []string `json:"events"`
		IsActive *bool    `json:"is_active"`
	}
	c.ShouldBindJSON(&req)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	eventsJSON, _ := json.Marshal(req.Events)

	h.db.Exec(`UPDATE webhooks SET
		name=COALESCE(NULLIF($1,''),name),
		url=COALESCE(NULLIF($2,''),url),
		secret=COALESCE(NULLIF($3,''),secret),
		events=COALESCE(NULLIF($4,'[]'),events)::text[],
		is_active=$5,
		fail_count=CASE WHEN $5=true THEN 0 ELSE fail_count END
		WHERE id=$6`,
		req.Name, req.URL, req.Secret, string(eventsJSON), isActive, id)
	c.JSON(http.StatusOK, gin.H{"message": "webhook updated"})
}

// DeleteWebhook removes a webhook and its logs.
func (h *Handler) DeleteWebhook(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`DELETE FROM webhook_logs WHERE webhook_id=$1`, id)
	h.db.Exec(`DELETE FROM webhooks WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "webhook deleted"})
}

// TestWebhook sends a test payload to the webhook URL.
func (h *Handler) TestWebhook(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var url, secret string
	var secretPtr *string
	h.db.QueryRow(`SELECT url, COALESCE(secret,'') FROM webhooks WHERE id=$1`, id).Scan(&url, &secret)
	if url == "" {
		respondError(c, http.StatusNotFound, "webhook not found")
		return
	}
	if secret != "" {
		secretPtr = &secret
	}

	payload := gin.H{
		"event":     "test",
		"timestamp": time.Now().Unix(),
		"message":   "This is a test delivery from RADIUS Manager",
	}

	statusCode, deliveryErr := deliverWebhook(url, secretPtr, "test", payload)
	success := deliveryErr == nil

	h.db.Exec(`INSERT INTO webhook_logs (webhook_id,event,payload,status_code,success) VALUES ($1,'test',$2,$3,$4)`,
		id, mustJSON(payload), statusCode, success)

	if !success {
		respondError(c, http.StatusBadGateway, fmt.Sprintf("delivery failed: %v", deliveryErr))
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "test delivered", "status_code": statusCode})
}

// ListWebhookLogs returns recent delivery logs for a webhook.
func (h *Handler) ListWebhookLogs(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	rows, _ := h.db.Query(`SELECT id,event,status_code,success,created_at
		FROM webhook_logs WHERE webhook_id=$1 ORDER BY created_at DESC LIMIT 50`, id)
	type Log struct {
		ID         int       `json:"id"`
		Event      string    `json:"event"`
		StatusCode int       `json:"status_code"`
		Success    bool      `json:"success"`
		CreatedAt  time.Time `json:"created_at"`
	}
	logs := []Log{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l Log
			rows.Scan(&l.ID, &l.Event, &l.StatusCode, &l.Success, &l.CreatedAt)
			logs = append(logs, l)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}

// dispatchWebhook delivers an event to all matching active webhooks (non-blocking).
func (h *Handler) dispatchWebhook(event string, payload gin.H) {
	rows, err := h.db.Query(`SELECT id,url,secret FROM webhooks
		WHERE is_active=true AND (events='{}' OR $1=ANY(events))`, event)
	if err != nil || rows == nil {
		return
	}
	defer rows.Close()

	type hook struct {
		id     int
		url    string
		secret *string
	}
	hooks := []hook{}
	for rows.Next() {
		var wh hook
		var sec *string
		rows.Scan(&wh.id, &wh.url, &sec)
		wh.secret = sec
		hooks = append(hooks, wh)
	}

	for _, wh := range hooks {
		go func(wh hook) {
			statusCode, deliveryErr := deliverWebhook(wh.url, wh.secret, event, payload)
			success := deliveryErr == nil

			h.db.Exec(`INSERT INTO webhook_logs (webhook_id,event,payload,status_code,success) VALUES ($1,$2,$3,$4,$5)`,
				wh.id, event, mustJSON(payload), statusCode, success)
			h.db.Exec(`UPDATE webhooks SET last_triggered=NOW(),
				fail_count=CASE WHEN $1 THEN 0 ELSE fail_count+1 END WHERE id=$2`,
				success, wh.id)
		}(wh)
	}
}

// deliverWebhook sends a signed JSON POST to the given URL.
func deliverWebhook(url string, secret *string, event string, payload interface{}) (int, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"event":     event,
		"timestamp": time.Now().Unix(),
		"data":      payload,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-RADIUS-Event", event)

	if secret != nil && *secret != "" {
		mac := hmac.New(sha256.New, []byte(*secret))
		mac.Write(body)
		req.Header.Set("X-RADIUS-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
