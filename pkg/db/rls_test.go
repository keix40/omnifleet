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

func appDSN() string {
	if v := os.Getenv("APP_DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://omnifleet_app:omnifleet_app@localhost:5432/omnifleet?sslmode=disable"
}

func TestRLS_TenantIsolation(t *testing.T) {
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
	globex := "22222222-2222-2222-2222-222222222222"

	var acmeVehicles, globexVehicles int
	err = db.WithTenant(ctx, pool, acme, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM vehicles`).Scan(&acmeVehicles)
	})
	if err != nil {
		t.Fatalf("acme query: %v", err)
	}
	err = db.WithTenant(ctx, pool, globex, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM vehicles`).Scan(&globexVehicles)
	})
	if err != nil {
		t.Fatalf("globex query: %v", err)
	}
	if acmeVehicles != 1 || globexVehicles != 1 {
		t.Fatalf("expected 1 vehicle per tenant, got acme=%d globex=%d", acmeVehicles, globexVehicles)
	}

	var label string
	err = db.WithTenant(ctx, pool, acme, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT label FROM vehicles WHERE id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
		`).Scan(&label)
	})
	if err == nil {
		t.Fatal("expected no rows when reading other tenant vehicle under RLS")
	}
}

func TestRLS_NoTenantContextReturnsEmpty(t *testing.T) {
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

	var count int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM vehicles`).Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Fatalf("without app.tenant_id, RLS must hide all rows; got count=%d", count)
	}
}

func TestRLS_AppRoleCannotBypassAsOwner(t *testing.T) {
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

	var owner string
	err = pool.QueryRow(ctx, `
		SELECT pg_catalog.pg_get_userbyid(c.relowner)
		FROM pg_catalog.pg_class c
	 JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relname = 'vehicles'
	`).Scan(&owner)
	if err != nil {
		t.Fatalf("owner lookup: %v", err)
	}
	if owner == "omnifleet_app" {
		t.Fatal("vehicles must not be owned by omnifleet_app")
	}

	var forced bool
	err = pool.QueryRow(ctx, `
		SELECT c.relforcerowsecurity
		FROM pg_catalog.pg_class c
	 JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relname = 'vehicles'
	`).Scan(&forced)
	if err != nil {
		t.Fatalf("force rls lookup: %v", err)
	}
	if !forced {
		t.Fatal("vehicles must have FORCE ROW LEVEL SECURITY enabled")
	}
}
