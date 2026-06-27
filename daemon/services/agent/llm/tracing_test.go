package llm

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type fakeProvider struct{}

func (fakeProvider) Name() string { return "openai" }
func (fakeProvider) Chat(_ context.Context, _ ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Text: "ok", InputTokens: 10, OutputTokens: 3}, nil
}

func TestTracingProviderEmitsGeneration(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	p := NewTracingProvider(fakeProvider{}, "test-model", tp.Tracer("test"))

	_, err := p.Chat(context.Background(), ChatRequest{System: "sys", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	spans := sr.Ended()
	if len(spans) != 1 || spans[0].Name() != "llm-generation" {
		t.Fatalf("expected 1 llm-generation span, got %#v", spans)
	}
	attrs := map[string]string{}
	for _, kv := range spans[0].Attributes() {
		attrs[string(kv.Key)] = kv.Value.Emit()
	}
	if attrs["langfuse.observation.type"] != "generation" {
		t.Errorf("missing generation type: %v", attrs)
	}
	if attrs["gen_ai.usage.output_tokens"] != "3" {
		t.Errorf("missing token usage: %v", attrs)
	}
	if attrs["gen_ai.request.model"] != "test-model" {
		t.Errorf("expected gen_ai.request.model=test-model, got %v", attrs)
	}
}
