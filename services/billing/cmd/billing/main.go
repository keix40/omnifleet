package main

import (
	"context"
	"log"
	"net"
	"os"

	billingv1 "github.com/keix40/omnifleet/gen/go/billing/v1"
	"github.com/keix40/omnifleet/services/billing/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	addr := env("BILLING_GRPC_ADDR", ":50056")
	dsn := env("DATABASE_URL", "")

	svc, cleanup, err := server.New(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	gs := grpc.NewServer()
	billingv1.RegisterBillingServiceServer(gs, svc)
	hs := health.NewServer()
	healthpb.RegisterHealthServer(gs, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	log.Printf("billing service on %s provider=%s", addr, env("STRIPE_PROVIDER", "fake"))
	log.Fatal(gs.Serve(lis))
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
