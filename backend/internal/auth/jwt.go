package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims contains JWT payload fields.
type Claims struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

var (
	ErrTokenExpired   = errors.New("token expired")
	ErrTokenInvalid   = errors.New("token invalid")
	ErrTokenMalformed = errors.New("token malformed")
)

// GenerateAccessToken creates a short-lived JWT for API access.
func GenerateAccessToken(userID int, username, role, sessionID string) (string, error) {
	secret := getJWTSecret()
	expiry := getAccessExpiry()

	claims := Claims{
		UserID:    userID,
		Username:  username,
		Role:      role,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "radius-manager",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken creates an opaque refresh token and its hash for storage.
func GenerateRefreshToken() (token string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	token = hex.EncodeToString(b)
	h := sha256.Sum256([]byte(token))
	hash = hex.EncodeToString(h[:])
	return
}

// HashRefreshToken returns the SHA-256 hash of a refresh token.
func HashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// ValidateAccessToken parses and validates a JWT, returning the claims.
func ValidateAccessToken(tokenString string) (*Claims, error) {
	secret := getJWTSecret()

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenMalformed
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// RefreshExpiry returns the refresh token duration.
func RefreshExpiry() time.Duration {
	raw := os.Getenv("JWT_REFRESH_EXPIRY")
	if raw == "" {
		return 168 * time.Hour
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 168 * time.Hour
	}
	return d
}

func getJWTSecret() string {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		return "change-this-in-production-very-insecure"
	}
	return s
}

// AccessExpiry returns the access token duration.
func AccessExpiry() time.Duration {
	return getAccessExpiry()
}

func getAccessExpiry() time.Duration {
	raw := os.Getenv("JWT_ACCESS_EXPIRY")
	if raw == "" {
		return time.Hour
	}
	// Support both duration strings and plain minutes
	d, err := time.ParseDuration(raw)
	if err != nil {
		mins, err2 := strconv.Atoi(raw)
		if err2 != nil {
			return time.Hour
		}
		return time.Duration(mins) * time.Minute
	}
	return d
}
