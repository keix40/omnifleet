package channels_test

import (
	"context"
	"testing"

	"github.com/keix40/omnifleet/services/notifications/internal/channels"
)

func TestLogChannel_Deliver(t *testing.T) {
	c := channels.LogChannel{}
	if err := c.Deliver(context.Background(), "", "hello", map[string]string{"event_type": "dispatch"}); err != nil {
		t.Fatal(err)
	}
}
