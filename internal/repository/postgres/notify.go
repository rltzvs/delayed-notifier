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

func (r *NotifyDBRepository) GetReadyNotifies(ctx context.Context) ([]entity.Notify, error) {
	query := `
		SELECT id, send_at, message, status
		FROM notify
		WHERE send_at <= NOW() AND status = $1
	`

	rows, err := r.Pool.Query(ctx, query, entity.StatusScheduled)
	if err != nil {
		return nil, fmt.Errorf("GetReadyNotifies query: %w", err)
	}
	defer rows.Close()

	var notifies []entity.Notify
	for rows.Next() {
		var notify entity.Notify
		if err := rows.Scan(
			&notify.ID,
			&notify.SendAt,
			&notify.Message,
			&notify.Status,
		); err != nil {
			return nil, fmt.Errorf("GetReadyNotifies scan: %w", err)
		}
		notifies = append(notifies, notify)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetReadyNotifies iteration: %w", err)
	}

	return notifies, nil
}

func (r *NotifyDBRepository) UpdateNotifyStatus(ctx context.Context, notifyID string, status string) error {
	query := `
		UPDATE notify
		SET status = $1
		WHERE id = $2
	`

	cmdTag, err := r.Pool.Exec(ctx, query, status, notifyID)
	if err != nil {
		return fmt.Errorf("UpdateNotifyStatus: exec: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("UpdateNotifyStatus: no rows affected for ID=%s", notifyID)
	}

	return nil
}
