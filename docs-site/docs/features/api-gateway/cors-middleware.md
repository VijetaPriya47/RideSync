---
sidebar_position: 2
title: CORS Middleware
---

# CORS Middleware

The API Gateway utilizes custom middleware to handle Cross-Origin Resource Sharing (CORS). This allows the frontend (typically running on a different port like `3000` or a different domain) to communicate with the API Gateway without being blocked by the browser.

## Middleware Implementation

The implementation is a straightforward HTTP handler wrapper found in `services/api-gateway/middleware.go`:

```go
package main

import "net/http"

func enableCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// allow preflight requests from the browser API
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}
```

### Preflight Requests

When browsers prepare to make cross-origin requests, they first send an HTTP `OPTIONS` request known as a preflight request. The middleware detects this and responds with a `200 OK` without passing the request down to the actual business logic handler.

```go
		// allow preflight requests from the browser API
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
```

## Applying the Middleware

In `main.go`, the `enableCORS` middleware is wrapped around the endpoints that require it, often chained together with other middleware like tracing:

```go
	mux.Handle("/trip/preview", tracing.WrapHandlerFunc(enableCORS(handleTripPreview), "/trip/preview"))
	mux.Handle("/trip/start", tracing.WrapHandlerFunc(enableCORS(handleTripStart), "/trip/start"))
```

> [!WARNING]
> In this implementation, `Access-Control-Allow-Origin` is set to `*`. While acceptable for development, a hardened production setup should restrict this to specific trusted domains using environment variables.
