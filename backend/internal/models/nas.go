package models

import "time"

// NAS represents a Network Access Server (router, AP, VPN gateway).
type NAS struct {
	ID          int       `json:"id"`
	NASName     string    `json:"nasname"`
	ShortName   string    `json:"shortname"`
	Type        string    `json:"type"`
	Ports       *int      `json:"ports"`
	Secret      string    `json:"secret,omitempty"`
	Server      string    `json:"server"`
	Community   string    `json:"community"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateNASRequest is the request body for adding a NAS client.
type CreateNASRequest struct {
	NASName     string `json:"nasname" binding:"required"`
	ShortName   string `json:"shortname" binding:"required,max=32"`
	Type        string `json:"type"`
	Ports       *int   `json:"ports"`
	Secret      string `json:"secret" binding:"required,min=8"`
	Server      string `json:"server"`
	Community   string `json:"community"`
	Description string `json:"description"`
}

// UpdateNASRequest is the request body for updating a NAS client.
type UpdateNASRequest struct {
	ShortName   string `json:"shortname"`
	Type        string `json:"type"`
	Ports       *int   `json:"ports"`
	Secret      string `json:"secret" binding:"omitempty,min=8"`
	Server      string `json:"server"`
	Community   string `json:"community"`
	Description string `json:"description"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive"`
}

// NASTestResult is returned after testing a NAS connection.
type NASTestResult struct {
	Success     bool    `json:"success"`
	Message     string  `json:"message"`
	LatencyMs   float64 `json:"latency_ms"`
	AuthTime    float64 `json:"auth_time_ms"`
	NASName     string  `json:"nasname"`
	RadiusPort  int     `json:"radius_port"`
}

// DiscoverRequest specifies a subnet to scan for NAS devices.
type DiscoverRequest struct {
	Subnet string `json:"subnet" binding:"required"`
}
