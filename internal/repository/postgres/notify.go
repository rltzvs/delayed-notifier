package postgres

import (
	"context"
	"delayed-notifier/internal/entity"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotifyDBRepository struct {
	Pool *pgxpool.Pool
}

func NewNotifyDBRepository(pool *pgxpool.Pool) *NotifyDBRepository {
	return &NotifyDBRepository{Pool: pool}
}

func (r *NotifyDBRepository) CreateNotify(ctx context.Context, notify entity.Notify) (entity.Notify, error) {
	query := `
		INSERT INTO notify (send_at, message, status)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id string
	err := r.Pool.QueryRow(ctx, query, notify.SendAt, notify.Message, notify.Status).Scan(&id)
	if err != nil {
		return entity.Notify{}, fmt.Errorf("CreateNotify: %w", err)
	}

	notify.ID = id
	return notify, nil
}

func (r *NotifyDBRepository) GetNotify(ctx context.Context, notifyID string) (entity.Notify, error) {
	query := `
		SELECT id, send_at, message, status
		FROM notify
		WHERE id = $1
	`

	var notify entity.Notify
	err := r.Pool.QueryRow(ctx, query, notifyID).Scan(
		&notify.ID,
		&notify.SendAt,
		&notify.Message,
		&notify.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Notify{}, fmt.Errorf("GetNotify: %w", entity.ErrNotifyNotFound)
		}
		return entity.Notify{}, fmt.Errorf("GetNotify: %w", err)
	}

	return notify, nil
}

func (r *NotifyDBRepository) DeleteNotify(ctx context.Context, notifyID string) error {
	query := `
		DELETE FROM notify
		WHERE id = $1
	`

	_, err := r.Pool.Exec(ctx, query, notifyID)
	if err != nil {
		return fmt.Errorf("DeleteNotify: %w", err)
	}

	return nil
}
