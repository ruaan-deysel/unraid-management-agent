package collectors

import (
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

type SystemCollector struct {
	ctx *domain.Context
}

func NewSystemCollector(ctx *domain.Context) *SystemCollector {
	return &SystemCollector{ctx: ctx}
}

func (c *SystemCollector) Start(interval time.Duration) {
	logger.Info("Starting system collector (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.Collect()
	}
}

func (c *SystemCollector) Collect() {
	if c.ctx.MockMode {
		logger.Debug("Mock mode: system collection skipped")
		return
	}
	logger.Debug("Collecting system data...")
}
