package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Publisher struct {
	js jetstream.JetStream
}

func ConnectPublisher(natsURL string) (*Publisher, *nats.Conn, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, nil, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, err
	}
	return &Publisher{js: js}, nc, nil
}

func (p *Publisher) EnsureStream(ctx context.Context) error {
	_, err := p.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     StreamFleet,
		Subjects: []string{"fleet.>"},
		Storage:  jetstream.FileStorage,
	})
	return err
}

func (p *Publisher) PublishPosition(ctx context.Context, ev PositionEvent) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	subj := PositionSubject(ev.TenantID)
	_, err = p.js.Publish(ctx, subj, data)
	return err
}

func (p *Publisher) PublishAlert(ctx context.Context, ev AlertEvent) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	subj := AlertSubject(ev.TenantID)
	_, err = p.js.Publish(ctx, subj, data)
	return err
}

func SubscribeTenant(ctx context.Context, js jetstream.JetStream, tenantID string, handler func(PositionEvent) error) (jetstream.ConsumeContext, error) {
	subj := PositionSubject(tenantID)
	cons, err := js.CreateOrUpdateConsumer(ctx, StreamFleet, jetstream.ConsumerConfig{
		FilterSubject: subj,
		Durable:       fmt.Sprintf("gateway-%s", tenantID),
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return nil, err
	}
	return cons.Consume(func(msg jetstream.Msg) {
		var ev PositionEvent
		if err := json.Unmarshal(msg.Data(), &ev); err != nil {
			_ = msg.Term()
			return
		}
		if err := handler(ev); err != nil {
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})
}
