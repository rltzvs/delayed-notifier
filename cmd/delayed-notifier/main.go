package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"delayed-notifier/internal/config"
	httpHandlers "delayed-notifier/internal/controller/http"
	"delayed-notifier/internal/controller/http/middleware"
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

	// Repository and service
	notifyRepo := postgres.NewNotifyDBRepository(db.Pool)
	cacheRepo := redis.NewNotifyRedisRepository(redisClient, logg)
	notifierRepo := email.NewMailer(cfg.Mail)
	notifyService := service.NewNotifyService(notifyRepo, cacheRepo, producer, notifierRepo, logg)

	// worker
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
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

	// Router and middleware
	r := chi.NewRouter()
	r.Use(middleware.LoggingMiddleware(logg))

	notifyHandler := httpHandlers.NewNotifyHandler(notifyService, logg)
	r.Route("/notify", func(r chi.Router) {
		r.Post("/", notifyHandler.CreateNotify)
		r.Route("/{notifyID}", func(r chi.Router) {
			r.Get("/", notifyHandler.GetNotify)
			r.Delete("/", notifyHandler.DeleteNotify)
		})
	})

	// HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	logg.Info("server started", slog.String("addr", server.Addr))
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logg.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logg.Error("server shutdown failed", slog.Any("error", err))
	} else {
		logg.Info("server gracefully shutdown")
	}
}
