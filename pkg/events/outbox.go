package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

// RelayPending publishes unpublished outbox rows (call outside tenant-scoped transactions).
func RelayPending(ctx context.Context, pool *pgxpool.Pool, pub *Publisher, batchSize int) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, subject, payload
		FROM event_outbox
		WHERE published_at IS NULL
		ORDER BY created_at
		LIMIT $1
	`, batchSize)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type row struct {
		id, subject string
		payload     []byte
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.subject, &r.payload); err != nil {
			return 0, err
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	published := 0
	for _, r := range pending {
		if err := pub.PublishRaw(ctx, r.subject, r.payload); err != nil {
			return published, err
		}
		tag, err := pool.Exec(ctx, `
			UPDATE event_outbox SET published_at = $2 WHERE id = $1::uuid AND published_at IS NULL
		`, r.id, time.Now().UTC())
		if err != nil {
			return published, err
		}
		if tag.RowsAffected() > 0 {
			published++
		}
	}
	return published, nil
}
