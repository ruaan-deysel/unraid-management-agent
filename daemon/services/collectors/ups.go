package collectors

import (
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

type UPSCollector struct {
	ctx *domain.Context
}

func NewUPSCollector(ctx *domain.Context) *UPSCollector {
	return &UPSCollector{ctx: ctx}
}

func (c *UPSCollector) Start(interval time.Duration) {
	logger.Info("Starting ups collector (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.Collect()
	}
}

func (c *UPSCollector) Collect() {
	if c.ctx.MockMode {
		logger.Debug("Mock mode: ups collection skipped")
		return
	}
	logger.Debug("Collecting ups data...")
}
