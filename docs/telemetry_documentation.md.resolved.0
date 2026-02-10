# Telemetry and Observability Documentation

> [!NOTE]
> This document provides comprehensive coverage of the distributed tracing and observability implementation in the ride-sharing microservices system using OpenTelemetry and Jaeger.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Core Components](#core-components)
3. [Jaeger Setup](#jaeger-setup)
4. [OpenTelemetry Initialization](#opentelemetry-initialization)
5. [HTTP Instrumentation](#http-instrumentation)
6. [gRPC Tracing](#grpc-tracing)
7. [RabbitMQ Async Tracing](#rabbitmq-async-tracing)
8. [Integration Examples](#integration-examples)
9. [Configuration Guide](#configuration-guide)
10. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

The ride-sharing application implements distributed tracing using **OpenTelemetry** (OTel) as the instrumentation framework and **Jaeger** as the tracing backend. This enables end-to-end observability across multiple microservices communicating via HTTP, gRPC, and RabbitMQ.

### High-Level Architecture

```mermaid
graph TB
    subgraph "Services Layer"
        API[API Gateway]
        TRIP[Trip Service]
        DRIVER[Driver Service]
        PAYMENT[Payment Service]
    end
    
    subgraph "Communication Protocols"
        HTTP[HTTP/REST]
        GRPC[gRPC]
        RABBIT[RabbitMQ]
    end
    
    subgraph "Tracing Infrastructure"
        OTEL[OpenTelemetry SDK]
        PROP[Context Propagator]
        EXPORT[Jaeger Exporter]
    end
    
    subgraph "Observability Backend"
        JAEGER[Jaeger All-in-One]
        COLLECTOR[Collector :14268]
        UI[Jaeger UI :16686]
    end
    
    API --> HTTP
    API --> GRPC
    API --> RABBIT
    TRIP --> GRPC
    TRIP --> RABBIT
    DRIVER --> RABBIT
    PAYMENT --> RABBIT
    
    HTTP --> OTEL
    GRPC --> OTEL
    RABBIT --> OTEL
    
    OTEL --> PROP
    OTEL --> EXPORT
    EXPORT --> COLLECTOR
    COLLECTOR --> JAEGER
    JAEGER --> UI
    
    style OTEL fill:#4285f4
    style JAEGER fill:#60d7a7
    style RABBIT fill:#ff6600
```

### Trace Context Propagation Flow

```mermaid
sequenceDiagram
    participant Client
    participant API Gateway
    participant Trip Service
    participant RabbitMQ
    participant Driver Service
    participant Jaeger
    
    Note over Client,Jaeger: Synchronous HTTP → gRPC Flow
    Client->>API Gateway: HTTP POST /trip/preview
    activate API Gateway
    Note right of API Gateway: Create root span<br/>Inject trace context
    API Gateway->>Trip Service: gRPC PreviewTrip(ctx)
    activate Trip Service
    Note right of Trip Service: Extract context<br/>Create child span
    Trip Service-->>API Gateway: Response
    deactivate Trip Service
    API Gateway-->>Client: JSON Response
    deactivate API Gateway
    
    Note over Client,Jaeger: Asynchronous Messaging Flow
    Client->>API Gateway: HTTP POST /trip/start
    activate API Gateway
    API Gateway->>RabbitMQ: Publish TripCreated
    Note right of API Gateway: Inject trace context<br/>into AMQP headers
    deactivate API Gateway
    
    RabbitMQ->>Driver Service: Deliver message
    activate Driver Service
    Note right of Driver Service: Extract context<br/>from headers<br/>Create child span
    Driver Service->>Jaeger: Export spans
    deactivate Driver Service
    
    Note over Jaeger: All spans linked<br/>via trace ID
```

### Key Design Decisions

> [!IMPORTANT]
> **Context Propagation Across Async Boundaries**: The implementation uses custom AMQP header carriers to propagate trace context through RabbitMQ, maintaining trace continuity across asynchronous message flows.

> [!TIP]
> **Centralized Tracing Package**: All tracing logic is consolidated in [shared/tracing/](file:///home/vijetapriya/ride-sharing-1/shared/tracing) to ensure consistent instrumentation patterns across all services.

---

## Core Components

The telemetry implementation is organized into the following modules:

| Component | File | Purpose |
|-----------|------|---------|
| Core Tracer | [tracing.go](file:///home/vijetapriya/ride-sharing-1/shared/tracing/tracing.go) | Initializes OTel SDK, Jaeger exporter, and tracer provider |
| HTTP Instrumentation | [http.go](file:///home/vijetapriya/ride-sharing-1/shared/tracing/http.go) | Wraps HTTP handlers with automatic tracing |
| gRPC Instrumentation | [grpc.go](file:///home/vijetapriya/ride-sharing-1/shared/tracing/grpc.go) | Provides gRPC interceptors for client/server tracing |
| RabbitMQ Instrumentation | [rabbitmq.go](file:///home/vijetapriya/ride-sharing-1/shared/tracing/rabbitmq.go) | Custom trace context propagation for async messaging |

### Dependencies

```go
// OpenTelemetry Core
go.opentelemetry.io/otel v1.34.0
go.opentelemetry.io/otel/sdk v1.34.0
go.opentelemetry.io/otel/trace v1.34.0

// Jaeger Exporter
go.opentelemetry.io/otel/exporters/jaeger v1.17.0

// Auto-Instrumentation Libraries
go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0
go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.59.0
```

---

## Jaeger Setup

Jaeger runs as a unified "all-in-one" deployment providing collection, storage, and UI in a single container.

### Kubernetes Deployment

#### Development Environment

[infra/development/k8s/jaeger.yaml](file:///home/vijetapriya/ride-sharing-1/infra/development/k8s/jaeger.yaml):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
  labels:
    app: jaeger
spec:
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
        - name: jaeger
          image: jaegertracing/all-in-one:1.49
          ports:
            - containerPort: 16686 # UI
            - containerPort: 14268 # Collector HTTP
          env:
            - name: COLLECTOR_OTLP_ENABLED
              value: "true"
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger
spec:
  selector:
    app: jaeger
  ports:
    - name: ui
      port: 16686
      targetPort: 16686
    - name: collector
      port: 14268
      targetPort: 14268
  type: ClusterIP
```

#### Production Environment

[infra/production/k8s/jaeger-deployment.yaml](file:///home/vijetapriya/ride-sharing-1/infra/production/k8s/jaeger-deployment.yaml) adds resource constraints:

```yaml
resources:
  requests:
    cpu: "250m"
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "256Mi"
```

### Port Reference

| Port | Purpose | Protocol |
|------|---------|----------|
| 16686 | Jaeger UI | HTTP |
| 14268 | Collector HTTP endpoint | HTTP |
| 6831 | Thrift compact protocol (UDP) | UDP |
| 6832 | Thrift binary protocol (UDP) | UDP |

> [!TIP]
> Access the Jaeger UI at `http://jaeger:16686` within the cluster or via port-forwarding: `kubectl port-forward svc/jaeger 16686:16686`

---

## OpenTelemetry Initialization

### Tracer Configuration

The core initialization is handled by [InitTracer](file:///home/vijetapriya/ride-sharing-1/shared/tracing/tracing.go#L22-L41):

```go
package tracing

import (
    "context"
    "fmt"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
    "go.opentelemetry.io/otel/trace"
)

type Config struct {
    ServiceName    string  // Identifies the service in Jaeger UI
    Environment    string  // e.g., "development", "production"
    JaegerEndpoint string  // Jaeger collector URL
}

func InitTracer(cfg Config) (func(context.Context) error, error) {
    // 1. Create Jaeger exporter
    traceExporter, err := newExporter(cfg.JaegerEndpoint)
    if err != nil {
        return nil, err
    }

    // 2. Create trace provider with resource attributes
    traceProvider, err := newTraceProvider(cfg, traceExporter)
    if err != nil {
        return nil, err
    }
    
    // 3. Set global tracer provider
    otel.SetTracerProvider(traceProvider)

    // 4. Configure trace context propagation
    prop := newPropagator()
    otel.SetTextMapPropagator(prop)

    // Return shutdown function for graceful cleanup
    return traceProvider.Shutdown, nil
}

func GetTracer(name string) trace.Tracer {
    return otel.GetTracerProvider().Tracer(name)
}
```

### Implementation Details

#### 1. Exporter Creation

```go
func newExporter(endpoint string) (sdktrace.SpanExporter, error) {
    return jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint(endpoint),
    ))
}
```

The exporter sends trace data to Jaeger's HTTP collector endpoint.

#### 2. Resource Attributes

```go
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
        sdktrace.WithBatcher(exporter),  // Batch spans for efficiency
        sdktrace.WithResource(res),
    )

    return traceProvider, nil
}
```

> [!IMPORTANT]
> **Resource Attributes**: Service name and environment are attached to ALL spans from this service, enabling filtering in Jaeger UI.

#### 3. Context Propagation

```go
func newPropagator() propagation.TextMapPropagator {
    return propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},  // W3C Trace Context standard
        propagation.Baggage{},       // W3C Baggage for custom metadata
    )
}
```

Uses **W3C Trace Context** standard for interoperability with other observability tools.

---

## HTTP Instrumentation

### Handler Wrapping

[http.go](file:///home/vijetapriya/ride-sharing-1/shared/tracing/http.go) provides a simple wrapper using OpenTelemetry's `otelhttp`:

```go
package tracing

import (
    "net/http"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func WrapHandlerFunc(handler http.HandlerFunc, operation string) http.Handler {
    return otelhttp.NewHandler(handler, operation)
}
```

### Features Provided by `otelhttp.NewHandler`

| Feature | Description |
|---------|-------------|
| **Automatic span creation** | Creates a span for each HTTP request |
| **HTTP attributes** | Captures method, URL, status code, user agent |
| **Context injection** | Injects `traceparent` and `tracestate` headers in responses |
| **Context extraction** | Extracts trace context from incoming request headers |
| **Error recording** | Marks spans with error status on HTTP 5xx responses |

### Usage Example

From [api-gateway/main.go](file:///home/vijetapriya/ride-sharing-1/services/api-gateway/main.go#L52-L62):

```go
mux := http.NewServeMux()

mux.Handle("POST /trip/preview", 
    tracing.WrapHandlerFunc(enableCORS(handleTripPreview), "/trip/preview"))

mux.Handle("POST /trip/start", 
    tracing.WrapHandlerFunc(enableCORS(handleTripStart), "/trip/start"))

mux.Handle("/ws/drivers", 
    tracing.WrapHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        handleDriversWebSocket(w, r, rabbitmq)
    }, "/ws/drivers"))
```

### Request Flow

```mermaid
sequenceDiagram
    participant Client
    participant otelhttp
    participant Handler
    participant Jaeger
    
    Client->>otelhttp: HTTP Request
    Note right of otelhttp: Extract trace context<br/>from headers
    otelhttp->>otelhttp: Create span
    Note right of otelhttp: span.name = operation<br/>span.attributes = HTTP metadata
    otelhttp->>Handler: r.Context() with span
    Handler->>Handler: Process request
    Handler-->>otelhttp: Response + error
    Note right of otelhttp: Set span status<br/>Record HTTP status
    otelhttp-->>Client: HTTP Response
    otelhttp->>Jaeger: Export span
```

---

## gRPC Tracing

### Interceptor Configuration

[grpc.go](file:///home/vijetapriya/ride-sharing-1/shared/tracing/grpc.go) uses OpenTelemetry's `otelgrpc` stats handlers:

```go
package tracing

import (
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
    "go.opentelemetry.io/otel"
    "google.golang.org/grpc"
    "google.golang.org/grpc/stats"
)

// For gRPC servers
func WithTracingInterceptors() []grpc.ServerOption {
    return []grpc.ServerOption{
        grpc.StatsHandler(newServerHandler()),
    }
}

// For gRPC clients
func DialOptionsWithTracing() []grpc.DialOption {
    return []grpc.DialOption{
        grpc.WithStatsHandler(newClientHandler()),
    }
}

func newClientHandler() stats.Handler {
    return otelgrpc.NewClientHandler(
        otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
    )
}

func newServerHandler() stats.Handler {
    return otelgrpc.NewServerHandler(
        otelgrpc.WithTracerProvider(otel.GetTracerProvider()),
    )
}
```

### gRPC Server Usage

From [trip-service/cmd/main.go](file:///home/vijetapriya/ride-sharing-1/services/trip-service/cmd/main.go#L87):

```go
import (
    "google.golang.org/grpc"
    "ride-sharing/shared/tracing"
)

// Create gRPC server with tracing enabled
grpcServer := grpc.NewServer(tracing.WithTracingInterceptors()...)

// Register service implementations
pb.RegisterTripServiceServer(grpcServer, tripServiceImpl)

// Start serving
if err := grpcServer.Serve(lis); err != nil {
    log.Fatal(err)
}
```

### gRPC Client Usage

```go
import (
    "google.golang.org/grpc"
    "ride-sharing/shared/tracing"
)

conn, err := grpc.Dial(
    "trip-service:9093",
    grpc.WithInsecure(),
    tracing.DialOptionsWithTracing()...,
)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewTripServiceClient(conn)

// Context from parent span is automatically propagated
resp, err := client.PreviewTrip(ctx, req)
```

### What Gets Traced

| Attribute | Description | Example Value |
|-----------|-------------|---------------|
| `rpc.system` | RPC system | `grpc` |
| `rpc.service` | Service name | `trip.TripService` |
| `rpc.method` | Method name | `PreviewTrip` |
| `rpc.grpc.status_code` | gRPC status code | `0` (OK) |
| `net.peer.name` | Peer hostname | `trip-service` |
| `net.peer.port` | Peer port | `9093` |

---

## RabbitMQ Async Tracing

### Challenge: Async Context Propagation

Unlike HTTP/gRPC where context flows through function calls, RabbitMQ messages are asynchronous. Trace context must be **explicitly injected** into message headers by the publisher and **extracted** by the consumer.

### Custom AMQP Header Carrier

[rabbitmq.go](file:///home/vijetapriya/ride-sharing-1/shared/tracing/rabbitmq.go#L16-L38) implements the `propagation.TextMapCarrier` interface:

```go
package tracing

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
    amqp "github.com/rabbitmq/amqp091-go"
)

// amqpHeadersCarrier adapts AMQP headers to OTel's TextMapCarrier interface
type amqpHeadersCarrier amqp.Table

func (c amqpHeadersCarrier) Get(key string) string {
    if v, ok := c[key]; ok {
        if s, ok := v.(string); ok {
            return s
        }
    }
    return ""
}

func (c amqpHeadersCarrier) Set(key string, value string) {
    c[key] = value
}

func (c amqpHeadersCarrier) Keys() []string {
    keys := make([]string, 0, len(c))
    for k := range c {
        keys = append(keys, k)
    }
    return keys
}
```

### Publisher-Side Tracing

[rabbitmq.go](file:///home/vijetapriya/ride-sharing-1/shared/tracing/rabbitmq.go#L40-L74):

```go
func TracedPublisher(
    ctx context.Context,
    exchange, routingKey string,
    msg amqp.Publishing,
    publish func(context.Context, string, string, amqp.Publishing) error,
) error {
    tracer := otel.GetTracerProvider().Tracer("rabbitmq")

    // 1. Create a new span for the publish operation
    ctx, span := tracer.Start(ctx, "rabbitmq.publish",
        trace.WithAttributes(
            attribute.String("messaging.destination", exchange),
            attribute.String("messaging.routing_key", routingKey),
        ),
    )
    defer span.End()

    // 2. Extract business context (optional, for better observability)
    var msgBody contracts.AmqpMessage
    if err := json.Unmarshal(msg.Body, &msgBody); err == nil {
        if msgBody.OwnerID != "" {
            span.SetAttributes(attribute.String("messaging.owner_id", msgBody.OwnerID))
        }
    }

    // 3. Inject trace context into message headers
    if msg.Headers == nil {
        msg.Headers = make(amqp.Table)
    }
    carrier := amqpHeadersCarrier(msg.Headers)
    otel.GetTextMapPropagator().Inject(ctx, carrier)
    msg.Headers = amqp.Table(carrier)

    // 4. Publish the message
    if err := publish(ctx, exchange, routingKey, msg); err != nil {
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    return nil
}
```

### Consumer-Side Tracing

[rabbitmq.go](file:///home/vijetapriya/ride-sharing-1/shared/tracing/rabbitmq.go#L76-L106):

```go
func TracedConsumer(
    delivery amqp.Delivery,
    handler func(context.Context, amqp.Delivery) error,
) error {
    // 1. Extract trace context from message headers
    carrier := amqpHeadersCarrier(delivery.Headers)
    ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)

    tracer := otel.GetTracerProvider().Tracer("rabbitmq")

    // 2. Create a new span linked to the publisher's trace
    ctx, span := tracer.Start(ctx, "rabbitmq.consume",
        trace.WithAttributes(
            attribute.String("messaging.destination", delivery.Exchange),
            attribute.String("messaging.routing_key", delivery.RoutingKey),
        ),
    )
    defer span.End()

    // 3. Extract business context
    var msgBody contracts.AmqpMessage
    if err := json.Unmarshal(delivery.Body, &msgBody); err == nil {
        if msgBody.OwnerID != "" {
            span.SetAttributes(attribute.String("messaging.owner_id", msgBody.OwnerID))
        }
    }

    // 4. Execute the handler
    if err := handler(ctx, delivery); err != nil {
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    return nil
}
```

### Integration with RabbitMQ Client

From [shared/messaging/rabbitmq.go](file:///home/vijetapriya/ride-sharing-1/shared/messaging/rabbitmq.go):

#### Publisher Integration (L123-L138)

```go
func (r *RabbitMQ) PublishMessage(
    ctx context.Context, 
    routingKey string, 
    message contracts.AmqpMessage,
) error {
    jsonMsg, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %v", err)
    }

    msg := amqp.Publishing{
        DeliveryMode: amqp.Persistent,
        ContentType:  "application/json",
        Body:         jsonMsg,
    }

    // Automatically traced!
    return tracing.TracedPublisher(ctx, TripExchange, routingKey, msg, r.publish)
}
```

#### Consumer Integration (L79-L114)

```go
func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {
    msgs, err := r.Channel.Consume(queueName, ...)
    if err != nil {
        return err
    }

    go func() {
        for msg := range msgs {
            // Wrap handler with tracing
            if err := tracing.TracedConsumer(msg, func(ctx context.Context, d amqp.Delivery) error {
                // Retry logic with traced context
                return retry.WithBackoff(ctx, cfg, func() error {
                    return handler(ctx, d)  // Context includes trace info!
                })
            }); err != nil {
                log.Printf("Error processing message: %v", err)
            }
        }
    }()

    return nil
}
```

### Trace Visualization

```mermaid
graph LR
    A[HTTP Request] --> B[api-gateway:8081]
    B --> C[Span: POST /trip/start]
    C --> D[rabbitmq.publish]
    D --> E[RabbitMQ Exchange]
    E --> F[Queue: trip.created]
    F --> G[trip-service]
    G --> H[rabbitmq.consume]
    H --> I[Span: handleTripCreated]
    
    style C fill:#4285f4
    style D fill:#60d7a7
    style H fill:#60d7a7
    style I fill:#4285f4
    
    linkStyle 3,7 stroke:#ff6600,stroke-width:3px
```

> [!IMPORTANT]
> **Trace Continuity**: Even though messages are processed asynchronously (potentially seconds later), the consumer's span is **linked** to the publisher's span via the propagated trace ID.

---

## Integration Examples

### Complete Service Initialization

From [api-gateway/main.go](file:///home/vijetapriya/ride-sharing-1/services/api-gateway/main.go#L22-L39):

```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "ride-sharing/shared/env"
    "ride-sharing/shared/tracing"
)

func main() {
    log.Println("Starting API Gateway")

    // 1. Initialize OpenTelemetry tracing
    tracerCfg := tracing.Config{
        ServiceName:    "api-gateway",
        Environment:    env.GetString("ENVIRONMENT", "development"),
        JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
    }

    shutdown, err := tracing.InitTracer(tracerCfg)
    if err != nil {
        log.Fatalf("Failed to initialize the tracer: %v", err)
    }

    // 2. Ensure graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    defer shutdown(ctx)  // Flush remaining spans on exit

    // 3. Setup HTTP handlers with tracing
    mux := http.NewServeMux()
    mux.Handle("POST /trip/preview", 
        tracing.WrapHandlerFunc(handleTripPreview, "/trip/preview"))

    // 4. Start server
    server := &http.Server{Addr: ":8081", Handler: mux}
    log.Fatal(server.ListenAndServe())
}
```

### Multi-Protocol Service (gRPC + RabbitMQ)

From [trip-service/cmd/main.go](file:///home/vijetapriya/ride-sharing-1/services/trip-service/cmd/main.go#L24-L88):

```go
package main

import (
    "context"
    "log"
    "net"
    
    "ride-sharing/shared/tracing"
    "ride-sharing/shared/messaging"
    "google.golang.org/grpc"
)

func main() {
    // 1. Initialize tracing
    tracerCfg := tracing.Config{
        ServiceName:    "trip-service",
        Environment:    env.GetString("ENVIRONMENT", "development"),
        JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
    }
    shutdown, err := tracing.InitTracer(tracerCfg)
    if err != nil {
        log.Fatalf("Failed to initialize tracer: %v", err)
    }
    defer shutdown(context.Background())

    // 2. Setup gRPC server with tracing
    lis, err := net.Listen("tcp", ":9093")
    if err != nil {
        log.Fatal(err)
    }

    grpcServer := grpc.NewServer(tracing.WithTracingInterceptors()...)
    pb.RegisterTripServiceServer(grpcServer, serviceImpl)

    // 3. Setup RabbitMQ consumers (automatically traced)
    rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
    if err != nil {
        log.Fatal(err)
    }
    defer rabbitmq.Close()

    driverConsumer := events.NewDriverConsumer(rabbitmq, svc)
    go driverConsumer.Listen()  // Uses TracedConsumer internally

    // 4. Start gRPC server
    log.Printf("Starting gRPC server on :9093")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatal(err)
    }
}
```

### End-to-End Trace Example

**Scenario**: User requests trip preview → API Gateway calls Trip Service via gRPC

```mermaid
graph TB
    subgraph "Span Hierarchy"
        ROOT[Root Span: POST /trip/preview<br/>Service: api-gateway]
        GRPC_CLIENT[Child Span: grpc.client.PreviewTrip<br/>Service: api-gateway]
        GRPC_SERVER[Child Span: grpc.server.PreviewTrip<br/>Service: trip-service]
        DB[Child Span: mongodb.find<br/>Service: trip-service]
    end
    
    ROOT --> GRPC_CLIENT
    GRPC_CLIENT --> GRPC_SERVER
    GRPC_SERVER --> DB
    
    style ROOT fill:#4285f4
    style GRPC_CLIENT fill:#60d7a7
    style GRPC_SERVER fill:#60d7a7
    style DB fill:#fbbc04
```

**Resulting Jaeger Trace**:
- **Trace ID**: `a1b2c3d4e5f6g7h8` (shared across all spans)
- **Span 1**: `POST /trip/preview` (duration: 245ms)
  - **Span 2**: `grpc.client.PreviewTrip` (duration: 240ms)
    - **Span 3**: `grpc.server.PreviewTrip` (duration: 235ms)
      - **Span 4**: `mongodb.find` (duration: 180ms)

---

## Configuration Guide

### Environment Variables

Each service requires the following environment variables:

```bash
# Service identification
SERVICE_NAME=api-gateway
ENVIRONMENT=production

# Jaeger collector endpoint
JAEGER_ENDPOINT=http://jaeger:14268/api/traces

# Optional: Sampling configuration (defaults to 100%)
OTEL_TRACES_SAMPLER=always_on
```

### Kubernetes ConfigMap Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: telemetry-config
data:
  JAEGER_ENDPOINT: "http://jaeger:14268/api/traces"
  ENVIRONMENT: "production"
  OTEL_TRACES_SAMPLER: "always_on"
```

Reference in Deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
spec:
  template:
    spec:
      containers:
      - name: api-gateway
        image: api-gateway:latest
        env:
        - name: SERVICE_NAME
          value: "api-gateway"
        envFrom:
        - configMapRef:
            name: telemetry-config
```

### Sampling Strategies

| Strategy | Use Case | Configuration |
|----------|----------|---------------|
| `always_on` | Development, debugging | `OTEL_TRACES_SAMPLER=always_on` |
| `always_off` | Disable tracing | `OTEL_TRACES_SAMPLER=always_off` |
| `traceidratio` | Production (sample 10%) | `OTEL_TRACES_SAMPLER=traceidratio`<br/>`OTEL_TRACES_SAMPLER_ARG=0.1` |
| `parentbased_always_on` | Trace if parent is sampled | `OTEL_TRACES_SAMPLER=parentbased_always_on` |

> [!WARNING]
> **Production Sampling**: In high-traffic production environments, consider using `traceidratio` with a value like `0.05` (5%) to reduce overhead while maintaining observability.

---

## Troubleshooting

### Common Issues

#### 1. No Traces Appearing in Jaeger UI

**Symptoms**: Jaeger UI shows no services or traces

**Diagnosis**:
```bash
# Check if Jaeger is running
kubectl get pods -l app=jaeger

# Check service logs for connection errors
kubectl logs <service-pod> | grep -i "jaeger\|tracer"

# Verify Jaeger endpoint is reachable
kubectl exec <service-pod> -- curl http://jaeger:14268/
```

**Solutions**:
- Verify `JAEGER_ENDPOINT` environment variable is correct
- Ensure Jaeger service is running and healthy
- Check network policies allow traffic to Jaeger
- Confirm tracer initialization doesn't return errors

#### 2. Broken Trace Chains (Missing Parent-Child Links)

**Symptoms**: Spans appear as separate traces instead of linked hierarchy

**Diagnosis**:
```bash
# Check if context is being passed correctly
# Look for logs indicating context propagation failures
kubectl logs <service-pod> | grep -i "context\|propagat"
```

**Solutions**:

**HTTP**: Ensure [WrapHandlerFunc](file:///home/vijetapriya/ride-sharing-1/shared/tracing/http.go#9-12) is used for all handlers
```go
// ✅ Correct
mux.Handle("/path", tracing.WrapHandlerFunc(handler, "operation"))

// ❌ Incorrect
mux.HandleFunc("/path", handler)
```

**gRPC**: Verify interceptors are registered
```go
// ✅ Correct
grpc.NewServer(tracing.WithTracingInterceptors()...)

// ❌ Incorrect
grpc.NewServer()
```

**RabbitMQ**: Confirm [TracedPublisher](file:///home/vijetapriya/ride-sharing-1/shared/tracing/rabbitmq.go#40-75) and [TracedConsumer](file:///home/vijetapriya/ride-sharing-1/shared/tracing/rabbitmq.go#76-107) are used
```go
// ✅ Correct
tracing.TracedPublisher(ctx, exchange, key, msg, publishFunc)

// ❌ Incorrect
channel.Publish(exchange, key, false, false, msg)
```

#### 3. High Memory Usage

**Symptoms**: Service memory grows over time

**Possible Causes**:
- Span batching not flushing properly
- Too many custom attributes per span
- Sampler set to `always_on` in production

**Solutions**:
```go
// Ensure shutdown is called on service termination
defer shutdown(context.Background())

// Reduce attribute cardinality
// ❌ High cardinality
span.SetAttributes(attribute.String("user.id", userID))

// ✅ Low cardinality
span.SetAttributes(attribute.String("user.type", "premium"))

// Configure sampling
export OTEL_TRACES_SAMPLER=traceidratio
export OTEL_TRACES_SAMPLER_ARG=0.1
```

#### 4. gRPC Context Not Propagating

**Symptoms**: gRPC client spans don't link to server spans

**Root Cause**: Context not passed to gRPC client call

**Fix**:
```go
// ❌ Wrong: Creates new context without trace info
resp, err := client.PreviewTrip(context.Background(), req)

// ✅ Correct: Uses existing context with trace
resp, err := client.PreviewTrip(ctx, req)
```

---

## Best Practices

### 1. Always Pass Context

> [!IMPORTANT]
> **Context is King**: Always pass `context.Context` through your application layers. This ensures trace continuity across service boundaries.

```go
// ✅ Good
func ProcessTrip(ctx context.Context, tripID string) error {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(attribute.String("trip.id", tripID))
    
    // Pass ctx to all downstream calls
    driver, err := findDriver(ctx, tripID)
    if err != nil {
        return err
    }
    
    return notifyDriver(ctx, driver)
}

// ❌ Bad
func ProcessTrip(tripID string) error {
    // Context not available, cannot trace!
    driver, err := findDriver(tripID)
    return notifyDriver(driver)
}
```

### 2. Add Meaningful Attributes

```go
span := trace.SpanFromContext(ctx)

// Business context
span.SetAttributes(
    attribute.String("trip.id", tripID),
    attribute.String("driver.id", driverID),
    attribute.Float64("trip.distance_km", 12.5),
    attribute.String("payment.method", "card"),
)

// Technical context
span.SetAttributes(
    attribute.Int("retry.attempt", attemptNum),
    attribute.String("database.table", "trips"),
)
```

### 3. Handle Errors Consistently

```go
if err != nil {
    span := trace.SpanFromContext(ctx)
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    return err
}
```

### 4. Use Descriptive Operation Names

```go
// ✅ Descriptive
ctx, span := tracer.Start(ctx, "trip.calculate_fare")
ctx, span := tracer.Start(ctx, "driver.search_nearby")
ctx, span := tracer.Start(ctx, "payment.process_card")

// ❌ Too generic
ctx, span := tracer.Start(ctx, "process")
ctx, span := tracer.Start(ctx, "handler")
```

### 5. Leverage Span Events for Milestones

```go
span.AddEvent("driver_search_started")
// ... search logic ...
span.AddEvent("driver_found", trace.WithAttributes(
    attribute.String("driver.id", driverID),
    attribute.Float64("search.duration_ms", elapsed),
))
```

---

## Summary

This telemetry implementation provides:

✅ **End-to-end tracing** across HTTP, gRPC, and RabbitMQ  
✅ **Automatic instrumentation** with minimal code changes  
✅ **Trace context propagation** across sync and async boundaries  
✅ **Centralized observability** via Jaeger UI  
✅ **Production-ready** with configurable sampling and resource limits

### Quick Reference Card

| Task | Code |
|------|------|
| Initialize tracing | `shutdown, _ := tracing.InitTracer(cfg)` |
| Wrap HTTP handler | `tracing.WrapHandlerFunc(handler, "op")` |
| Add gRPC server tracing | `grpc.NewServer(tracing.WithTracingInterceptors()...)` |
| Add gRPC client tracing | `grpc.Dial(addr, tracing.DialOptionsWithTracing()...)` |
| Trace RabbitMQ publish | `tracing.TracedPublisher(ctx, ex, key, msg, fn)` |
| Trace RabbitMQ consume | `tracing.TracedConsumer(delivery, handler)` |
| Get tracer | `tracer := tracing.GetTracer("my-component")` |
| Create manual span | `ctx, span := tracer.Start(ctx, "operation")` |
| Add attributes | `span.SetAttributes(attribute.String(k, v))` |
| Record error | `span.RecordError(err); span.SetStatus(codes.Error, msg)` |

### Further Reading

- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/instrumentation/go/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [W3C Trace Context Specification](https://www.w3.org/TR/trace-context/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)

---

**Document Version**: 1.0  
**Last Updated**: 2026-02-10  
**Maintained By**: Platform Engineering Team
