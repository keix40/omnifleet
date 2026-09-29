package main

import (
	"context"
	"log"
	"net"
	"os"

	etav1 "github.com/keix40/omnifleet/gen/go/eta/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type etaServer struct {
	etav1.UnimplementedETAServiceServer
}

func (etaServer) Health(_ context.Context, _ *etav1.HealthRequest) (*etav1.HealthResponse, error) {
	// TODO: integrate OSRM/GraphHopper and historical Timescale aggregates for ETA confidence bands.
	return &etav1.HealthResponse{Status: "stub"}, nil
}

func main() {
	addr := env("ETA_GRPC_ADDR", ":50054")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	etav1.RegisterETAServiceServer(s, etaServer{})
	hs := health.NewServer()
	healthpb.RegisterHealthServer(s, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	log.Printf("eta service (stub) on %s", addr)
	log.Fatal(s.Serve(lis))
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
