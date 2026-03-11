---
sidebar_position: 1
title: Stripe Checkout
---

# Stripe Checkout Session

When a driver formally accepts a trip, the backend must immediately provision a Stripe Checkout interface for the rider to pay before the trip officially begins.

## AMQP Trigger

The `Payment Service` runs a background consumer (`services/payment-service/internal/events/trip_consumer.go`) that listens on the `PaymentTripResponseQueue`. When it receives a `PaymentCmdCreateSession` routing key (published by the independent Trip Service), it triggers the session creation:

```go
func (c *TripConsumer) handleTripAccepted(ctx context.Context, payload messaging.PaymentTripResponseData) error {
	log.Printf("Handling trip accepted by driver: %s", payload.TripID)

	paymentSession, err := c.service.CreatePaymentSession(
		ctx,
		payload.TripID,
		payload.UserID,
		payload.DriverID,
		int64(payload.Amount), // In cents!
		payload.Currency,
	)
    // ...
```

## Session Creation

The underlying service layer contacts the Stripe REST API to generate the Hosted Checkout URL id. Crucially, the system attaches specific `metadata` to the Stripe payload. This metadata (trip_id, user_id, driver_id) is the only way the system can reconcile the payment once the user completes the flow on Stripe's external servers.

```go
func (s *paymentService) CreatePaymentSession(
	ctx context.Context,
	tripID string,
	userID string,
	driverID string,
	amount int64,
	currency string,
) (*types.PaymentIntent, error) {
	metadata := map[string]string{
		"trip_id":   tripID,
		"user_id":   userID,
		"driver_id": driverID,
	}

	sessionID, err := s.paymentProcessor.CreatePaymentSession(ctx, amount, currency, metadata)
    // ...
```

## Broadcasting the Session URL

Once Stripe responds with a valid `sessionID`, the Payment Service encapsulates this into a new RabbitMQ message `PaymentEventSessionCreated`:

```go
	// Publish payment session created event
	paymentPayload := messaging.PaymentEventSessionCreatedData{
		TripID:    payload.TripID,
		SessionID: paymentSession.StripeSessionID,
		Amount:    float64(paymentSession.Amount) / 100.0, // Frontend expects dollars
		Currency:  paymentSession.Currency,
	}
    // ... publishes PaymentEventSessionCreated
```

The API Gateway picks up this message and pipes it directly over the active WebSocket connection to the Rider's Next.js frontend, telling React to redirect the browser to the Stripe Hosted Checkout page.

## Stripe API Resources

- [Stripe Checkout: Getting Started](https://docs.stripe.com/get-started)

