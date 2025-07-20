package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"delayed-notifier/internal/controller"
	"delayed-notifier/internal/entity"
)

type OrderConsumer struct {
	reader    *kafka.Reader
	dlqWriter *kafka.Writer
	service   controller.NotifyService
	logger    *slog.Logger
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

	dlqWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{brokers},
		Topic:        topic + "-dlq",
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: 1,
		Async:        false,
		BatchTimeout: 50 * time.Millisecond,
		MaxAttempts:  3,
	})

	return &OrderConsumer{
		reader:    reader,
		dlqWriter: dlqWriter,
		service:   service,
		logger:    logger,
	}
}

func (c *OrderConsumer) Start(ctx context.Context) {
	defer func() {
		if err := c.reader.Close(); err != nil {
			c.logger.Error("failed to close notify consumer", slog.Any("error", err))
		} else {
			c.logger.Info("notify consumer closed")
		}
		if err := c.dlqWriter.Close(); err != nil {
			c.logger.Error("failed to close dlq writer", slog.Any("error", err))
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
			// Отправляем невалидное сообщение в DLQ
			if err := c.sendToDLQ(ctx, m, "invalid_json", err.Error()); err != nil {
				c.logger.Error("failed to send invalid message to DLQ", slog.Any("error", err))
			}
			// Коммитим сообщение с невалидным JSON
			if err := c.reader.CommitMessages(ctx, m); err != nil {
				c.logger.Error("failed to commit invalid message offset", slog.Any("error", err))
			}
			continue
		}

		c.logger.Debug("handling notify message", slog.Any("message", notify))

		if err := c.service.ProcessNotify(ctx, notify); err != nil {
			c.logger.Error("failed to process notify message",
				slog.String("notify_id", notify.ID),
				slog.Any("error", err),
			)
			// Отправляем неудачное сообщение в DLQ
			if err := c.sendToDLQ(ctx, m, "processing_failed", err.Error()); err != nil {
				c.logger.Error("failed to send failed message to DLQ", slog.Any("error", err))
			}
			// Коммитим сообщение даже при ошибке
			if err := c.reader.CommitMessages(ctx, m); err != nil {
				c.logger.Error("failed to commit failed message offset", slog.Any("error", err))
			}
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

func (c *OrderConsumer) sendToDLQ(ctx context.Context, msg kafka.Message, reason, errorMsg string) error {
	dlqMessage := struct {
		OriginalMessage kafka.Message `json:"original_message"`
		Reason          string        `json:"reason"`
		ErrorMessage    string        `json:"error_message"`
		Timestamp       time.Time     `json:"timestamp"`
	}{
		OriginalMessage: msg,
		Reason:          reason,
		ErrorMessage:    errorMsg,
		Timestamp:       time.Now(),
	}

	dlqData, err := json.Marshal(dlqMessage)
	if err != nil {
		return err
	}

	return c.dlqWriter.WriteMessages(ctx, kafka.Message{
		Key:   msg.Key,
		Value: dlqData,
	})
}
