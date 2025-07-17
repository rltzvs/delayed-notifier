package redis

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"delayed-notifier/internal/entity"
)

type NotifyRedisRepository struct {
	client *RedisClient
	logger *slog.Logger
}

func NewNotifyRedisRepository(client *RedisClient, logger *slog.Logger) *NotifyRedisRepository {
	return &NotifyRedisRepository{
		client: client,
		logger: logger,
	}
}

func (r *NotifyRedisRepository) SetNotify(ctx context.Context, notify entity.Notify, ttl time.Duration) error {
	data, err := json.Marshal(notify)
	if err != nil {
		r.logger.Error("failed to marshal notify", slog.Any("error", err))
		return err
	}
	if err := r.client.Client.Set(ctx, notify.ID, data, ttl).Err(); err != nil {
		r.logger.Error("failed to set notify in redis", slog.Any("error", err))
		return err
	}
	r.logger.Info("notify cached in redis", slog.String("id", notify.ID))
	return nil
}

func (r *NotifyRedisRepository) GetNotify(ctx context.Context, id string) (entity.Notify, error) {
	val, err := r.client.Client.Get(ctx, id).Result()
	if err != nil {
		r.logger.Error("failed to get notify from redis", slog.Any("error", err))
		return entity.Notify{}, err
	}
	var notify entity.Notify
	if err := json.Unmarshal([]byte(val), &notify); err != nil {
		r.logger.Error("failed to unmarshal notify from redis", slog.Any("error", err))
		return entity.Notify{}, err
	}
	return notify, nil
}

func (r *NotifyRedisRepository) DeleteNotify(ctx context.Context, id string) error {
	if err := r.client.Client.Del(ctx, id).Err(); err != nil {
		r.logger.Error("failed to delete notify from redis", slog.Any("error", err))
		return err
	}
	r.logger.Info("notify deleted from redis", slog.String("id", id))
	return nil
}
