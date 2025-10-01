package collectors

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/dto"
	"github.com/ruaandeysel/unraid-management-agent/daemon/lib"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

type GPUCollector struct {
	ctx *domain.Context
}

func NewGPUCollector(ctx *domain.Context) *GPUCollector {
	return &GPUCollector{ctx: ctx}
}

func (c *GPUCollector) Start(interval time.Duration) {
	logger.Info("Starting gpu collector (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.Collect()
	}
}

func (c *GPUCollector) Collect() {
	if c.ctx.MockMode {
		logger.Debug("Mock mode: gpu collection skipped")
		return
	}

	logger.Debug("Collecting gpu data...")

	// Check if nvidia-smi is available
	if !lib.CommandExists("nvidia-smi") {
		logger.Debug("nvidia-smi not found, no GPU monitoring available")
		return
	}

	// Collect GPU metrics
	gpuMetrics, err := c.collectNvidiaGPU()
	if err != nil {
		logger.Warning("Failed to collect GPU metrics", "error", err)
		return
	}

	// Publish event
	c.ctx.Hub.Pub(gpuMetrics, "gpu_metrics_update")
	logger.Debug("Published gpu_metrics_update event for %d GPUs", len(gpuMetrics))
}

func (c *GPUCollector) collectNvidiaGPU() ([]*dto.GPUMetrics, error) {
	// Query nvidia-smi with CSV output for easy parsing
	// Format: index, name, temperature.gpu, utilization.gpu, memory.used, memory.total, power.draw
	output, err := lib.ExecCommandOutput(
		"nvidia-smi",
		"--query-gpu=index,name,temperature.gpu,utilization.gpu,memory.used,memory.total,power.draw,power.limit",
		"--format=csv,noheader,nounits",
	)
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi query failed: %w", err)
	}

	// Parse CSV output
	reader := csv.NewReader(strings.NewReader(output))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV output: %w", err)
	}

	gpus := make([]*dto.GPUMetrics, 0, len(records))

	for _, record := range records {
		if len(record) < 8 {
			continue
		}

		gpu := &dto.GPUMetrics{
			Timestamp: time.Now(),
		}

		// Index
		if _, err := strconv.Atoi(strings.TrimSpace(record[0])); err == nil {
			// gpu.Index (not in DTO) = idx
		}

		// Name
		gpu.Name = strings.TrimSpace(record[1])

		// Temperature (°C)
		if temp, err := strconv.ParseFloat(strings.TrimSpace(record[2]), 64); err == nil {
			gpu.Temperature = temp
		}

		// Utilization (%)
		if util, err := strconv.ParseFloat(strings.TrimSpace(record[3]), 64); err == nil {
			gpu.UtilizationGPU = util
		}

		// Memory Used (MiB)
		if memUsed, err := strconv.ParseFloat(strings.TrimSpace(record[4]), 64); err == nil {
			gpu.MemoryUsed = uint64(memUsed * 1024 * 1024) // Convert MiB to bytes
		}

		// Memory Total (MiB)
		if memTotal, err := strconv.ParseFloat(strings.TrimSpace(record[5]), 64); err == nil {
			gpu.MemoryTotal = uint64(memTotal * 1024 * 1024) // Convert MiB to bytes
			if gpu.MemoryTotal > 0 {
				gpu.UtilizationMemory = float64(gpu.MemoryUsed) / float64(gpu.MemoryTotal) * 100
			}
		}

		// Power Draw (W)
		if power, err := strconv.ParseFloat(strings.TrimSpace(record[6]), 64); err == nil {
			gpu.PowerDraw = power
		}

		// Power Limit (W)
		if _, err := strconv.ParseFloat(strings.TrimSpace(record[7]), 64); err == nil {
			// gpu.PowerLimit (not in DTO) = powerLimit
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}
