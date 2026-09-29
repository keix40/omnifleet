package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keix40/omnifleet/services/gateway/internal/httpapi"
)

func main() {
	addr := env("GATEWAY_HTTP_ADDR", "0.0.0.0:8080")
	authAddr := env("AUTH_GRPC_ADDR", "localhost:50051")
	trackingAddr := env("TRACKING_GRPC_ADDR", "localhost:50052")
	natsURL := env("NATS_URL", "nats://localhost:4222")
	jwtSecret := env("JWT_SECRET", "dev-only-change-me")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv, cleanup, err := httpapi.NewServer(ctx, httpapi.Config{
		AuthAddr:     authAddr,
		TrackingAddr: trackingAddr,
		NatsURL:      natsURL,
		JWTSecret:    jwtSecret,
	})
	if err != nil {
		log.Fatalf("gateway: %v", err)
	}
	defer cleanup()

	server := &http.Server{
		Addr:              addr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("gateway HTTP listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
