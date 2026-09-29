package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
)

func TestLogin_WrongTenantSlugRejected(t *testing.T) {
	if os.Getenv("E2E_COMPOSE") != "1" {
		t.Skip("set E2E_COMPOSE=1 when running against docker compose stack")
	}
	body, _ := json.Marshal(map[string]string{
		"email":       "dispatcher@acme.test",
		"password":    "demo-password-change-me",
		"tenant_slug": "globex-freight",
	})
	resp, err := http.Post(gatewayURL()+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected login failure for wrong tenant, got: %s", string(data))
	}
}
