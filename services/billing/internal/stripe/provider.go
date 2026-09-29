package stripe

import "context"

type SubscriptionInfo struct {
	PlanID     string
	PlanName   string
	Status     string
	CustomerID string
	SubID      string
}

type Provider interface {
	Name() string
	GetSubscription(ctx context.Context, tenantID string) (SubscriptionInfo, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) (string, error)
}
