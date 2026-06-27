package telemetry

import (
	"context"
	"strings"
	"testing"

	"github.com/ruaan-deysel/unraid-management-agent/daemon/domain"
)

func TestNewDisabledIsNoop(t *testing.T) {
	p, err := New(domain.Config{}) // no keys
	if err != nil {
		t.Fatalf("New(disabled) error: %v", err)
	}
	if p.Enabled() {
		t.Fatal("expected disabled provider")
	}
	_ = p.Tracer("test")
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(noop) error: %v", err)
	}
}

func TestNewEnabledBuildsProvider(t *testing.T) {
	p, err := New(domain.Config{LangfusePublicKey: "pk", LangfuseSecretKey: "sk", LangfuseBaseURL: "https://example.com"})
	if err != nil {
		t.Fatalf("New(enabled) error: %v", err)
	}
	if !p.Enabled() {
		t.Fatal("expected enabled provider")
	}
	// Shutdown must not hang/panic even though nothing was exported.
	if err := p.Shutdown(context.Background()); err != nil {
		t.Logf("shutdown returned (acceptable if export endpoint unreachable): %v", err)
	}
}

func TestMaskRedactsSecrets(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		leaked string
	}{
		{"registration key", "registration key ABCD-1234 rest", "ABCD-1234"},
		{"wireguard", "wireguard-privatekey=secret", "secret"},
		{"password=", "password=hunter2 other", "hunter2"},
		{"token whitespace", "token sk-live-xyz end", "sk-live-xyz"},
		{"token=", "token=mytoken rest", "mytoken"},
		{"sk-key", "sk-abc123-def rest", "sk-abc123-def"},
		{"pk-lf key", "pk-lf-mypublickey rest", "pk-lf-mypublickey"},
		{"sk-lf key", "sk-lf-deadbeef rest", "sk-lf-deadbeef"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Mask(tc.input)
			if strings.Contains(got, tc.leaked) {
				t.Errorf("Mask leaked %q in %q", tc.leaked, got)
			}
		})
	}
}
