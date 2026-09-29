package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
	authv1 "github.com/keix40/omnifleet/gen/go/auth/v1"
	commonv1 "github.com/keix40/omnifleet/gen/go/common/v1"
	trackingv1 "github.com/keix40/omnifleet/gen/go/tracking/v1"
	"github.com/keix40/omnifleet/pkg/auth"
	"github.com/keix40/omnifleet/pkg/events"
	"github.com/keix40/omnifleet/pkg/validate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	AuthAddr     string
	TrackingAddr string
	NatsURL      string
	DatabaseURL  string
	JWT          auth.JWTSettings
	ReplicaID    string
}

const wsTicketSubprotocolPrefix = "omnifleet.v1."

type Server struct {
	authClient     authv1.AuthServiceClient
	trackingClient trackingv1.TrackingServiceClient
	tokens         *auth.TokenIssuer
	wsTickets      *auth.WSTicketStore
	hub            *wsHub
	nc             *nats.Conn
	loginLimiter   *auth.LoginRateLimiter
	dbPool         *pgxpool.Pool
	replicaID      string
}

func NewServer(ctx context.Context, cfg Config) (*Server, func(), error) {
	authConn, err := grpc.NewClient(cfg.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, func() {}, err
	}
	trackConn, err := grpc.NewClient(cfg.TrackingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		authConn.Close()
		return nil, func() {}, err
	}
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		trackConn.Close()
		authConn.Close()
		return nil, func() {}, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		trackConn.Close()
		authConn.Close()
		return nil, func() {}, err
	}

	var dbPool *pgxpool.Pool
	var ticketStore auth.SingleUseStore
	if cfg.DatabaseURL != "" {
		dbPool, err = pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			nc.Close()
			trackConn.Close()
			authConn.Close()
			return nil, func() {}, err
		}
		ticketStore = auth.NewPostgresSingleUseStore(dbPool)
	}

	replicaID := cfg.ReplicaID
	if replicaID == "" {
		replicaID = os.Getenv("HOSTNAME")
	}
	if replicaID == "" {
		replicaID = fmt.Sprintf("gateway-%d", time.Now().UnixNano())
	}

	hub := newHub()
	s := &Server{
		authClient:     authv1.NewAuthServiceClient(authConn),
		trackingClient: trackingv1.NewTrackingServiceClient(trackConn),
		tokens:         auth.NewTokenIssuerFromSettings(cfg.JWT, 24*time.Hour),
		wsTickets:      auth.NewWSTicketStore(cfg.JWT.Secret, 30*time.Second, ticketStore),
		hub:            hub,
		nc:             nc,
		loginLimiter:   auth.NewLoginRateLimiter(30, time.Minute),
		dbPool:         dbPool,
		replicaID:      replicaID,
	}
	go s.bridgeNATS(ctx, js)
	cleanup := func() {
		if dbPool != nil {
			dbPool.Close()
		}
		nc.Close()
		trackConn.Close()
		authConn.Close()
	}
	return s, cleanup, nil
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", s.handleReadyz)

	r.Post("/api/v1/auth/login", s.handleLogin)
	r.Group(func(protected chi.Router) {
		protected.Use(s.authMiddleware)
		protected.Post("/api/v1/tracking/positions", s.handleIngestPosition)
		protected.Post("/api/v1/ws/fleet/ticket", s.handleWSTicket)
		protected.Get("/api/v1/ws/fleet", s.handleWebSocket)
	})
	// Browser WebSocket: short-lived ticket via Sec-WebSocket-Protocol (see README security model).
	r.Get("/api/v1/ws/fleet/live", s.handleWebSocketLive)
	return r
}

type loginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TenantSlug string `json:"tenant_slug"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = strings.Split(r.RemoteAddr, ":")[0]
	}
	if !s.loginLimiter.Allow("ip:" + ip) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	var body loginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.TenantSlug) == "" {
		http.Error(w, "tenant_slug required", http.StatusBadRequest)
		return
	}
	resp, err := s.authClient.Login(r.Context(), &authv1.LoginRequest{
		Email:      body.Email,
		Password:   body.Password,
		TenantSlug: body.TenantSlug,
	})
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, resp)
}

type ingestRequest struct {
	VehicleID string  `json:"vehicle_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	SpeedMPS  float64 `json:"speed_mps"`
	Heading   float64 `json:"heading_deg"`
}

