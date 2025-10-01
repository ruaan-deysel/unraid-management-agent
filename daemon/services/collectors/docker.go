package collectors

import (
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

type DockerCollector struct {
	ctx *domain.Context
}

func NewDockerCollector(ctx *domain.Context) *DockerCollector {
	return &DockerCollector{ctx: ctx}
}

func (c *DockerCollector) Start(interval time.Duration) {
	logger.Info("Starting docker collector (interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.Collect()
	}
}

func (c *DockerCollector) Collect() {
	if c.ctx.MockMode {
		logger.Debug("Mock mode: docker collection skipped")
		return
	}
	logger.Debug("Collecting docker data...")
}
