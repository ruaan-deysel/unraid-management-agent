package collectors

import (
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

type DiskCollector struct {
	ctx *domain.Context
}

func NewDiskCollector(ctx *domain.Context) *DiskCollector {
	return &DiskCollector{ctx: ctx}
}

func (c *DiskCollector) Start(interval time.Duration) {
	logger.Info("Starting disk collector (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.Collect()
	}
}

func (c *DiskCollector) Collect() {
	if c.ctx.MockMode {
		logger.Debug("Mock mode: disk collection skipped")
		return
	}
	logger.Debug("Collecting disk data...")
}
