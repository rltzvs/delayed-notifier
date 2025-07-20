package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"delayed-notifier/internal/config"
	"delayed-notifier/internal/controller/consumer"
	"delayed-notifier/internal/logger"
	"delayed-notifier/internal/repository/email"
	"delayed-notifier/internal/repository/postgres"
	"delayed-notifier/internal/repository/producer"
	"delayed-notifier/internal/repository/redis"
	"delayed-notifier/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Config
	cfg, err := config.New()
	if err != nil {
		log.Fatal("config error: ", err) //nolint:gocritic
	}

	// Logger
	logg := logger.New(cfg.Logger.Level)
	logg.Info("logger initialized")

	// DB connection
	db, err := postgres.NewDbConnection(cfg)
	if err != nil {
		logg.Error("db connection error", slog.Any("error", err))
		os.Exit(1)
	}
	logg.Info("db connection initialized")

	redisClient, err := redis.NewRedisClient(cfg)
	if err != nil {
		logg.Error("redis connection error", slog.Any("error", err))
		os.Exit(1)
	}
	logg.Info("redis connection initialized")

	// Kafka producer
	producer := producer.NewNotifyProducer(cfg.Kafka.Host+":"+cfg.Kafka.Port, cfg.Kafka.Topic, logg)
	if err != nil {
		logg.Error("failed to initialize notify producer", slog.Any("error", err))
		os.Exit(1)
	}
	logg.Info("notify producer initialized")

	notifyRepo := postgres.NewNotifyDBRepository(db.Pool)
	cacheRepo := redis.NewNotifyRedisRepository(redisClient, logg)
	notifierRepo := email.NewMailer(cfg.Mail)
	notifyService := service.NewNotifyService(notifyRepo, cacheRepo, producer, notifierRepo, logg)

	kafkaConsumer := consumer.NewOrderConsumer(cfg.Kafka.Host+":"+cfg.Kafka.Port, cfg.Kafka.Topic, notifyService, logg)

	// worker
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := notifyService.ScheduleReadyNotifies(ctx); err != nil {
					logg.Error("schedule error", slog.Any("error", err))
				}
			case <-ctx.Done():
				logg.Info("scheduler stopped")
				return
			}
		}
	}()

	go func() {
		kafkaConsumer.Start(ctx)
	}()

	<-ctx.Done()
	logg.Info("shutdown signal received")

	_, cancel := context.WithTimeout(ctx, time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer cancel()

	logg.Info("server gracefully shutdown")
}
