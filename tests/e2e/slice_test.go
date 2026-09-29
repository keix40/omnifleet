package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func gatewayURL() string {
	if v := os.Getenv("GATEWAY_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func wsURL() string {
	if v := os.Getenv("GATEWAY_WS_URL"); v != "" {
		return v
	}
	return strings.Replace(gatewayURL(), "http://", "ws://", 1)
}

func login(t *testing.T, email, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := http.Post(gatewayURL()+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login %s failed: %s", email, string(data))
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &out); err != nil || out.AccessToken == "" {
		t.Fatalf("parse login: %s", string(data))
	}
	return out.AccessToken
}

func wsTicket(t *testing.T, bearer string) string {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, gatewayURL()+"/api/v1/ws/fleet/ticket", nil)
	req.Header.Set("Authorization", "Bearer "+bearer)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ticket: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ticket failed: %s", string(data))
	}
	var out struct {
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(data, &out); err != nil || out.Ticket == "" {
		t.Fatalf("ticket parse: %s", string(data))
	}
	return out.Ticket
}

func listenWS(t *testing.T, ticket string) (*websocket.Conn, chan json.RawMessage) {
	t.Helper()
	sub := fmt.Sprintf("omnifleet.v1.%s", ticket)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL()+"/api/v1/ws/fleet/live", http.Header{
		"Sec-WebSocket-Protocol": []string{sub},
	})
	if err != nil {
		if resp != nil {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("ws dial: %v body=%s", err, string(b))
		}
		t.Fatalf("ws dial: %v", err)
	}
	ch := make(chan json.RawMessage, 32)
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			ch <- json.RawMessage(msg)
		}
	}()
	return conn, ch
}

func postPositions(t *testing.T, driverToken string, steps int) {
	t.Helper()
	startLat, startLon := 37.7700, -122.4300
	endLat, endLon := 37.7900, -122.4100
	for i := 0; i <= steps; i++ {
		p := float64(i) / float64(steps)
		lat := startLat + (endLat-startLat)*p
		lon := startLon + (endLon-startLon)*p
		payload, _ := json.Marshal(map[string]any{
			"vehicle_id":  "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
			"latitude":    lat,
			"longitude":   lon,
			"speed_mps":   8.0,
			"heading_deg": 90.0,
		})
		req, _ := http.NewRequest(http.MethodPost, gatewayURL()+"/api/v1/tracking/positions", bytes.NewReader(payload))
		req.Header.Set("Authorization", "Bearer "+driverToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("ingest: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			t.Fatalf("ingest status %d", resp.StatusCode)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func TestVerticalSlice_TenantScopedWebSocket(t *testing.T) {
	if os.Getenv("E2E_COMPOSE") != "1" {
		t.Skip("set E2E_COMPOSE=1 when running against docker compose stack")
	}

	acmeDisp := login(t, "dispatcher@acme.test", "demo-password-change-me")
	globexDisp := login(t, "dispatcher@globex.test", "demo-password-change-me")
	acmeDriver := login(t, "driver@acme.test", "demo-password-change-me")

	acmeWS, acmeCh := listenWS(t, wsTicket(t, acmeDisp))
	defer acmeWS.Close()
	globexWS, globexCh := listenWS(t, wsTicket(t, globexDisp))
	defer globexWS.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		postPositions(t, acmeDriver, 25)
	}()

	acmePositions := 0
	acmeAlerts := 0
	globexEvents := 0
	deadline := time.Now().Add(60 * time.Second)

	for time.Now().Before(deadline) {
		select {
		case msg := <-acmeCh:
			if bytes.Contains(msg, []byte(`"latitude"`)) {
				acmePositions++
			}
			if bytes.Contains(msg, []byte(`"event_type"`)) {
				acmeAlerts++
			}
		case msg := <-globexCh:
			if bytes.Contains(msg, []byte(`"latitude"`)) || bytes.Contains(msg, []byte(`"event_type"`)) {
				globexEvents++
			}
		case <-time.After(200 * time.Millisecond):
		}
		if acmePositions >= 2 && acmeAlerts >= 1 {
			break
		}
	}
	wg.Wait()

	if acmePositions < 2 {
		t.Fatalf("expected Acme position events, got %d", acmePositions)
	}
	if acmeAlerts < 1 {
		t.Fatalf("expected at least one geofence alert for Acme, got %d", acmeAlerts)
	}
	if globexEvents != 0 {
		t.Fatalf("Globex dispatcher must not receive Acme events, got %d", globexEvents)
	}
}
