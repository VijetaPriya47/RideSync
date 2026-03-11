---
sidebar_position: 2
title: Webhook Verification
---

# Webhook Verification

Handling asynchronous payments securely is critical. Users might try to bypass payment by sending fake "payment success" payloads directly to the backend. The Hybrid Logistics Engine prevents this using Stripe Webhook Signature Verification.

## The Webhook Endpoint

Stripe sends an HTTP POST to the `/webhook/stripe` endpoint whenever a user finishes interacting with their hosted checkout system. Because this is an external HTTP call, it is handled directly inside the `API Gateway` (`services/api-gateway/http.go`).

## Constructing the Event Safely

Instead of simply unwrapping the JSON payload and trusting its contents, the server extracts the raw HTTP body bytes and the cryptographic `Stripe-Signature` header. It compares this against the heavily restricted `STRIPE_WEBHOOK_KEY` environment variable secret:

```go
	body, err := io.ReadAll(r.Body)
    // ...

	webhookKey := env.GetString("STRIPE_WEBHOOK_KEY", "")

	event, err := webhook.ConstructEventWithOptions(
		body,
		r.Header.Get("Stripe-Signature"),
		webhookKey,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true, // Graceful degradation for API updates
		},
	)
	if err != nil {
		log.Printf("Error verifying webhook signature: %v", err)
		http.Error(w, "Invalid signature", http.StatusBadRequest)
		return
	}
```

If the signature math fails, the request is immediately dropped with an `HTTP 400 Bad Request`.

## Fulfilling the Event

If the cryptographic signature is valid, the server trusts that the event absolutely originated from Stripe's internal servers. It intercepts the `checkout.session.completed` event to finalize the trip states:

```go
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession

		err := json.Unmarshal(event.Data.Raw, &session)
		
        // Rehydrate the metadata we attached during Session Creation
		payload := messaging.PaymentStatusUpdateData{
			TripID:   session.Metadata["trip_id"],
			UserID:   session.Metadata["user_id"],
			DriverID: session.Metadata["driver_id"],
		}
```

The gateway extracts the custom metadata we appended during the Session Creation phase. It wraps this into a `PaymentEventSuccess` RabbitMQ packet. The Trip Service consumes this exact packet to finalize the MongoDB state (transitioning from `accepted` to `payed`) without trusting the end-user's device whatsoever.
