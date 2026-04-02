---
sidebar_position: 1
title: Shared Infrastructure Overview
---

# Shared Infrastructure

The `shared/` package provides cross-cutting infrastructure that is imported by every microservice — RabbitMQ messaging, distributed telemetry, environment utilities, and retry logic.

## 1. RabbitMQ Event Bus

Rather than having each microservice natively interact with `github.com/rabbitmq/amqp091-go` plumbing, the `shared/messaging/rabbitmq.go` package outlines a unified wrapper:

```go
type RabbitMQ struct {
    Conn    *amqp.Connection
    Channel *amqp.Channel
}
```

Calling `NewRabbitMQ` creates a unified `TripExchange`, declares all 11 queues, and wires up their bindings automatically based on predefined constants in `shared/messaging/events.go` and `shared/contracts/amqp.go`.

### Routing Keys

All events use consistent naming conventions:

| Pattern | Example |
|---|---|
| `trip.event.*` | `trip.event.created`, `trip.event.driver_assigned` |
| `driver.cmd.*` | `driver.cmd.trip_request`, `driver.cmd.trip_decline` |
| `payment.event.*` | `payment.event.session_created`, `payment.event.success` |
| `payment.cmd.*` | `payment.cmd.create_session` |

### Publishing Messages

Services publish via:

```go
r.PublishMessage(ctx, contracts.TripEventCreated, contracts.AmqpMessage{
    OwnerID: trip.UserID,
    Data:    tripEventJSON,
})
```

The `OwnerID` is used by the API Gateway's queue consumers to route the message to the correct WebSocket connection (finding the right rider or driver by their UUID).

### Delayed Publishing (Search Retry)

The `PublishDelayMessage` method publishes directly to the `search_retry_queue` — a headless queue with a 10s TTL and a dead-letter routing key pointing back to `TripExchange`. This implements the driver search retry loop without any cron or polling worker:

```go
func (r *RabbitMQ) PublishDelayMessage(ctx context.Context, message contracts.AmqpMessage) error {
    // Publishes directly to the search_retry_queue (no exchange)
    return r.Channel.PublishWithContext(ctx, "", SearchRetryQueue, false, false, msg)
}
```

---

## 2. Distributed Telemetry (OpenTelemetry)

Tracking a request as it hops from the Next.js frontend → API Gateway → RabbitMQ → Trip Service requires centralized distributed tracing. Every microservice uses `shared/tracing` backed by OpenTelemetry (OTel) and Jaeger.

### Initialization

Every `main.go` initializes a service-scoped tracer:

```go
tracerCfg := tracing.Config{
    ServiceName:    "driver-service",
    Environment:    env.GetString("ENVIRONMENT", "development"),
    JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
}
sh, err := tracing.InitTracer(tracerCfg)
```

### Context Propagation — Three Protocols

OTel propagates trace IDs across all three communication channels used in this system:

| Protocol | Mechanism |
|---|---|
| **HTTP** | `tracing.WrapHandlerFunc` — extracts W3C TraceContext headers on API Gateway routes |
| **gRPC** | `tracing.WithTracingInterceptors()` — injects/extracts trace IDs in HTTP/2 gRPC metadata |
| **AMQP** | Trace IDs injected into RabbitMQ message headers before publish, extracted on consume |

The composite propagator that handles all three:

```go
func newPropagator() propagation.TextMapPropagator {
    return propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    )
}
```

Because the propagator interface is consistent across protocols, Jaeger can connect the full causal chain — from the rider's browser HTTP POST to the AMQP message hitting the Driver Service — with exact millisecond latency at each hop.
