package public

import (
	"context"

	notificationsv1 "github.com/keix40/omnifleet/gen/go/notifications/v1"
	"github.com/keix40/omnifleet/services/notifications/internal/server"
	"google.golang.org/grpc"
)

type Server = server.Server

func New(ctx context.Context, dsn, natsURL string) (*Server, func(), error) {
	return server.New(ctx, dsn, natsURL)
}

func Register(gs *grpc.Server, s *Server) {
	notificationsv1.RegisterNotificationsServiceServer(gs, s)
}
