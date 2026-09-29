package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	geofencingv1 "github.com/keix40/omnifleet/gen/go/geofencing/v1"
	"github.com/keix40/omnifleet/pkg/db"
	"github.com/keix40/omnifleet/pkg/events"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Server struct {
	geofencingv1.UnimplementedGeofencingServiceServer
	pool      *pgxpool.Pool
	js        jetstream.JetStream
	publisher *events.Publisher
	nc        *nats.Conn
}

func New(ctx context.Context, dsn, natsURL string) (*Server, func(), error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, func() {}, err
	}
	pub, nc, err := events.ConnectPublisher(natsURL)
	if err != nil {
		pool.Close()
		return nil, func() {}, err
	}
	if err := pub.EnsureStream(ctx); err != nil {
		nc.Close()
		pool.Close()
		return nil, func() {}, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		pool.Close()
		return nil, func() {}, err
	}
	s := &Server{pool: pool, publisher: pub, nc: nc, js: js}
	cleanup := func() {
		nc.Close()
		pool.Close()
	}
	return s, cleanup, nil
}

func (s *Server) RunConsumer(ctx context.Context) error {
	_, err := s.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     events.StreamFleet,
		Subjects: []string{"fleet.>"},
	})
	if err != nil {
		return err
	}
	cons, err := s.js.CreateOrUpdateConsumer(ctx, events.StreamFleet, jetstream.ConsumerConfig{
		Durable:       "geofencing-positions",
		FilterSubject: "fleet.>",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}
	_, err = cons.Consume(func(msg jetstream.Msg) {
		if !strings.HasSuffix(msg.Subject(), ".positions") {
			_ = msg.Ack()
			return
		}
		var ev events.PositionEvent
		if err := json.Unmarshal(msg.Data(), &ev); err != nil {
			_ = msg.Term()
			return
		}
		if _, err := s.evaluateInternal(ctx, ev.TenantID, ev.VehicleID, ev.Latitude, ev.Longitude); err != nil {
			log.Printf("geofence evaluate: %v", err)
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})
	return err
}

func (s *Server) EvaluatePosition(ctx context.Context, req *geofencingv1.EvaluatePositionRequest) (*geofencingv1.EvaluatePositionResponse, error) {
	eventsOut, err := s.evaluateInternal(ctx, req.TenantId, req.VehicleId, req.Point.Latitude, req.Point.Longitude)
	if err != nil {
		return nil, err
	}
	var pbEvents []*geofencingv1.GeofenceEvent
	for _, e := range eventsOut {
		pbEvents = append(pbEvents, &geofencingv1.GeofenceEvent{
			GeofenceId:   e.GeofenceID,
			GeofenceName: e.GeofenceName,
			EventType:    e.EventType,
		})
	}
	return &geofencingv1.EvaluatePositionResponse{Events: pbEvents}, nil
}

type geofenceEval struct {
	GeofenceID   string
	GeofenceName string
	EventType    string
}

func (s *Server) evaluateInternal(ctx context.Context, tenantID, vehicleID string, lat, lon float64) ([]geofenceEval, error) {
	var out []geofenceEval
	err := db.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT g.id::text, g.name,
			       ST_Intersects(g.boundary, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS inside
			FROM geofences g
		`, lon, lat)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var gfID, name string
			var inside bool
			if err := rows.Scan(&gfID, &name, &inside); err != nil {
				return err
			}
			var wasInside bool
			err := tx.QueryRow(ctx, `
				SELECT inside FROM vehicle_geofence_state
				WHERE tenant_id = $1::uuid AND vehicle_id = $2::uuid AND geofence_id = $3::uuid
			`, tenantID, vehicleID, gfID).Scan(&wasInside)
			if err != nil && err != pgx.ErrNoRows {
				return err
			}
			if err == pgx.ErrNoRows {
				wasInside = false
			}

			if inside != wasInside {
				eventType := "enter"
				if !inside {
					eventType = "exit"
				}
				out = append(out, geofenceEval{GeofenceID: gfID, GeofenceName: name, EventType: eventType})
				_, err = tx.Exec(ctx, `
					INSERT INTO vehicle_geofence_state (tenant_id, vehicle_id, geofence_id, inside, updated_at)
					VALUES ($1::uuid, $2::uuid, $3::uuid, $4, now())
					ON CONFLICT (tenant_id, vehicle_id, geofence_id)
					DO UPDATE SET inside = EXCLUDED.inside, updated_at = now()
				`, tenantID, vehicleID, gfID, inside)
				if err != nil {
					return err
				}
				alert := events.AlertEvent{
					TenantID:     tenantID,
					VehicleID:    vehicleID,
					GeofenceID:   gfID,
					GeofenceName: name,
					EventType:    eventType,
					Message:      fmt.Sprintf("Vehicle %s %s geofence %s", vehicleID, eventType, name),
				}
				_ = s.publisher.PublishAlert(ctx, alert)
			}
		}
		return rows.Err()
	})
	return out, err
}

func (s *Server) Health(ctx context.Context, _ *geofencingv1.HealthRequest) (*geofencingv1.HealthResponse, error) {
	if err := s.pool.Ping(ctx); err != nil {
		return &geofencingv1.HealthResponse{Status: "degraded"}, nil
	}
	return &geofencingv1.HealthResponse{Status: "ok"}, nil
}
