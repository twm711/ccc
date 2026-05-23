package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

// Producer wraps kafka-go writer for CDR event publishing.
type Producer struct {
	writer *kafka.Writer
	logger zerolog.Logger
}

type Config struct {
	Brokers []string
	Topic   string
	Logger  zerolog.Logger
}

func NewProducer(cfg Config) *Producer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Brokers...),
		Topic:    cfg.Topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &Producer{writer: w, logger: cfg.Logger}
}

// PublishCDR writes a call detail record event to Kafka.
func (p *Producer) PublishCDR(ctx context.Context, callID int64, event interface{}) error {
	b, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka: marshal: %w", err)
	}

	key := fmt.Sprintf("call-%d", callID)
	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: b,
	})
	if err != nil {
		return fmt.Errorf("kafka: write: %w", err)
	}

	p.logger.Debug().Int64("call_id", callID).Msg("CDR event published")
	return nil
}

func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
