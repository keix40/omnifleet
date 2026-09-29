package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keix40/omnifleet/pkg/auth"
	"github.com/keix40/omnifleet/pkg/events"
	authpub "github.com/keix40/omnifleet/services/auth/public"
	billingpub "github.com/keix40/omnifleet/services/billing/public"
	dispatchpub "github.com/keix40/omnifleet/services/dispatch/public"
	etapub "github.com/keix40/omnifleet/services/eta/public"
	geopub "github.com/keix40/omnifleet/services/geofencing/public"
	gatewaypub "github.com/keix40/omnifleet/services/gateway/public"
	notifpub "github.com/keix40/omnifleet/services/notifications/public"
	trackpub "github.com/keix40/omnifleet/services/tracking/public"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	_ = os.Setenv("OMNIFLEET_SHARED_NATS", "1")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dsn := env("DATABASE_URL", "")
	if dsn == "" {
		log.Fatal("DATABASE_URL required")
	}
	natsURL := env("NATS_URL", "nats://localhost:4222")
	_ = os.Setenv("NATS_URL", natsURL)

	jwtSettings, err := auth.LoadJWTSettingsFromEnv()
	if err != nil {
		log.Fatalf("jwt: %v", err)
	}

	authSvc, err := authpub.New(dsn, jwtSettings, 24*time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	defer authpub.Close(authSvc)

	trackSvc, trackCleanup, err := trackpub.New(dsn, natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer trackCleanup()

	geoSvc, geoCleanup, err := geopub.New(ctx, dsn, natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer geoCleanup()

	etaSvc, etaCleanup, err := etapub.New(ctx, dsn, natsURL, env("OSRM_BASE_URL", ""))
	if err != nil {
		log.Fatal(err)
	}
	defer etaCleanup()

	dispatchSvc, dispatchCleanup, err := dispatchpub.New(ctx, dsn, natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer dispatchCleanup()

	billingSvc, billingCleanup, err := billingpub.New(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer billingCleanup()

	notifSvc, notifCleanup, err := notifpub.New(ctx, dsn, natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer notifCleanup()

	authAddr := mustServeGRPC(ctx, func(gs *grpc.Server) { authpub.Register(gs, authSvc) })
	trackAddr := mustServeGRPC(ctx, func(gs *grpc.Server) { trackpub.Register(gs, trackSvc) })
	_ = mustServeGRPC(ctx, func(gs *grpc.Server) {
		geopub.Register(gs, geoSvc)
		registerHealth(gs)
	})
	etaAddr := mustServeGRPC(ctx, func(gs *grpc.Server) { etapub.Register(gs, etaSvc) })
	dispatchAddr := mustServeGRPC(ctx, func(gs *grpc.Server) { dispatchpub.Register(gs, dispatchSvc) })
	billingAddr := mustServeGRPC(ctx, func(gs *grpc.Server) { billingpub.Register(gs, billingSvc) })
	notifAddr := mustServeGRPC(ctx, func(gs *grpc.Server) { notifpub.Register(gs, notifSvc) })

	go func() {
		if err := geopub.RunConsumer(ctx, geoSvc); err != nil && ctx.Err() == nil {
			log.Printf("geofencing consumer: %v", err)
		}
	}()
	go geopub.RunOutboxRelay(ctx, geoSvc)

	httpSrv, httpCleanup, err := gatewaypub.NewServer(ctx, gatewaypub.Config{
		AuthAddr:          authAddr,
		TrackingAddr:      trackAddr,
		ETAAddr:           etaAddr,
		DispatchAddr:      dispatchAddr,
		BillingAddr:       billingAddr,
		NotificationsAddr: notifAddr,
		NatsURL:           natsURL,
		DatabaseURL:       dsn,
		JWT:               jwtSettings,
		ReplicaID:         env("HOSTNAME", "omnifleet-all"),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer httpCleanup()

	port := env("PORT", env("GATEWAY_HTTP_ADDR", "8080"))
	if len(port) > 0 && port[0] == ':' {
		port = port[1:]
	}
	addr := "0.0.0.0:" + port

	server := &http.Server{
		Addr:              addr,
		Handler:           httpSrv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("omnifleet-all listening on %s (loopback gRPC)", addr)
		log.Printf("NATS: %s", events.SharedConnectionCountHint())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	events.CloseSharedNATS()
}

func mustServeGRPC(ctx context.Context, register func(*grpc.Server)) string {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	gs := grpc.NewServer()
	register(gs)
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Printf("grpc serve: %v", err)
		}
	}()
	go func() {
		<-ctx.Done()
		gs.GracefulStop()
	}()
	return lis.Addr().String()
}

func registerHealth(gs *grpc.Server) {
	hs := health.NewServer()
	healthpb.RegisterHealthServer(gs, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
