package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/tinkerologie/unraid-management-agent/daemon/services/scheduler"
)

// freshnessTimeout bounds how long a read waits for a refresh before it
// returns the cached data.
const freshnessTimeout = 2 * time.Second

// dynamicCollectors is the Dynamic-mode part of the collector manager.
type dynamicCollectors interface {
	Dynamic() bool
	RequestRefresh(name string) (<-chan error, bool)
	Demand() *scheduler.Demand
}

// Freshness describes the cached data a read returns.
type Freshness struct {
	// UpdatedAt is when the oldest of the read's collectors last reached the
	// cache; zero when none did or the read has no collectors.
	UpdatedAt time.Time
	// Stale is true when a collector's data is older than its staleness
	// limit because its refresh failed or timed out.
	Stale bool
}

// EnsureFresh makes the named collectors' cached data fresh before a read
// (#280). For each collector whose policy allows it and whose data is older
// than its staleness limit, it requests one shared refresh and waits, within
// freshnessTimeout, for the run and for the new data to reach the cache.
// recordDemand also marks the collectors as read, which runs them at their
// active rate for the read window; /metrics scrapes pass false. In Manual
// mode it only reports the data's age.
func (s *Server) EnsureFresh(ctx context.Context, recordDemand bool, collectors ...string) Freshness {
	dc := s.dynamic
	if dc != nil && recordDemand {
		dc.Demand().RecordRead(collectors...)
	}
	ctx, cancel := context.WithTimeout(ctx, freshnessTimeout)
	defer cancel()

	var (
		mu  sync.Mutex
		out Freshness
		wg  sync.WaitGroup
	)
	for _, name := range collectors {
		policy, ok := scheduler.PolicyFor(name)
		if !ok {
			continue
		}
		wg.Go(func() {
			at, stale := s.freshen(ctx, dc, name, policy)
			mu.Lock()
			defer mu.Unlock()
			if !at.IsZero() && (out.UpdatedAt.IsZero() || at.Before(out.UpdatedAt)) {
				out.UpdatedAt = at
			}
			out.Stale = out.Stale || stale
		})
	}
	wg.Wait()
	return out
}

// freshen refreshes one collector if its data is stale and returns the time
// its data reached the cache and whether it is still stale.
func (s *Server) freshen(ctx context.Context, dc dynamicCollectors, name string, policy scheduler.Policy) (time.Time, bool) {
	topic := policy.Topics[0]
	gen, at := s.CacheInstalled(topic)
	if dc == nil || !policy.RefreshOnRead || (!at.IsZero() && time.Since(at) <= policy.Staleness) {
		return at, false
	}
	published := s.publishCount(topic)
	done, running := dc.RequestRefresh(name)
	if !running {
		return at, false // stopped collectors have no fresh data to wait for
	}
	select {
	case err := <-done:
		if err != nil {
			return at, true
		}
	case <-ctx.Done():
		return at, true
	}
	// A run that published nothing (no data, or unchanged data a collector
	// does not republish) leaves the cache as current as it gets.
	if s.publishCount(topic) == published {
		return at, false
	}
	if !s.waitInstalled(ctx, topic, gen) {
		return at, true
	}
	_, at = s.CacheInstalled(topic)
	return at, false
}

// publishCount returns how many updates were published to topic.
func (s *Server) publishCount(topic string) uint64 {
	if s.ctx == nil || s.ctx.Hub == nil {
		return 0
	}
	return s.ctx.Hub.PublishCount(topic)
}

// freshnessMiddleware runs EnsureFresh for GET routes that serve collector
// data, then sets X-Cache-Updated-At and, when the data is still stale,
// X-Cache-Stale: true.
func (s *Server) freshnessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		route := mux.CurrentRoute(r)
		if route == nil {
			next.ServeHTTP(w, r)
			return
		}
		tmpl, err := route.GetPathTemplate()
		names, ok := routeCollectors[tmpl]
		if err != nil || !ok {
			next.ServeHTTP(w, r)
			return
		}
		f := s.EnsureFresh(r.Context(), tmpl != "/metrics", names...)
		if !f.UpdatedAt.IsZero() {
			w.Header().Set("X-Cache-Updated-At", f.UpdatedAt.UTC().Format(time.RFC3339))
		}
		if f.Stale {
			w.Header().Set("X-Cache-Stale", "true")
		}
		next.ServeHTTP(w, r)
	})
}

// routeCollectors maps each GET route that serves cached collector data to
// those collectors. TestRouteCollectorsCoverGETRoutes makes every new GET
// route choose between this table and the test's routesWithoutCollectors.
var routeCollectors = map[string][]string{
	"/metrics":                         {"system", "array", "disk", "shares", "docker", "docker_update", "vm", "ups", "gpu", "fancontrol"},
	"/api/v1/health/report":            {"array", "disk", "docker", "docker_update"},
	"/api/v1/system":                   {"system"},
	"/api/v1/temperatures":             {"system"},
	"/api/v1/array":                    {"array"},
	"/api/v1/disks":                    {"disk"},
	"/api/v1/disks/{id}":               {"disk"},
	"/api/v1/shares":                   {"shares"},
	"/api/v1/docker":                   {"docker", "docker_update"},
	"/api/v1/docker/{id}":              {"docker", "docker_update"},
	"/api/v1/docker/stats":             {"docker", "docker_update"},
	"/api/v1/docker/networks":          {"docker_networks"},
	"/api/v1/docker/updates":           {"docker_update"},
	"/api/v1/vm":                       {"vm"},
	"/api/v1/vm/{id}":                  {"vm"},
	"/api/v1/vm/guest":                 {"vm"},
	"/api/v1/ups":                      {"ups"},
	"/api/v1/nut":                      {"nut"},
	"/api/v1/gpu":                      {"gpu"},
	"/api/v1/network":                  {"network"},
	"/api/v1/network/cards":            {"nic"},
	"/api/v1/zfs/pools":                {"zfs"},
	"/api/v1/zfs/pools/{name}":         {"zfs"},
	"/api/v1/zfs/datasets":             {"zfs"},
	"/api/v1/zfs/snapshots":            {"zfs"},
	"/api/v1/zfs/arc":                  {"zfs"},
	"/api/v1/hardware/full":            {"hardware"},
	"/api/v1/hardware/bios":            {"hardware"},
	"/api/v1/hardware/baseboard":       {"hardware"},
	"/api/v1/hardware/cpu":             {"hardware"},
	"/api/v1/hardware/cache":           {"hardware"},
	"/api/v1/hardware/memory-array":    {"hardware"},
	"/api/v1/hardware/memory-devices":  {"hardware"},
	"/api/v1/plugins/check-updates":    {"plugin_update"},
	"/api/v1/updates":                  {"os_update"},
	"/api/v1/os/update":                {"os_update"},
	"/api/v1/mover":                    {"mover"},
	"/api/v1/storage/topology":         {"storage_topology"},
	"/api/v1/registration":             {"registration"},
	"/api/v1/notifications":            {"notification"},
	"/api/v1/notifications/unread":     {"notification"},
	"/api/v1/notifications/archive":    {"notification"},
	"/api/v1/notifications/overview":   {"notification"},
	"/api/v1/notifications/{id}":       {"notification"},
	"/api/v1/unassigned":               {"unassigned"},
	"/api/v1/unassigned/devices":       {"unassigned"},
	"/api/v1/unassigned/remote-shares": {"unassigned"},
	"/api/v1/fans":                     {"fancontrol"},
	"/api/v1/tuning":                   {"tuning"},
}
