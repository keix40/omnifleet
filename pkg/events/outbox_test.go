package events_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/keix40/omnifleet/pkg/db"
	"github.com/keix40/omnifleet/pkg/events"
)

func appDSN() string {
	if v := os.Getenv("APP_DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://omnifleet_app:omnifleet_app@localhost:5432/omnifleet?sslmode=disable"
}

func TestOutbox_EnqueueUnderTenantRLS(t *testing.T) {
	if os.Getenv("CI") == "" && os.Getenv("DATABASE_URL") == "" {
		t.Skip("database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, appDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tenant := "11111111-1111-1111-1111-111111111111"
	alert := events.AlertEvent{
		TenantID:     tenant,
		VehicleID:    "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
		GeofenceID:   "99999999-9999-9999-9999-999999999901",
		GeofenceName: "Acme SF Depot",
		EventType:    "enter",
		Message:      "test",
	}

	err = db.WithTenant(ctx, pool, tenant, func(tx pgx.Tx) error {
		return events.EnqueueAlert(ctx, tx, alert)
	})
	if err != nil {
		t.Fatal(err)
	}

	var unpublished int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM event_outbox WHERE published_at IS NULL`).Scan(&unpublished); err != nil {
		t.Fatal(err)
	}
	if unpublished < 1 {
		t.Fatalf("expected outbox row, got %d", unpublished)
	}
}
