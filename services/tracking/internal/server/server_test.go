package server_test

import (
	"errors"
	"testing"

	"github.com/keix40/omnifleet/pkg/validate"
)

func TestIngestValidationMessagesAreSanitized(t *testing.T) {
	if msg := validate.PublicMessage(validate.ErrInvalidVehicleID); msg != "invalid vehicle_id" {
		t.Fatalf("got %q", msg)
	}
	if msg := validate.PublicMessage(validate.WrapInternal(errors.New("pq: connection refused"))); msg != "request failed" {
		t.Fatalf("internal details must not leak: %q", msg)
	}
}
