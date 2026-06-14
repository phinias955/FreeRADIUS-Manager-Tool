package models

import "time"

// RadiusUser represents a RADIUS network-access user.
type RadiusUser struct {
	ID                  int        `json:"id"`
	Username            string     `json:"username"`
	Password            string     `json:"-"`
	Email               string     `json:"email"`
	FullName            string     `json:"full_name"`
	Department          string     `json:"department"`
	Status              string     `json:"status"`
	DeviceLimit         int        `json:"device_limit"`
	AccountExpiry       *time.Time `json:"account_expiry"`
	PasswordExpiry      *time.Time `json:"password_expiry"`
	ForcePasswordChange bool       `json:"force_password_change"`
	CreatedBy           *int       `json:"created_by"`
	CreatedByUsername   string     `json:"created_by_username,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	ActiveSessions      int        `json:"active_sessions,omitempty"`
}

// CreateRadiusUserRequest is the request body for creating a RADIUS user.
type CreateRadiusUserRequest struct {
	Username            string  `json:"username" binding:"required,min=2,max=64"`
	Password            string  `json:"password" binding:"required,min=6"`
	Email               string  `json:"email" binding:"omitempty,email"`
	FullName            string  `json:"full_name"`
	Department          string  `json:"department"`
	DeviceLimit         int     `json:"device_limit" binding:"omitempty,min=1,max=500"`
	AccountExpiry       *string `json:"account_expiry"`
	ForcePasswordChange bool    `json:"force_password_change"`
}

// UpdateRadiusUserRequest is the request body for updating a RADIUS user.
type UpdateRadiusUserRequest struct {
	Email               string  `json:"email" binding:"omitempty,email"`
	FullName            string  `json:"full_name"`
	Department          string  `json:"department"`
	DeviceLimit         *int    `json:"device_limit" binding:"omitempty,min=1,max=500"`
	AccountExpiry       *string `json:"account_expiry"`
	ForcePasswordChange *bool   `json:"force_password_change"`
}

// ResetPasswordRequest is used to reset a RADIUS user's password.
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// RadiusUserSession represents an active session for a RADIUS user.
type RadiusUserSession struct {
	SessionID       string    `json:"session_id"`
	Username        string    `json:"username"`
	NASIPAddress    string    `json:"nas_ip"`
	NASPortID       string    `json:"nas_port"`
	FramedIPAddress string    `json:"framed_ip"`
	CallingStation  string    `json:"calling_station"`
	CalledStation   string    `json:"called_station"`
	StartTime       time.Time `json:"start_time"`
	SessionDuration int64     `json:"session_duration"`
	InputOctets     int64     `json:"input_octets"`
	OutputOctets    int64     `json:"output_octets"`
}

// ImportCSVUser represents a single row from a CSV bulk import.
type ImportCSVUser struct {
	Username    string `csv:"username"`
	Password    string `csv:"password"`
	Email       string `csv:"email"`
	FullName    string `csv:"full_name"`
	Department  string `csv:"department"`
	DeviceLimit int    `csv:"device_limit"`
}
