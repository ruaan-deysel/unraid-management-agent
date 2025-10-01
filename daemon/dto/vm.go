package dto

import "time"

// VMInfo contains virtual machine information
type VMInfo struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	State           string    `json:"state"`
	CPUCount        int       `json:"cpu_count"`
	MemoryAllocated uint64    `json:"memory_allocated_bytes"`
	MemoryUsed      uint64    `json:"memory_used_bytes"`
	DiskPath        string    `json:"disk_path"`
	DiskSize        uint64    `json:"disk_size_bytes"`
	Autostart       bool      `json:"autostart"`
	PersistentState bool      `json:"persistent"`
	Timestamp       time.Time `json:"timestamp"`
}
