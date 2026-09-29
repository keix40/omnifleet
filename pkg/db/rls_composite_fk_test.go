package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/keix40/omnifleet/pkg/db"
)

func TestRLS_CrossTenantVehiclePositionRejected(t *testing.T) {
	dsn := appDSN()
	if os.Getenv("CI") == "" && os.Getenv("DATABASE_URL") == "" {
		t.Skip("set DATABASE_URL or APP_DATABASE_URL for integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	acme := "11111111-1111-1111-1111-111111111111"
	globexVehicle := "ffffffff-ffff-ffff-ffff-ffffffffffff"

	err = db.WithTenant(ctx, pool, acme, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO gps_positions (tenant_id, vehicle_id, location, recorded_at)
			VALUES ($1::uuid, $2::uuid,
			        ST_SetSRID(ST_MakePoint(-122.42, 37.77), 4326)::geography,
			        now())
		`, acme, globexVehicle)
		return err
	})
	if err == nil {
		t.Fatal("expected FK violation when inserting another tenant's vehicle_id")
	}
}

func TestRLS_CrossTenantGeofenceStateRejected(t *testing.T) {
	dsn := appDSN()
	if os.Getenv("CI") == "" && os.Getenv("DATABASE_URL") == "" {
		t.Skip("set DATABASE_URL or APP_DATABASE_URL for integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	acme := "11111111-1111-1111-1111-111111111111"
	globexGeofence := "99999999-9999-9999-9999-999999999902"
	acmeVehicle := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"

	err = db.WithTenant(ctx, pool, acme, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO vehicle_geofence_state (tenant_id, vehicle_id, geofence_id, inside)
			VALUES ($1::uuid, $2::uuid, $3::uuid, true)
		`, acme, acmeVehicle, globexGeofence)
		return err
	})
	if err == nil {
		t.Fatal("expected FK violation when referencing another tenant's geofence_id")
	}
}
