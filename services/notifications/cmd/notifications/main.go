package main

import (
	"context"
	"log"
	"net"
	"os"

	notificationsv1 "github.com/keix40/omnifleet/gen/go/notifications/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type notificationsServer struct {
	notificationsv1.UnimplementedNotificationsServiceServer
}

func (notificationsServer) Health(_ context.Context, _ *notificationsv1.HealthRequest) (*notificationsv1.HealthResponse, error) {
	// TODO: fan-out geofence alerts to email/SMS/push providers with tenant-branded templates.
	return &notificationsv1.HealthResponse{Status: "stub"}, nil
}

func main() {
	addr := env("NOTIFICATIONS_GRPC_ADDR", ":50057")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	notificationsv1.RegisterNotificationsServiceServer(s, notificationsServer{})
	hs := health.NewServer()
	healthpb.RegisterHealthServer(s, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	log.Printf("notifications service (stub) on %s", addr)
	log.Fatal(s.Serve(lis))
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
