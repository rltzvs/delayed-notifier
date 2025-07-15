package service

import (
	"context"
	"delayed-notifier/internal/entity"
)

type NotifyDBRepository interface {
	CreateNotify(ctx context.Context, notify entity.Notify) (entity.Notify, error)
	GetNotify(ctx context.Context, notifyID string) (entity.Notify, error)
	DeleteNotify(ctx context.Context, notifyID string) error
}

type NotifyService struct {
	db NotifyDBRepository
}

func NewNotifyService(db NotifyDBRepository) *NotifyService {
	return &NotifyService{db: db}
}

func (s *NotifyService) CreateNotify(ctx context.Context, notify entity.Notify) (entity.Notify, error) {
	return s.db.CreateNotify(ctx, notify)
}

func (s *NotifyService) GetNotify(ctx context.Context, notifyID string) (entity.Notify, error) {
	return s.db.GetNotify(ctx, notifyID)
}

func (s *NotifyService) DeleteNotify(ctx context.Context, notifyID string) error {
	return s.db.DeleteNotify(ctx, notifyID)
}
