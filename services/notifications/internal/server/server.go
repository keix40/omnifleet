package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	notificationsv1 "github.com/keix40/omnifleet/gen/go/notifications/v1"
	"github.com/keix40/omnifleet/pkg/db"
	"github.com/keix40/omnifleet/pkg/events"
	"github.com/keix40/omnifleet/services/notifications/internal/channels"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	"os"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	notificationsv1.UnimplementedNotificationsServiceServer
	pool *pgxpool.Pool
	js   jetstream.JetStream
}

func New(ctx context.Context, dsn, natsURL string) (*Server, func(), error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, func() {}, err
	}
	if natsURL != "" && os.Getenv("NATS_URL") == "" {
		_ = os.Setenv("NATS_URL", natsURL)
	}
	nc, err := events.ConnectNATSFromEnv()
	if err != nil {
		pool.Close()
		return nil, func() {}, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		events.ReleaseNATS(nc)
		pool.Close()
		return nil, func() {}, err
	}
	_, _ = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: events.StreamFleet, Subjects: []string{"fleet.>"},
	})
	s := &Server{pool: pool, js: js}
	go s.RunConsumer(context.Background())
	go s.RunRetryWorker(context.Background())
	cleanup := func() {
		events.ReleaseNATS(nc)
		pool.Close()
	}
	return s, cleanup, nil
}

func (s *Server) GetPreferences(ctx context.Context, req *notificationsv1.GetPreferencesRequest) (*notificationsv1.GetPreferencesResponse, error) {
	p, err := s.loadPrefs(ctx, req.TenantId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "preferences not found")
	}
	return &notificationsv1.GetPreferencesResponse{Preferences: p}, nil
}

