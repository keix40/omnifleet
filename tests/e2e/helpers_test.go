package e2e_test

import (
	"os"
	"os/exec"
	"testing"
)

func resetGeofenceState(t *testing.T) {
	t.Helper()
	if os.Getenv("E2E_COMPOSE") != "1" {
		return
	}
	cmd := exec.Command("docker", "compose", "exec", "-T", "postgres",
		"psql", "-U", "omnifleet", "-d", "omnifleet", "-c", "DELETE FROM vehicle_geofence_state;")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Logf("reset geofence state: %v %s", err, out)
	}
}
