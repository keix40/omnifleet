package public

import (
	"context"

	etav1 "github.com/keix40/omnifleet/gen/go/eta/v1"
	"github.com/keix40/omnifleet/services/eta/internal/server"
	"google.golang.org/grpc"
)

type Server = server.Server

func New(ctx context.Context, dsn, natsURL, osrmURL string) (*Server, func(), error) {
	return server.New(ctx, dsn, natsURL, osrmURL)
}

func Register(gs *grpc.Server, s *Server) {
	etav1.RegisterETAServiceServer(gs, s)
}
