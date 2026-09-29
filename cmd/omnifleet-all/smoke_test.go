//go:build smoke

package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func baseURL() string {
	if u := os.Getenv("SMOKE_GATEWAY_URL"); u != "" {
		return u
	}
	return "http://127.0.0.1:" + env("SMOKE_PORT", "18080")
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func TestSmoke_LoginPositionWebSocket(t *testing.T) {
	if os.Getenv("SMOKE_TEST") != "1" {
		t.Skip("set SMOKE_TEST=1")
	}

	body, _ := json.Marshal(map[string]string{
		"email":       "dispatcher@acme.test",
		"password":    "demo-password-change-me",
		"tenant_slug": "acme-logistics",
	})
	resp, err := http.Post(baseURL()+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: %s", data)
	}
	var login struct {
		Token string `json:"access_token"`
	}
	_ = json.Unmarshal(data, &login)
	if login.Token == "" {
		t.Fatalf("no token: %s", data)
	}

	ticketReq, _ := http.NewRequest(http.MethodPost, baseURL()+"/api/v1/ws/fleet/ticket", nil)
	ticketReq.Header.Set("Authorization", "Bearer "+login.Token)
	ticketResp, err := http.DefaultClient.Do(ticketReq)
	if err != nil {
		t.Fatal(err)
	}
	ticketData, _ := io.ReadAll(ticketResp.Body)
	ticketResp.Body.Close()
	var ticketOut struct {
		Ticket string `json:"ticket"`
	}
	_ = json.Unmarshal(ticketData, &ticketOut)

	wsBase := os.Getenv("SMOKE_WS_URL")
	if wsBase == "" {
		wsBase = "ws" + baseURL()[4:]
	}
	sub := fmt.Sprintf("omnifleet.v1.%s", ticketOut.Ticket)
	conn, _, err := websocket.DefaultDialer.Dial(wsBase+"/api/v1/ws/fleet/live", http.Header{
		"Sec-WebSocket-Protocol": []string{sub},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

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

	pos, _ := json.Marshal(map[string]any{
		"vehicle_id":  "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
		"latitude":    37.772,
		"longitude":   -122.425,
		"speed_mps":   8.0,
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

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			continue
		}
		if bytes.Contains(msg, []byte(`"latitude"`)) {
			return
		}
	}
	t.Fatal("no position event on websocket")
}
