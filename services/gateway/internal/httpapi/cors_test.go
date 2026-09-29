package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_AllowedOriginGetsHeadersAndPreflight(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://omnifleet-dashboard.vercel.app,http://localhost:3000")

	s := &Server{}
	r := s.Router()

	origin := "https://omnifleet-dashboard.vercel.app"

	opt := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	opt.Header.Set("Origin", origin)
	opt.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, opt)

	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Fatalf("preflight status=%d want 200/204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("Allow-Origin=%q want %q", got, origin)
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("expected Allow-Credentials=true")
	}

	get := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	get.Header.Set("Origin", origin)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, get)
	if w2.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Fatalf("GET Allow-Origin=%q want %q", w2.Header().Get("Access-Control-Allow-Origin"), origin)
	}
}

func TestCORS_DisallowedOriginGetsNoHeaders(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")

	s := &Server{}
	r := s.Router()

	bad := "https://evil.example"
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", bad)
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("disallowed origin must not get Allow-Origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestOriginAllowed_ExactMatch(t *testing.T) {
	allowed := []string{"http://localhost:3000"}
	if !originAllowed("http://localhost:3000", allowed) {
		t.Fatal("expected match")
	}
	if originAllowed("http://localhost:3001", allowed) {
		t.Fatal("expected no match")
	}
}
