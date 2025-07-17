package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"delayed-notifier/internal/config"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedisClient(config *config.Config) (*RedisClient, error) {
	redisCfg := config.Redis
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisCfg.Host, redisCfg.Port),
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &RedisClient{Client: client}, nil
}
