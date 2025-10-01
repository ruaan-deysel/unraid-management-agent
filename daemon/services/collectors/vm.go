package collectors

import (
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

type VMCollector struct {
	ctx *domain.Context
}

func NewVMCollector(ctx *domain.Context) *VMCollector {
	return &VMCollector{ctx: ctx}
}

func (c *VMCollector) Start(interval time.Duration) {
	logger.Info("Starting vm collector (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.Collect()
	}
}

func (c *VMCollector) Collect() {
	if c.ctx.MockMode {
		logger.Debug("Mock mode: vm collection skipped")
		return
	}
	logger.Debug("Collecting vm data...")
}
