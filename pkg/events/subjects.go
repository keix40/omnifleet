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

// DispatchSubject is tenant-scoped dispatch job lifecycle events.
func DispatchSubject(tenantID string) string {
	return fmt.Sprintf("fleet.%s.dispatch", tenantID)
}

// ETASubject is tenant-scoped ETA update events.
func ETASubject(tenantID string) string {
	return fmt.Sprintf("fleet.%s.eta", tenantID)
}

// NotificationSubject is tenant-scoped notification delivery audit (optional fan-out).
func NotificationSubject(tenantID string) string {
	return fmt.Sprintf("fleet.%s.notifications", tenantID)
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

type DispatchEvent struct {
	TenantID  string `json:"tenant_id"`
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	VehicleID string `json:"vehicle_id,omitempty"`
	DriverID  string `json:"driver_id,omitempty"`
	Message   string `json:"message"`
}

type ETAEvent struct {
	TenantID         string  `json:"tenant_id"`
	VehicleID        string  `json:"vehicle_id"`
	DistanceMeters   float64 `json:"distance_meters"`
	DurationSeconds  float64 `json:"duration_seconds"`
	ETAUnix          int64   `json:"eta_unix"`
	RoutingProvider  string  `json:"routing_provider"`
	DestLatitude     float64 `json:"dest_latitude"`
	DestLongitude    float64 `json:"dest_longitude"`
}

type NotificationEvent struct {
	TenantID  string `json:"tenant_id"`
	Channel   string `json:"channel"`
	EventType string `json:"event_type"`
	Message   string `json:"message"`
}
