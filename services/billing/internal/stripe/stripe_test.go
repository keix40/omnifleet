package stripe_test

import (
	"context"
	"testing"

	"github.com/keix40/omnifleet/services/billing/internal/stripe"
)

func TestFakeProvider_Webhook(t *testing.T) {
	p := stripe.FakeProvider{}
	id, err := p.HandleWebhook(context.Background(), []byte(`{"id":"evt_1","type":"invoice.paid"}`), "sig")
	if err != nil {
		t.Fatal(err)
	}
	if id != "evt_1" {
		t.Fatalf("id=%s", id)
	}
}

func TestNewFromEnv_DefaultFake(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "")
	p := stripe.NewFromEnv()
	if p.Name() != "fake" {
		t.Fatalf("provider=%s", p.Name())
	}
}
