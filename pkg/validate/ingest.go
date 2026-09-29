package validate

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrInvalidVehicleID = errors.New("invalid vehicle_id")
	ErrInvalidLatitude  = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude = errors.New("longitude must be between -180 and 180")
)

func IngestPosition(vehicleID string, lat, lon float64) error {
	if _, err := uuid.Parse(vehicleID); err != nil {
		return ErrInvalidVehicleID
	}
	if lat < -90 || lat > 90 {
		return ErrInvalidLatitude
	}
	if lon < -180 || lon > 180 {
		return ErrInvalidLongitude
	}
	return nil
}

// PublicMessage returns a safe client-facing error string.
func PublicMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidVehicleID):
		return "invalid vehicle_id"
	case errors.Is(err, ErrInvalidLatitude):
		return "invalid latitude"
	case errors.Is(err, ErrInvalidLongitude):
		return "invalid longitude"
	default:
		return "request failed"
	}
}

func WrapInternal(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("internal: %w", err)
}
