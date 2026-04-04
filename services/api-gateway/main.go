package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-sharing/services/api-gateway/grpc_clients"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	httpAddr    = env.GetString("HTTP_ADDR", ":8081")
	rabbitMqURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
)

func main() {
	log.Println("Starting API Gateway")

	tracerCfg := tracing.Config{
		ServiceName:    "api-gateway",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize the tracer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer sh(ctx)

	mux := http.NewServeMux()

	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	startDriverSearchExpiredConsumer(rabbitmq)

	tripGRPC, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatalf("Failed to create trip service gRPC client: %v", err)
	}
	defer tripGRPC.Close()

	driverGRPC, err := grpc_clients.NewDriverServiceClient()
	if err != nil {
		log.Fatalf("Failed to create driver service gRPC client: %v", err)
	}
	defer driverGRPC.Close()

	platformGRPC, err := grpc_clients.NewPlatformGRPC()
	if err != nil {
		log.Fatalf("Failed to create platform-service gRPC client: %v", err)
	}
	defer platformGRPC.Close()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		handleAuthLogin(w, r, platformGRPC.Auth)
	})
	mux.HandleFunc("/api/auth/google", func(w http.ResponseWriter, r *http.Request) {
		handleAuthGoogle(w, r, platformGRPC.Auth)
	})
	mux.HandleFunc("/api/auth/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		handleAuthForgotPassword(w, r, platformGRPC.Auth)
	})
	mux.HandleFunc("/api/auth/reset-password", func(w http.ResponseWriter, r *http.Request) {
		handleAuthResetPassword(w, r, platformGRPC.Auth)
	})

	mux.HandleFunc("/api/finance/me", func(w http.ResponseWriter, r *http.Request) {
		handleFinanceMe(w, r, platformGRPC.Finance)
	})
	mux.HandleFunc("/api/finance/dashboard/revenue", func(w http.ResponseWriter, r *http.Request) {
		handleFinanceDashboardRevenue(w, r, platformGRPC.Finance)
	})
	mux.HandleFunc("/api/finance/dashboard/regions", func(w http.ResponseWriter, r *http.Request) {
		handleFinanceDashboardRegions(w, r, platformGRPC.Finance)
	})
	mux.HandleFunc("/api/finance/dashboard/categories", func(w http.ResponseWriter, r *http.Request) {
		handleFinanceDashboardCategories(w, r, platformGRPC.Finance)
	})

	mux.HandleFunc("/api/admin/system-logs", func(w http.ResponseWriter, r *http.Request) {
		handleAdminSystemLogs(w, r, platformGRPC.Auth)
	})
	mux.HandleFunc("/api/admin/users/business", func(w http.ResponseWriter, r *http.Request) {
		handleAdminRegisterBusiness(w, r, platformGRPC.Auth)
	})
	mux.HandleFunc("/api/admin/users/admin", func(w http.ResponseWriter, r *http.Request) {
		handleAdminRegisterAdmin(w, r, platformGRPC.Auth)
	})

	mux.Handle("/trip/preview", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleTripPreview(w, r, tripGRPC)
	}))
	mux.Handle("/trip/start", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleTripStart(w, r, tripGRPC)
	}))
	mux.Handle("/trip/increase-fare", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleIncreaseTripFare(w, r, tripGRPC)
	}))
	mux.Handle("/trip/update-seats", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleUpdateTripSeats(w, r, tripGRPC)
	}))
	mux.Handle("/trip/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleGetTripStatus(w, r, tripGRPC)
	}))
	mux.Handle("/api/trips/book", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleTripStart(w, r, tripGRPC)
	}))

	mux.Handle("/ws/drivers", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleDriversWebSocket(w, r, rabbitmq, driverGRPC)
	}))
	mux.Handle("/ws/riders", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRidersWebSocket(w, r, rabbitmq)
	}))
	mux.Handle("/webhook/stripe", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleStripeWebhook(w, r, rabbitmq)
	}))

	stack := chainHTTP([]func(http.Handler) http.Handler{
		corsMiddleware,
		jwtMiddleware,
		rbacMiddleware,
		auditMiddleware(rabbitmq),
	}, mux)

	server := &http.Server{
		Addr:    httpAddr,
		Handler: otelhttp.NewHandler(stack, "api-gateway"),
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Server listening on %s", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

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
}
