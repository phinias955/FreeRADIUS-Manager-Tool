package models

import "time"

// NetworkScanRequest starts an nmap-based network discovery scan.
type NetworkScanRequest struct {
	Subnet   string `json:"subnet" binding:"required"`
	ScanType string `json:"scan_type" binding:"omitempty,oneof=ping discovery standard ap full"`
}

// NetworkScan represents a stored scan job.
type NetworkScan struct {
	ID           int        `json:"id"`
	Subnet       string     `json:"subnet"`
	ScanType     string     `json:"scan_type"`
	Status       string     `json:"status"`
	HostCount    int        `json:"host_count"`
	ErrorMessage *string    `json:"error_message"`
	StartedBy    *int       `json:"started_by"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
}

// NetworkScanHost is a device discovered during a scan.
type NetworkScanHost struct {
	ID              int       `json:"id"`
	ScanID          int       `json:"scan_id"`
	IPAddress       string    `json:"ip_address"`
	Hostname        string    `json:"hostname"`
	MACAddress      string    `json:"mac_address"`
	Vendor          string    `json:"vendor"`
	DeviceType      string    `json:"device_type"`
	OSGuess         string    `json:"os_guess"`
	OpenPorts       []int     `json:"open_ports"`
	IsAccessPoint   bool      `json:"is_access_point"`
	IsRadiusCapable bool      `json:"is_radius_capable"`
	LatencyMs       float64   `json:"latency_ms"`
	CreatedAt       time.Time `json:"created_at"`
}
