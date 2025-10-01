package cmd

import (
	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/services"
)

type Boot struct{}

func (b *Boot) Run(ctx *domain.Context) error {
	return services.CreateOrchestrator(ctx).Run()
}
