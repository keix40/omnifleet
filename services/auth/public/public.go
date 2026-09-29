package public

import (
	"time"

	authv1 "github.com/keix40/omnifleet/gen/go/auth/v1"
	"github.com/keix40/omnifleet/pkg/auth"
	"github.com/keix40/omnifleet/services/auth/internal/server"
	"google.golang.org/grpc"
)

type Server = server.Server

func New(dsn string, jwt auth.JWTSettings, ttl time.Duration) (*Server, error) {
	return server.New(dsn, jwt, ttl)
}

func Register(gs *grpc.Server, s *Server) {
	authv1.RegisterAuthServiceServer(gs, s)
}

func Close(s *Server) {
	s.Close()
}
