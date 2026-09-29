package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/keix40/omnifleet/pkg/auth"
)

type ctxKey int

const claimsKey ctxKey = 1

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := s.tokens.Parse(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func claimsFromContext(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(claimsKey).(*auth.Claims)
	return c
}
