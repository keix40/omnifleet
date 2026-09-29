package events_test

import (
	"testing"

	"github.com/keix40/omnifleet/pkg/events"
)

func TestSharedNATSMode(t *testing.T) {
	t.Setenv("OMNIFLEET_SHARED_NATS", "1")
	if !events.SharedNATSMode() {
		t.Fatal("expected shared mode")
	}
}
