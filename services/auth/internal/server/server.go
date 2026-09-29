package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	authv1 "github.com/keix40/omnifleet/gen/go/auth/v1"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/keix40/omnifleet/pkg/auth"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	authv1.UnimplementedAuthServiceServer
	pool         *pgxpool.Pool
	tokens       *auth.TokenIssuer
	accountLimit *auth.LoginRateLimiter
}

func New(dsn string, jwtSettings auth.JWTSettings, ttl time.Duration) (*Server, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	return &Server{
		pool:         pool,
		tokens:       auth.NewTokenIssuerFromSettings(jwtSettings, ttl),
		accountLimit: auth.NewLoginRateLimiter(10, time.Minute),
	}, nil
}

func (s *Server) Close() {
	s.pool.Close()
}

func (s *Server) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	tenantSlug := strings.TrimSpace(req.TenantSlug)
	if tenantSlug == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_slug required")
	}
	accountKey := fmt.Sprintf("acct:%s:%s", tenantSlug, strings.ToLower(strings.TrimSpace(req.Email)))
	if !s.accountLimit.Allow(accountKey) {
		return nil, status.Error(codes.ResourceExhausted, "too many login attempts")
	}

	var userID, tenantID, role, hash string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, role::text, password_hash
		FROM auth_lookup_user($1, $2)
	`, req.Email, tenantSlug).Scan(&userID, &tenantID, &role, &hash)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		return nil, errors.New("invalid credentials")
	}
	parsedRole, err := auth.ParseRole(role)
	if err != nil {
		return nil, err
	}
	token, exp, err := s.tokens.Issue(userID, tenantID, req.Email, parsedRole)
	if err != nil {
		return nil, err
	}
	return &authv1.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(exp).Seconds()),
		Claims: &authv1.TokenClaims{
			UserId:   userID,
			TenantId: tenantID,
			Role:     roleToProto(parsedRole),
			Email:    req.Email,
		},
	}, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	claims, err := s.tokens.Parse(req.AccessToken)
	if err != nil {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}
	return &authv1.ValidateTokenResponse{
		Valid: true,
		Claims: &authv1.TokenClaims{
			UserId:   claims.UserID,
			TenantId: claims.TenantID,
			Role:     roleToProto(claims.Role),
			Email:    claims.Email,
		},
	}, nil
}

func (s *Server) Health(ctx context.Context, _ *authv1.HealthRequest) (*authv1.HealthResponse, error) {
	if err := s.pool.Ping(ctx); err != nil {
		return &authv1.HealthResponse{Status: "degraded"}, nil
	}
	return &authv1.HealthResponse{Status: "ok"}, nil
}

func roleToProto(r auth.Role) authv1.Role {
	switch r {
	case auth.RoleAdmin:
		return authv1.Role_ROLE_ADMIN
	case auth.RoleDispatcher:
		return authv1.Role_ROLE_DISPATCHER
	case auth.RoleDriver:
		return authv1.Role_ROLE_DRIVER
	case auth.RoleCustomer:
		return authv1.Role_ROLE_CUSTOMER
	default:
		return authv1.Role_ROLE_UNSPECIFIED
	}
}
