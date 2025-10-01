package collectors

import (
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
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
}
