package server

import (
	"context"

	billingv1 "github.com/keix40/omnifleet/gen/go/billing/v1"
	"github.com/keix40/omnifleet/pkg/db"
	"github.com/keix40/omnifleet/services/billing/internal/stripe"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	billingv1.UnimplementedBillingServiceServer
	pool     *pgxpool.Pool
	provider stripe.Provider
}

func New(ctx context.Context, dsn string) (*Server, func(), error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, func() {}, err
	}
	return &Server{pool: pool, provider: stripe.NewFromEnv()}, func() { pool.Close() }, nil
}

func (s *Server) GetSubscription(ctx context.Context, req *billingv1.GetSubscriptionRequest) (*billingv1.GetSubscriptionResponse, error) {
	var planID, planName, subStatus string
	var vehicleLimit int32
	var positions int64
	err := db.WithTenant(ctx, s.pool, req.TenantId, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT bs.plan_id, bp.name, bs.status, bp.vehicle_limit, bp.positions_included_monthly
			FROM billing_subscriptions bs
			JOIN billing_plans bp ON bp.id = bs.plan_id
			WHERE bs.tenant_id = $1::uuid
		`, req.TenantId).Scan(&planID, &planName, &subStatus, &vehicleLimit, &positions)
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, "subscription not found")
	}
	return &billingv1.GetSubscriptionResponse{
		PlanId:                   planID,
		PlanName:                 planName,
		Status:                   subStatus,
		VehicleLimit:             vehicleLimit,
		PositionsIncludedMonthly: positions,
	}, nil
}

func (s *Server) GetUsage(ctx context.Context, req *billingv1.GetUsageRequest) (*billingv1.GetUsageResponse, error) {
	start, end := stripe.CurrentPeriod()
	var vehicles int32
	var positions int64
	err := db.WithTenant(ctx, s.pool, req.TenantId, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM vehicles`).Scan(&vehicles); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM gps_positions
			WHERE tenant_id = $1::uuid
			  AND recorded_at >= $2::date
			  AND recorded_at < ($3::date + interval '1 day')
		`, req.TenantId, start, end).Scan(&positions)
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "usage query failed")
	}
	return &billingv1.GetUsageResponse{
		VehicleCount:        vehicles,
		PositionCountPeriod: positions,
		PeriodStart:         start,
		PeriodEnd:           end,
	}, nil
}

func (s *Server) RecordUsageSnapshot(ctx context.Context, req *billingv1.RecordUsageSnapshotRequest) (*billingv1.RecordUsageSnapshotResponse, error) {
	u, err := s.GetUsage(ctx, &billingv1.GetUsageRequest{TenantId: req.TenantId})
	if err != nil {
		return nil, err
	}
	err = db.WithTenant(ctx, s.pool, req.TenantId, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO billing_usage_snapshots (tenant_id, period_start, period_end, vehicle_count, position_count)
			VALUES ($1::uuid, $2::date, $3::date, $4, $5)
			ON CONFLICT (tenant_id, period_start) DO UPDATE
			SET vehicle_count = EXCLUDED.vehicle_count, position_count = EXCLUDED.position_count, recorded_at = now()
		`, req.TenantId, u.PeriodStart, u.PeriodEnd, u.VehicleCount, u.PositionCountPeriod)
		return err
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "snapshot failed")
	}
	return &billingv1.RecordUsageSnapshotResponse{
		VehicleCount:  u.VehicleCount,
		PositionCount: u.PositionCountPeriod,
	}, nil
}

func (s *Server) HandleStripeWebhook(ctx context.Context, req *billingv1.HandleStripeWebhookRequest) (*billingv1.HandleStripeWebhookResponse, error) {
	eventID, err := s.provider.HandleWebhook(ctx, req.Payload, req.StripeSignature)
	if err != nil {
		return &billingv1.HandleStripeWebhookResponse{Accepted: false, Message: err.Error()}, nil
	}
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO stripe_webhook_events (idempotency_key, event_type)
		VALUES ($1, 'stripe')
		ON CONFLICT (idempotency_key) DO NOTHING
	`, eventID)
	if err != nil {
		return nil, status.Error(codes.Internal, "idempotency store failed")
	}
	if tag.RowsAffected() == 0 {
		return &billingv1.HandleStripeWebhookResponse{Accepted: true, Message: "duplicate ignored"}, nil
	}
	return &billingv1.HandleStripeWebhookResponse{Accepted: true, Message: "processed"}, nil
}

func (s *Server) Health(ctx context.Context, _ *billingv1.HealthRequest) (*billingv1.HealthResponse, error) {
	if err := s.pool.Ping(ctx); err != nil {
		return &billingv1.HealthResponse{Status: "degraded", Provider: s.provider.Name()}, nil
	}
	return &billingv1.HealthResponse{Status: "ok", Provider: s.provider.Name()}, nil
}
