package service

import (
	"context"
	"delayed-notifier/internal/entity"
	"fmt"
	"log/slog"
	"time"
)

type NotifyDBRepository interface {
	CreateNotify(ctx context.Context, notify entity.Notify) (entity.Notify, error)
	GetNotify(ctx context.Context, notifyID string) (entity.Notify, error)
	DeleteNotify(ctx context.Context, notifyID string) error
	GetReadyNotifies(ctx context.Context) ([]entity.Notify, error)
	UpdateNotifyStatus(ctx context.Context, notifyID string, status string) error
}

type NotifyCacheRepository interface {
	SetNotify(ctx context.Context, notify entity.Notify, ttl time.Duration) error
	GetNotify(ctx context.Context, notifyID string) (entity.Notify, error)
	DeleteNotify(ctx context.Context, notifyID string) error
}

type NotifyProducer interface {
	Send(ctx context.Context, notify entity.Notify) error
}

type NotifyService struct {
	db       NotifyDBRepository
	cache    NotifyCacheRepository
	producer NotifyProducer
	logger   *slog.Logger
}

func NewNotifyService(db NotifyDBRepository, cache NotifyCacheRepository, producer NotifyProducer, logger *slog.Logger) *NotifyService {
	return &NotifyService{db: db, cache: cache, producer: producer, logger: logger}
}

func (s *NotifyService) CreateNotify(ctx context.Context, notify entity.Notify) (entity.Notify, error) {
	created, err := s.db.CreateNotify(ctx, notify)
	if err != nil {
		return entity.Notify{}, err
	}
	_ = s.cache.SetNotify(ctx, created, 24*time.Hour)
	return created, nil
}

func (s *NotifyService) GetNotify(ctx context.Context, notifyID string) (entity.Notify, error) {
	notify, err := s.cache.GetNotify(ctx, notifyID)
	if err == nil {
		return notify, nil
	}
	notify, err = s.db.GetNotify(ctx, notifyID)
	if err != nil {
		return entity.Notify{}, err
	}
	_ = s.cache.SetNotify(ctx, notify, 24*time.Hour)
	return notify, nil
}

func (s *NotifyService) DeleteNotify(ctx context.Context, notifyID string) error {
	_ = s.cache.DeleteNotify(ctx, notifyID)
	return s.db.DeleteNotify(ctx, notifyID)
}

func (s *NotifyService) UpdateNotifyStatus(ctx context.Context, notifyID string, status string) error {
	return s.db.UpdateNotifyStatus(ctx, notifyID, status)
}

func (s *NotifyService) ScheduleReadyNotifies(ctx context.Context) error {
	notifies, err := s.db.GetReadyNotifies(ctx)
	if err != nil {
		return fmt.Errorf("ScheduleReadyNotifies: get ready notifies: %w", err)
	}

	for _, notify := range notifies {
		if err := s.producer.Send(ctx, notify); err != nil {
			s.logger.Error("ScheduleReadyNotifies: failed to send notify", slog.String("ID", notify.ID), slog.Any("error", err))
			continue
		}

		if err := s.UpdateNotifyStatus(ctx, notify.ID, entity.StatusQueued); err != nil {
			return fmt.Errorf("ScheduleReadyNotifies: update status for ID=%s: %w", notify.ID, err)
		}
	}

	return nil
}
