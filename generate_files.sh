#!/bin/bash
# This script generates all remaining source files for the Unraid Management Agent

set -e

PROJECT_DIR="/Users/ruaandeysel/Github/unraid-management-agent"
cd "$PROJECT_DIR"

echo "Generating remaining DTO files..."

# Array DTO
cat > daemon/dto/array.go << 'EOF'
package dto

import "time"

// ArrayStatus contains Unraid array status information
type ArrayStatus struct {
	State                string    `json:"state"`
	UsedPercent          float64   `json:"used_percent"`
	FreeBytes            uint64    `json:"free_bytes"`
	TotalBytes           uint64    `json:"total_bytes"`
	ParityValid          bool      `json:"parity_valid"`
	ParityCheckStatus    string    `json:"parity_check_status"`
	ParityCheckProgress  float64   `json:"parity_check_progress"`
	NumDisks             int       `json:"num_disks"`
	NumDataDisks         int       `json:"num_data_disks"`
	NumParityDisks       int       `json:"num_parity_disks"`
	Timestamp            time.Time `json:"timestamp"`
}
EOF

# Disk DTO
cat > daemon/dto/disk.go << 'EOF'
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
EOF

# Docker DTO
cat > daemon/dto/docker.go << 'EOF'
package dto

import "time"

// ContainerInfo contains Docker container information
type ContainerInfo struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Image       string        `json:"image"`
	State       string        `json:"state"`
	Status      string        `json:"status"`
	CPUPercent  float64       `json:"cpu_percent"`
	MemoryUsage uint64        `json:"memory_usage_bytes"`
	MemoryLimit uint64        `json:"memory_limit_bytes"`
	NetworkRX   uint64        `json:"network_rx_bytes"`
	NetworkTX   uint64        `json:"network_tx_bytes"`
	Ports       []PortMapping `json:"ports"`
	Timestamp   time.Time     `json:"timestamp"`
}

// PortMapping represents a port mapping
type PortMapping struct {
	PrivatePort int    `json:"private_port"`
	PublicPort  int    `json:"public_port"`
	Type        string `json:"type"`
}
EOF

# VM DTO
cat > daemon/dto/vm.go << 'EOF'
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
EOF

# UPS DTO
cat > daemon/dto/ups.go << 'EOF'
package dto

import "time"

// UPSStatus contains UPS status information
type UPSStatus struct {
	Connected     bool      `json:"connected"`
	Status        string    `json:"status"`
	LoadPercent   float64   `json:"load_percent"`
	BatteryCharge float64   `json:"battery_charge_percent"`
	RuntimeLeft   int       `json:"runtime_left_seconds"`
	PowerWatts    float64   `json:"power_watts"`
	NominalPower  float64   `json:"nominal_power_watts"`
	Model         string    `json:"model"`
	Timestamp     time.Time `json:"timestamp"`
}
EOF

# GPU DTO
cat > daemon/dto/gpu.go << 'EOF'
package dto

import "time"

// GPUMetrics contains GPU metrics
type GPUMetrics struct {
	Available         bool      `json:"available"`
	Name              string    `json:"name"`
	DriverVersion     string    `json:"driver_version"`
	Temperature       float64   `json:"temperature_celsius"`
	UtilizationGPU    float64   `json:"utilization_gpu_percent"`
	UtilizationMemory float64   `json:"utilization_memory_percent"`
	MemoryTotal       uint64    `json:"memory_total_bytes"`
	MemoryUsed        uint64    `json:"memory_used_bytes"`
	PowerDraw         float64   `json:"power_draw_watts"`
	Timestamp         time.Time `json:"timestamp"`
}
EOF

# Share DTO
cat > daemon/dto/share.go << 'EOF'
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
EOF

# WebSocket DTO
cat > daemon/dto/websocket.go << 'EOF'
package dto

import "time"

// WSEvent represents a WebSocket event
type WSEvent struct {
	Event     string      `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// Response represents a standard API response
type Response struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}
EOF

echo "DTO files generated successfully!"
echo ""
echo "Note: This script has created the DTO layer."
echo "To complete the project, you still need to implement:"
echo "  1. HTTP/WebSocket server (daemon/services/api/)"
echo "  2. Data collectors (daemon/services/collectors/)"
echo "  3. Controllers (daemon/services/controllers/)"
echo "  4. Service orchestrator (daemon/services/orchestrator.go)"
echo "  5. Plugin packaging files (meta/)"
echo "  6. Documentation (docs/ and README.md)"
echo ""
echo "Run this script with: bash generate_files.sh"
