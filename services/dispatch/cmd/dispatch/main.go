package main

import (
	"context"
	"log"
	"net"
	"os"

	dispatchv1 "github.com/keix40/omnifleet/gen/go/dispatch/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type dispatchServer struct {
	dispatchv1.UnimplementedDispatchServiceServer
}

func (dispatchServer) Health(_ context.Context, _ *dispatchv1.HealthRequest) (*dispatchv1.HealthResponse, error) {
	// TODO: job assignment, capacity constraints, dispatcher approval flows.
	return &dispatchv1.HealthResponse{Status: "stub"}, nil
}

func main() {
	addr := env("DISPATCH_GRPC_ADDR", ":50055")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	dispatchv1.RegisterDispatchServiceServer(s, dispatchServer{})
	hs := health.NewServer()
	healthpb.RegisterHealthServer(s, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	log.Printf("dispatch service (stub) on %s", addr)
	log.Fatal(s.Serve(lis))
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
