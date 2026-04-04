package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"ride-sharing/services/user-auth-service/internal/auditconsumer"
	"ride-sharing/services/user-auth-service/internal/grpcsvc"
	"ride-sharing/services/user-auth-service/internal/repo"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/auth"
	"ride-sharing/shared/sqlmigrate"
	"ride-sharing/shared/tracing"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "user-auth-service",
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

	rep := &repo.Repo{Pool: pool}
	if err := rep.EnsureSuperAdmin(ctx, env.GetString("SUPER_ADMIN_EMAIL", "vijeta.admin@ridesync.com"), env.GetString("SUPER_ADMIN_PASSWORD", "change-me")); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	rabbitURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	rmq, err := messaging.NewRabbitMQ(rabbitURI)
	if err != nil {
		log.Fatalf("rabbitmq: %v", err)
	}
	defer rmq.Close()

	if err := auditconsumer.Listen(rmq, rep); err != nil {
		log.Fatalf("audit consumer: %v", err)
	}

	grpcAddr := env.GetString("GRPC_ADDR", ":9095")
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer(tracing.WithTracingInterceptors()...)
	pb.RegisterUserAuthServiceServer(srv, &grpcsvc.Server{Repo: rep})

	go func() {
		log.Printf("user-auth-service gRPC on %s", grpcAddr)
		if err := srv.Serve(lis); err != nil {
			log.Printf("grpc: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	srv.GracefulStop()
}
