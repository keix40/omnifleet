package public

import (
	trackingv1 "github.com/keix40/omnifleet/gen/go/tracking/v1"
	"github.com/keix40/omnifleet/services/tracking/internal/server"
	"google.golang.org/grpc"
)

type Server = server.Server

func New(dsn, natsURL string) (*Server, func(), error) {
	return server.New(dsn, natsURL)
}

func Register(gs *grpc.Server, s *Server) {
	trackingv1.RegisterTrackingServiceServer(gs, s)
}
