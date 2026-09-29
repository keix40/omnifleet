package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"time"
)

func main() {
	gateway := env("GATEWAY_URL", "http://localhost:8080")
	email := env("DRIVER_EMAIL", "driver@acme.test")
	password := env("DRIVER_PASSWORD", "demo-password-change-me")
	vehicleID := env("VEHICLE_ID", "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")

	token := login(gateway, email, password)
	fmt.Printf("simulating GPS for vehicle %s via %s\n", vehicleID, gateway)

	// Path through Acme SF depot geofence (enter/exit demo).
	startLat, startLon := 37.7700, -122.4300
	endLat, endLon := 37.7900, -122.4100
	steps := 40
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		lat := startLat + (endLat-startLat)*t
		lon := startLon + (endLon-startLon)*t
		heading := math.Atan2(endLat-startLat, endLon-startLon) * 180 / math.Pi
		postPosition(gateway, token, vehicleID, lat, lon, 8.5, heading)
		time.Sleep(2 * time.Second)
	}
}

func login(base, email, password string) string {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp, err := http.Post(base+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("login failed: %s", string(data)))
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(data, &out)
	if out.AccessToken == "" {
		panic("missing access token")
	}
	return out.AccessToken
}

func postPosition(base, token, vehicleID string, lat, lon, speed, heading float64) {
	payload, _ := json.Marshal(map[string]any{
		"vehicle_id":  vehicleID,
		"latitude":    lat,
		"longitude":   lon,
		"speed_mps":   speed,
		"heading_deg": heading,
	})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/tracking/positions", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		panic(fmt.Sprintf("ingest failed: %s", string(b)))
	}
	fmt.Printf("position lat=%.4f lon=%.4f\n", lat, lon)
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
