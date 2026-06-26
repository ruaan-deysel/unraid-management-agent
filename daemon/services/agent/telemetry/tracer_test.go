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
	in := `registration key ABCD-1234; password=hunter2; token sk-live-xyz; sk-lf-deadbeef`
	got := Mask(in)
	for _, leak := range []string{"ABCD-1234", "hunter2", "sk-live-xyz", "sk-lf-deadbeef"} {
		if strings.Contains(got, leak) {
			t.Errorf("Mask leaked %q in %q", leak, got)
		}
	}
}
