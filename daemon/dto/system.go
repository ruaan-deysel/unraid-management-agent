package dto

import "time"

// SystemInfo contains system-level metrics
type SystemInfo struct {
	Hostname        string    `json:"hostname"`
	Version         string    `json:"version"`
	Uptime          int64     `json:"uptime_seconds"`
	CPUUsage        float64   `json:"cpu_usage_percent"`
	RAMUsage        float64   `json:"ram_usage_percent"`
	RAMTotal        uint64    `json:"ram_total_bytes"`
	RAMUsed         uint64    `json:"ram_used_bytes"`
	RAMFree         uint64    `json:"ram_free_bytes"`
	CPUTemp         float64   `json:"cpu_temp_celsius"`
	MotherboardTemp float64   `json:"motherboard_temp_celsius"`
	Fans            []FanInfo `json:"fans"`
	Timestamp       time.Time `json:"timestamp"`
}

// FanInfo contains fan speed information
type FanInfo struct {
	Name string `json:"name"`
	RPM  int    `json:"rpm"`
}
