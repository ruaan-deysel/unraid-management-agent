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
