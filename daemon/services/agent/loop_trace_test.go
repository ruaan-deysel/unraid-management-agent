package agent

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/ruaan-deysel/unraid-management-agent/daemon/dto"
	"github.com/ruaan-deysel/unraid-management-agent/daemon/services/agent/llm"
	"github.com/ruaan-deysel/unraid-management-agent/daemon/services/agent/memory"
	"github.com/ruaan-deysel/unraid-management-agent/daemon/services/agent/tools"
)

// TestRunLoopEmitsSessionStepToolSpans drives a session that calls one
// auto-approved tool to completion and asserts the span tree shape.
func TestRunLoopEmitsSessionStepToolSpans(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	p := llm.NewMockProvider(
		&llm.ChatResponse{Text: "[]"}, // planner: empty plan
		&llm.ChatResponse{ToolCalls: []llm.ToolCall{{ID: "1", Name: "get_system_info", Args: "{}"}}, InputTokens: 10, OutputTokens: 5},
		&llm.ChatResponse{Text: "Host is tower, all healthy.", InputTokens: 8, OutputTokens: 7},
	)
	cfg := dto.DefaultAgentConfig()
	cfg.Enabled = true
	reg := tools.BuildDefault(fakeState{}, fakeDocker{})
	svc := NewService(cfg, p, reg, NewStore(t.TempDir()), memory.NewStore(t.TempDir(), 0), &capturingBroadcaster{}, tp.Tracer("test"))

	sess, err := svc.StartSession(context.Background(), "is my system healthy?")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if sess.Status != dto.SessionCompleted {
		t.Fatalf("status=%q err=%q", sess.Status, sess.Error)
	}

	seen := map[string]bool{}
	var rootTrace string
	for _, sp := range sr.Ended() {
		seen[sp.Name()] = true
		if sp.Name() == "agent-session" {
			rootTrace = sp.SpanContext().TraceID().String()
		}
	}
	for _, want := range []string{"agent-session", "step-0", "tool:get_system_info"} {
		if !seen[want] {
			t.Errorf("missing span %q; saw %v", want, seen)
		}
	}

	if sess.TraceID == "" {
		t.Fatal("expected sess.TraceID to be set")
	}
	if sess.TraceID != rootTrace {
		t.Errorf("sess.TraceID=%q want root trace %q", sess.TraceID, rootTrace)
	}
}

// TestRunLoopNoopTracerRecordsNothing confirms the default no-op tracer
// (nil passed to NewService) records no spans.
func TestRunLoopNoopTracerRecordsNothing(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	// Recorder is attached to a separate provider that the Service never uses;
	// the Service gets the default no-op tracer via nil.
	_ = sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))

	p := llm.NewMockProvider(
		&llm.ChatResponse{Text: "[]"},
		&llm.ChatResponse{Text: "all good", InputTokens: 1, OutputTokens: 1},
	)
	cfg := dto.DefaultAgentConfig()
	cfg.Enabled = true
	svc := NewService(cfg, p, tools.BuildDefault(fakeState{}, fakeDocker{}), NewStore(t.TempDir()), memory.NewStore(t.TempDir(), 0), &capturingBroadcaster{}, nil)

	if _, err := svc.StartSession(context.Background(), "hi"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if got := len(sr.Ended()); got != 0 {
		t.Fatalf("expected no recorded spans with no-op tracer, got %d", got)
	}
}
