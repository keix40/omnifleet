package public

import (
	"context"
	"net/http"

	"github.com/keix40/omnifleet/pkg/auth"
	"github.com/keix40/omnifleet/services/gateway/internal/httpapi"
)

type Config = httpapi.Config

type Server struct {
	inner *httpapi.Server
}

func NewServer(ctx context.Context, cfg Config) (*Server, func(), error) {
	s, cleanup, err := httpapi.NewServer(ctx, cfg)
	if err != nil {
		return nil, cleanup, err
	}
	return &Server{inner: s}, cleanup, nil
}

func (s *Server) Router() http.Handler {
	return s.inner.Router()
}

func JWTFromEnv() (auth.JWTSettings, error) {
	return auth.LoadJWTSettingsFromEnv()
}
