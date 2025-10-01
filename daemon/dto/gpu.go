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
