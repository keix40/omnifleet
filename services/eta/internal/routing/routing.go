package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/keix40/omnifleet/pkg/geo"
)

type Point struct {
	Lat, Lon float64
}

type Result struct {
	DistanceMeters  float64
	DurationSeconds float64
	Provider        string
}

type Provider interface {
	Route(ctx context.Context, from, to Point) (Result, error)
}

const defaultRoadFactor = 1.35

type HaversineProvider struct {
	RoadFactor   float64
	SpeedMetersS float64
}

func (h HaversineProvider) Route(_ context.Context, from, to Point) (Result, error) {
	factor := h.RoadFactor
	if factor <= 0 {
		factor = defaultRoadFactor
	}
	speed := h.SpeedMetersS
	if speed <= 0 {
		speed = 8.0
	}
	dist := geo.HaversineMeters(geo.Point{Lat: from.Lat, Lon: from.Lon}, geo.Point{Lat: to.Lat, Lon: to.Lon})
	roadDist := dist * factor
	return Result{
		DistanceMeters:  roadDist,
		DurationSeconds: roadDist / speed,
		Provider:        "haversine",
	}, nil
}

type OSRMProvider struct {
	BaseURL string
	Client  *http.Client
}

func (o OSRMProvider) Route(ctx context.Context, from, to Point) (Result, error) {
	base := o.BaseURL
	if base == "" {
		base = "http://router.project-osrm.org"
	}
	client := o.Client
	if client == nil {
		client = http.DefaultClient
	}
	u := fmt.Sprintf("%s/route/v1/driving/%f,%f;%f,%f?overview=false", base, from.Lon, from.Lat, to.Lon, to.Lat)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Result{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("osrm status %d", resp.StatusCode)
	}
	var parsed struct {
		Routes []struct {
			Distance float64 `json:"distance"`
			Duration float64 `json:"duration"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Result{}, err
	}
	if len(parsed.Routes) == 0 {
		return Result{}, fmt.Errorf("osrm: no routes")
	}
	return Result{
		DistanceMeters:  parsed.Routes[0].Distance,
		DurationSeconds: parsed.Routes[0].Duration,
		Provider:        "osrm",
	}, nil
}

func NewFromEnv(osrmURL string) Provider {
	if u := osrmURL; u != "" {
		return OSRMProvider{BaseURL: u, Client: &http.Client{Timeout: 10 * time.Second}}
	}
	return HaversineProvider{RoadFactor: defaultRoadFactor, SpeedMetersS: 8}
}

// ParseOSRMURL returns trimmed URL or empty.
func ParseOSRMURL(raw string) string {
	if raw == "" {
		return ""
	}
	if _, err := url.Parse(raw); err != nil {
		return ""
	}
	return raw
}
