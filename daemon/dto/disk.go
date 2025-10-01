package dto

import "time"

// DiskInfo contains disk information
type DiskInfo struct {
	ID            string    `json:"id"`
	Device        string    `json:"device"`
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	Size          uint64    `json:"size_bytes"`
	Used          uint64    `json:"used_bytes"`
	Free          uint64    `json:"free_bytes"`
	Temperature   float64   `json:"temperature_celsius"`
	SMARTStatus   string    `json:"smart_status"`
	SMARTErrors   int       `json:"smart_errors"`
	SpindownDelay int       `json:"spindown_delay"`
	FileSystem    string    `json:"filesystem"`
	Timestamp     time.Time `json:"timestamp"`
}
