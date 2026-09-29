package main

import (
	"context"
	"log"
	"net"
	"os"

	billingv1 "github.com/keix40/omnifleet/gen/go/billing/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type billingServer struct {
	billingv1.UnimplementedBillingServiceServer
}

func (billingServer) Health(_ context.Context, _ *billingv1.HealthRequest) (*billingv1.HealthResponse, error) {
	// TODO: Stripe Checkout, Customer Portal, webhook signature verification (test mode keys via env).
	return &billingv1.HealthResponse{Status: "stub"}, nil
}

func main() {
	addr := env("BILLING_GRPC_ADDR", ":50056")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	billingv1.RegisterBillingServiceServer(s, billingServer{})
	hs := health.NewServer()
	healthpb.RegisterHealthServer(s, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	log.Printf("billing service (stub) on %s", addr)
	log.Fatal(s.Serve(lis))
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
