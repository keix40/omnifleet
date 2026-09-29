package events

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxPublisher publishes relayed outbox payloads (implemented by *Publisher).
type OutboxPublisher interface {
	PublishRaw(ctx context.Context, subject string, data []byte) error
}

// SecondaryOutboxRelayDisabled returns true when dispatch (or other secondary relays)
// must not poll event_outbox — used by omnifleet-all where geofencing runs the relay.
func SecondaryOutboxRelayDisabled() bool {
	return os.Getenv("OMNIFLEET_DISABLE_SECONDARY_OUTBOX_RELAY") == "1"
}

// EnqueueJSON writes an arbitrary payload to the outbox inside tx.
func EnqueueJSON(ctx context.Context, tx pgx.Tx, tenantID, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO event_outbox (tenant_id, subject, payload)
		VALUES ($1::uuid, $2, $3::jsonb)
	`, tenantID, subject, data)
	return err
}

// EnqueueAlert writes a geofence alert to the transactional outbox inside tx.
func EnqueueAlert(ctx context.Context, tx pgx.Tx, alert AlertEvent) error {
	return EnqueueJSON(ctx, tx, alert.TenantID, AlertSubject(alert.TenantID), alert)
}

// EnqueueDispatch writes a dispatch lifecycle event to the outbox inside tx.
func EnqueueDispatch(ctx context.Context, tx pgx.Tx, ev DispatchEvent) error {
	return EnqueueJSON(ctx, tx, ev.TenantID, DispatchSubject(ev.TenantID), ev)
}

// RelayPending claims unpublished outbox rows with FOR UPDATE SKIP LOCKED so multiple
// relay loops (e.g. geofencing + dispatch in omnifleet-all) cannot publish the same row twice.
func RelayPending(ctx context.Context, pool *pgxpool.Pool, pub OutboxPublisher, batchSize int) (int, error) {
	if batchSize < 1 {
		batchSize = 32
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id::text, subject, payload
		FROM event_outbox
		WHERE published_at IS NULL
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, batchSize)
	if err != nil {
		return 0, err
	}

	type row struct {
		id, subject string
		payload     []byte
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.subject, &r.payload); err != nil {
			rows.Close()
			return 0, err
		}
		pending = append(pending, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	published := 0
	for _, r := range pending {
		if err := pub.PublishRaw(ctx, r.subject, r.payload); err != nil {
			return published, err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE event_outbox SET published_at = $2 WHERE id = $1::uuid AND published_at IS NULL
		`, r.id, now)
		if err != nil {
			return published, err
		}
		if tag.RowsAffected() > 0 {
			published++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return published, nil
}
