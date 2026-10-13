package services

import (
	"github.com/tinkerologie/unraid-management-agent/daemon/constants"
	"github.com/tinkerologie/unraid-management-agent/daemon/dto"
	"github.com/tinkerologie/unraid-management-agent/daemon/logger"
	"github.com/tinkerologie/unraid-management-agent/daemon/services/collectors"
)

// autoDisableAfter is the number of consecutive "absent" results that disable
// an auto-enabled collector, so one flaky check does not stop it.
const autoDisableAfter = 2

// collectorAutoPolicy enables and disables collectors in automatic mode from
// each capability snapshot (#261): present enables the collector at
// IntervalAutoEnable (x4 in low power mode), absent twice in a row disables
// it, and unknown or missing results change nothing. Collectors not in
// automatic mode (explicit intervals, the disabled list, manual runtime
// changes) are never touched. Apply is called with refreshes serialised.
type collectorAutoPolicy struct {
	cm       *CollectorManager
	lowPower bool
	absent   map[string]int
}

func newCollectorAutoPolicy(cm *CollectorManager, lowPower bool) *collectorAutoPolicy {
	return &collectorAutoPolicy{cm: cm, lowPower: lowPower, absent: map[string]int{}}
}

// Apply applies the policy to one capability snapshot.
func (p *collectorAutoPolicy) Apply(caps dto.Capabilities) {
	interval := constants.IntervalAutoEnable
	if p.lowPower {
		interval *= 4
	}
	for _, class := range collectors.HardwareClasses {
		if class.Collector == "" {
			continue
		}
		status, err := p.cm.GetStatus(class.Collector)
		if err != nil || !status.Auto {
			continue
		}
		item, ok := caps.Item(class.Capability)
		if !ok {
			continue
		}
		switch item.State {
		case dto.CapabilityPresent:
			p.absent[class.Collector] = 0
			if p.cm.SetAutoInterval(class.Collector, interval) {
				logger.Info("Auto collectors: enabled %s, %s detected (%s)", class.Collector, class.Capability, item.Detail)
			}
		case dto.CapabilityAbsent:
			p.absent[class.Collector]++
			if p.absent[class.Collector] >= autoDisableAfter && p.cm.SetAutoInterval(class.Collector, 0) {
				logger.Info("Auto collectors: disabled %s, %s not detected (%s)", class.Collector, class.Capability, item.Detail)
			}
		default:
			p.absent[class.Collector] = 0
		}
	}
}
