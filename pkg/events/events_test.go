package events_test

import (
	"strings"
	"testing"

	"github.com/keix40/omnifleet/pkg/events"
)

func TestEventSubjects_TenantScoped(t *testing.T) {
	a := events.PositionSubject("11111111-1111-1111-1111-111111111111")
	b := events.PositionSubject("22222222-2222-2222-2222-222222222222")
	if a == b {
		t.Fatal("subjects must differ per tenant")
	}
	if !strings.Contains(a, "11111111") {
		t.Fatal("subject should embed tenant id")
	}
}

func TestAlertSubject(t *testing.T) {
	s := events.AlertSubject("tenant-a")
	if s != "fleet.tenant-a.alerts" {
		t.Fatalf("unexpected subject: %s", s)
	}
}
