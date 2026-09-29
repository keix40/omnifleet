package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthLookup_UserFoundForTenantSlug(t *testing.T) {
	dsn := appDSN()
	if os.Getenv("CI") == "" && os.Getenv("DATABASE_URL") == "" {
		t.Skip("database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var userID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM auth_lookup_user($1, $2)
	`, "dispatcher@acme.test", "acme-logistics").Scan(&userID)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if userID == "" {
		t.Fatal("expected user id")
	}
}

func TestAuthLookup_WrongTenantReturnsNoRows(t *testing.T) {
	dsn := appDSN()
	if os.Getenv("CI") == "" && os.Getenv("DATABASE_URL") == "" {
		t.Skip("database required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var userID string
	err = pool.QueryRow(ctx, `
		SELECT id::text FROM auth_lookup_user($1, $2)
	`, "dispatcher@acme.test", "globex-freight").Scan(&userID)
	if err == nil {
		t.Fatal("expected no row for wrong tenant slug")
	}
}
