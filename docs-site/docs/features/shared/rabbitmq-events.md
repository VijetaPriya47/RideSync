---
sidebar_position: 1
title: RabbitMQ Event Bus
---

# RabbitMQ Event Bus

To decouple microservices and improve resilience against cascading failures, the Hybrid Logistics Engine heavily relies on asynchronous messaging via RabbitMQ (AMQP 0.9.1).

## Publisher / Consumer Abstraction

Rather than having each microservice natively interact with `github.com/rabbitmq/amqp091-go` plumbing, the `shared/messaging/rabbitmq.go` package outlines a unified wrapper struct:

```go
type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}
```

This ensures that whenever any service calls `NewRabbitMQ`, the underlying setup logic creates a unified `TripExchange` and wires up all queue bindings dynamically based on predefined constants:

```go
	if err := r.declareAndBindQueue(
		DriverCmdTripRequestQueue,
		[]string{contracts.DriverCmdTripRequest},
		TripExchange,
	); err != nil {
		return err
	}
```

## Retry and Dead Letter Queues (DLQ)

Transient hardware failures or database deadlocks shouldn't result in dropped messages. The messaging abstraction utilizes a custom exponential backoff retry loop before defaulting to a Dead Letter Queue (DLQ).

```go
// Inside r.ConsumeMessages

        cfg := retry.DefaultConfig()
        err := retry.WithBackoff(ctx, cfg, func() error {
            return handler(ctx, d)
        })

        if err != nil {
            // Add failure context before sending to the DLQ
            headers := amqp.Table{}
            headers["x-death-reason"] = err.Error()
            headers["x-retry-count"] = cfg.MaxRetries
            d.Headers = headers

            // Reject without requeue - message will go to the DLQ
            _ = d.Reject(false)
            return err
        }
```

If the internal handler (`handler(ctx, d)`) fails, the loop will progressively retry until exhausted. If the error still persists, `d.Reject(false)` forces the AMQP broker to funnel the poisoned message specifically into the `DeadLetterExchange`, preserving the raw payload alongside the `x-death-reason` header for manual engineering review.

## Quality of Service (QoS)

To ensure fair dispatch across horizontally scaled containers (e.g., three instances of `trip-service` running simultaneously), the consumers restrict their "prefetch" to exactly 1 message at a time. This prevents a single fast instance from hoarding 100 messages into its local buffer while the other instances idle.

```go
	err := r.Channel.Qos(
		1,     // prefetchCount: Limit to 1 unacknowledged message per consumer
		0,     // prefetchSize: No specific limit on message size
		false, // global: Apply prefetchCount to each consumer individually
	)
```
