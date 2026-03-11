---
sidebar_position: 3
title: Graceful Shutdown
---

# Graceful Shutdown

To prevent dropping active connections or interrupting in-flight requests during deployments or scale-downs, the API Gateway implements a graceful shutdown routine.

## Signal Handling

The server listens for operating system interrupt signals (`SIGINT` and `SIGTERM`) instead of crashing immediately:

```go
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
```

## Shutdown Orchestration

A `select` block handles either server startup errors or the interception of a shutdown signal:

```go
	select {
	case err := <-serverErrors:
		log.Printf("Error starting the server: %v", err)

	case sig := <-shutdown:
		log.Printf("Server is shutting down due to %v signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Could not stop the server gracefully: %v", err)
			server.Close()
		}
	}
```

### Context Timeout

1. A background context is created with a strict 10-second timeout: `context.WithTimeout(context.Background(), 10*time.Second)`.
2. The `server.Shutdown(ctx)` command is called, which stops the server from accepting new connections while allowing existing ones to finish.
3. If the active requests take longer than 10 seconds to finish, the context expires, and `server.Shutdown` returns an error, forcing `server.Close()` to forcefully kill any hanging connections.

> [!NOTE]
> This pattern is applied across all microservices to ensure reliable deployments and avoid frustrating client-side errors when pods restart.
