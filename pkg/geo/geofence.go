package geo

import "math"

// Point is WGS84 coordinates in degrees.
type Point struct {
	Lat float64
	Lon float64
}

// Polygon is a simple closed ring (first point need not repeat last).
type Polygon []Point

// Contains uses ray-casting for point-in-polygon (planar approximation; demo slice).
// Production would use PostGIS ST_Contains on geography types.
func (poly Polygon) Contains(p Point) bool {
	if len(poly) < 3 {
		return false
	}
	inside := false
	j := len(poly) - 1
	for i := 0; i < len(poly); i++ {
		xi, yi := poly[i].Lon, poly[i].Lat
		xj, yj := poly[j].Lon, poly[j].Lat
		intersect := ((yi > p.Lat) != (yj > p.Lat)) &&
			(p.Lon < (xj-xi)*(p.Lat-yi)/(yj-yi+1e-15)+xi)
		if intersect {
			inside = !inside
		}
		j = i
	}
	return inside
}

// HaversineMeters returns distance between two points in meters.
func HaversineMeters(a, b Point) float64 {
	const earthRadius = 6371000.0
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadius * math.Asin(math.Sqrt(h))
}

// Transition detects enter/exit when crossing geofence boundary.
type Transition struct {
	GeofenceID string
	EventType  string // enter | exit
}

func DetectTransitions(geofenceID string, poly Polygon, wasInside, nowInside bool) []Transition {
	var out []Transition
	if !wasInside && nowInside {
		out = append(out, Transition{GeofenceID: geofenceID, EventType: "enter"})
	}
	if wasInside && !nowInside {
		out = append(out, Transition{GeofenceID: geofenceID, EventType: "exit"})
	}
	return out
}
