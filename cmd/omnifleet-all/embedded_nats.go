package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

const (
	embeddedJetStreamMemory = 64 * 1024 * 1024  // 64 MiB
	embeddedJetStreamStore  = 256 * 1024 * 1024 // 256 MiB
)

type embeddedNATS struct {
	ns       *server.Server
	storeDir string
}

func startEmbeddedNATS() (*embeddedNATS, error) {
	storeDir, err := os.MkdirTemp("", "omnifleet-nats-js-*")
	if err != nil {
		return nil, err
	}
	opts := &server.Options{
		Host:                "127.0.0.1",
		Port:                server.RANDOM_PORT,
		JetStream:           true,
		StoreDir:            storeDir,
		JetStreamMaxMemory:  embeddedJetStreamMemory,
		JetStreamMaxStore:   embeddedJetStreamStore,
		NoSigs:              true,
		NoLog:               false,
		Debug:               false,
		MaxControlLine:      4096,
		MaxPayload:          1024 * 1024,
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		_ = os.RemoveAll(storeDir)
		return nil, err
	}
	ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		ns.Shutdown()
		_ = os.RemoveAll(storeDir)
		return nil, fmt.Errorf("embedded NATS not ready")
	}
	log.Printf("embedded NATS JetStream on %s (store=%s memory=%dMiB file=%dMiB)",
		ns.ClientURL(), storeDir, embeddedJetStreamMemory/(1024*1024), embeddedJetStreamStore/(1024*1024))
	log.Print("embedded NATS: events are ephemeral and lost on process restart")
	return &embeddedNATS{ns: ns, storeDir: storeDir}, nil
}

func (e *embeddedNATS) clientURL() string {
	if e == nil || e.ns == nil {
		return ""
	}
	return e.ns.ClientURL()
}

func (e *embeddedNATS) shutdown() {
	if e == nil || e.ns == nil {
		return
	}
	e.ns.Shutdown()
	if e.storeDir != "" {
		_ = os.RemoveAll(e.storeDir)
	}
}

func envBool(key string) bool {
	switch os.Getenv(key) {
	case "1", "true", "TRUE", "yes", "YES":
		return true
	default:
		return false
	}
}
