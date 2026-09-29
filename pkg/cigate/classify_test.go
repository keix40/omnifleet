package cigate_test

import (
	"testing"

	"github.com/keix40/omnifleet/pkg/cigate"
)

func TestClassify_PkgRequiresGoAndE2E(t *testing.T) {
	req := cigate.ClassifyChangedFiles([]string{"pkg/auth/jwt.go"})
	if !req.Go || !req.E2E {
		t.Fatalf("pkg changes must require go and e2e: %+v", req)
	}
	if req.Web || req.Infra {
		t.Fatalf("unexpected web/infra: %+v", req)
	}
}

func TestClassify_ServicesRequiresGoAndE2E(t *testing.T) {
	req := cigate.ClassifyChangedFiles([]string{"services/gateway/internal/httpapi/server.go"})
	if !req.Go || !req.E2E {
		t.Fatalf("services changes must require go and e2e: %+v", req)
	}
}

func TestClassify_DbRequiresGoAndE2E(t *testing.T) {
	req := cigate.ClassifyChangedFiles([]string{"db/migrations/006_foo.sql"})
	if !req.Go || !req.E2E {
		t.Fatalf("db changes must require go and e2e: %+v", req)
	}
}

func TestClassify_WebOnly(t *testing.T) {
	req := cigate.ClassifyChangedFiles([]string{"web/dashboard/app/page.tsx"})
	if !req.Web || req.Go || req.E2E {
		t.Fatalf("web-only diff: %+v", req)
	}
}

func TestClassify_DashboardOnlyDoesNotRequireE2E(t *testing.T) {
	req := cigate.ClassifyChangedFiles([]string{"web/dashboard/package.json"})
	if req.E2E {
		t.Fatal("dashboard-only change should not require e2e")
	}
}
