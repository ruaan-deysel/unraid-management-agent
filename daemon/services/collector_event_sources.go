package services

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/tinkerologie/unraid-management-agent/daemon/constants"
	"github.com/tinkerologie/unraid-management-agent/daemon/services/collectors"
)

// Debounce windows of the file-based change sources. Plugin installs write
// several files, so they get a longer window.
const (
	fileEventDebounce   = 500 * time.Millisecond
	pluginEventDebounce = 5 * time.Second
)

// defaultEventSources returns the change-event sources that trigger a
// collector in Dynamic mode. Each event-driven collector also keeps its
// safety-net poll, because events can be missed.
func defaultEventSources(name string) []eventSource {
	switch name {
	case "array", "disk":
		return []eventSource{
			collectors.NewFileEventSource(fileEventDebounce, constants.VarIni),
			disksIniChanges(),
		}
	case "shares":
		return []eventSource{collectors.NewFileEventSource(fileEventDebounce, constants.SharesIni)}
	case "mover":
		return []eventSource{collectors.NewFileEventSource(fileEventDebounce, constants.VarIni)}
	case "notification":
		unread, archive := collectors.ResolveNotificationDirs(constants.DynamixCfg)
		return []eventSource{
			collectors.NewDirEventSource(fileEventDebounce, unread, ""),
			collectors.NewDirEventSource(fileEventDebounce, archive, ""),
		}
	case "unassigned":
		return []eventSource{
			disksIniChanges(),
			collectors.NewDirEventSource(fileEventDebounce, filepath.Dir(constants.UnassignedSambaMountCfg), ""),
			collectors.NewUeventSource(collectors.BlockDeviceUevent),
		}
	case "plugin_update":
		return []eventSource{collectors.NewDirEventSource(pluginEventDebounce, constants.PluginsConfigDir, "*.plg")}
	case "docker":
		return []eventSource{collectors.NewDockerEventSource()}
	case "vm":
		return []eventSource{collectors.NewLibvirtEventSource()}
	case "hardware":
		return []eventSource{collectors.NewUeventSource(collectors.HardwareDeviceUevent)}
	default:
		return nil
	}
}

// disksIniVolatileKeys are the disks.ini keys emhttp updates every few
// seconds. Changes to them alone do not trigger a collection; the active and
// safety-net polls pick them up.
var disksIniVolatileKeys = []string{"numReads", "numWrites", "temp"}

// disksIniChanges triggers on disks.ini changes other than its I/O counters
// and temperatures: emhttp rewrites the file every few seconds to update them.
func disksIniChanges() eventSource {
	return &contentChanges{
		src:    collectors.NewFileEventSource(fileEventDebounce, constants.DisksIni),
		file:   constants.DisksIni,
		ignore: disksIniVolatileKeys,
	}
}

// contentChanges passes on its source's triggers only when the file's
// content, without the ignored INI keys, changed since the last trigger.
type contentChanges struct {
	src    eventSource
	file   string
	ignore []string

	mu   sync.Mutex
	last [sha256.Size]byte
}

// Run runs the source until ctx is cancelled.
func (c *contentChanges) Run(ctx context.Context, trigger func()) {
	c.mu.Lock()
	c.last = c.digest()
	c.mu.Unlock()
	c.src.Run(ctx, func() {
		c.mu.Lock()
		d := c.digest()
		changed := d != c.last
		c.last = d
		c.mu.Unlock()
		if changed {
			trigger()
		}
	})
}

// digest hashes the file without the ignored keys; an unreadable file hashes
// as empty, so it changing to or from unreadable is a change.
func (c *contentChanges) digest() [sha256.Size]byte {
	data, err := os.ReadFile(c.file)
	if err != nil {
		return sha256.Sum256(nil)
	}
	var kept bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		key, _, _ := strings.Cut(line, "=")
		if !slices.Contains(c.ignore, strings.TrimSpace(key)) {
			kept.WriteString(line)
			kept.WriteByte('\n')
		}
	}
	return sha256.Sum256(kept.Bytes())
}
