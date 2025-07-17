package controller

import (
	"context"

	"delayed-notifier/internal/entity"
)

type NotifyService interface {
	CreateNotify(ctx context.Context, notify entity.Notify) (entity.Notify, error)
	GetNotify(ctx context.Context, notifyID string) (entity.Notify, error)
	DeleteNotify(ctx context.Context, notifyID string) error
	UpdateNotifyStatus(ctx context.Context, notifyID, status string) error
}
