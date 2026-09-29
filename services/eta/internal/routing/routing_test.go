package routing_test

import (
	"context"
	"testing"

	"github.com/keix40/omnifleet/services/eta/internal/routing"
)

func TestHaversineProvider_Route(t *testing.T) {
	p := routing.HaversineProvider{RoadFactor: 1.35, SpeedMetersS: 10}
	from := routing.Point{Lat: 37.77, Lon: -122.42}
	to := routing.Point{Lat: 37.78, Lon: -122.41}
	res, err := p.Route(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if res.DistanceMeters <= 0 || res.DurationSeconds <= 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.Provider != "haversine" {
		t.Fatalf("provider=%s", res.Provider)
	}
}
