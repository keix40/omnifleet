package server

import (
	"context"
	"fmt"
	"log"
	"time"

	commonv1 "github.com/keix40/omnifleet/gen/go/common/v1"
	dispatchv1 "github.com/keix40/omnifleet/gen/go/dispatch/v1"
	"github.com/google/uuid"
	"github.com/keix40/omnifleet/pkg/db"
	"github.com/keix40/omnifleet/pkg/events"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	dispatchv1.UnimplementedDispatchServiceServer
	pool      *pgxpool.Pool
	publisher *events.Publisher
}

func New(ctx context.Context, dsn, natsURL string) (*Server, func(), error) {
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
	s := &Server{pool: pool, publisher: pub}
	go s.RunOutboxRelay(ctx)
	cleanup := func() {
		events.ReleaseNATS(nc)
		pool.Close()
	}
	return s, cleanup, nil
}

func (s *Server) RunOutboxRelay(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := events.RelayPending(ctx, s.pool, s.publisher, 32)
			if err != nil {
				log.Printf("dispatch outbox relay: %v", err)
			}
		}
	}
}

func (s *Server) CreateJob(ctx context.Context, req *dispatchv1.CreateJobRequest) (*dispatchv1.CreateJobResponse, error) {
	if req.Pickup == nil || req.Dropoff == nil {
		return nil, status.Error(codes.InvalidArgument, "pickup and dropoff required")
	}
	jobID := uuid.NewString()
	var job *dispatchv1.Job
	err := db.WithTenant(ctx, s.pool, req.TenantId, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO dispatch_jobs (id, tenant_id, status, pickup, dropoff, pickup_label, dropoff_label)
			VALUES ($1::uuid, $2::uuid, 'created',
			        ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography,
			        ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography,
			        $7, $8)
		`, jobID, req.TenantId,
			req.Pickup.Longitude, req.Pickup.Latitude,
			req.Dropoff.Longitude, req.Dropoff.Latitude,
			req.PickupLabel, req.DropoffLabel)
		if err != nil {
			return err
		}
		if err := events.EnqueueDispatch(ctx, tx, events.DispatchEvent{
			TenantID: req.TenantId,
			JobID:    jobID,
			Status:   "created",
			Message:  "Job created",
		}); err != nil {
			return err
		}
		job, err = scanJob(ctx, tx, req.TenantId, jobID)
		return err
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "create job failed")
	}
	resp := &dispatchv1.CreateJobResponse{Job: job}
	if req.AutoAssign {
		ar, err := s.autoAssignInternal(ctx, req.TenantId, jobID)
		if err == nil {
			resp.Job = ar
		}
	}
	return resp, nil
}

func (s *Server) GetJob(ctx context.Context, req *dispatchv1.GetJobRequest) (*dispatchv1.GetJobResponse, error) {
	var job *dispatchv1.Job
	err := db.WithTenant(ctx, s.pool, req.TenantId, func(tx pgx.Tx) error {
		var err error
		job, err = scanJob(ctx, tx, req.TenantId, req.JobId)
		return err
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, "job not found")
	}
	return &dispatchv1.GetJobResponse{Job: job}, nil
}

func (s *Server) ListJobs(ctx context.Context, req *dispatchv1.ListJobsRequest) (*dispatchv1.ListJobsResponse, error) {
	limit := int32(50)
	if req.Page != nil && req.Page.PageSize > 0 {
		limit = req.Page.PageSize
	}
	var jobs []*dispatchv1.Job
	err := db.WithTenant(ctx, s.pool, req.TenantId, func(tx pgx.Tx) error {
		query := `
			SELECT id::text, tenant_id::text, status::text,
			       ST_Y(pickup::geometry), ST_X(pickup::geometry),
			       ST_Y(dropoff::geometry), ST_X(dropoff::geometry),
			       COALESCE(pickup_label,''), COALESCE(dropoff_label,''),
			       COALESCE(vehicle_id::text,''), COALESCE(driver_id::text,''),
			       EXTRACT(EPOCH FROM created_at)::bigint, EXTRACT(EPOCH FROM updated_at)::bigint
			FROM dispatch_jobs
			WHERE tenant_id = $1::uuid
		`
		args := []any{req.TenantId}
		if req.DriverId != "" {
			query += ` AND driver_id = $2::uuid`
			args = append(args, req.DriverId)
		}
		if req.StatusFilter != dispatchv1.JobStatus_JOB_STATUS_UNSPECIFIED {
			query += fmt.Sprintf(` AND status = $%d::job_status`, len(args)+1)
			args = append(args, statusToDB(req.StatusFilter))
		}
		query += fmt.Sprintf(` ORDER BY updated_at DESC LIMIT $%d`, len(args)+1)
		args = append(args, limit)
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			j, err := scanJobRow(rows)
			if err != nil {
				return err
			}
			jobs = append(jobs, j)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "list jobs failed")
	}
	return &dispatchv1.ListJobsResponse{Jobs: jobs}, nil
}

func (s *Server) AssignJob(ctx context.Context, req *dispatchv1.AssignJobRequest) (*dispatchv1.AssignJobResponse, error) {
	var job *dispatchv1.Job
	err := db.WithTenant(ctx, s.pool, req.TenantId, func(tx pgx.Tx) error {
		var curStatus string
		if err := tx.QueryRow(ctx, `SELECT status::text FROM dispatch_jobs WHERE id = $1::uuid`, req.JobId).Scan(&curStatus); err != nil {
			return err
		}
		if curStatus != "created" && curStatus != "assigned" {
			return fmt.Errorf("cannot assign job in status %s", curStatus)
		}
		driverID := req.DriverId
		if driverID == "" {
			err := tx.QueryRow(ctx, `
				SELECT id::text FROM users WHERE tenant_id = $1::uuid AND role = 'driver' LIMIT 1
			`, req.TenantId).Scan(&driverID)
			if err != nil {
				return fmt.Errorf("driver required")
			}
		}
		_, err := tx.Exec(ctx, `
			UPDATE dispatch_jobs
			SET vehicle_id = $3::uuid, driver_id = $4::uuid, status = 'assigned', updated_at = now()
			WHERE id = $1::uuid AND tenant_id = $2::uuid
		`, req.JobId, req.TenantId, req.VehicleId, driverID)
		if err != nil {
			return err
		}
		if err := events.EnqueueDispatch(ctx, tx, events.DispatchEvent{
			TenantID:  req.TenantId,
			JobID:     req.JobId,
			Status:    "assigned",
			VehicleID: req.VehicleId,
			DriverID:  driverID,
			Message:   "Job assigned",
		}); err != nil {
			return err
		}
		job, err = scanJob(ctx, tx, req.TenantId, req.JobId)
		return err
	})
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &dispatchv1.AssignJobResponse{Job: job}, nil
}

func (s *Server) AutoAssignNearest(ctx context.Context, req *dispatchv1.AutoAssignNearestRequest) (*dispatchv1.AutoAssignNearestResponse, error) {
	job, err := s.autoAssignInternal(ctx, req.TenantId, req.JobId)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &dispatchv1.AutoAssignNearestResponse{Job: job, VehicleId: job.VehicleId}, nil
}

func (s *Server) autoAssignInternal(ctx context.Context, tenantID, jobID string) (*dispatchv1.Job, error) {
	var job *dispatchv1.Job
	err := db.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var vehicleID, driverID string
		err := tx.QueryRow(ctx, `
			WITH job AS (
				SELECT pickup FROM dispatch_jobs WHERE id = $1::uuid AND tenant_id = $2::uuid
			),
			latest AS (
				SELECT DISTINCT ON (p.vehicle_id)
					p.vehicle_id, p.location
				FROM gps_positions p
				WHERE p.tenant_id = $2::uuid
				ORDER BY p.vehicle_id, p.recorded_at DESC
			)
			SELECT v.id::text, u.id::text
			FROM latest l
			JOIN vehicles v ON v.id = l.vehicle_id AND v.tenant_id = $2::uuid
			LEFT JOIN users u ON u.tenant_id = $2::uuid AND u.role = 'driver'
			CROSS JOIN job j
			ORDER BY ST_Distance(l.location, j.pickup)
			LIMIT 1
		`, jobID, tenantID).Scan(&vehicleID, &driverID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE dispatch_jobs
			SET vehicle_id = $3::uuid, driver_id = NULLIF($4,'')::uuid, status = 'assigned', updated_at = now()
			WHERE id = $1::uuid AND tenant_id = $2::uuid AND status = 'created'
		`, jobID, tenantID, vehicleID, driverID)
		if err != nil {
			return err
		}
		if err := events.EnqueueDispatch(ctx, tx, events.DispatchEvent{
			TenantID:  tenantID,
			JobID:     jobID,
			Status:    "assigned",
			VehicleID: vehicleID,
			DriverID:  driverID,
			Message:   "Auto-assigned nearest vehicle",
		}); err != nil {
			return err
		}
		job, err = scanJob(ctx, tx, tenantID, jobID)
		return err
	})
	return job, err
}

