package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/divord97/ccc/pkg/metrics"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
)

// MessageHandler processes a single NATS message. Returning nil acknowledges
// the message; returning an error triggers NAK with backoff.
type MessageHandler func(ctx context.Context, subject string, data []byte) error

// Consumer subscribes to JetStream subjects and dispatches messages to a handler.
type Consumer struct {
	js      jetstream.JetStream
	logger  zerolog.Logger
	handler MessageHandler
}

// NewConsumer creates a consumer bound to the given client and handler.
func NewConsumer(client *Client, handler MessageHandler) *Consumer {
	return &Consumer{
		js:      client.js,
		logger:  client.logger,
		handler: handler,
	}
}

// Subscribe creates a durable consumer on the given stream and starts consuming
// messages. It blocks until ctx is cancelled.
func (c *Consumer) Subscribe(ctx context.Context, stream, durable, filterSubject string) error {
	cons, err := c.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:       durable,
		FilterSubject: filterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("nats: create consumer %s: %w", durable, err)
	}

	c.logger.Info().Str("durable", durable).Str("filter", filterSubject).Msg("nats: consumer started")

	iter, err := cons.Messages(jetstream.PullMaxMessages(10))
	if err != nil {
		return fmt.Errorf("nats: messages iterator: %w", err)
	}
	defer iter.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := iter.Next()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.Warn().Err(err).Msg("nats: consumer next")
			continue
		}

		if meta, err := msg.Metadata(); err == nil && meta.NumDelivered > 1 {
			metrics.NATSRedeliveries.WithLabelValues(msg.Subject()).Inc()
		}
		if err := c.handler(ctx, msg.Subject(), msg.Data()); err != nil {
			c.logger.Warn().Err(err).Str("subject", msg.Subject()).Msg("nats: handler error, nacking")
			_ = msg.Nak()
		} else {
			_ = msg.Ack()
		}
	}
}
