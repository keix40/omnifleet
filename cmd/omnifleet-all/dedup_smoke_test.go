//go:build smoke

package main_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func appDatabaseURL() string {
	if u := os.Getenv("APP_DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://omnifleet_app:omnifleet_app@localhost:5432/omnifleet?sslmode=disable"
}

func TestSmoke_GeofenceAlertAndNotificationDedup(t *testing.T) {
	if os.Getenv("SMOKE_TEST") != "1" {
		t.Skip("set SMOKE_TEST=1")
	}
	if os.Getenv("NATS_EMBEDDED") != "1" {
		t.Skip("embedded NATS dedup regression")
	}

	db, err := sql.Open("pgx", appDatabaseURL())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var before int
	if err := db.QueryRow(`SELECT count(*) FROM notification_deliveries WHERE event_type = 'alerts'`).Scan(&before); err != nil {
		t.Fatal(err)
	}

	driverBody, _ := json.Marshal(map[string]string{
		"email":       "driver@acme.test",
		"password":    "demo-password-change-me",
		"tenant_slug": "acme-logistics",
	})
	dresp, err := http.Post(baseURL()+"/api/v1/auth/login", "application/json", bytes.NewReader(driverBody))
	if err != nil {
		t.Fatal(err)
	}
	dData, _ := io.ReadAll(dresp.Body)
	dresp.Body.Close()
	var driverLogin struct {
		Token string `json:"access_token"`
	}
	_ = json.Unmarshal(dData, &driverLogin)
	if driverLogin.Token == "" {
		t.Fatalf("driver login: %s", dData)
	}

	postPosition := func(lat, lon float64) {
		t.Helper()
		pos, _ := json.Marshal(map[string]any{
			"vehicle_id":  "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
			"latitude":    lat,
			"longitude":   lon,
			"speed_mps":   5.0,
			"heading_deg": 90.0,
		})
		preq, _ := http.NewRequest(http.MethodPost, baseURL()+"/api/v1/tracking/positions", bytes.NewReader(pos))
		preq.Header.Set("Authorization", "Bearer "+driverLogin.Token)
		preq.Header.Set("Content-Type", "application/json")
		presp, err := http.DefaultClient.Do(preq)
		if err != nil {
			t.Fatal(err)
		}
		presp.Body.Close()
		if presp.StatusCode >= 300 {
			t.Fatalf("ingest status %d", presp.StatusCode)
		}
	}

	// Outside Acme SF depot geofence, then inside — should produce one enter alert.
	postPosition(37.75, -122.50)
	time.Sleep(1500 * time.Millisecond)
	postPosition(37.78, -122.425)

	deadline := time.Now().Add(30 * time.Second)
	var after int
	for time.Now().Before(deadline) {
		if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM notification_deliveries WHERE event_type = 'alerts'`).Scan(&after); err != nil {
			t.Fatal(err)
		}
		if after-before >= 1 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if after-before != 1 {
		t.Fatalf("expected 1 new alert notification delivery, got %d (before=%d after=%d)", after-before, before, after)
	}

}
