// Package telemetry wires OpenTelemetry tracing to Langfuse (OTLP/HTTP).
// It is opt-in: with no Langfuse keys it is a zero-cost no-op.
package telemetry

import (
	"context"
	"encoding/base64"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/ruaan-deysel/unraid-management-agent/daemon/domain"
)

// Provider owns the tracer lifecycle. Safe to use when disabled.
type Provider struct {
	tp      *sdktrace.TracerProvider // nil when disabled
	tracer  trace.Tracer
	enabled bool
}

// New builds a Provider. When Langfuse is not configured it returns a no-op
// provider that performs no network I/O.
func New(cfg domain.Config) (*Provider, error) {
	if !cfg.LangfuseEnabled() {
		return &Provider{tracer: noop.NewTracerProvider().Tracer("unraid-agent")}, nil
	}
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.LangfusePublicKey + ":" + cfg.LangfuseSecretKey))
	exp, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpointURL(cfg.LangfuseBaseURL+"/api/public/otel/v1/traces"),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization":                "Basic " + auth,
			"x-langfuse-ingestion-version": "4",
		}),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", "unraid-management-agent"))),
	)
	return &Provider{tp: tp, tracer: tp.Tracer("unraid-agent"), enabled: true}, nil
}

func (p *Provider) Enabled() bool              { return p.enabled }
func (p *Provider) Tracer(string) trace.Tracer { return p.tracer }

// Shutdown flushes pending spans within a bounded timeout. Best-effort.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tp == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return p.tp.Shutdown(ctx)
}
