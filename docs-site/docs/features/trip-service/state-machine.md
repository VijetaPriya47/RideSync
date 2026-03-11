---
sidebar_position: 3
title: Trip State Machine
---

# Trip State Machine

A ride goes through several status changes throughout its lifecycle. The Trip Service acts as the authoritative truth for the current state of a Trip.

## 1. Pending

When a user formally accepts a ride fare and requests a driver, the trip is created in MongoDB with an initial `pending` status. At this phase, there is no assigned driver.

```go
func (s *service) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	t := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		Status:   "pending",
		RideFare: fare,
		Driver:   nil, // Driver is initially nil until assigned
	}

	return s.repo.CreateTrip(ctx, t)
}
```

## 2. Accepted

The `Driver Service` consumes the trip request and pushes it to available driver WebSockets. When a driver clicks "Accept," the `Driver Consumer` inside the Trip Service receives the AMQP message and transitions the Trip status to `accepted`, cementing the relationship between Rider and Driver.

```go
// Inside events/driver_consumer.go -> handleTripAccepted
func (c *driverConsumer) handleTripAccepted(ctx context.Context, tripID string, driver *pbd.Driver) error {
	// ... validation ...
	
	// Update the trip with the assigned Driver structure
	if err := c.service.UpdateTrip(ctx, tripID, "accepted", driver); err != nil {
		log.Printf("Failed to update the trip: %v", err)
		return err
	}
    // ...
```

This interaction triggers two cascading actions via RabbitMQ:
1. `TripEventDriverAssigned`: Notifies the Rider WebSocket that a driver has accepted.
2. `PaymentCmdCreateSession`: Notifies the Payment Service to spin up a Stripe checkout.

## 3. Payed

Once the system completes the Stripe webhook callback verifying a successful capture, the Payment Service throws a `NotifyPaymentSuccessQueue` message. The Trip Service consumes this and transitions the trip state to `payed`.

```go
// Inside events/payment_consumer.go -> Listen()
		log.Printf("Trip has been completed and payed.")

		return c.service.UpdateTrip(
			ctx,
			payload.TripID,
			"payed",   // Status transition
			nil,
		)
```

> [!NOTE]
> If a driver declines (`DriverCmdTripDecline`), the Trip remains in the `pending` state, but the service broadcasts a `TripEventDriverNotInterested` back to the Rider so the frontend UI can react accordingly.
