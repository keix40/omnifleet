package validate_test

import (
	"testing"

	"github.com/keix40/omnifleet/pkg/validate"
)

func TestIngestPosition_Valid(t *testing.T) {
	if err := validate.IngestPosition("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", 37.77, -122.42); err != nil {
		t.Fatal(err)
	}
}

func TestIngestPosition_InvalidUUID(t *testing.T) {
	if err := validate.IngestPosition("not-a-uuid", 0, 0); err != validate.ErrInvalidVehicleID {
		t.Fatalf("got %v", err)
	}
}

func TestIngestPosition_InvalidLatLon(t *testing.T) {
	if err := validate.IngestPosition("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", 91, 0); err != validate.ErrInvalidLatitude {
		t.Fatalf("lat: %v", err)
	}
	if err := validate.IngestPosition("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", 0, 181); err != validate.ErrInvalidLongitude {
		t.Fatalf("lon: %v", err)
	}
}
