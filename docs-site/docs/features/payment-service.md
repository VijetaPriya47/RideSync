---
sidebar_position: 1
title: Payment Service Overview
---

# Payment Service

The Payment Service handles all financial transactions using Stripe. It is a fully decoupled consumer that only activates after a driver accepts a trip, ensuring payment never blocks the core dispatch loop.

## 1. Stripe Checkout Session

### AMQP Trigger

The Payment Service runs a background consumer (`services/payment-service/internal/events/trip_consumer.go`) on the `PaymentTripResponseQueue`. When it receives a `PaymentCmdCreateSession` routing key (published by the Trip Service after a driver accepts), it creates the Stripe session:

```go
func (c *TripConsumer) handleTripAccepted(ctx context.Context, payload messaging.PaymentTripResponseData) error {
	paymentSession, err := c.service.CreatePaymentSession(
		ctx,
		payload.TripID,
		payload.UserID,
		payload.DriverID,
		int64(payload.Amount), // In cents!
		payload.Currency,
	)
```

### Session Creation & Metadata

The service contacts the Stripe REST API to generate a Hosted Checkout URL. Critically, it attaches `metadata` (trip_id, user_id, driver_id) to the Stripe payload. This metadata is the only way the backend can reconcile the payment after the user completes the checkout on Stripe's external servers:

```go
metadata := map[string]string{
	"trip_id":   tripID,
	"user_id":   userID,
	"driver_id": driverID,
}
sessionID, err := s.paymentProcessor.CreatePaymentSession(ctx, amount, currency, metadata)
```

### Broadcasting the Session URL

Once Stripe responds with a valid `sessionID`, the Payment Service publishes a `PaymentEventSessionCreated` event. The API Gateway picks this up and sends the Stripe Checkout URL directly to the rider's browser over WebSocket:

```go
paymentPayload := messaging.PaymentEventSessionCreatedData{
	TripID:    payload.TripID,
	SessionID: paymentSession.StripeSessionID,
	Amount:    float64(paymentSession.Amount) / 100.0, // Frontend expects rupees/dollars
	Currency:  paymentSession.Currency,
}
```

---

## 2. Webhook Verification

Handling asynchronous payments securely is critical. The system prevents fake "payment success" payloads using Stripe Webhook Signature Verification.

### The Webhook Endpoint

Stripe sends an HTTP POST to `/webhook/stripe` in the API Gateway whenever a rider completes a checkout. The server does **not** trust the raw JSON body. Instead, it extracts the raw bytes and the `Stripe-Signature` header and validates them cryptographically against the `STRIPE_WEBHOOK_KEY` secret:

```go
body, err := io.ReadAll(r.Body)
// ...
webhookKey := env.GetString("STRIPE_WEBHOOK_KEY", "")

event, err := webhook.ConstructEventWithOptions(
	body,
	r.Header.Get("Stripe-Signature"),
	webhookKey,
	webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true, // Graceful degradation for API version updates
	},
)
if err != nil {
	http.Error(w, "Invalid signature", http.StatusBadRequest)
	return
}
```

If the signature math fails, the request is dropped with `HTTP 400 Bad Request`.

### Completing the Trip

On a valid `checkout.session.completed` event, the server rehydrates the trip metadata embedded during session creation and publishes `PaymentEventSuccess` to RabbitMQ. The Trip Service consumes this and transitions the trip status from `accepted` → `payed`:

```go
switch event.Type {
case "checkout.session.completed":
	payload := messaging.PaymentStatusUpdateData{
		TripID:   session.Metadata["trip_id"],
		UserID:   session.Metadata["user_id"],
		DriverID: session.Metadata["driver_id"],
	}
	// → publishes PaymentEventSuccess
```

> [!NOTE]
> The Trip Service never trusts the end-user device for payment confirmation. The only source of truth is Stripe's cryptographically signed webhook, rehydrated with our own metadata.

**Rider transaction history (`GET /api/finance/me`)** is written by **platform-service** when it consumes the same `PaymentEventSuccess` message and inserts into Postgres. If `trip_id` or `user_id` is missing on the Checkout Session metadata, the gateway does **not** publish that event (and logs a clear reason). Common causes: **`STRIPE_WEBHOOK_KEY` unset** on the gateway (webhook returns 503), Stripe Dashboard webhook URL not pointing at **`/webhook/stripe`**, or **mock Checkout session IDs** from payment-service when Stripe is disabled, errors out, or times out (those sessions never complete in Stripe).

---

## Reliability Note {#reliability-note}

The Payment Service currently relies on the shared **Go-level exponential backoff** (3 retries: 1s → 2s → 4s) built into `shared/messaging/rabbitmq.go`. If the Payment Service crashes mid-retry, any in-flight retry is lost.

For a production financial system, the ideal approach is **broker-level retry** using a dedicated `PaymentRetryExchange` and headless `PaymentWaitQueue`, which preserves messages in RabbitMQ across pod restarts.

See the full design and comparison in [Reliability → Ideal Pattern for Financial Events](./reliability#ideal-pattern-for-financial-events-broker-level-retry).

---

## Resources

- [Stripe Checkout: Getting Started](https://docs.stripe.com/get-started)
- [Stripe Webhook Signature Verification](https://docs.stripe.com/webhooks#verify-official-libraries)