func (s *Server) UpdatePreferences(ctx context.Context, req *notificationsv1.UpdatePreferencesRequest) (*notificationsv1.UpdatePreferencesResponse, error) {
	if req.Preferences == nil {
		return nil, status.Error(codes.InvalidArgument, "preferences required")
	}
	p := req.Preferences
	err := db.WithTenant(ctx, s.pool, p.TenantId, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO notification_preferences (tenant_id, alerts_enabled, dispatch_enabled, eta_enabled, channel, webhook_url, email_to, updated_at)
			VALUES ($1::uuid, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''), now())
			ON CONFLICT (tenant_id) DO UPDATE SET
			  alerts_enabled = EXCLUDED.alerts_enabled,
			  dispatch_enabled = EXCLUDED.dispatch_enabled,
			  eta_enabled = EXCLUDED.eta_enabled,
			  channel = EXCLUDED.channel,
			  webhook_url = EXCLUDED.webhook_url,
			  email_to = EXCLUDED.email_to,
			  updated_at = now()
		`, p.TenantId, p.AlertsEnabled, p.DispatchEnabled, p.EtaEnabled, p.Channel, p.WebhookUrl, p.EmailTo)
		return err
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "update failed")
	}
	out, _ := s.loadPrefs(ctx, p.TenantId)
	return &notificationsv1.UpdatePreferencesResponse{Preferences: out}, nil
}

func (s *Server) loadPrefs(ctx context.Context, tenantID string) (*notificationsv1.NotificationPreferences, error) {
	var p notificationsv1.NotificationPreferences
	p.TenantId = tenantID
	err := db.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT alerts_enabled, dispatch_enabled, eta_enabled, channel,
			       COALESCE(webhook_url,''), COALESCE(email_to,'')
			FROM notification_preferences WHERE tenant_id = $1::uuid
		`, tenantID).Scan(&p.AlertsEnabled, &p.DispatchEnabled, &p.EtaEnabled, &p.Channel, &p.WebhookUrl, &p.EmailTo)
	})
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Server) RunConsumer(ctx context.Context) {
	cons, err := s.js.CreateOrUpdateConsumer(ctx, events.StreamFleet, jetstream.ConsumerConfig{
		Durable:       "notifications-events",
		FilterSubject: "fleet.>",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		log.Printf("notifications consumer: %v", err)
		return
	}
	_, _ = cons.Consume(func(msg jetstream.Msg) {
		subj := msg.Subject()
		if !(strings.HasSuffix(subj, ".alerts") || strings.HasSuffix(subj, ".dispatch") || strings.HasSuffix(subj, ".eta")) {
			_ = msg.Ack()
			return
		}
		parts := strings.Split(subj, ".")
		if len(parts) < 2 {
			_ = msg.Ack()
			return
		}
		tenantID := parts[1]
		eventType := strings.TrimPrefix(subj, "fleet."+tenantID+".")
		if err := s.enqueueDelivery(context.Background(), tenantID, eventType, msg.Data()); err != nil {
			log.Printf("enqueue delivery: %v", err)
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})
	<-ctx.Done()
}

func (s *Server) enqueueDelivery(ctx context.Context, tenantID, eventType string, payload []byte) error {
	prefs, err := s.loadPrefs(ctx, tenantID)
	if err != nil {
		return err
	}
	if !s.enabledFor(prefs, eventType) {
		return nil
	}
	return db.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO notification_deliveries (tenant_id, event_type, channel, payload, status)
			VALUES ($1::uuid, $2, $3, $4::jsonb, 'pending')
		`, tenantID, eventType, prefs.Channel, payload)
		return err
	})
}

func (s *Server) enabledFor(p *notificationsv1.NotificationPreferences, eventType string) bool {
	switch eventType {
	case "alerts":
		return p.AlertsEnabled
	case "dispatch":
		return p.DispatchEnabled
	case "eta":
		return p.EtaEnabled
	default:
		return true
	}
}

func (s *Server) RunRetryWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processPending(ctx)
		}
	}
}

func (s *Server) processPending(ctx context.Context) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, tenant_id::text, event_type, channel, payload::text, attempts
		FROM notification_deliveries
		WHERE status = 'pending' AND attempts < 5
		ORDER BY created_at
		LIMIT 20
	`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, tenantID, eventType, channelName, payload string
		var attempts int
		if err := rows.Scan(&id, &tenantID, &eventType, &channelName, &payload, &attempts); err != nil {
			continue
		}
		prefs, err := s.loadPrefs(ctx, tenantID)
		if err != nil {
			continue
		}
		meta := map[string]string{
			"event_type":  eventType,
			"webhook_url": prefs.WebhookUrl,
		}
		deliverer := channels.ForName(channelName)
		msg := payload
		var pretty map[string]any
		if json.Unmarshal([]byte(payload), &pretty) == nil {
			msg = fmt.Sprintf("%s: %v", eventType, pretty)
		}
		err = deliverer.Deliver(ctx, prefs.EmailTo, msg, meta)
		if err != nil {
			s.markAttempt(ctx, tenantID, id, attempts+1, err)
			continue
		}
		s.markDelivered(ctx, tenantID, id)
	}
}

func (s *Server) markAttempt(ctx context.Context, tenantID, id string, attempts int, err error) {
	if attempts >= 5 {
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO notification_dlq (tenant_id, event_type, channel, payload, error)
			SELECT tenant_id, event_type, channel, payload, $2 FROM notification_deliveries WHERE id = $1::uuid
		`, id, err.Error())
		_, _ = s.pool.Exec(ctx, `UPDATE notification_deliveries SET status = 'dead', last_error = $2 WHERE id = $1::uuid`, id, err.Error())
		return
	}
	_, _ = s.pool.Exec(ctx, `
		UPDATE notification_deliveries SET attempts = $2, last_error = $3 WHERE id = $1::uuid
	`, id, attempts, err.Error())
}

func (s *Server) markDelivered(ctx context.Context, tenantID, id string) {
	_, _ = s.pool.Exec(ctx, `
		UPDATE notification_deliveries SET status = 'delivered', delivered_at = now() WHERE id = $1::uuid
	`, id)
}

func (s *Server) Health(ctx context.Context, _ *notificationsv1.HealthRequest) (*notificationsv1.HealthResponse, error) {
	if err := s.pool.Ping(ctx); err != nil {
		return &notificationsv1.HealthResponse{Status: "degraded", DefaultChannel: "log"}, nil
	}
	return &notificationsv1.HealthResponse{Status: "ok", DefaultChannel: "log"}, nil
}