func (s *Server) UpdateJobStatus(ctx context.Context, req *dispatchv1.UpdateJobStatusRequest) (*dispatchv1.UpdateJobStatusResponse, error) {
	if req.Status == dispatchv1.JobStatus_JOB_STATUS_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "status required")
	}
	var job *dispatchv1.Job
	err := db.WithTenant(ctx, s.pool, req.TenantId, func(tx pgx.Tx) error {
		var cur string
		if err := tx.QueryRow(ctx, `SELECT status::text FROM dispatch_jobs WHERE id = $1::uuid`, req.JobId).Scan(&cur); err != nil {
			return err
		}
		from := statusFromDB(cur)
		if !canTransition(from, req.Status) {
			return fmt.Errorf("invalid transition %s -> %s", cur, statusToDB(req.Status))
		}
		if req.ActorUserId != "" {
			var driverID *string
			_ = tx.QueryRow(ctx, `SELECT driver_id::text FROM dispatch_jobs WHERE id = $1::uuid`, req.JobId).Scan(&driverID)
			if driverID != nil && *driverID != "" && req.ActorUserId != *driverID {
				return fmt.Errorf("driver can only update own jobs")
			}
		}
		newStatus := statusToDB(req.Status)
		_, err := tx.Exec(ctx, `
			UPDATE dispatch_jobs SET status = $3::job_status, updated_at = now()
			WHERE id = $1::uuid AND tenant_id = $2::uuid
		`, req.JobId, req.TenantId, newStatus)
		if err != nil {
			return err
		}
		if err := events.EnqueueDispatch(ctx, tx, events.DispatchEvent{
			TenantID: req.TenantId,
			JobID:    req.JobId,
			Status:   newStatus,
			Message:  fmt.Sprintf("Job status -> %s", newStatus),
		}); err != nil {
			return err
		}
		job, err = scanJob(ctx, tx, req.TenantId, req.JobId)
		return err
	})
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &dispatchv1.UpdateJobStatusResponse{Job: job}, nil
}

