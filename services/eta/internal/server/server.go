package server

import (
	"context"
	"encoding/json"
	"time"

	commonv1 "github.com/keix40/omnifleet/gen/go/common/v1"
	etav1 "github.com/keix40/omnifleet/gen/go/eta/v1"
	"github.com/keix40/omnifleet/pkg/db"
	"github.com/keix40/omnifleet/pkg/events"
	"github.com/keix40/omnifleet/services/eta/internal/routing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	etav1.UnimplementedETAServiceServer
	pool      *pgxpool.Pool
	router    routing.Provider
	publisher *events.Publisher
	osrmURL   string
}

func New(ctx context.Context, dsn, natsURL, osrmURL string) (*Server, func(), error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, func() {}, err
	}
	pub, nc, err := events.ConnectPublisher(natsURL)
	if err != nil {
		pool.Close()
		return nil, func() {}, err
	}
	if err := pub.EnsureStream(ctx); err != nil {
		nc.Close()
		pool.Close()
		return nil, func() {}, err
	}
	cleanup := func() {
		nc.Close()
		pool.Close()
	}
	return &Server{
		pool:      pool,
		router:    routing.NewFromEnv(routing.ParseOSRMURL(osrmURL)),
		publisher: pub,
		osrmURL:   osrmURL,
	}, cleanup, nil
}

func (s *Server) ComputeETA(ctx context.Context, req *etav1.ComputeETARequest) (*etav1.ComputeETAResponse, error) {
	if req.Destination == nil {
		return nil, status.Error(codes.InvalidArgument, "destination required")
	}
	var lat, lon, speed float64
	err := db.WithTenant(ctx, s.pool, req.TenantId, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT ST_Y(location::geometry), ST_X(location::geometry), COALESCE(speed_mps, 0)
			FROM gps_positions
			WHERE tenant_id = $1::uuid AND vehicle_id = $2::uuid
			ORDER BY recorded_at DESC
			LIMIT 1
		`, req.TenantId, req.VehicleId).Scan(&lat, &lon, &speed)
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, "no position for vehicle")
	}
	if speed <= 0 {
		speed = s.avgSpeed(ctx, req.TenantId, req.VehicleId)
	}
	if speed <= 0 {
		speed = 8.0
	}
	from := routing.Point{Lat: lat, Lon: lon}
	to := routing.Point{Lat: req.Destination.Latitude, Lon: req.Destination.Longitude}
	res, err := s.route(ctx, from, to, speed)
	if err != nil {
		return nil, status.Error(codes.Internal, "routing failed")
	}
	etaUnix := time.Now().UTC().Add(time.Duration(res.DurationSeconds * float64(time.Second))).Unix()
	if req.PublishUpdate {
		ev := events.ETAEvent{
			TenantID:        req.TenantId,
			VehicleID:       req.VehicleId,
			DistanceMeters:  res.DistanceMeters,
			DurationSeconds: res.DurationSeconds,
			ETAUnix:         etaUnix,
			RoutingProvider: res.Provider,
			DestLatitude:    to.Lat,
			DestLongitude:   to.Lon,
		}
		data, _ := json.Marshal(ev)
		_ = s.publisher.PublishRaw(ctx, events.ETASubject(req.TenantId), data)
	}
	return &etav1.ComputeETAResponse{
		VehicleId:       req.VehicleId,
		DistanceMeters:  res.DistanceMeters,
		DurationSeconds: res.DurationSeconds,
		EtaUnix:         float64(etaUnix),
		RoutingProvider: res.Provider,
		Origin:          &commonv1.GeoPoint{Latitude: lat, Longitude: lon, SpeedMps: speed},
	}, nil
}

func (s *Server) route(ctx context.Context, from, to routing.Point, speed float64) (routing.Result, error) {
	if s.osrmURL == "" {
		return routing.HaversineProvider{RoadFactor: 1.35, SpeedMetersS: speed}.Route(ctx, from, to)
	}
	res, err := s.router.Route(ctx, from, to)
	if err != nil {
		return routing.HaversineProvider{RoadFactor: 1.35, SpeedMetersS: speed}.Route(ctx, from, to)
	}
	return res, nil
}

func (s *Server) avgSpeed(ctx context.Context, tenantID, vehicleID string) float64 {
	var avg *float64
	err := db.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT AVG(speed_mps) FROM (
				SELECT speed_mps FROM gps_positions
				WHERE tenant_id = $1::uuid AND vehicle_id = $2::uuid AND speed_mps > 0
				ORDER BY recorded_at DESC LIMIT 20
			) s
		`, tenantID, vehicleID).Scan(&avg)
	})
	if err != nil || avg == nil {
		return 0
	}
	return *avg
}

func (s *Server) Health(ctx context.Context, _ *etav1.HealthRequest) (*etav1.HealthResponse, error) {
	if err := s.pool.Ping(ctx); err != nil {
		return &etav1.HealthResponse{Status: "degraded"}, nil
	}
	return &etav1.HealthResponse{Status: "ok"}, nil
}
