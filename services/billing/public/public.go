package public

import (
	"context"

	billingv1 "github.com/keix40/omnifleet/gen/go/billing/v1"
	"github.com/keix40/omnifleet/services/billing/internal/server"
	"google.golang.org/grpc"
)

type Server = server.Server

func New(ctx context.Context, dsn string) (*Server, func(), error) {
	return server.New(ctx, dsn)
}

func Register(gs *grpc.Server, s *Server) {
	billingv1.RegisterBillingServiceServer(gs, s)
}
