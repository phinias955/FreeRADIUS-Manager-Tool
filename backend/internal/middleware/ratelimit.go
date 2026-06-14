package middleware

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type visitor struct {
	count    int
	lastSeen time.Time
	blocked  bool
	blockedUntil time.Time
}

// RateLimiter is a simple in-memory rate limiter per IP.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter() *RateLimiter {
	limit := 100
	if raw := os.Getenv("RATE_LIMIT"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}

	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   time.Minute,
	}

	// Periodically clean up old visitors
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// Middleware returns the Gin rate limiting handler.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if rl.isLimited(ip) {
			c.Header("X-RateLimit-Limit", strconv.Itoa(rl.limit))
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, please try again later",
			})
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		c.Next()
	}
}

func (rl *RateLimiter) isLimited(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[ip]
	now := time.Now()

	if !ok {
		rl.visitors[ip] = &visitor{count: 1, lastSeen: now}
		return false
	}

	// Check if blocked
	if v.blocked && now.Before(v.blockedUntil) {
		return true
	}
	if v.blocked && now.After(v.blockedUntil) {
		v.blocked = false
		v.count = 0
	}

	// Reset window
	if now.Sub(v.lastSeen) > rl.window {
		v.count = 0
		v.lastSeen = now
	}

	v.count++
	v.lastSeen = now

	if v.count > rl.limit {
		v.blocked = true
		v.blockedUntil = now.Add(rl.window)
		return true
	}

	return false
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, v := range rl.visitors {
		if v.lastSeen.Before(cutoff) && !v.blocked {
			delete(rl.visitors, ip)
		}
	}
}
