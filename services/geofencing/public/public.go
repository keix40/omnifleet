package public

import (
	"context"

	geofencingv1 "github.com/keix40/omnifleet/gen/go/geofencing/v1"
	"github.com/keix40/omnifleet/services/geofencing/internal/server"
	"google.golang.org/grpc"
)

type Server = server.Server

func New(ctx context.Context, dsn, natsURL string) (*Server, func(), error) {
	return server.New(ctx, dsn, natsURL)
}

func Register(gs *grpc.Server, s *Server) {
	geofencingv1.RegisterGeofencingServiceServer(gs, s)
}

func RunConsumer(ctx context.Context, s *Server) error {
	return s.RunConsumer(ctx)
}

func RunOutboxRelay(ctx context.Context, s *Server) {
	s.RunOutboxRelay(ctx)
}
