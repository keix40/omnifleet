package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenant runs fn inside a transaction with app.tenant_id set for RLS.
func WithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SetTenantOnConn sets RLS context on an existing connection (local transaction scope).
func SetTenantOnConn(ctx context.Context, conn pgx.Tx, tenantID string) error {
	_, err := conn.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID)
	return err
}