func (s *Server) Health(ctx context.Context, _ *dispatchv1.HealthRequest) (*dispatchv1.HealthResponse, error) {
	if err := s.pool.Ping(ctx); err != nil {
		return &dispatchv1.HealthResponse{Status: "degraded"}, nil
	}
	return &dispatchv1.HealthResponse{Status: "ok"}, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJobRow(rows rowScanner) (*dispatchv1.Job, error) {
	var j dispatchv1.Job
	var statusStr string
	var plat, plon, dlat, dlon float64
	var created, updated int64
	if err := rows.Scan(&j.Id, &j.TenantId, &statusStr, &plat, &plon, &dlat, &dlon,
		&j.PickupLabel, &j.DropoffLabel, &j.VehicleId, &j.DriverId, &created, &updated); err != nil {
		return nil, err
	}
	j.Status = statusFromDB(statusStr)
	j.Pickup = &commonv1.GeoPoint{Latitude: plat, Longitude: plon}
	j.Dropoff = &commonv1.GeoPoint{Latitude: dlat, Longitude: dlon}
	j.CreatedAtUnix = created
	j.UpdatedAtUnix = updated
	return &j, nil
}

func scanJob(ctx context.Context, tx pgx.Tx, tenantID, jobID string) (*dispatchv1.Job, error) {
	row := tx.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, status::text,
		       ST_Y(pickup::geometry), ST_X(pickup::geometry),
		       ST_Y(dropoff::geometry), ST_X(dropoff::geometry),
		       COALESCE(pickup_label,''), COALESCE(dropoff_label,''),
		       COALESCE(vehicle_id::text,''), COALESCE(driver_id::text,''),
		       EXTRACT(EPOCH FROM created_at)::bigint, EXTRACT(EPOCH FROM updated_at)::bigint
		FROM dispatch_jobs WHERE tenant_id = $1::uuid AND id = $2::uuid
	`, tenantID, jobID)
	return scanJobRow(row)
}
