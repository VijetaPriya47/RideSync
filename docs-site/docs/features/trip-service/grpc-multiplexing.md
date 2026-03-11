---
sidebar_position: 4
title: gRPC and HTTP Multiplexing
---

# gRPC and HTTP Multiplexing

To simplify deployment topologies and reduce the number of exposed ports per microservice container, the Trip Service employs a multiplexer to serve both `HTTP/1.1` health checks and `HTTP/2` gRPC traffic over a single unified port.

## The Problem

Traditional gRPC servers bind exclusively to a port (`:9093`) and expect strict HTTP/2 frames. However, infrastructure tools like Kubernetes or AWS Application Load Balancers often send simple `HTTP GET /` probes to assess container health. If they hit a strict gRPC port, the health check fails.

## The Solution

In `services/trip-service/cmd/main.go`, we use the `golang.org/x/net/http2/h2c` package to sniff the incoming request protocol and route it dynamically.

### 1. HTTP Handler Setup

First, standard HTTP routes for health-checking are instantiated on a standard HTTP multiplexer:

```go
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Trip Service is Healthy"))
	})
```

### 2. Protocol Sniffing Handler

Next, the `h2c` (HTTP/2 Cleartext) handler intercepts traffic and inspects the `ProtoMajor` version and the `Content-Type` header:

```go
	h2Handler := h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
            // Forward strict gRPC frames to the gRPC handler
			grpcServer.ServeHTTP(w, r)
		} else {
            // Fallback plain HTTP/1.1 requests to the health-check handler
			mux.ServeHTTP(w, r)
		}
	}), &http2.Server{})
```

### 3. Unified Server

The single `http.Server` starts listening on the configured `$PORT` (default `:8080`), avoiding the need to bind both an HTTP listener and a distinct `net.Listen("tcp", GrpcAddr)`.

```go
	server := &http.Server{
		Addr:    ":" + port,
		Handler: h2Handler,
	}

	go func() {
		log.Printf("Starting Multiplexed Server (gRPC + HTTP) on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()
```

> [!TIP]
> This pattern allows Kubernetes readiness/liveness probes to safely query `:8080/` via HTTP while allowing internal API Gateway clusters to establish persistent long-lived gRPC connections to the exact same `:8080` port.
