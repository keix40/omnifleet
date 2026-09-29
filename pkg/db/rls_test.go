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

func TestRLS_TenantIsolation(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; run via docker-compose or CI postgres service")
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

	// Acme tenant must not see Globex vehicle labels.
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
