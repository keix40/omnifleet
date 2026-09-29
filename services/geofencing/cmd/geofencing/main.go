package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	geofencingv1 "github.com/keix40/omnifleet/gen/go/geofencing/v1"
	"github.com/keix40/omnifleet/services/geofencing/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	addr := env("GEOFENCING_GRPC_ADDR", ":50053")
	dsn := env("DATABASE_URL", "postgres://omnifleet:omnifleet@localhost:5432/omnifleet?sslmode=disable")
	natsURL := env("NATS_URL", "nats://localhost:4222")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	svc, cleanup, err := server.New(ctx, dsn, natsURL)
	if err != nil {
		log.Fatalf("geofencing: %v", err)
	}
	defer cleanup()

	grpcServer := grpc.NewServer()
	geofencingv1.RegisterGeofencingServiceServer(grpcServer, svc)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	go func() {
		if err := svc.RunConsumer(ctx); err != nil && ctx.Err() == nil {
			log.Printf("consumer stopped: %v", err)
		}
	}()

	log.Printf("geofencing gRPC listening on %s", addr)
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
