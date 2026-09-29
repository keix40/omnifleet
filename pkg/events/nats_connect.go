package events

import (
	"crypto/tls"
	"os"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
)

var (
	sharedMu     sync.Mutex
	sharedNC     *nats.Conn
	sharedActive bool
)

// SharedNATSMode returns true when services must reuse one NATS connection.
func SharedNATSMode() bool {
	return os.Getenv("OMNIFLEET_SHARED_NATS") == "1"
}

// ConnectNATSFromEnv connects using NATS_URL and optional NATS_CREDS / TLS settings.
func ConnectNATSFromEnv() (*nats.Conn, error) {
	if SharedNATSMode() {
		sharedMu.Lock()
		defer sharedMu.Unlock()
		if sharedNC != nil && sharedNC.IsConnected() {
			return sharedNC, nil
		}
		nc, err := dialNATS()
		if err != nil {
			return nil, err
		}
		sharedNC = nc
		sharedActive = true
		return nc, nil
	}
	return dialNATS()
}

func dialNATS() (*nats.Conn, error) {
	url := strings.TrimSpace(os.Getenv("NATS_URL"))
	if url == "" {
		url = nats.DefaultURL
	}
	opts, err := natsOptionsFromEnv()
	if err != nil {
		return nil, err
	}
	return nats.Connect(url, opts...)
}

func natsOptionsFromEnv() ([]nats.Option, error) {
	var opts []nats.Option
	opts = append(opts, nats.MaxReconnects(-1))

	if creds := strings.TrimSpace(os.Getenv("NATS_CREDS")); creds != "" {
		if strings.Contains(creds, "-----BEGIN NATS USER JWT-----") {
			path, err := writeTempCreds(creds)
			if err != nil {
				return nil, err
			}
			opts = append(opts, nats.UserCredentials(path))
		} else {
			opts = append(opts, nats.UserCredentials(creds))
		}
	}

	useTLS := strings.EqualFold(os.Getenv("NATS_TLS"), "true") ||
		strings.HasPrefix(strings.ToLower(os.Getenv("NATS_URL")), "tls://")
	if useTLS {
		opts = append(opts, nats.Secure(&tls.Config{MinVersion: tls.VersionTLS12}))
	}
	return opts, nil
}

func writeTempCreds(content string) (string, error) {
	f, err := os.CreateTemp("", "omnifleet-nats-*.creds")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// ReleaseNATS closes nc unless shared mode keeps a single connection open.
func ReleaseNATS(nc *nats.Conn) {
	if nc == nil {
		return
	}
	if SharedNATSMode() {
		return
	}
	nc.Close()
}

// CloseSharedNATS closes the shared connection (call on process shutdown).
func CloseSharedNATS() {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if sharedNC != nil {
		sharedNC.Close()
		sharedNC = nil
		sharedActive = false
	}
}

// SharedConnectionCountHint documents expected NATS connections for operators.
func SharedConnectionCountHint() string {
	if sharedActive {
		return "1 shared NATS connection (OMNIFLEET_SHARED_NATS=1)"
	}
	return "one NATS connection per service process"
}
