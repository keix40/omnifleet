package main

import (
	"context"
	"log"
	"net"
	"os"

	trackingv1 "github.com/keix40/omnifleet/gen/go/tracking/v1"
	"github.com/keix40/omnifleet/services/tracking/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	addr := env("TRACKING_GRPC_ADDR", ":50052")
	dsn := env("DATABASE_URL", "postgres://omnifleet:omnifleet@localhost:5432/omnifleet?sslmode=disable")
	natsURL := env("NATS_URL", "nats://localhost:4222")
	geoAddr := env("GEOFENCING_GRPC_ADDR", "localhost:50053")

	svc, cleanup, err := server.New(dsn, natsURL, geoAddr)
	if err != nil {
		log.Fatalf("tracking: %v", err)
	}
	defer cleanup()

	grpcServer := grpc.NewServer()
	trackingv1.RegisterTrackingServiceServer(grpcServer, svc)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("tracking gRPC listening on %s", addr)
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
