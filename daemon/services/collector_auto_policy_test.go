package services

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tinkerologie/unraid-management-agent/daemon/constants"
	"github.com/tinkerologie/unraid-management-agent/daemon/domain"
	"github.com/tinkerologie/unraid-management-agent/daemon/dto"
	"github.com/tinkerologie/unraid-management-agent/daemon/platform"
	"github.com/tinkerologie/unraid-management-agent/daemon/services/collectors"
)

// newAutoTestManager registers mock collectors: nic and storage_topology in
// automatic mode, gpu with an explicit interval, zfs explicitly disabled.
func newAutoTestManager(t *testing.T) (*CollectorManager, map[string]*mockCollector) {
	t.Helper()
	ctx := &domain.Context{
		Hub:            domain.NewEventBus(100),
		AutoCollectors: map[string]bool{"nic": true, "storage_topology": true, "gpu": true},
	}
	var wg sync.WaitGroup
	cm := NewCollectorManager(ctx, &wg)
	mocks := map[string]*mockCollector{}
	for name, interval := range map[string]int{"nic": 0, "storage_topology": 0, "gpu": 60, "zfs": 0} {
		m := newMockCollector()
		mocks[name] = m
		cm.Register(name, func(*domain.Context) Collector { return m }, interval, false)
	}
	t.Cleanup(func() {
		cm.StopAll()
		wg.Wait()
	})
	return cm, mocks
}

func snapshot(states map[string]dto.CapabilityState) dto.Capabilities {
	caps := dto.Capabilities{}
	for name, state := range states {
		caps.Items = append(caps.Items, dto.Capability{Name: name, State: state, Available: state == dto.CapabilityPresent, Detail: "test"})
	}
	return caps
}

