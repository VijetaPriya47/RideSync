package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"ride-sharing/services/finance-service/internal/infrastructure/events"
	grpcinfra "ride-sharing/services/finance-service/internal/infrastructure/grpc"
	"ride-sharing/services/finance-service/internal/infrastructure/repository"
	"ride-sharing/services/finance-service/internal/service"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"ride-sharing/shared/sqlmigrate"
	"ride-sharing/shared/tracing"

	"github.com/jackc/pgx/v5/pgxpool"
	grpcserver "google.golang.org/grpc"
)

func main() {
	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "finance-service",
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

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	dsn := env.GetString("DATABASE_URL", "postgres://ridesync:ridesync@postgres:5432/ridesync?sslmode=disable")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	schemaPath := env.GetString("SQL_SCHEMA_PATH", "infra/sql/001_schema.sql")
	if err := sqlmigrate.ApplyFile(ctx, pool, schemaPath); err != nil {
		log.Fatalf("Failed to apply SQL schema: %v", err)
	}

	rabbitURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	rmq, err := messaging.NewRabbitMQ(rabbitURI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	ledgerRepo := repository.NewPostgresLedger(pool)
	financeSvc := service.NewFinanceService(ledgerRepo)

	paymentConsumer := events.NewPaymentConsumer(rmq, financeSvc)
	if err := paymentConsumer.Listen(); err != nil {
		log.Fatalf("Failed to start finance payment consumer: %v", err)
	}

	grpcAddr := env.GetString("GRPC_ADDR", ":9094")
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	srv := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	grpcinfra.NewGRPCHandler(srv, financeSvc)

	go func() {
		log.Printf("Starting gRPC server finance-service on %s", grpcAddr)
		if err := srv.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down finance-service...")
	srv.GracefulStop()
}
