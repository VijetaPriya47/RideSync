package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"ride-sharing/services/payment-service/internal/events"
	"ride-sharing/services/payment-service/internal/infrastructure/stripe"
	"ride-sharing/services/payment-service/internal/service"
	"ride-sharing/services/payment-service/pkg/types"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
)

var GrpcAddr = env.GetString("GRPC_ADDR", ":9004")

func main() {
	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "payment-service",
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

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	// Setup graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	appURL := env.GetString("APP_URL", "http://localhost:3000")

	// Stripe config
	stripeCfg := &types.PaymentConfig{
		StripeSecretKey: env.GetString("STRIPE_SECRET_KEY", ""),
		SuccessURL:      env.GetString("STRIPE_SUCCESS_URL", appURL+"?payment=success"),
		CancelURL:       env.GetString("STRIPE_CANCEL_URL", appURL+"?payment=cancel"),
		UseStripeAPI:    env.GetBool("USE_STRIPE_API", false), // Toggle this to true to use real Stripe API
	}

	if stripeCfg.StripeSecretKey == "" {
		// Log warning instead of fatal if we are mocking
		if stripeCfg.UseStripeAPI {
			log.Fatalf("STRIPE_SECRET_KEY is not set")
			return
		} else {
			log.Printf("STRIPE_SECRET_KEY is not set (running in mock mode)")
		}
	}

	// Stripe processor
	paymentProcessor := stripe.NewStripeClient(stripeCfg)

	// Service
	svc := service.NewPaymentService(paymentProcessor)

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	// Trip Consumer
	tripConsumer := events.NewTripConsumer(rabbitmq, svc)
	go tripConsumer.Listen()

	// Start a dummy HTTP server for Render Web Service compatibility
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Payment Service is Healthy"))
		})
		log.Printf("Starting health check HTTP server for Render on port %s", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Printf("Failed to start health check HTTP server: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down payment service...")
}
