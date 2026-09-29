package main

import (
	"testing"

	"github.com/keix40/omnifleet/pkg/events"
)

func TestEmbeddedNATS_StartAndConnect(t *testing.T) {
	t.Setenv("OMNIFLEET_SHARED_NATS", "1")
	emb, err := startEmbeddedNATS()
	if err != nil {
		t.Fatal(err)
	}
	defer emb.shutdown()

	t.Setenv("NATS_URL", emb.clientURL())
	nc, err := events.ConnectNATSFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	defer events.ReleaseNATS(nc)
	if !nc.IsConnected() {
		t.Fatal("not connected")
	}
}

func TestEnvBool(t *testing.T) {
	t.Setenv("NATS_EMBEDDED", "1")
	if !envBool("NATS_EMBEDDED") {
		t.Fatal("expected true")
	}
}
