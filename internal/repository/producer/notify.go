package producer

import (
	"context"
	"delayed-notifier/internal/entity"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

type NotifyProducer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

func NewNotifyProducer(brokerURL string, topic string) *NotifyProducer {
	return &NotifyProducer{
		writer: kafka.NewWriter(kafka.WriterConfig{
			Brokers:      []string{brokerURL},
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: 1,
			Async:        false,
			BatchTimeout: 10 * time.Millisecond,
		}),
	}
}

func (p *NotifyProducer) Send(ctx context.Context, notify entity.Notify) error {
	msg, err := json.Marshal(notify)
	if err != nil {
		p.logger.Error("failed to marshal notify", slog.Any("error", err))
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(notify.ID),
		Value: msg,
	})
}

func (p *NotifyProducer) Close() error {
	return p.writer.Close()
}
