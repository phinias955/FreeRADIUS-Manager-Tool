package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"strconv"
	"time"
)

// GenerateSessionID creates a unique session identifier stored in JWT and refresh_tokens.
func GenerateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SessionTimeout returns how long a session stays valid with no API activity before logout.
func SessionTimeout() time.Duration {
	raw := os.Getenv("SESSION_TIMEOUT")
	if raw == "" {
		return time.Hour
	}
	secs, err := strconv.Atoi(raw)
	if err != nil || secs < 60 {
		return time.Hour
	}
	return time.Duration(secs) * time.Second
}

// SessionConcurrentBlock returns how recently the account must have been used to
// reject a second login. After this idle period the previous session is treated
// as closed (browser tab closed without sign-out).
func SessionConcurrentBlock() time.Duration {
	raw := os.Getenv("SESSION_CONCURRENT_BLOCK")
	if raw == "" {
		return 5 * time.Minute
	}
	if d, err := time.ParseDuration(raw); err == nil && d >= time.Minute {
		return d
	}
	if secs, err := strconv.Atoi(raw); err == nil && secs >= 60 {
		return time.Duration(secs) * time.Second
	}
	return 5 * time.Minute
}

// RevokeAllUserSessions invalidates every active session for a user.
func RevokeAllUserSessions(db *sql.DB, userID int) {
	db.Exec(`UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = $1 AND revoked = FALSE`, userID)
}

// HasActiveSession reports whether the user is actively signed in elsewhere
// (API activity within the concurrent block window).
func HasActiveSession(db *sql.DB, userID int) (bool, error) {
	cutoff := time.Now().Add(-SessionConcurrentBlock())
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM refresh_tokens
			WHERE user_id = $1
			  AND revoked = FALSE
			  AND session_id <> ''
			  AND expires_at > NOW()
			  AND last_activity > $2
		)`, userID, cutoff).Scan(&exists)
	return exists, err
}

// ValidateSession checks that the JWT session is still the user's active login.
func ValidateSession(db *sql.DB, userID int, sessionID string) (bool, error) {
	if sessionID == "" {
		return false, nil
	}
	idleCutoff := time.Now().Add(-SessionTimeout())
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM refresh_tokens
			WHERE user_id = $1
			  AND session_id = $2
			  AND revoked = FALSE
			  AND expires_at > NOW()
			  AND last_activity > $3
		)`, userID, sessionID, idleCutoff).Scan(&exists)
	return exists, err
}

// TouchSession updates last_activity for the active session.
func TouchSession(db *sql.DB, userID int, sessionID string) {
	if sessionID == "" {
		return
	}
	db.Exec(`
		UPDATE refresh_tokens SET last_activity = NOW()
		WHERE user_id = $1 AND session_id = $2 AND revoked = FALSE`,
		userID, sessionID)
}
