package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	authv1 "github.com/keix40/omnifleet/gen/go/auth/v1"
	"github.com/keix40/omnifleet/services/auth/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	addr := env("AUTH_GRPC_ADDR", ":50051")
	dsn := env("DATABASE_URL", "postgres://omnifleet:omnifleet@localhost:5432/omnifleet?sslmode=disable")
	jwtSecret := env("JWT_SECRET", "dev-only-change-me")
	jwtIssuer := env("JWT_ISSUER", "omnifleet-auth")

	svc, err := server.New(dsn, jwtSecret, jwtIssuer, 24*time.Hour)
	if err != nil {
		log.Fatalf("auth server: %v", err)
	}
	defer svc.Close()

	grpcServer := grpc.NewServer()
	authv1.RegisterAuthServiceServer(grpcServer, svc)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("auth gRPC listening on %s", addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

var _ = context.Background
