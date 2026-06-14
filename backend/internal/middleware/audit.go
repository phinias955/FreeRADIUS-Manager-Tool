package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/freeradius-manager/backend/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// responseWriter wraps gin.ResponseWriter to capture the status code.
type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// AuditLogger records all write operations (POST, PUT, DELETE) to the audit_log table.
func AuditLogger(db *database.DB, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		// Only audit mutating requests
		if method != http.MethodPost && method != http.MethodPut &&
			method != http.MethodDelete && method != http.MethodPatch {
			c.Next()
			return
		}

		// Read body for audit record
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Replace writer to capture status
		rw := &responseWriter{c.Writer, &bytes.Buffer{}, http.StatusOK}
		c.Writer = rw

		c.Next()

		claims, ok := GetClaims(c)
		if !ok {
			return
		}

		// Sanitize body - remove password fields
		sanitized := sanitizeBody(bodyBytes)

		action := method + " " + c.FullPath()
		ipStr := c.ClientIP()

		details, _ := json.Marshal(map[string]interface{}{
			"method":    method,
			"path":      c.Request.URL.Path,
			"status":    rw.status,
			"body":      string(sanitized),
		})

		_, err := db.Exec(`
			INSERT INTO audit_log (user_id, action, details, ip_address, user_agent)
			VALUES ($1, $2, $3, $4::inet, $5)`,
			claims.UserID,
			action,
			string(details),
			ipStr,
			c.Request.UserAgent(),
		)
		if err != nil {
			log.WithError(err).Warn("Failed to write audit log entry")
		}
	}
}

// sanitizeBody removes sensitive fields from the JSON body for audit storage.
func sanitizeBody(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return []byte("[non-JSON body]")
	}
	sensitiveFields := []string{"password", "new_password", "current_password", "secret", "mfa_secret", "token"}
	for _, f := range sensitiveFields {
		if _, ok := m[f]; ok {
			m[f] = "***REDACTED***"
		}
	}
	out, _ := json.Marshal(m)
	return out
}
