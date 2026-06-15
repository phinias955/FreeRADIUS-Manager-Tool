package models

import "time"

// AppUser represents an application admin/operator user.
type AppUser struct {
	ID             int        `json:"id" db:"id"`
	Username       string     `json:"username" db:"username"`
	PasswordHash   string     `json:"-" db:"password_hash"`
	Email          string     `json:"email" db:"email"`
	FullName       string     `json:"full_name" db:"full_name"`
	Role           string     `json:"role" db:"role"`
	MFASecret      string     `json:"-" db:"mfa_secret"`
	MFAEnabled     bool       `json:"mfa_enabled" db:"mfa_enabled"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	LastLogin      *time.Time `json:"last_login" db:"last_login"`
	FailedAttempts int        `json:"-" db:"failed_attempts"`
	LockedUntil    *time.Time `json:"-" db:"locked_until"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateAdminUserRequest is the request body for creating an admin user.
type CreateAdminUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50,alphanum"`
	Password string `json:"password" binding:"required,min=12"`
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=super_admin admin operator"`
}

// UpdateAdminUserRequest is the request body for updating an admin user.
type UpdateAdminUserRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	FullName string `json:"full_name"`
	Role     string `json:"role" binding:"omitempty,oneof=super_admin admin operator"`
	IsActive *bool  `json:"is_active"`
}

// LoginRequest is the login payload.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	MFACode  string `json:"mfa_code"`
}

// ChangePasswordRequest is used for changing one's own password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=12"`
}

// UpdateProfileRequest is used for updating the current user's own profile.
type UpdateProfileRequest struct {
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name" binding:"required,min=2,max=100"`
}

// RefreshRequest carries the refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// TokenResponse is returned on successful login.
type TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	User         *AppUser `json:"user"`
}
