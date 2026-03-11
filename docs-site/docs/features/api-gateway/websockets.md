---
sidebar_position: 1
title: WebSockets
---

# WebSockets in API Gateway

The API Gateway provides real-time bidirectional communication using WebSockets. This is crucial for keeping both drivers and riders updated with live trip events, location tracking, and dispatching.

## Connection Management

The `connManager` (a `ConnectionManager` from `shared/messaging`) handles upgrading HTTP requests to WebSocket connections and managing active client sessions.

```go
var (
	connManager = messaging.NewConnectionManager()
)
```

### Upgrading the Connection

When a client hits the `/ws/riders` or `/ws/drivers` endpoints, the HTTP connection is upgraded:

```go
func handleRidersWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {
	conn, err := connManager.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()
	// ...
```

The `userID` is extracted from the query parameters to uniquely identify the connection. If successful, the connection is added to the connection manager:

```go
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Println("No user ID provided")
		return
	}

	// Add connection to manager
	connManager.Add(userID, conn)
	defer connManager.Remove(userID)
```

## Bridging WebSockets and RabbitMQ

A major feature of the API Gateway is bridging synchronous WebSocket connections with asynchronous RabbitMQ message queues.

### Consuming Messages

For a rider connection, the gateway initializes queue consumers that listen for events related to that rider and push them down the specific WebSocket connection:

```go
	// Initialize queue consumers
	queues := []string{
		messaging.NotifyDriverNoDriversFoundQueue,
		messaging.NotifyDriverAssignQueue,
		messaging.NotifyPaymentSessionCreatedQueue,
		messaging.NotifyTripCreatedQueue,
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s: err: %v", q, err)
		}
	}
```

### Publishing Messages

When the gateway receives a message from a driver over the WebSocket connection (like accepting or declining a trip), it parses it and publishes it back into RabbitMQ to be handled asynchronously by the Driver Service:

```go
		var driverMsg driverMessage
		if err := json.Unmarshal(message, &driverMsg); err != nil {
			log.Printf("Error unmarshaling driver message: %v", err)
			continue
		}

		// Handle the different message type
		switch driverMsg.Type {
		case contracts.DriverCmdLocation:
			// Handle driver location update in the future
			continue
		case contracts.DriverCmdTripAccept, contracts.DriverCmdTripDecline:
			// Forward the message to RabbitMQ
			if err := rb.PublishMessage(ctx, driverMsg.Type, contracts.AmqpMessage{
				OwnerID: userID,
				Data:    driverMsg.Data,
			}); err != nil {
				log.Printf("Error publishing message to RabbitMQ: %v", err)
			}
		default:
			log.Printf("Unknown message type: %s", driverMsg.Type)
		}
```

## Driver Registration via gRPC

For drivers, connecting to the WebSocket also triggers a gRPC call to register them as available in the internal system:

```go
	driverService, err := grpc_clients.NewDriverServiceClient()
	// ...
	driverData, err := driverService.Client.RegisterDriver(ctx, &driver.RegisterDriverRequest{
		DriverID:    userID,
		PackageSlug: packageSlug,
	})
```

Upon WebSocket disconnection, a deferred function automatically calls the gRPC `UnregisterDriver` endpoint to remove the driver from the available pool:

```go
	// Closing connections
	defer func() {
		connManager.Remove(userID)

		driverService.Client.UnregisterDriver(ctx, &driver.RegisterDriverRequest{
			DriverID:    userID,
			PackageSlug: packageSlug,
		})

		driverService.Close()
		log.Println("Driver unregistered: ", userID)
	}()

## Further Reading & Resources

- [MDN WebSockets API Reference](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API)
- [Gorilla WebSocket Documentation](https://pkg.go.dev/github.com/gorilla/websocket#section-readme)
- [What is CORS? - AWS](https://aws.amazon.com/what-is/cross-origin-resource-sharing/)
- [MDN CORS Deep Dive](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS#specifications)

```
