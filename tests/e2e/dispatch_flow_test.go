package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func authPost(t *testing.T, token, path string, body any) (*http.Response, []byte) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(http.MethodPost, gatewayURL()+path, reader)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, data
}

func authPatch(t *testing.T, token, path string, body any) (*http.Response, []byte) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPatch, gatewayURL()+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, data
}

func deliveredNotifications(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("docker", "compose", "exec", "-T", "postgres",
		"psql", "-U", "omnifleet", "-d", "omnifleet", "-tAc",
		"SELECT count(*) FROM notification_deliveries WHERE status='delivered'")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("notification count query: %v %s", err, out)
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n
}

func TestVerticalSlice_DispatchETAAndNotification(t *testing.T) {
	if os.Getenv("E2E_COMPOSE") != "1" {
		t.Skip("set E2E_COMPOSE=1 when running against docker compose stack")
	}

	resetGeofenceState(t)

	disp := login(t, "dispatcher@acme.test", "demo-password-change-me", "acme-logistics")
	driver := login(t, "driver@acme.test", "demo-password-change-me", "acme-logistics")

	// Seed a fresh position near pickup.
	_, _ = authPost(t, driver, "/api/v1/tracking/positions", map[string]any{
		"vehicle_id":  "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
		"latitude":    37.7720,
		"longitude":   -122.4250,
		"speed_mps":   10.0,
		"heading_deg": 90.0,
	})
	time.Sleep(time.Second)

	resp, data := authPost(t, disp, "/api/v1/dispatch/jobs", map[string]any{
		"pickup":        map[string]float64{"latitude": 37.7720, "longitude": -122.4250},
		"dropoff":       map[string]float64{"latitude": 37.7850, "longitude": -122.4100},
		"pickup_label":  "Acme pickup",
		"dropoff_label": "Acme dropoff",
		"auto_assign":   true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create job: %d %s", resp.StatusCode, string(data))
	}
	var job struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &job); err != nil || job.ID == "" {
		t.Fatalf("parse job: %s", string(data))
	}

	etaResp, etaData := authPost(t, disp, "/api/v1/eta/compute", map[string]any{
		"vehicle_id":     "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
		"publish_update": true,
		"destination":    map[string]float64{"latitude": 37.7850, "longitude": -122.4100},
	})
	if etaResp.StatusCode != http.StatusOK {
		t.Fatalf("eta: %d %s", etaResp.StatusCode, string(etaData))
	}

	for _, st := range []string{"en_route", "picked_up", "delivered"} {
		patchResp, patchData := authPatch(t, driver, "/api/v1/dispatch/jobs/"+job.ID+"/status", map[string]string{
			"status": st,
		})
		if patchResp.StatusCode != http.StatusOK {
			t.Fatalf("status %s: %d %s", st, patchResp.StatusCode, string(patchData))
		}
		time.Sleep(500 * time.Millisecond)
	}

	deadline := time.Now().Add(60 * time.Second)
	delivered := 0
	for time.Now().Before(deadline) {
		delivered = deliveredNotifications(t)
		if delivered >= 1 {
			break
		}
		time.Sleep(time.Second)
	}
	if delivered < 1 {
		t.Fatalf("expected at least one delivered notification, got %d", delivered)
	}
}
