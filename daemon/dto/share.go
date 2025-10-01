package dto

import "time"

// ShareInfo contains share information
type ShareInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Used      uint64    `json:"used_bytes"`
	Free      uint64    `json:"free_bytes"`
	Total     uint64    `json:"total_bytes"`
	Timestamp time.Time `json:"timestamp"`
}