func (s *Server) handleIngestPosition(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermIngestPosition) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := validate.IngestPosition(body.VehicleID, body.Latitude, body.Longitude); err != nil {
		http.Error(w, validate.PublicMessage(err), http.StatusBadRequest)
		return
	}
	resp, err := s.trackingClient.IngestPosition(r.Context(), &trackingv1.IngestPositionRequest{
		TenantId:  claims.TenantID,
		VehicleId: body.VehicleID,
		DriverId:  claims.UserID,
		Point: &commonv1.GeoPoint{
			Latitude:   body.Latitude,
			Longitude:  body.Longitude,
			SpeedMps:   body.SpeedMPS,
			HeadingDeg: body.Heading,
		},
		RecordedAtUnix: time.Now().Unix(),
	})
	if err != nil {
		http.Error(w, validate.PublicMessage(validate.WrapInternal(err)), http.StatusBadGateway)
		return
	}
	writeJSON(w, resp)
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if s.nc == nil || !s.nc.IsConnected() {
		http.Error(w, "nats unavailable", http.StatusServiceUnavailable)
		return
	}
	if _, err := s.authClient.Health(ctx, &authv1.HealthRequest{}); err != nil {
		http.Error(w, "auth unavailable", http.StatusServiceUnavailable)
		return
	}
	if _, err := s.trackingClient.Health(ctx, &trackingv1.HealthRequest{}); err != nil {
		http.Error(w, "tracking unavailable", http.StatusServiceUnavailable)
		return
	}
	if s.dbPool != nil {
		if err := s.dbPool.Ping(ctx); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleWSTicket(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	if !auth.HasPermission(claims.Role, auth.PermViewFleet) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	ticket, exp, err := s.wsTickets.Issue(claims)
	if err != nil {
		http.Error(w, "ticket issue failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"ticket":     ticket,
		"expiresIn":  int(time.Until(exp).Seconds()),
		"protocol":   "omnifleet.v1",
		"wsPath":     "/api/v1/ws/fleet/live",
	})
}

func (s *Server) handleWebSocketLive(w http.ResponseWriter, r *http.Request) {
	var ticketID, selectedProtocol string
	for _, p := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, wsTicketSubprotocolPrefix) {
			ticketID = strings.TrimPrefix(p, wsTicketSubprotocolPrefix)
			selectedProtocol = p
			break
		}
	}
	if ticketID == "" {
		http.Error(w, "missing omnifleet.v1.<ticket> subprotocol", http.StatusUnauthorized)
		return
	}
	claims, ok := s.wsTickets.Redeem(ticketID)
	if !ok {
		http.Error(w, "invalid or expired ticket", http.StatusUnauthorized)
		return
	}
	header := http.Header{}
	header.Set("Sec-WebSocket-Protocol", selectedProtocol)
	s.serveWebSocket(w, r, claims, header)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromContext(r.Context())
	s.serveWebSocket(w, r, claims, nil)
}

func (s *Server) serveWebSocket(w http.ResponseWriter, r *http.Request, claims *auth.Claims, responseHeader http.Header) {
	if claims == nil || !auth.HasPermission(claims.Role, auth.PermViewFleet) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	conn, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return
	}
	client := s.hub.register(claims.TenantID, conn)
	defer func() {
		s.hub.unregister(claims.TenantID, client)
		_ = conn.Close()
	}()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (s *Server) bridgeNATS(ctx context.Context, js jetstream.JetStream) {
	_, _ = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     events.StreamFleet,
		Subjects: []string{"fleet.>"},
	})
	cons, err := js.CreateOrUpdateConsumer(ctx, events.StreamFleet, jetstream.ConsumerConfig{
		Name:          fmt.Sprintf("gateway-live-%s", s.replicaID),
		FilterSubject: "fleet.>",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return
	}
	_, _ = cons.Consume(func(msg jetstream.Msg) {
		subj := msg.Subject()
		parts := strings.Split(subj, ".")
		if len(parts) < 2 {
			_ = msg.Ack()
			return
		}
		tenantID := parts[1]
		s.hub.broadcast(tenantID, msg.Data())
		_ = msg.Ack()
	})
	<-ctx.Done()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
