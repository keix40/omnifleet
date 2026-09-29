package server_test

import (
	"context"
	"os"
	"testing"

	commonv1 "github.com/keix40/omnifleet/gen/go/common/v1"
	etav1 "github.com/keix40/omnifleet/gen/go/eta/v1"
	"github.com/keix40/omnifleet/services/eta/internal/server"
)

func TestComputeETA_RequiresDestination(t *testing.T) {
	dsn := os.Getenv("APP_DATABASE_URL")
	if dsn == "" {
		t.Skip("APP_DATABASE_URL not set")
	}
	svc, cleanup, err := server.New(context.Background(), dsn, os.Getenv("NATS_URL"), "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	_, err = svc.ComputeETA(context.Background(), &etav1.ComputeETARequest{
		TenantId:  "11111111-1111-1111-1111-111111111111",
		VehicleId: "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
	})
	if err == nil {
		t.Fatal("expected error without destination")
	}
	_, err = svc.ComputeETA(context.Background(), &etav1.ComputeETARequest{
		TenantId:  "11111111-1111-1111-1111-111111111111",
		VehicleId: "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
		Destination: &commonv1.GeoPoint{
			Latitude:  37.78,
			Longitude: -122.41,
		},
	})
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
}
