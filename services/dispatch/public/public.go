package public

import (
	"context"

	dispatchv1 "github.com/keix40/omnifleet/gen/go/dispatch/v1"
	"github.com/keix40/omnifleet/services/dispatch/internal/server"
	"google.golang.org/grpc"
)

type Server = server.Server

func New(ctx context.Context, dsn, natsURL string) (*Server, func(), error) {
	return server.New(ctx, dsn, natsURL)
}

func Register(gs *grpc.Server, s *Server) {
	dispatchv1.RegisterDispatchServiceServer(gs, s)
}
