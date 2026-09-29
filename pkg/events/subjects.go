package events

import "fmt"

const (
	StreamFleet = "FLEET"
)

// PositionSubject is tenant-scoped NATS subject for live GPS.
func PositionSubject(tenantID string) string {
	return fmt.Sprintf("fleet.%s.positions", tenantID)
}

// AlertSubject is tenant-scoped geofence / dispatch alerts.
func AlertSubject(tenantID string) string {
	return fmt.Sprintf("fleet.%s.alerts", tenantID)
}

type PositionEvent struct {
	TenantID   string  `json:"tenant_id"`
	VehicleID  string  `json:"vehicle_id"`
	DriverID   string  `json:"driver_id"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	SpeedMPS   float64 `json:"speed_mps"`
	HeadingDeg float64 `json:"heading_deg"`
	RecordedAt int64   `json:"recorded_at_unix"`
}

type AlertEvent struct {
	TenantID     string `json:"tenant_id"`
	VehicleID    string `json:"vehicle_id"`
	GeofenceID   string `json:"geofence_id"`
	GeofenceName string `json:"geofence_name"`
	EventType    string `json:"event_type"`
	Message      string `json:"message"`
}
