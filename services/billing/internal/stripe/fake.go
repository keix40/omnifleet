package stripe

import (
	"context"
	"encoding/json"
	"fmt"
)

type FakeProvider struct{}

func (FakeProvider) Name() string { return "fake" }

func (FakeProvider) GetSubscription(_ context.Context, _ string) (SubscriptionInfo, error) {
	return SubscriptionInfo{PlanID: "growth", PlanName: "Growth", Status: "active"}, nil
}

func (FakeProvider) HandleWebhook(_ context.Context, payload []byte, signature string) (string, error) {
	if signature == "" {
		return "", fmt.Errorf("missing signature")
	}
	var evt struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return "", err
	}
	if evt.ID == "" {
		evt.ID = "evt_fake_" + signature
	}
	return evt.ID, nil
}
