package collectors

import (
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

type ShareCollector struct {
	ctx *domain.Context
}

func NewShareCollector(ctx *domain.Context) *ShareCollector {
	return &ShareCollector{ctx: ctx}
}

func (c *ShareCollector) Start(interval time.Duration) {
	logger.Info("Starting share collector (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.Collect()
	}
}

func (c *ShareCollector) Collect() {
	if c.ctx.MockMode {
		logger.Debug("Mock mode: share collection skipped")
		return
	}
	logger.Debug("Collecting share data...")
}
