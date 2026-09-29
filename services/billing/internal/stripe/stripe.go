package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type LiveProvider struct {
	WebhookSecret string
	APIKey        string
}

func (LiveProvider) Name() string { return "stripe" }

func (l LiveProvider) GetSubscription(ctx context.Context, tenantID string) (SubscriptionInfo, error) {
	_ = ctx
	_ = tenantID
	return SubscriptionInfo{Status: "unknown"}, fmt.Errorf("stripe live lookup not implemented in slice")
}

func (l LiveProvider) HandleWebhook(_ context.Context, payload []byte, signature string) (string, error) {
	if l.WebhookSecret == "" {
		return "", fmt.Errorf("stripe webhook secret not configured")
	}
	if !verifyStripeSignature(payload, signature, l.WebhookSecret) {
		return "", fmt.Errorf("invalid signature")
	}
	var evt struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return "", err
	}
	return evt.ID, nil
}

func verifyStripeSignature(payload []byte, header, secret string) bool {
	// Minimal Stripe v1 signature check for tests/production wiring.
	const prefix = "t="
	var ts, sig string
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "t=") {
			ts = strings.TrimPrefix(part, "t=")
		}
		if strings.HasPrefix(part, "v1=") {
			sig = strings.TrimPrefix(part, "v1=")
		}
	}
	if ts == "" || sig == "" {
		return false
	}
	_ = ts
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

func NewFromEnv() Provider {
	mode := strings.ToLower(os.Getenv("STRIPE_PROVIDER"))
	if mode == "fake" || mode == "noop" || os.Getenv("STRIPE_SECRET_KEY") == "" {
		return FakeProvider{}
	}
	return LiveProvider{
		WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		APIKey:        os.Getenv("STRIPE_SECRET_KEY"),
	}
}

func CurrentPeriod() (start, end string) {
	now := time.Now().UTC()
	startT := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endT := startT.AddDate(0, 1, 0).Add(-time.Second)
	return startT.Format("2006-01-02"), endT.Format("2006-01-02")
}
