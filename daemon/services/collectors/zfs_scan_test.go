package collectors

import (
	"math"
	"testing"
	"time"

	"github.com/tinkerologie/unraid-management-agent/daemon/dto"
)

func zfsTime(year int, month time.Month, day, h, m, s int) time.Time {
	return time.Date(year, month, day, h, m, s, 0, time.Local)
}

func TestParseScanInfoFormats(t *testing.T) {
	end := zfsTime(2026, time.September, 17, 22, 6, 13)
	tests := []struct {
		name     string
		line     string
		status   string
		state    string
		repaired uint64
		errors   int
		start    time.Time
		end      time.Time
		pct      float64
	}{
		{
			name: "completed scrub with days", status: "scrub completed", state: "finished", pct: 100,
			line:  "scan: scrub repaired 0B in 2 days 00:06:12 with 0 errors on Thu Sep 17 22:06:13 2026",
			start: end.Add(-(48*time.Hour + 6*time.Minute + 12*time.Second)), end: end,
		},
		{
			name: "completed scrub with repairs and errors", status: "scrub completed", state: "finished", pct: 100,
			line:     "scan: scrub repaired 512K in 01:23:45 with 5 errors on Thu Sep 17 22:06:13 2026",
			repaired: 512 * 1024, errors: 5,
			start: end.Add(-(time.Hour + 23*time.Minute + 45*time.Second)), end: end,
		},
		{
			name: "completed resilver", status: "resilver completed", state: "finished", pct: 100,
			line:     "scan: resilvered 1.21T in 10:11:12 with 0 errors on Thu Sep 17 22:06:13 2026",
			repaired: uint64(math.Round(1.21 * math.Pow(1024, 4))),
			start:    end.Add(-(10*time.Hour + 11*time.Minute + 12*time.Second)), end: end,
		},
		{
			name: "canceled scrub", status: "scrub canceled", state: "canceled",
			line: "scan: scrub canceled on Thu Sep 17 22:06:13 2026", end: end,
		},
		{
			name: "canceled resilver", status: "resilver canceled", state: "canceled",
			line: "scan: resilver canceled on Thu Sep 17 22:06:13 2026", end: end,
		},
		{
			name: "scrub in progress, ctime single-digit day", status: "scrub in progress", state: "scanning",
			line:  "scan: scrub in progress since Sun Nov  9 02:39:43 2025",
			start: zfsTime(2025, time.November, 9, 2, 39, 43),
		},
		{
			name: "scrub paused", status: "scrub paused", state: "paused",
			line: "scan: scrub paused since Thu Sep 17 22:06:13 2026",
		},
		{name: "none requested", line: "scan: none requested"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &dto.ZFSPool{}
			parseScanInfo(pool, tt.line)
			if pool.ScanStatus != tt.status || pool.ScanState != tt.state {
				t.Errorf("status/state = %q/%q, want %q/%q", pool.ScanStatus, pool.ScanState, tt.status, tt.state)
			}
			if pool.ScanRepairedBytes != tt.repaired || pool.ScanErrors != tt.errors || pool.ScanProgressPct != tt.pct {
				t.Errorf("repaired/errors/pct = %d/%d/%v, want %d/%d/%v",
					pool.ScanRepairedBytes, pool.ScanErrors, pool.ScanProgressPct, tt.repaired, tt.errors, tt.pct)
			}
			if !pool.ScanStartTime.Equal(tt.start) || !pool.ScanEndTime.Equal(tt.end) {
				t.Errorf("start/end = %v/%v, want %v/%v", pool.ScanStartTime, pool.ScanEndTime, tt.start, tt.end)
			}
		})
	}
}

func TestParseZPoolStatusScanProgress(t *testing.T) {
	inProgress := `  pool: tank
 state: ONLINE
  scan: scrub in progress since Thu Sep 17 22:06:13 2026
	1.23T / 4.56T scanned at 1.2G/s, 1.00T / 4.56T issued at 1.0G/s
	64K repaired, 21.93% done, 01:02:03 to go
config:

	NAME        STATE     READ WRITE CKSUM
	tank        ONLINE       0     0     0
	  sdb       ONLINE       0     0     0

errors: No known data errors
`
	pool := &dto.ZFSPool{}
	if err := parseZPoolStatusOutput(pool, inProgress); err != nil {
		t.Fatal(err)
	}
	if pool.ScanStatus != "scrub in progress" || pool.ScanProgressPct != 21.93 || pool.ScanRepairedBytes != 64*1024 {
		t.Errorf("in progress: status=%q pct=%v repaired=%d", pool.ScanStatus, pool.ScanProgressPct, pool.ScanRepairedBytes)
	}
	if !pool.ScanStartTime.Equal(zfsTime(2026, time.September, 17, 22, 6, 13)) {
		t.Errorf("in progress start = %v", pool.ScanStartTime)
	}
	if len(pool.VDEVs) == 0 {
		t.Error("vdevs after the scan block were not parsed")
	}

	paused := `  pool: tank
 state: ONLINE
  scan: scrub paused since Thu Sep 17 22:06:13 2026
	scrub started on Wed Sep 16 08:00:00 2026
	2.00T / 4.56T scanned, 1.50T / 4.56T issued
	0B repaired, 32.89% done
config:
`
	pool = &dto.ZFSPool{}
	if err := parseZPoolStatusOutput(pool, paused); err != nil {
		t.Fatal(err)
	}
	if pool.ScanStatus != "scrub paused" || pool.ScanState != "paused" || pool.ScanProgressPct != 32.89 {
		t.Errorf("paused: status=%q state=%q pct=%v", pool.ScanStatus, pool.ScanState, pool.ScanProgressPct)
	}
	if !pool.ScanStartTime.Equal(zfsTime(2026, time.September, 16, 8, 0, 0)) {
		t.Errorf("paused start = %v", pool.ScanStartTime)
	}
}

func TestParseZFSSize(t *testing.T) {
	tests := map[string]uint64{"0B": 0, "512B": 512, "4K": 4096, "1.5M": 1572864, "2G": 2 << 30, "1T": 1 << 40}
	for in, want := range tests {
		if got, ok := parseZFSSize(in); !ok || got != want {
			t.Errorf("parseZFSSize(%q) = %d, %v; want %d", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "abc", "-1K"} {
		if _, ok := parseZFSSize(bad); ok {
			t.Errorf("parseZFSSize(%q) accepted invalid input", bad)
		}
	}
}

func TestParseScanDuration(t *testing.T) {
	tests := map[string]time.Duration{
		"00:00:01":        time.Second,
		"01:23:45":        time.Hour + 23*time.Minute + 45*time.Second,
		"1 days 00:00:00": 24 * time.Hour,
		"2 days 00:06:12": 48*time.Hour + 6*time.Minute + 12*time.Second,
	}
	for in, want := range tests {
		if got, ok := parseScanDuration(in); !ok || got != want {
			t.Errorf("parseScanDuration(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	if _, ok := parseScanDuration("soon"); ok {
		t.Error("parseScanDuration accepted invalid input")
	}
}
