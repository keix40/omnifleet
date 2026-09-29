package auth_test

import (
	"testing"
	"time"

	"github.com/keix40/omnifleet/pkg/auth"
)

func TestWSTicket_SingleUse(t *testing.T) {
	store := auth.NewWSTicketStore([]byte("01234567890123456789012345678901"), time.Minute, auth.NewMemorySingleUseStore())
	claims := &auth.Claims{UserID: "u1", TenantID: "t1", Role: auth.RoleDispatcher}
	ticket, _, err := store.Issue(claims)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := store.Redeem(ticket)
	if !ok || got.TenantID != "t1" {
		t.Fatalf("first redeem failed: ok=%v claims=%+v", ok, got)
	}
	_, ok = store.Redeem(ticket)
	if ok {
		t.Fatal("ticket must be single-use")
	}
}

func TestWSTicket_TamperedSignatureRejected(t *testing.T) {
	store := auth.NewWSTicketStore([]byte("01234567890123456789012345678901"), time.Minute, auth.NewMemorySingleUseStore())
	claims := &auth.Claims{UserID: "u1", TenantID: "t1", Role: auth.RoleDispatcher}
	ticket, _, err := store.Issue(claims)
	if err != nil {
		t.Fatal(err)
	}
	ticket = ticket + "x"
	if _, ok := store.Redeem(ticket); ok {
		t.Fatal("tampered ticket must be rejected")
	}
}