func status(t *testing.T, cm *CollectorManager, name string) *dto.CollectorStatus {
	t.Helper()
	s, err := cm.GetStatus(name)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCollectorManagerAutoMode(t *testing.T) {
	cm, mocks := newAutoTestManager(t)

	if s := status(t, cm, "nic"); !s.Auto || s.Enabled || s.Status != "disabled" {
		t.Fatalf("nic should register as auto and disabled: %+v", s)
	}
	if s := status(t, cm, "gpu"); s.Auto {
		t.Fatalf("gpu has an explicit interval and must not be auto: %+v", s)
	}
	if s := status(t, cm, "zfs"); s.Auto {
		t.Fatalf("zfs is not in the auto set: %+v", s)
	}
	if n := cm.StartAll(); n != 1 {
		t.Fatalf("StartAll started %d collectors, want 1 (gpu)", n)
	}
	if s := status(t, cm, "nic"); s.Status == "running" {
		t.Fatal("auto collector at interval 0 started")
	}

	if !cm.SetAutoInterval("nic", 300) {
		t.Fatal("SetAutoInterval(nic, 300) reported no change")
	}
	mocks["nic"].waitStarted(t)
	if s := status(t, cm, "nic"); s.Status != "running" || s.Interval != 300 || !s.Enabled || !s.Auto {
		t.Fatalf("nic after auto-enable: %+v", s)
	}
	if cm.SetAutoInterval("nic", 300) {
		t.Error("same interval reported a change")
	}
	if !cm.SetAutoInterval("nic", 0) {
		t.Fatal("SetAutoInterval(nic, 0) reported no change")
	}
	mocks["nic"].waitDone(t)
	if s := status(t, cm, "nic"); s.Status != "disabled" || s.Interval != 0 || s.Enabled || !s.Auto {
		t.Fatalf("nic after auto-disable: %+v", s)
	}
	if cm.SetAutoInterval("nic", 0) {
		t.Error("disabling a stopped collector reported a change")
	}

	if cm.SetAutoInterval("gpu", 0) || cm.SetAutoInterval("zfs", 300) || cm.SetAutoInterval("missing", 300) {
		t.Error("SetAutoInterval changed a collector not in automatic mode")
	}
}

func TestCollectorManagerAutoRestartsAtNewInterval(t *testing.T) {
	cm, mocks := newAutoTestManager(t)
	cm.SetAutoInterval("nic", 300)
	mocks["nic"].waitStarted(t)
	if !cm.SetAutoInterval("nic", 1200) {
		t.Fatal("interval change reported no change")
	}
	mocks["nic"].waitStarted(t)
	if s := status(t, cm, "nic"); s.Interval != 1200 || s.Status != "running" {
		t.Fatalf("nic after interval change: %+v", s)
	}
}

func TestCollectorManagerManualControlEndsAutoMode(t *testing.T) {
	tests := []struct {
		name string
		act  func(cm *CollectorManager) error
	}{
		{"enable", func(cm *CollectorManager) error { return cm.EnableCollector("nic") }},
		{"disable", func(cm *CollectorManager) error { return cm.DisableCollector("nic") }},
		{"interval", func(cm *CollectorManager) error { return cm.UpdateInterval("nic", 120) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, _ := newAutoTestManager(t)
			if err := tt.act(cm); err != nil {
				t.Fatal(err)
			}
			if status(t, cm, "nic").Auto {
				t.Fatal("manual control left the collector in automatic mode")
			}
			for _, c := range cm.GetAllStatus().Collectors {
				if c.Name == "storage_topology" && !c.Auto {
					t.Fatal("manual control of nic cleared another collector's auto flag")
				}
			}
			if cm.SetAutoInterval("nic", 300) {
				t.Fatal("policy changed a manually controlled collector")
			}
		})
	}
}

func TestCollectorManagerInvalidIntervalKeepsAutoMode(t *testing.T) {
	cm, _ := newAutoTestManager(t)
	if err := cm.UpdateInterval("nic", 1); err == nil {
		t.Fatal("expected invalid interval error")
	}
	if !status(t, cm, "nic").Auto {
		t.Fatal("a rejected interval change ended automatic mode")
	}
}

func TestCollectorAutoPolicy(t *testing.T) {
	cm, mocks := newAutoTestManager(t)
	policy := newCollectorAutoPolicy(cm, false)
	nic, storage := collectors.CapabilityNICOptics, collectors.CapabilityStorageTopology

	// Reference server: optics and SAS absent. Nothing starts, nothing changes.
	policy.Apply(snapshot(map[string]dto.CapabilityState{nic: dto.CapabilityAbsent, storage: dto.CapabilityAbsent, collectors.CapabilityGPU: dto.CapabilityAbsent}))
	for _, name := range []string{"nic", "storage_topology"} {
		if s := status(t, cm, name); s.Status == "running" || s.Interval != 0 {
			t.Fatalf("%s runs although its hardware is absent: %+v", name, s)
		}
	}
	if s := status(t, cm, "gpu"); s.Interval != 60 {
		t.Fatalf("explicit gpu interval changed: %+v", s)
	}

	// SAS HBA added: storage topology is enabled at the default interval.
	policy.Apply(snapshot(map[string]dto.CapabilityState{nic: dto.CapabilityAbsent, storage: dto.CapabilityPresent}))
	mocks["storage_topology"].waitStarted(t)
	if s := status(t, cm, "storage_topology"); s.Interval != constants.IntervalAutoEnable || s.Status != "running" {
		t.Fatalf("storage_topology after detection: %+v", s)
	}

	// Unknown and missing results change nothing.
	policy.Apply(snapshot(map[string]dto.CapabilityState{storage: dto.CapabilityUnknown}))
	policy.Apply(dto.Capabilities{})
	if s := status(t, cm, "storage_topology"); s.Status != "running" {
		t.Fatalf("unknown result stopped the collector: %+v", s)
	}

	// One absent result keeps it; an unknown in between resets the count.
	policy.Apply(snapshot(map[string]dto.CapabilityState{storage: dto.CapabilityAbsent}))
	policy.Apply(snapshot(map[string]dto.CapabilityState{storage: dto.CapabilityUnknown}))
	policy.Apply(snapshot(map[string]dto.CapabilityState{storage: dto.CapabilityAbsent}))
	if s := status(t, cm, "storage_topology"); s.Status != "running" {
		t.Fatalf("non-consecutive absent results stopped the collector: %+v", s)
	}

	// Two consecutive absent results: the hardware was removed.
	policy.Apply(snapshot(map[string]dto.CapabilityState{storage: dto.CapabilityAbsent}))
	mocks["storage_topology"].waitDone(t)
	if s := status(t, cm, "storage_topology"); s.Status != "disabled" || s.Interval != 0 {
		t.Fatalf("storage_topology after removal: %+v", s)
	}

	// Explicit settings are never touched, even when hardware is present.
	policy.Apply(snapshot(map[string]dto.CapabilityState{collectors.CapabilityGPU: dto.CapabilityPresent, collectors.CapabilityZFS: dto.CapabilityPresent}))
	if s := status(t, cm, "gpu"); s.Interval != 60 {
		t.Fatalf("policy changed explicit gpu interval: %+v", s)
	}
	if s := status(t, cm, "zfs"); s.Status == "running" {
		t.Fatalf("policy enabled explicitly disabled zfs: %+v", s)
	}
}

func TestCollectorAutoPolicyLowPower(t *testing.T) {
	cm, mocks := newAutoTestManager(t)
	newCollectorAutoPolicy(cm, true).Apply(snapshot(map[string]dto.CapabilityState{collectors.CapabilityNICOptics: dto.CapabilityPresent}))
	mocks["nic"].waitStarted(t)
	if s := status(t, cm, "nic"); s.Interval != constants.IntervalAutoEnable*4 {
		t.Fatalf("low power interval = %d, want %d", s.Interval, constants.IntervalAutoEnable*4)
	}
}

// TestHiddenSectionsNeverRun asserts the core rule of #261 for the reference
// server: every hardware class detected as absent leaves its automatic
// collector disabled.
func TestHiddenSectionsNeverRun(t *testing.T) {
	auto := map[string]bool{}
	for _, c := range collectors.HardwareClasses {
		if c.Collector != "" {
			auto[c.Collector] = true
		}
	}
	ctx := &domain.Context{Hub: domain.NewEventBus(100), AutoCollectors: auto}
	var wg sync.WaitGroup
	cm := NewCollectorManager(ctx, &wg)
	for name := range auto {
		cm.Register(name, func(*domain.Context) Collector { return newMockCollector() }, 0, false)
	}
	t.Cleanup(func() { cm.StopAll(); wg.Wait() })
	cm.StartAll()

	absent := map[string]dto.CapabilityState{}
	for _, c := range collectors.HardwareClasses {
		absent[c.Capability] = dto.CapabilityAbsent
	}
	policy := newCollectorAutoPolicy(cm, false)
	policy.Apply(snapshot(absent))
	policy.Apply(snapshot(absent))
	for name := range auto {
		if s := status(t, cm, name); s.Status == "running" || s.Enabled || s.Interval != 0 {
			t.Errorf("%s runs although its hardware is absent: %+v", name, s)
		}
	}
}

func TestCapabilityRefresher(t *testing.T) {
	reg := platform.NewRegistry()
	var checks atomic.Int32
	state := atomic.Value{}
	state.Store(dto.CapabilityAbsent)
	probes := func() []platform.Probe {
		return []platform.Probe{{Name: "hardware.test", Kind: platform.ProbeCheck, Check: func(ctx context.Context) (dto.CapabilityState, string) {
			checks.Add(1)
			if _, ok := ctx.Deadline(); !ok {
				t.Error("check ran without a timeout")
			}
			return state.Load().(dto.CapabilityState), "detail"
		}}}
	}
	r := NewCapabilityRefresher(reg, probes)
	clock := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	r.Now = func() time.Time { return clock }
	var hooked []dto.Capabilities
	r.SetHook(func(c dto.Capabilities) { hooked = append(hooked, c) })

	first := r.Refresh(context.Background())
	if checks.Load() != 1 || len(hooked) != 1 {
		t.Fatalf("first refresh: checks=%d hooks=%d", checks.Load(), len(hooked))
	}
	if it, _ := reg.Capabilities().Item("hardware.test"); it.State != dto.CapabilityAbsent {
		t.Fatalf("registry not updated: %+v", reg.Capabilities())
	}

	// Within MinInterval: cached, no check, no hook.
	state.Store(dto.CapabilityPresent)
	clock = clock.Add(10 * time.Second)
	if c := r.Refresh(context.Background()); checks.Load() != 1 || len(hooked) != 1 || c.Items[0].State != dto.CapabilityAbsent {
		t.Fatalf("rate limit: checks=%d hooks=%d state=%q", checks.Load(), len(hooked), c.Items[0].State)
	}

	// After MinInterval: a new snapshot; the old one is unchanged.
	clock = clock.Add(constants.CapabilityMinRefreshInterval)
	second := r.Refresh(context.Background())
	if second.Items[0].State != dto.CapabilityPresent || len(hooked) != 2 {
		t.Fatalf("second refresh: %+v hooks=%d", second.Items, len(hooked))
	}
	if first.Items[0].State != dto.CapabilityAbsent {
		t.Fatal("a refresh modified an earlier snapshot")
	}

	// A cancelled request context does not leave a half-checked snapshot.
	clock = clock.Add(time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if c := r.Refresh(ctx); c.Items[0].State != dto.CapabilityPresent {
		t.Fatalf("cancelled caller: state = %q, want present", c.Items[0].State)
	}
}

func TestCapabilityRefresherConcurrentCallers(t *testing.T) {
	reg := platform.NewRegistry()
	var checks atomic.Int32
	release := make(chan struct{})
	r := NewCapabilityRefresher(reg, func() []platform.Probe {
		return []platform.Probe{{Name: "hardware.slow", Kind: platform.ProbeCheck, Check: func(context.Context) (dto.CapabilityState, string) {
			checks.Add(1)
			<-release
			return dto.CapabilityPresent, ""
		}}}
	})
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() { r.Refresh(context.Background()) })
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	if n := checks.Load(); n != 1 {
		t.Fatalf("concurrent refreshes ran %d checks, want 1", n)
	}
}

func TestCapabilityRefresherStart(t *testing.T) {
	reg := platform.NewRegistry()
	ticks := make(chan struct{}, 10)
	r := NewCapabilityRefresher(reg, func() []platform.Probe {
		return []platform.Probe{{Name: "hardware.tick", Kind: platform.ProbeCheck, Check: func(context.Context) (dto.CapabilityState, string) {
			ticks <- struct{}{}
			return dto.CapabilityAbsent, ""
		}}}
	})
	r.Interval = 10 * time.Millisecond
	r.MinInterval = 0
	r.SetHook(func(dto.Capabilities) { panic("hook panic is recovered") })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { r.Start(ctx); close(done) }()
	for range 2 {
		select {
		case <-ticks:
		case <-time.After(2 * time.Second):
			t.Fatal("refresher did not tick")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("refresher did not stop on cancel")
	}
}
