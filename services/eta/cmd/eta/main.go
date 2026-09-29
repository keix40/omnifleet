package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	etav1 "github.com/keix40/omnifleet/gen/go/eta/v1"
	"github.com/keix40/omnifleet/services/eta/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	addr := env("ETA_GRPC_ADDR", ":50054")
	dsn := env("DATABASE_URL", "")
	natsURL := env("NATS_URL", "nats://localhost:4222")
	osrmURL := env("OSRM_BASE_URL", "")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svc, cleanup, err := server.New(ctx, dsn, natsURL, osrmURL)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	gs := grpc.NewServer()
	etav1.RegisterETAServiceServer(gs, svc)
	hs := health.NewServer()
	healthpb.RegisterHealthServer(gs, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	log.Printf("eta service on %s (osrm=%v)", addr, osrmURL != "")
	log.Fatal(gs.Serve(lis))
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
