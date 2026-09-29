package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	trackingv1 "github.com/keix40/omnifleet/gen/go/tracking/v1"
	"github.com/keix40/omnifleet/pkg/db"
	"github.com/keix40/omnifleet/pkg/events"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	trackingv1.UnimplementedTrackingServiceServer
	pool      *pgxpool.Pool
	publisher *events.Publisher
}

func New(dsn, natsURL string) (*Server, func(), error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, func() {}, err
	}
	pub, nc, err := events.ConnectPublisher(natsURL)
	if err != nil {
		pool.Close()
		return nil, func() {}, err
	}
	if err := pub.EnsureStream(context.Background()); err != nil {
		nc.Close()
		pool.Close()
		return nil, func() {}, err
	}
	cleanup := func() {
		nc.Close()
		pool.Close()
	}
	return &Server{pool: pool, publisher: pub}, cleanup, nil
}

func (s *Server) IngestPosition(ctx context.Context, req *trackingv1.IngestPositionRequest) (*trackingv1.IngestPositionResponse, error) {
	positionID := uuid.NewString()
	recordedAt := time.Unix(req.RecordedAtUnix, 0)
	if req.RecordedAtUnix == 0 {
		recordedAt = time.Now().UTC()
	}

	err := db.WithTenant(ctx, s.pool, req.TenantId, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO gps_positions (id, tenant_id, vehicle_id, driver_id, location, speed_mps, heading_deg, recorded_at)
			VALUES ($1, $2, $3, NULLIF($4, '')::uuid,
			        ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography,
			        $7, $8, $9)
		`, positionID, req.TenantId, req.VehicleId, req.DriverId,
			req.Point.Longitude, req.Point.Latitude,
			req.Point.SpeedMps, req.Point.HeadingDeg, recordedAt)
		return err
	})
	if err != nil {
		return nil, err
	}

	ev := events.PositionEvent{
		TenantID:   req.TenantId,
		VehicleID:  req.VehicleId,
		DriverID:   req.DriverId,
		Latitude:   req.Point.Latitude,
		Longitude:  req.Point.Longitude,
		SpeedMPS:   req.Point.SpeedMps,
		HeadingDeg: req.Point.HeadingDeg,
		RecordedAt: recordedAt.Unix(),
	}
	_ = s.publisher.PublishPosition(ctx, ev)

	return &trackingv1.IngestPositionResponse{PositionId: positionID, Accepted: true}, nil
}

func (s *Server) Health(ctx context.Context, _ *trackingv1.HealthRequest) (*trackingv1.HealthResponse, error) {
	if err := s.pool.Ping(ctx); err != nil {
		return &trackingv1.HealthResponse{Status: "degraded"}, nil
	}
	return &trackingv1.HealthResponse{Status: "ok"}, nil
}
