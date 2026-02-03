package tracing

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	ServiceName    string
	Environment    string
	JaegerEndpoint string // Keeping this name for backward compatibility in arguments, but it will be used as OTLP endpoint
}

func InitTracer(cfg Config) (func(context.Context) error, error) {
	// Exporter
	traceExporter, err := newExporter(cfg.JaegerEndpoint)
	if err != nil {
		return nil, err
	}

	// Trace Provider
	traceProvider, err := newTraceProvider(cfg, traceExporter)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(traceProvider)

	// Propagator
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	return traceProvider.Shutdown, nil
}

func GetTracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

func newExporter(endpoint string) (sdktrace.SpanExporter, error) {
	var opts []otlptracehttp.Option

	// Check if standard env var is set
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// If the argument is provided, it takes precedence (backward compatibility)
	targetEndpoint := endpoint
	if targetEndpoint == "" {
		targetEndpoint = otelEndpoint
	}

	// Logic for Insecure:
	// 1. If we have an explicit HTTP endpoint (arg or env), use insecure.
	// 2. If we have NO endpoint (default localhost), use insecure.
	// 3. If we have HTTPS endpoint, use secure (default).
	if targetEndpoint == "" || strings.HasPrefix(targetEndpoint, "http://") {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	// Only apply WithEndpointURL if we have an explicit non-OTEL-env endpoint passed as arg,
	// OR if we want to ensure specific logic.
	// Actually, otlptracehttp.New() picks up OTEL_EXPORTER_OTLP_ENDPOINT automatically if we don't pass WithEndpointURL.
	// But since we have logic depending on the protocol, we might as well be explicit if an arg was passed.
	if endpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpointURL(endpoint))
	}

	return otlptracehttp.New(context.Background(), opts...)
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider(cfg Config, exporter sdktrace.SpanExporter) (*sdktrace.TracerProvider, error) {
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.DeploymentEnvironmentKey.String(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	return traceProvider, nil
}
