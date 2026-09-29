package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSingleUseStore enforces single-use tickets in Postgres for multi-replica gateways.
type PostgresSingleUseStore struct {
	pool *pgxpool.Pool
}

func NewPostgresSingleUseStore(pool *pgxpool.Pool) *PostgresSingleUseStore {
	return &PostgresSingleUseStore{pool: pool}
}

func (p *PostgresSingleUseStore) RedeemOnce(jti string, expiresAt time.Time) (bool, error) {
	if time.Now().After(expiresAt) {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tag, err := p.pool.Exec(ctx, `
		INSERT INTO ws_ticket_redemptions (jti, expires_at)
		VALUES ($1, $2)
		ON CONFLICT (jti) DO NOTHING
	`, jti, expiresAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
