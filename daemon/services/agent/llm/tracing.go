package llm

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ruaan-deysel/unraid-management-agent/daemon/services/agent/telemetry"
)

// tracingProvider decorates a Provider with Langfuse generation spans.
type tracingProvider struct {
	delegate Provider
	tracer   trace.Tracer
}

// NewTracingProvider wraps delegate so each Chat emits a generation span.
func NewTracingProvider(delegate Provider, tracer trace.Tracer) Provider {
	return &tracingProvider{delegate: delegate, tracer: tracer}
}

func (t *tracingProvider) Name() string { return t.delegate.Name() }

func (t *tracingProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	ctx, span := t.tracer.Start(ctx, "llm-generation")
	defer span.End()
	start := time.Now()

	span.SetAttributes(
		attribute.String("langfuse.observation.type", "generation"),
		attribute.String("gen_ai.system", t.delegate.Name()),
		attribute.String("gen_ai.request.model", t.delegate.Name()),
		attribute.String("langfuse.observation.input", telemetry.Mask(renderMessages(req))),
	)

	resp, err := t.delegate.Chat(ctx, req)
	span.SetAttributes(attribute.Int64("langfuse.latency_ms", time.Since(start).Milliseconds()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return resp, err
	}
	span.SetAttributes(
		attribute.Int("gen_ai.usage.input_tokens", resp.InputTokens),
		attribute.Int("gen_ai.usage.output_tokens", resp.OutputTokens),
		attribute.String("langfuse.observation.output", telemetry.Mask(resp.Text)),
	)
	return resp, nil
}

// renderMessages produces a compact, maskable string of the request.
func renderMessages(req ChatRequest) string {
	out := "system: " + req.System
	for _, m := range req.Messages {
		out += "\n" + m.Role + ": " + m.Content
	}
	return out
}
