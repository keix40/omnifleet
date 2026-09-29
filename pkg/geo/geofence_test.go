package geo_test

import (
	"testing"

	"github.com/keix40/omnifleet/pkg/geo"
)

func TestGeofence_Contains(t *testing.T) {
	box := geo.Polygon{
		{Lat: 37.7749, Lon: -122.4194},
		{Lat: 37.7749, Lon: -122.4094},
		{Lat: 37.7849, Lon: -122.4094},
		{Lat: 37.7849, Lon: -122.4194},
	}
	inside := geo.Point{Lat: 37.7799, Lon: -122.4144}
	outside := geo.Point{Lat: 37.70, Lon: -122.50}
	if !box.Contains(inside) {
		t.Fatal("expected inside point")
	}
	if box.Contains(outside) {
		t.Fatal("expected outside point")
	}
}

func TestGeofence_Transitions(t *testing.T) {
	poly := geo.Polygon{
		{Lat: -1, Lon: -1}, {Lat: -1, Lon: 1}, {Lat: 1, Lon: 1}, {Lat: 1, Lon: -1},
	}
	events := geo.DetectTransitions("gf-1", poly, false, true)
	if len(events) != 1 || events[0].EventType != "enter" {
		t.Fatalf("expected enter, got %+v", events)
	}
	events = geo.DetectTransitions("gf-1", poly, true, false)
	if len(events) != 1 || events[0].EventType != "exit" {
		t.Fatalf("expected exit, got %+v", events)
	}
}
