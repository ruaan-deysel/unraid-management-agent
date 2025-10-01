package collectors

import (
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

type ArrayCollector struct {
	ctx *domain.Context
}

func NewArrayCollector(ctx *domain.Context) *ArrayCollector {
	return &ArrayCollector{ctx: ctx}
}

func (c *ArrayCollector) Start(interval time.Duration) {
	logger.Info("Starting array collector (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.Collect()
	}
}

func (c *ArrayCollector) Collect() {
	if c.ctx.MockMode {
		logger.Debug("Mock mode: array collection skipped")
		return
	}
	logger.Debug("Collecting array data...")
}
