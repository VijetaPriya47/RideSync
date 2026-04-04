package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"ride-sharing/services/finance-service/internal/consumer"
	"ride-sharing/services/finance-service/internal/grpcsvc"
	"ride-sharing/services/finance-service/internal/repo"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/finance"
	"ride-sharing/shared/sqlmigrate"
	"ride-sharing/shared/tracing"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "finance-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}
	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("tracer: %v", err)
	}
	ctx := context.Background()
	defer sh(ctx)

	dsn := env.GetString("DATABASE_URL", "postgres://ridesync:ridesync@postgres:5432/ridesync?sslmode=disable")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	schemaPath := env.GetString("SQL_SCHEMA_PATH", "infra/sql/001_schema.sql")
	if err := sqlmigrate.ApplyFile(ctx, pool, schemaPath); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	rabbitURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	rmq, err := messaging.NewRabbitMQ(rabbitURI)
	if err != nil {
		log.Fatalf("rabbitmq: %v", err)
	}
	defer rmq.Close()

	rep := &repo.Repo{Pool: pool}
	if err := consumer.ListenPaymentSuccess(rmq, rep); err != nil {
		log.Fatalf("consumer: %v", err)
	}

	grpcAddr := env.GetString("GRPC_ADDR", ":9094")
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(tracing.WithTracingInterceptors()...)
	pb.RegisterFinanceServiceServer(srv, &grpcsvc.Server{Repo: rep})

	go func() {
		log.Printf("finance-service gRPC on %s", grpcAddr)
		if err := srv.Serve(lis); err != nil {
			log.Printf("grpc serve: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	srv.GracefulStop()
}
