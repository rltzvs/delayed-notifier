package consumer

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"

	"delayed-notifier/internal/controller"
	"delayed-notifier/internal/entity"
)

type OrderConsumer struct {
	reader  *kafka.Reader
	service controller.NotifyService
	logger  *slog.Logger
}

func NewOrderConsumer(brokers, topic string, service controller.NotifyService, logger *slog.Logger) *OrderConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{brokers},
		Topic:          topic,
		GroupID:        "notify-worker-group",
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: 0,
	})
	return &OrderConsumer{
		reader:  reader,
		service: service,
		logger:  logger,
	}
}

func (c *OrderConsumer) Start(ctx context.Context) {
	defer func() {
		if err := c.reader.Close(); err != nil {
			c.logger.Error("failed to close notify consumer", slog.Any("error", err))
		} else {
			c.logger.Info("notify consumer closed")
		}
	}()

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.logger.Info("consumer context cancelled, exiting")
				return
			}
			c.logger.Error("failed to fetch notify message", slog.Any("error", err))
			continue
		}

		c.logger.Info("received message",
			slog.String("topic", m.Topic),
			slog.Int("partition", m.Partition),
			slog.Int64("offset", m.Offset),
		)

		var notify entity.Notify
		if err := json.Unmarshal(m.Value, &notify); err != nil {
			c.logger.Warn("invalid json message",
				slog.Any("error", err),
				slog.Int64("offset", m.Offset),
			)
			continue
		}

		c.logger.Debug("handling notify message", slog.Any("message", notify))

		if err := c.service.ProcessNotify(ctx, notify); err != nil {
			c.logger.Error("failed to set process notify message",
				slog.String("notify_id", notify.ID),
				slog.Any("error", err),
			)
			continue
		}

		c.logger.Info("successfully sent notify",
			slog.String("notify_id", notify.ID),
		)

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			c.logger.Error("failed to commit notify offset",
				slog.Any("error", err),
				slog.Int64("offset", m.Offset),
			)
		} else {
			c.logger.Debug("committed notify offset", slog.Int64("offset", m.Offset))
		}
	}
}
