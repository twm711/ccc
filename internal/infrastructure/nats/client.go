package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
)

// Client wraps NATS connection with JetStream for event publishing.
type Client struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	logger zerolog.Logger
}

type Config struct {
	URL    string
	Logger zerolog.Logger
}

func NewClient(cfg Config) (*Client, error) {
	nc, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("nats: connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats: jetstream: %w", err)
	}

	return &Client{nc: nc, js: js, logger: cfg.Logger}, nil
}

// EnsureStream creates or updates a JetStream stream.
func (c *Client) EnsureStream(ctx context.Context, name string, subjects []string) error {
	_, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     name,
		Subjects: subjects,
	})
	return err
}

// Publish publishes an event to a NATS subject.
func (c *Client) Publish(ctx context.Context, subject string, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("nats: marshal: %w", err)
	}

	_, err = c.js.Publish(ctx, subject, b)
	if err != nil {
		return fmt.Errorf("nats: publish %s: %w", subject, err)
	}

	c.logger.Debug().Str("subject", subject).Msg("event published")
	return nil
}

func (c *Client) Close() {
	if c.nc != nil {
		c.nc.Close()
	}
}
