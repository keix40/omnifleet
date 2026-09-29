package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	geofencingv1 "github.com/keix40/omnifleet/gen/go/geofencing/v1"
	commonv1 "github.com/keix40/omnifleet/gen/go/common/v1"
	trackingv1 "github.com/keix40/omnifleet/gen/go/tracking/v1"
	"github.com/keix40/omnifleet/pkg/db"
	"github.com/keix40/omnifleet/pkg/events"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	trackingv1.UnimplementedTrackingServiceServer
	pool       *pgxpool.Pool
	publisher  *events.Publisher
	geoClient  geofencingv1.GeofencingServiceClient
	geoConn    *grpc.ClientConn
}

func New(dsn, natsURL, geofencingAddr string) (*Server, func(), error) {
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
	var geoClient geofencingv1.GeofencingServiceClient
	var geoConn *grpc.ClientConn
	if geofencingAddr != "" {
		conn, err := grpc.NewClient(geofencingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			nc.Close()
			pool.Close()
			return nil, func() {}, err
		}
		geoConn = conn
		geoClient = geofencingv1.NewGeofencingServiceClient(conn)
	}
	cleanup := func() {
		if geoConn != nil {
			geoConn.Close()
		}
		nc.Close()
		pool.Close()
	}
	return &Server{pool: pool, publisher: pub, geoClient: geoClient, geoConn: geoConn}, cleanup, nil
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

	if s.geoClient != nil {
		_, _ = s.geoClient.EvaluatePosition(ctx, &geofencingv1.EvaluatePositionRequest{
			TenantId:  req.TenantId,
			VehicleId: req.VehicleId,
			Point: &commonv1.GeoPoint{
				Latitude:  req.Point.Latitude,
				Longitude: req.Point.Longitude,
			},
		})
	}

	return &trackingv1.IngestPositionResponse{PositionId: positionID, Accepted: true}, nil
}

func (s *Server) Health(ctx context.Context, _ *trackingv1.HealthRequest) (*trackingv1.HealthResponse, error) {
	if err := s.pool.Ping(ctx); err != nil {
		return &trackingv1.HealthResponse{Status: "degraded"}, nil
	}
	return &trackingv1.HealthResponse{Status: "ok"}, nil
}
