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

func TestGeofence_DemoDepotContainsPoint(t *testing.T) {
	if os.Getenv("CI") == "" && os.Getenv("DATABASE_URL") == "" {
		t.Skip("requires bootstrapped postgres")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, appDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	acme := "11111111-1111-1111-1111-111111111111"
	var inside bool
	err = db.WithTenant(ctx, pool, acme, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT ST_Intersects(
				boundary,
				ST_SetSRID(ST_MakePoint(-122.4144, 37.7799), 4326)::geography
			)
			FROM geofences
			WHERE name = 'Acme SF Depot'
		`).Scan(&inside)
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !inside {
		t.Fatal("demo depot geofence should contain waypoint used in e2e")
	}
}
