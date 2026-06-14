package middleware

import (
	"net/http"
	"strings"

	"github.com/freeradius-manager/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

const (
	claimsKey = "claims"
)

// RequireAuth validates the Bearer JWT from the Authorization header.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		claims, err := auth.ValidateAccessToken(parts[1])
		if err != nil {
			switch err {
			case auth.ErrTokenExpired:
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired", "code": "TOKEN_EXPIRED"})
			default:
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			}
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}

// RequireRole returns a middleware that allows only users with specific roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		if !auth.HasRole(claims.Role, roles...) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "insufficient permissions",
				"required_roles": roles,
			})
			return
		}

		c.Next()
	}
}

// GetClaims extracts JWT claims from the Gin context.
func GetClaims(c *gin.Context) (*auth.Claims, bool) {
	raw, exists := c.Get(claimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := raw.(*auth.Claims)
	return claims, ok
}
