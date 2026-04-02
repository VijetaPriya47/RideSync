---
sidebar_position: 2
title: Dispatch Algorithm
---

# Dispatch Algorithm

When a rider locks in an OSRM-rated trip, the Trip Service broadcasts an asynchronous RabbitMQ message on the `FindAvailableDriversQueue`. The Driver Service is responsible for consuming this and distributing it out to the active driver pool.

## 1. AMQP Consumer

The background worker in `services/driver-service/trip_consumer.go` listens for both `TripEventCreated` and `TripEventDriverNotInterested` (re-dispatch) events:

```go
func (c *tripConsumer) Listen() error {
	return c.rabbitmq.ConsumeMessages(messaging.FindAvailableDriversQueue, func(ctx context.Context, msg amqp091.Delivery) error {
        // ... unmarshal payload
		switch msg.RoutingKey {
		case contracts.TripEventCreated, contracts.TripEventDriverNotInterested:
			return c.handleFindAndNotifyDrivers(ctx, payload)
		}
		return nil
	})
}
```

## 2. Filtering Suitable Drivers

The `handleFindAndNotifyDrivers` function delegates to the core service struct to filter the currently retained in-memory drivers:

```go
	// Check HTTP endpoint to ensure trip is still valid and fare is not outdated
	url := fmt.Sprintf("%s/trips/%s", c.tripSvcURL, payload.Trip.Id)
	// If tripStatus.RideFare.TotalPriceInCents > payload.Trip.SelectedFare.TotalPriceInCents { 
	//    return fmt.Errorf("outdated_fare") // Triggers AMQP rejection to DLQ
    // }

	allSuitableIDs := c.service.FindAvailableDrivers(payload.Trip.SelectedFare.PackageSlug, reqSeats, payload.Trip.Route)

	// Filter out already tried drivers (resolves infinite loop issue matching the same declining driver)
	var suitableIDs []string
	triedMap := make(map[string]bool)
	for _, id := range payload.TriedDriverIDs {
		triedMap[id] = true
	}
	for _, id := range allSuitableIDs {
		if !triedMap[id] {
			suitableIDs = append(suitableIDs, id)
		}
	}

	if len(suitableIDs) == 0 {
		// Respond backward if no capacity exists
		if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventNoDriversFound...
		return nil
	}
```

The dispatch sequence was recently upgraded to include `TriedDriverIDs` array tracking. When a driver declines a ride or naturally times out, their `driverId` is added to the event payload. The search algorithm filters these out, preventing the system from matching the same declining driver in an infinite loop. 

Additionally, if a rider clicks "Increase Fare", a brand new queue request is broadcast. The HTTP check explicitly cross-references the MongoDB fare; if the active AMQP payload fare is lower than the new increased API base fare, the outdated message throws a native error forcing it to be silently rejected into the Dead Letter Queue (DLQ).

The underlying `FindAvailableDrivers` locks the array and filters strictly by `PackageSlug` and `RequestedSeats`:

```go
func (s *Service) FindAvailableDrivers(packageType string) []string {
	var matchingDrivers []string

	for _, driver := range s.drivers {
		if driver.Driver.PackageSlug == packageType {
			matchingDrivers = append(matchingDrivers, driver.Driver.Id)
		}
	}

	return matchingDrivers
}
```

## 3. Dispersal

To simulate fair dispatch logic and prevent overloading a single driver, the system selects one matching driver randomly from the pool and publishes the `DriverCmdTripRequest`, which is eventually channeled through the API Gateway WebSocket directly to the singular dispatched driver's device screen.

```go
	// Get a random index from the matching drivers
	randomIndex := rand.Intn(len(suitableIDs))
	suitableDriverID := suitableIDs[randomIndex]

	// Notify the isolated driver about a potential trip
	if err := c.rabbitmq.PublishMessage(ctx, contracts.DriverCmdTripRequest, contracts.AmqpMessage{
		OwnerID: suitableDriverID,
		Data:    marshalledEvent,
	}); err != nil {
		return err
	}
```
