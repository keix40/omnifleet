package channels

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type Deliverer interface {
	Name() string
	Deliver(ctx context.Context, to, message string, meta map[string]string) error
}

type LogChannel struct{}

func (LogChannel) Name() string { return "log" }

func (LogChannel) Deliver(_ context.Context, to, message string, meta map[string]string) error {
	log.Printf("NOTIFY channel=log to=%s type=%s msg=%s", to, meta["event_type"], message)
	return nil
}

type WebhookChannel struct {
	Client *http.Client
}

func (WebhookChannel) Name() string { return "webhook" }

func (w WebhookChannel) Deliver(ctx context.Context, _, message string, meta map[string]string) error {
	url := meta["webhook_url"]
	if url == "" {
		return fmt.Errorf("webhook_url missing")
	}
	client := w.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(message))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return nil
}

type SMTPChannel struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

func (SMTPChannel) Name() string { return "email" }

func (s SMTPChannel) Deliver(_ context.Context, to, message string, meta map[string]string) error {
	if to == "" {
		return fmt.Errorf("email_to missing")
	}
	host := s.Host
	if host == "" {
		host = os.Getenv("SMTP_HOST")
	}
	port := s.Port
	if port == "" {
		port = os.Getenv("SMTP_PORT")
	}
	if port == "" {
		port = "587"
	}
	user := s.User
	if user == "" {
		user = os.Getenv("SMTP_USER")
	}
	pass := s.Pass
	if pass == "" {
		pass = os.Getenv("SMTP_PASSWORD")
	}
	from := s.From
	if from == "" {
		from = os.Getenv("SMTP_FROM")
	}
	if from == "" {
		from = "notifications@omnifleet.local"
	}
	if host == "" {
		return fmt.Errorf("smtp not configured")
	}
	addr := host + ":" + port
	body := []byte("Subject: OmniFleet " + meta["event_type"] + "\r\n\r\n" + message)
	auth := smtp.PlainAuth("", user, pass, host)
	return smtp.SendMail(addr, auth, from, []string{to}, body)
}

func ForName(name string) Deliverer {
	switch name {
	case "webhook":
		return WebhookChannel{}
	case "email":
		return SMTPChannel{}
	default:
		return LogChannel{}
	}
}
