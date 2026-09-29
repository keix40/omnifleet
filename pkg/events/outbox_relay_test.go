package events_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/keix40/omnifleet/pkg/db"
	"github.com/keix40/omnifleet/pkg/events"
)

type countingOutboxPublisher struct {
	publishCount atomic.Int32
}

func (c *countingOutboxPublisher) PublishRaw(_ context.Context, _ string, _ []byte) error {
	c.publishCount.Add(1)
	return nil
}

func TestRelayPending_ConcurrentRelaysPublishOnce(t *testing.T) {
	if os.Getenv("CI") == "" && os.Getenv("DATABASE_URL") == "" {
		t.Skip("database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, appDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `DELETE FROM event_outbox`); err != nil {
		t.Fatal(err)
	}

	tenant := "11111111-1111-1111-1111-111111111111"
	alert := events.AlertEvent{
		TenantID:     tenant,
		VehicleID:    "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
		GeofenceID:   "99999999-9999-9999-9999-999999999901",
		GeofenceName: "Acme SF Depot",
		EventType:    "enter",
		Message:      "dedup test",
	}
	err = db.WithTenant(ctx, pool, tenant, func(tx pgx.Tx) error {
		return events.EnqueueAlert(ctx, tx, alert)
	})
	if err != nil {
		t.Fatal(err)
	}

	pub := &countingOutboxPublisher{}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, _ = events.RelayPending(ctx, pool, pub, 8)
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}
	wg.Wait()

	if got := pub.publishCount.Load(); got != 1 {
		t.Fatalf("expected exactly 1 NATS publish for 1 outbox row, got %d", got)
	}
	var unpublished int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM event_outbox WHERE published_at IS NULL`).Scan(&unpublished); err != nil {
		t.Fatal(err)
	}
	if unpublished != 0 {
		t.Fatalf("expected outbox drained, unpublished=%d", unpublished)
	}
}
