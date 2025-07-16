package service

import (
	"context"
	"delayed-notifier/internal/entity"
	"time"
)

type NotifyDBRepository interface {
	CreateNotify(ctx context.Context, notify entity.Notify) (entity.Notify, error)
	GetNotify(ctx context.Context, notifyID string) (entity.Notify, error)
	DeleteNotify(ctx context.Context, notifyID string) error
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
}

func NewNotifyService(db NotifyDBRepository, cache NotifyCacheRepository, producer NotifyProducer) *NotifyService {
	return &NotifyService{db: db, cache: cache, producer: producer}
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
