---
sidebar_position: 2
title: Distributed Telemetry
---

# Distributed Telemetry

Tracking requests as they span from the `Next.js` frontend -> `API Gateway` -> `RabbitMQ` -> `Trip Service` requires centralized distributed tracing. The architecture relies completely on OpenTelemetry (OTel).

## Initialization

Every Go microservice initializes its tracing agent inside `main.go` using the `shared/tracing.go` package:

```go
	tracerCfg := tracing.Config{
		ServiceName:    "driver-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
```

The underlying pipeline boots up an OTLP HTTP exporter pointed towards the centralized Jaeger instance and creates a global Trace Provider.

## Context Propagation

To connect a span inside the `API Gateway` to a span inside the `Driver Service`, context headers must mathematically link them. OpenTelemetry handles this automatically through its composite propagators:

```go
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}
```

1. **HTTP**: External HTTP API routes use a generic `WrapHandlerFunc` wrapper pulling trace IDs from `W3C TraceContext` standard HTTP headers.
2. **gRPC**: Internal RPC calls utilize `tracing.WithTracingInterceptors()` to inject and extract those exact same trace credentials transparently over the HTTP/2 gRPC metadata streams.
3. **AMQP**: RabbitMQ messages inject standard trace IDs natively inside AMQP Message Headers.

Because the unified Propagator interfaces remain constant, Jaeger can successfully visualize exactly how many milliseconds it took for the Rider to HTTP POST `/trip/start`, for the `API Gateway` to gRPC ping the `Trip Service`, and for the `Trip Service` to finally AMQP publish the `TripEventCreated` request to the backend workers.
