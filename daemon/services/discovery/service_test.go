package discovery

import (
	"context"
	"errors"
	"net"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tinkerologie/unraid-management-agent/daemon/domain"
)

func TestInstanceName(t *testing.T) {
	tests := []struct {
		name     string
		config   domain.DiscoveryConfig
		hostname string
		want     string
	}{
		{
			name:     "falls back to hostname when no override",
			config:   domain.DiscoveryConfig{Enabled: true},
			hostname: "tower",
			want:     "tower",
		},
		{
			name:     "uses configured service name override",
			config:   domain.DiscoveryConfig{Enabled: true, ServiceName: "My Unraid"},
			hostname: "tower",
			want:     "My Unraid",
		},
		{
			name:     "empty hostname with no override returns empty",
			config:   domain.DiscoveryConfig{Enabled: true},
			hostname: "",
			want:     "",
		},
		{
			name:     "override wins even when hostname is empty",
			config:   domain.DiscoveryConfig{Enabled: true, ServiceName: "Main Unraid"},
			hostname: "",
			want:     "Main Unraid",
		},
		{
			name:     "special characters in hostname preserved",
			config:   domain.DiscoveryConfig{Enabled: true},
			hostname: "my-server_01",
			want:     "my-server_01",
		},
		{
			// instanceName returns values verbatim; mDNS-label encoding is the
			// zeroconf library's responsibility. These cases document that the
			// getter does not mangle or panic on unusual input.
			name:     "unicode hostname preserved",
			config:   domain.DiscoveryConfig{Enabled: true},
			hostname: "🏠-tower",
			want:     "🏠-tower",
		},
		{
			name:     "long hostname preserved",
			config:   domain.DiscoveryConfig{Enabled: true},
			hostname: strings.Repeat("a", 300),
			want:     strings.Repeat("a", 300),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService(tt.config, tt.hostname, 8043, "2026.06.01", "", false)
			if got := s.instanceName(); got != tt.want {
				t.Errorf("instanceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTxtRecords(t *testing.T) {
	tests := []struct {
		name        string
		tlsEnabled  bool
		authEnabled bool
		wantScheme  string
		wantMCPAuth string
	}{
		{name: "plain HTTP without auth advertises scheme=http and mcp_auth=none", tlsEnabled: false, authEnabled: false, wantScheme: "scheme=http", wantMCPAuth: "mcp_auth=none"},
		{name: "TLS enabled with auth advertises scheme=https and mcp_auth=bearer", tlsEnabled: true, authEnabled: true, wantScheme: "scheme=https", wantMCPAuth: "mcp_auth=bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewService(domain.DiscoveryConfig{Enabled: true}, "tower", 8043, "2026.06.01", "", tt.tlsEnabled)
			s.SetAuthEnabled(tt.authEnabled)
			got := s.txtRecords()

			want := []string{
				"version=2026.06.01",
				"path=/api/v1",
				"name=tower",
				tt.wantScheme,
				"mcp_path=/mcp",
				"mcp_transport=streamable-http",
				tt.wantMCPAuth,
			}
			if len(got) != len(want) {
				t.Errorf("txtRecords() returned %d records (%v), want %d (%v)", len(got), got, len(want), want)
			}
			for _, rec := range want {
				if !slices.Contains(got, rec) {
					t.Errorf("txtRecords() = %v, missing %q", got, rec)
				}
			}
			// The opposite scheme must never appear.
			otherScheme := "scheme=https"
			if tt.tlsEnabled {
				otherScheme = "scheme=http"
			}
			if slices.Contains(got, otherScheme) {
				t.Errorf("txtRecords() = %v, must not contain %q", got, otherScheme)
			}
			// Ensure no secret or token keys appear in mDNS TXT records.
			for _, rec := range got {
				if strings.Contains(rec, "secret") || strings.Contains(rec, "token") {
					t.Errorf("txtRecords() must never expose secret or token fields, got %q", rec)
				}
			}
		})
	}
}

func TestShutdownWithoutStartIsSafe(t *testing.T) {
	s := NewService(domain.DiscoveryConfig{Enabled: true}, "tower", 8043, "2026.06.01", "", false)
	// Shutdown before Start must be a no-op and must not panic.
	s.Shutdown()
}

func TestDefaultDiscoveryConfig(t *testing.T) {
	cfg := domain.DefaultDiscoveryConfig()
	if !cfg.Enabled {
		t.Error("DefaultDiscoveryConfig().Enabled = false, want true")
	}
	if cfg.ServiceName != "" {
		t.Errorf("DefaultDiscoveryConfig().ServiceName = %q, want empty", cfg.ServiceName)
	}
}

// lanFlags are the flags of an Ethernet interface that is up.
const lanFlags = net.FlagUp | net.FlagBroadcast | net.FlagMulticast

func fakeIface(index int, name string, flags net.Flags, addrs ...string) netInterface {
	ni := netInterface{iface: net.Interface{Index: index, Name: name, Flags: flags}}
	for _, a := range addrs {
		ni.addrs = append(ni.addrs, net.ParseIP(a))
	}
	return ni
}

// unraidHost mirrors an Unraid server with Docker and libvirt networks: the
// LAN on br0 (shared by the shim-br0 macvlan), Docker bridges, a veth pair,
// libvirt's bridge, a WireGuard tunnel and loopback.
func unraidHost() []netInterface {
	return []netInterface{
		fakeIface(1, "lo", net.FlagUp|net.FlagLoopback|net.FlagMulticast, "127.0.0.1", "::1"),
		fakeIface(2, "eth0", lanFlags),
		fakeIface(3, "br0", lanFlags, "192.168.20.21", "fe80::1"),
		fakeIface(4, "docker0", lanFlags, "172.17.0.1"),
		fakeIface(5, "br-3cc0fa14431c", lanFlags, "172.18.0.1"),
		fakeIface(6, "br-f063b8e6a1ab", lanFlags, "172.19.0.1"),
		fakeIface(7, "shim-br0", lanFlags, "192.168.20.21"),
		fakeIface(8, "veth1a2b3c", lanFlags, "fe80::2"),
		fakeIface(9, "virbr0", lanFlags, "192.168.122.1"),
		fakeIface(10, "wg0", net.FlagUp|net.FlagPointToPoint, "10.253.0.1"),
	}
}

// fakeNetwork replaces the interface list and default-route lookup.
func fakeNetwork(t *testing.T, ifaces []netInterface, route string) {
	t.Helper()
	origList, origRoute := listInterfaces, routeIPv4
	t.Cleanup(func() { listInterfaces, routeIPv4 = origList, origRoute })
	listInterfaces = func() ([]netInterface, error) { return ifaces, nil }
	routeIPv4 = func() net.IP { return net.ParseIP(route) }
}

type registration struct {
	ips    []string
	ifaces []string
}

type fakeServer struct{ shutdowns *atomic.Int32 }

func (f fakeServer) Shutdown() { f.shutdowns.Add(1) }

// fakeRegister records every registration instead of opening sockets.
func fakeRegister(t *testing.T) (*[]registration, *sync.Mutex, *atomic.Int32) {
	t.Helper()
	orig := registerProxy
	t.Cleanup(func() { registerProxy = orig })
	var (
		mu        sync.Mutex
		regs      []registration
		shutdowns atomic.Int32
	)
	registerProxy = func(_, _, _ string, _ int, _ string, ips, _ []string, ifaces []net.Interface) (advertiser, error) {
		mu.Lock()
		defer mu.Unlock()
		r := registration{ips: ips}
		for _, iface := range ifaces {
			r.ifaces = append(r.ifaces, iface.Name)
		}
		regs = append(regs, r)
		return fakeServer{&shutdowns}, nil
	}
	return &regs, &mu, &shutdowns
}

func TestSelectAdvertisement(t *testing.T) {
	tests := []struct {
		name        string
		ifaces      []netInterface
		route       string
		bindAddress string
		wantIP      string
		wantIface   string
		wantErr     bool
	}{
		{name: "route IP on br0", ifaces: unraidHost(), route: "192.168.20.21", wantIP: "192.168.20.21", wantIface: "br0"},
		{name: "unspecified bind uses the route", ifaces: unraidHost(), route: "192.168.20.21", bindAddress: "0.0.0.0", wantIP: "192.168.20.21", wantIface: "br0"},
		{name: "invalid bind uses the route", ifaces: unraidHost(), route: "192.168.20.21", bindAddress: "not-an-ip", wantIP: "192.168.20.21", wantIface: "br0"},
		{name: "no route falls back to br0, never a Docker bridge", ifaces: unraidHost(), wantIP: "192.168.20.21", wantIface: "br0"},
		{name: "route through a VPN falls back to br0", ifaces: unraidHost(), route: "10.253.0.1", wantIP: "192.168.20.21", wantIface: "br0"},
		{name: "route on a Docker bridge falls back to br0", ifaces: unraidHost(), route: "172.18.0.1", wantIP: "192.168.20.21", wantIface: "br0"},
		{name: "bind address on br0", ifaces: unraidHost(), route: "192.168.20.21", bindAddress: "192.168.20.21", wantIP: "192.168.20.21", wantIface: "br0"},
		{
			name:        "IPv6 bind address",
			ifaces:      []netInterface{fakeIface(3, "br0", lanFlags, "192.168.20.21", "2001:db8::10")},
			bindAddress: "2001:db8::10", wantIP: "2001:db8::10", wantIface: "br0",
		},
		{name: "bind address on a Docker bridge is rejected", ifaces: unraidHost(), bindAddress: "172.17.0.1", wantErr: true},
		{name: "bind address not on the host is rejected", ifaces: unraidHost(), bindAddress: "192.168.40.10", wantErr: true},
		{
			name: "bond preferred over eth without a route",
			ifaces: []netInterface{
				fakeIface(2, "eth1", lanFlags, "192.168.30.5"),
				fakeIface(3, "bond0", lanFlags, "192.168.20.21"),
			},
			wantIP: "192.168.20.21", wantIface: "bond0",
		},
		{
			name:   "other NIC names are allowed",
			ifaces: []netInterface{fakeIface(2, "enp3s0", lanFlags, "fe80::3", "192.168.20.21")},
			wantIP: "192.168.20.21", wantIface: "enp3s0",
		},
		{
			name:    "down interface is skipped",
			ifaces:  []netInterface{fakeIface(3, "br0", net.FlagBroadcast|net.FlagMulticast, "192.168.20.21")},
			wantErr: true,
		},
		{
			name:    "no route and br0 without an address skips an LXC bridge",
			ifaces:  []netInterface{fakeIface(3, "br0", lanFlags), fakeIface(12, "lxcbr0", lanFlags, "10.0.3.1")},
			wantErr: true,
		},
		{
			name: "only virtual interfaces",
			ifaces: []netInterface{
				fakeIface(4, "docker0", lanFlags, "172.17.0.1"),
				fakeIface(9, "virbr0", lanFlags, "192.168.122.1"),
				fakeIface(11, "tailscale0", lanFlags, "100.64.0.1"),
			},
			route: "172.17.0.1", wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeNetwork(t, tt.ifaces, tt.route)
			s := NewService(domain.DiscoveryConfig{Enabled: true}, "tower", 8043, "2026.06.01", tt.bindAddress, false)
			ip, iface, err := s.selectAdvertisement()
			if tt.wantErr {
				if !errors.Is(err, errNoLANInterface) {
					t.Fatalf("selectAdvertisement() = %v, %q, %v; want errNoLANInterface", ip, iface.Name, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("selectAdvertisement() error = %v", err)
			}
			if ip.String() != tt.wantIP || iface.Name != tt.wantIface {
				t.Errorf("selectAdvertisement() = %v on %q, want %s on %q", ip, iface.Name, tt.wantIP, tt.wantIface)
			}
		})
	}
}

func TestStartRegistersOnOneInterfaceWithOneAddress(t *testing.T) {
	fakeNetwork(t, unraidHost(), "")
	regs, mu, shutdowns := fakeRegister(t)

	s := NewService(domain.DiscoveryConfig{Enabled: true}, "tower", 8043, "2026.06.01", "", false)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	// A second Start is a no-op.
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	mu.Lock()
	got := slices.Clone(*regs)
	mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("registrations = %d, want 1", len(got))
	}
	if !slices.Equal(got[0].ips, []string{"192.168.20.21"}) || !slices.Equal(got[0].ifaces, []string{"br0"}) {
		t.Errorf("registered ips=%v ifaces=%v, want [192.168.20.21] on [br0]", got[0].ips, got[0].ifaces)
	}
	s.Shutdown()
	if shutdowns.Load() != 1 {
		t.Errorf("server shutdowns = %d, want 1", shutdowns.Load())
	}
}

func TestStartRetriesUntilALANInterfaceAppears(t *testing.T) {
	orig := retryInterval
	retryInterval = 10 * time.Millisecond
	t.Cleanup(func() { retryInterval = orig })

	var up atomic.Bool
	origList, origRoute := listInterfaces, routeIPv4
	t.Cleanup(func() { listInterfaces, routeIPv4 = origList, origRoute })
	routeIPv4 = func() net.IP { return nil }
	listInterfaces = func() ([]netInterface, error) {
		if up.Load() {
			return unraidHost(), nil
		}
		// Boot: br0 has no address yet, only Docker bridges do.
		return []netInterface{fakeIface(3, "br0", lanFlags), fakeIface(4, "docker0", lanFlags, "172.17.0.1")}, nil
	}
	regs, mu, _ := fakeRegister(t)
	count := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(*regs)
	}

	s := NewService(domain.DiscoveryConfig{Enabled: true}, "tower", 8043, "2026.06.01", "", false)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v, want nil while retrying", err)
	}
	time.Sleep(50 * time.Millisecond)
	if n := count(); n != 0 {
		t.Fatalf("registered %d times before a LAN address existed", n)
	}

	up.Store(true)
	deadline := time.Now().Add(2 * time.Second)
	for count() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	mu.Lock()
	got := slices.Clone(*regs)
	mu.Unlock()
	if len(got) != 1 || !slices.Equal(got[0].ips, []string{"192.168.20.21"}) || !slices.Equal(got[0].ifaces, []string{"br0"}) {
		t.Fatalf("registrations = %+v, want one on br0 with 192.168.20.21", got)
	}
	s.Shutdown()
}

func TestShutdownStopsRetries(t *testing.T) {
	orig := retryInterval
	retryInterval = 10 * time.Millisecond
	t.Cleanup(func() { retryInterval = orig })
	var up atomic.Bool
	origList, origRoute := listInterfaces, routeIPv4
	t.Cleanup(func() { listInterfaces, routeIPv4 = origList, origRoute })
	routeIPv4 = func() net.IP { return nil }
	listInterfaces = func() ([]netInterface, error) {
		if up.Load() {
			return unraidHost(), nil
		}
		return []netInterface{fakeIface(4, "docker0", lanFlags, "172.17.0.1")}, nil
	}
	regs, mu, _ := fakeRegister(t)

	s := NewService(domain.DiscoveryConfig{Enabled: true}, "tower", 8043, "2026.06.01", "", false)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	s.Shutdown()
	// Shutdown is idempotent and a later network change registers nothing.
	s.Shutdown()
	up.Store(true)
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(*regs) != 0 {
		t.Errorf("registered %d times after Shutdown", len(*regs))
	}
}

func TestStartReturnsRegistrationErrors(t *testing.T) {
	fakeNetwork(t, unraidHost(), "192.168.20.21")
	orig := registerProxy
	t.Cleanup(func() { registerProxy = orig })
	registerProxy = func(string, string, string, int, string, []string, []string, []net.Interface) (advertiser, error) {
		return nil, errors.New("socket failure")
	}
	s := NewService(domain.DiscoveryConfig{Enabled: true}, "tower", 8043, "2026.06.01", "", false)
	if err := s.Start(context.Background()); err == nil || !strings.Contains(err.Error(), "socket failure") {
		t.Errorf("Start() error = %v, want the registration error", err)
	}
	if s.server != nil || s.stop != nil {
		t.Error("failed Start() must leave no server or retry loop")
	}
}

func TestIsLANInterface(t *testing.T) {
	for _, name := range []string{"br0", "bond0", "eth0", "enp3s0", "eno1"} {
		if !isLANInterface(net.Interface{Name: name, Flags: lanFlags}) {
			t.Errorf("isLANInterface(%q) = false, want true", name)
		}
	}
	for _, name := range []string{
		"docker0", "br-3cc0fa14431c", "veth1a2b3c", "lxcbr0", "lxdbr0", "incusbr0", "podman0", "cni0",
		"flannel.1", "cali1234", "cilium_host", "kube-ipvs0", "virbr0", "vnet0", "shim-br0", "tun0",
		"tap0", "wg0", "tailscale0", "zt5u4y5n6a", "vhost0",
	} {
		if isLANInterface(net.Interface{Name: name, Flags: lanFlags}) {
			t.Errorf("isLANInterface(%q) = true, want false", name)
		}
	}
}

func TestStartSkipsAdvertisementOnLoopbackBind(t *testing.T) {
	for _, bindAddr := range []string{"127.0.0.1", "127.0.0.53", "::1"} {
		t.Run(bindAddr, func(t *testing.T) {
			s := NewService(domain.DiscoveryConfig{Enabled: true}, "tower", 8043, "2026.06.01", bindAddr, false)
			if err := s.Start(context.Background()); err != nil {
				t.Fatalf("Start() with loopback bind %q returned error: %v", bindAddr, err)
			}
			if s.server != nil {
				t.Errorf("Start() with loopback bind %q must not register an mDNS server", bindAddr)
			}
			s.Shutdown()
		})
	}
}
