package domain

import (
	"github.com/tinkerologie/unraid-management-agent/daemon/logger"
	"github.com/tinkerologie/unraid-management-agent/daemon/platform"
)

// Intervals holds collection interval settings in seconds.
type Intervals struct {
	System         int
	Array          int
	Disk           int
	Docker         int
	VM             int
	UPS            int
	NUT            int
	GPU            int
	Shares         int
	Network        int
	Hardware       int
	ZFS            int
	Notification   int
	Registration   int
	Unassigned     int
	FanControl     int
	Tuning         int
	DockerUpdate   int
	DockerNetworks int
	PluginUpdate   int
	OSUpdate       int
	Mover          int

	// StorageTopology is the storage_topology collector interval in seconds.
	StorageTopology int
	// NIC is the nic collector interval in seconds.
	NIC int
}

// Context holds the application runtime context including the event hub and configuration.
type Context struct {
	Hub                *EventBus
	Platform           *platform.Registry
	Intervals          Intervals
	MQTTConfig         MQTTConfig
	DiscoveryConfig    DiscoveryConfig
	DiagnosticLogger   *logger.DiagnosticLogger
	LogsDir            string
	DockerUpdateNotify bool
	// AutoCollectors are the hardware-gated collectors in automatic mode: they
	// start at interval 0 and run only while their hardware is detected (#261).
	AutoCollectors map[string]bool
	// LowPowerMode multiplies collector intervals by 4 (also auto-enabled ones).
	LowPowerMode bool
	Config
}
