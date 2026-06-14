package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/freeradius-manager/backend/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	db  *database.DB
	log *logrus.Logger
}

// New creates a Handler with injected dependencies.
func New(db *database.DB, log *logrus.Logger) *Handler {
	return &Handler{db: db, log: log}
}

// HealthCheck returns service health status.
func HealthCheck(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "ok"
		if err := db.Ping(); err != nil {
			dbStatus = "error: " + err.Error()
		}

		status := http.StatusOK
		if dbStatus != "ok" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"services": gin.H{
				"database": dbStatus,
			},
		})
	}
}

// Version returns the application version.
func Version() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": "1.0.0",
			"name":    "FreeRADIUS Manager",
			"build":   time.Now().Format("2006-01-02"),
		})
	}
}

// respondError sends a JSON error response.
func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// paginationParams extracts page and limit from query string.
func paginationParams(c *gin.Context) (offset, limit int) {
	page := 1
	limit = 20

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	offset = (page - 1) * limit
	return
}

// mustInt is a helper to parse URL param :id.
func mustInt(c *gin.Context, param string) (int, error) {
	raw := c.Param(param)
	return strconv.Atoi(raw)
}

