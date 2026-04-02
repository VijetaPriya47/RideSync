package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"ride-sharing/services/trip-service/internal/infrastructure/driverclient"
	"ride-sharing/services/trip-service/internal/infrastructure/events"
	"ride-sharing/services/trip-service/internal/infrastructure/grpc"
	"ride-sharing/services/trip-service/internal/infrastructure/repository"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/db"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/tracing"
	"strings"
	"syscall"

	grpcserver "google.golang.org/grpc"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var GrpcAddr = env.GetString("GRPC_ADDR", ":9093")

func main() {
	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "trip-service",
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

	// Initialize MongoDB
	mongoClient, err := db.NewMongoClient(ctx, db.NewMongoDefaultConfig())
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB, err: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	mongoDb := db.GetDatabase(mongoClient, db.NewMongoDefaultConfig())

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	mongoDBRepo := repository.NewMongoRepository(mongoDb)
	svc := service.NewService(mongoDBRepo)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Starting RabbitMQ connection")

	publisher := events.NewTripEventPublisher(rabbitmq)

	var seatSync events.SeatNotifier
	drvClient, err := driverclient.New()
	if err != nil {
		log.Printf("WARN: driver gRPC client: %v (seat sync disabled)", err)
	} else {
		defer func() { _ = drvClient.Close() }()
		seatSync = drvClient
	}

	driverConsumer := events.NewDriverConsumer(rabbitmq, svc, mongoDBRepo, seatSync)
	go driverConsumer.Listen()

	// Initialize the gRPC server
	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	grpc.NewGRPCHandler(grpcServer, svc, publisher)

	paymentConsumer := events.NewPaymentConsumer(rabbitmq, svc, drvClient)
	go paymentConsumer.Listen()

	log.Printf("Starting gRPC server Trip service on port %s", GrpcAddr)

	// Combine gRPC and HTTP Health Check on the same port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/trips/", func(w http.ResponseWriter, r *http.Request) {
		// Handle /trips/{id}/verify-otp
		path := strings.TrimPrefix(r.URL.Path, "/trips/")
		parts := strings.SplitN(path, "/", 2)
		tripID := parts[0]
		if tripID == "" {
			http.Error(w, "tripID is required", http.StatusBadRequest)
			return
		}

		if len(parts) == 2 && parts[1] == "verify-otp" && r.Method == http.MethodPost {
			var req struct {
				DriverID string `json:"driverID"`
				OTP      string `json:"otp"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			ok, err := svc.VerifyTripOTP(r.Context(), tripID, req.DriverID, req.OTP)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				json.NewEncoder(w).Encode(map[string]string{"message": "invalid OTP"})
				return
			}
			// OTP verified — now emit payment event
			trip, err := svc.GetTripByID(r.Context(), tripID)
			if err != nil || trip == nil || trip.Driver == nil || trip.RideFare == nil {
				http.Error(w, "trip or fare data missing", http.StatusInternalServerError)
				return
			}
			marshalledPayload, err := json.Marshal(messaging.PaymentTripResponseData{
				TripID:   tripID,
				UserID:   trip.UserID,
				DriverID: trip.Driver.ID,
				Amount:   trip.RideFare.TotalPriceInCents,
				Currency: "USD",
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := rabbitmq.PublishMessage(r.Context(), contracts.PaymentCmdCreateSession, contracts.AmqpMessage{
				OwnerID: trip.UserID,
				Data:    marshalledPayload,
			}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
			return
		}

		// Default: GET /trips/{id}
		trip, err := svc.GetTripByID(r.Context(), tripID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if trip == nil {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(trip)
	})

	mux.HandleFunc("/fares/update-seats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			FareID string `json:"fareID"`
			Seats  int32  `json:"seats"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := mongoDBRepo.UpdateRideFareSeats(r.Context(), req.FareID, req.Seats); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	h2Handler := h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			mux.ServeHTTP(w, r)
		}
	}), &http2.Server{})

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

	// wait for the shutdown signal
	<-ctx.Done()
	log.Println("Shutting down the server...")
	grpcServer.GracefulStop()
}
