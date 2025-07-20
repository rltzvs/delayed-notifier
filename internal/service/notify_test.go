package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"delayed-notifier/internal/entity"
	mock_email "delayed-notifier/internal/repository/email/mocks"
	mock_db "delayed-notifier/internal/repository/postgres/mocks"
	mock_producer "delayed-notifier/internal/repository/producer/mocks"
	mock_cache "delayed-notifier/internal/repository/redis/mocks"
)

func setupTestService(t *testing.T) (context.Context, *mock_db.NotifyDBRepository, *mock_cache.NotifyCacheRepository, *mock_producer.NotifyProducer, *NotifyService) {
	t.Helper()

	ctx := context.Background()

	db := new(mock_db.NotifyDBRepository)
	cache := new(mock_cache.NotifyCacheRepository)
	producer := new(mock_producer.NotifyProducer)
	notifier := new(mock_email.Notifier)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	s := NewNotifyService(db, cache, producer, notifier, logger)

	return ctx, db, cache, producer, s
}

func mustParseTime(t *testing.T, raw string) time.Time {
	t.Helper()
	ts, err := time.Parse("2006-01-02T15:04:05.999999", raw)
	assert.NoError(t, err)
	return ts
}

func TestCreateNotify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx, db, cache, _, s := setupTestService(t)

		sendAt := mustParseTime(t, "2025-10-25T10:10:10.555555")
		input := entity.Notify{SendAt: sendAt, Message: "test message", Email: "test@example.com"}
		expected := input
		expected.ID = "test-id"
		expected.Status = entity.StatusScheduled

		db.On("CreateNotify", ctx, input).Return(expected, nil).Once()
		cache.On("SetNotify", ctx, expected, 24*time.Hour).Return(nil).Once()

		result, err := s.CreateNotify(ctx, input)

		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		db.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("db error", func(t *testing.T) {
		ctx, db, cache, _, s := setupTestService(t)

		input := entity.Notify{Message: "fail", Email: "fail@example.com"}
		db.On("CreateNotify", ctx, input).Return(entity.Notify{}, assert.AnError).Once()

		result, err := s.CreateNotify(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, entity.Notify{}, result)
		db.AssertExpectations(t)
		cache.AssertNotCalled(t, "SetNotify", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestGetNotify(t *testing.T) {
	t.Run("cache hit", func(t *testing.T) {
		ctx, db, cache, _, s := setupTestService(t)

		n := entity.Notify{ID: "id1", Message: "msg", SendAt: mustParseTime(t, "2025-10-25T10:10:10.555555"), Status: entity.StatusQueued, Email: "cache@example.com"}
		cache.On("GetNotify", ctx, n.ID).Return(n, nil).Once()

		result, err := s.GetNotify(ctx, n.ID)

		assert.NoError(t, err)
		assert.Equal(t, n, result)
		cache.AssertExpectations(t)
		db.AssertNotCalled(t, "GetNotify", mock.Anything, mock.Anything)
	})

	t.Run("cache miss, db hit", func(t *testing.T) {
		ctx, db, cache, _, s := setupTestService(t)

		n := entity.Notify{ID: "id2", Message: "msg2", SendAt: mustParseTime(t, "2025-10-25T10:10:10.555555"), Status: entity.StatusQueued, Email: "db@example.com"}
		cache.On("GetNotify", ctx, n.ID).Return(entity.Notify{}, assert.AnError).Once()
		db.On("GetNotify", ctx, n.ID).Return(n, nil).Once()
		cache.On("SetNotify", ctx, n, 24*time.Hour).Return(nil).Once()

		result, err := s.GetNotify(ctx, n.ID)

		assert.NoError(t, err)
		assert.Equal(t, n, result)
		cache.AssertExpectations(t)
		db.AssertExpectations(t)
	})

	t.Run("cache miss, db error", func(t *testing.T) {
		ctx, db, cache, _, s := setupTestService(t)

		id := "id3"
		cache.On("GetNotify", ctx, id).Return(entity.Notify{}, assert.AnError).Once()
		db.On("GetNotify", ctx, id).Return(entity.Notify{}, assert.AnError).Once()

		result, err := s.GetNotify(ctx, id)

		assert.Error(t, err)
		assert.Equal(t, entity.Notify{}, result)
		cache.AssertExpectations(t)
		db.AssertExpectations(t)
	})
}

func TestDeleteNotify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx, db, cache, _, s := setupTestService(t)

		id := "id1"
		db.On("DeleteNotify", ctx, id).Return(nil).Once()
		cache.On("DeleteNotify", ctx, id).Return(nil).Once()

		err := s.DeleteNotify(ctx, id)

		assert.NoError(t, err)
		db.AssertExpectations(t)
		cache.AssertExpectations(t)
	})
}

func TestUpdateNotifyStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx, db, _, _, s := setupTestService(t)

		id := "id1"
		status := entity.StatusQueued
		db.On("UpdateNotifyStatus", ctx, id, status).Return(nil).Once()

		err := s.UpdateNotifyStatus(ctx, id, status)

		assert.NoError(t, err)
		db.AssertExpectations(t)
	})
}

func TestScheduleReadyNotifies(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx, db, _, producer, s := setupTestService(t)

		n1 := entity.Notify{ID: "id1", Message: "m1", SendAt: mustParseTime(t, "2025-10-25T10:10:10.555555"), Status: entity.StatusScheduled}
		n2 := entity.Notify{ID: "id2", Message: "m2", SendAt: mustParseTime(t, "2025-10-26T11:11:11.111111"), Status: entity.StatusScheduled}
		notifies := []entity.Notify{n1, n2}

		db.On("GetReadyNotifies", ctx).Return(notifies, nil).Once()
		producer.On("Send", ctx, n1).Return(nil).Once()
		db.On("UpdateNotifyStatus", ctx, n1.ID, entity.StatusQueued).Return(nil).Once()
		producer.On("Send", ctx, n2).Return(nil).Once()
		db.On("UpdateNotifyStatus", ctx, n2.ID, entity.StatusQueued).Return(nil).Once()

		err := s.ScheduleReadyNotifies(ctx)

		assert.NoError(t, err)
		db.AssertExpectations(t)
		producer.AssertExpectations(t)
	})

	t.Run("db error", func(t *testing.T) {
		ctx, db, _, producer, s := setupTestService(t)

		db.On("GetReadyNotifies", ctx).Return(nil, assert.AnError).Once()

		err := s.ScheduleReadyNotifies(ctx)

		assert.Error(t, err)
		db.AssertExpectations(t)
		producer.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
	})

	t.Run("producer error", func(t *testing.T) {
		ctx, db, _, producer, s := setupTestService(t)

		n1 := entity.Notify{ID: "id1", Message: "m1", SendAt: mustParseTime(t, "2025-10-25T10:10:10.555555"), Status: entity.StatusScheduled}
		n2 := entity.Notify{ID: "id2", Message: "m2", SendAt: mustParseTime(t, "2025-10-26T11:11:11.111111"), Status: entity.StatusScheduled}
		notifies := []entity.Notify{n1, n2}

		db.On("GetReadyNotifies", ctx).Return(notifies, nil).Once()
		producer.On("Send", ctx, n1).Return(assert.AnError).Once()
		producer.On("Send", ctx, n2).Return(assert.AnError).Once()

		err := s.ScheduleReadyNotifies(ctx)

		assert.NoError(t, err)
		db.AssertExpectations(t)
		producer.AssertExpectations(t)
		db.AssertNotCalled(t, "UpdateNotifyStatus", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("update status error", func(t *testing.T) {
		ctx, db, _, producer, s := setupTestService(t)

		n1 := entity.Notify{ID: "id1", Message: "m1", SendAt: mustParseTime(t, "2025-10-25T10:10:10.555555"), Status: entity.StatusScheduled}
		n2 := entity.Notify{ID: "id2", Message: "m2", SendAt: mustParseTime(t, "2025-10-26T11:11:11.111111"), Status: entity.StatusScheduled}
		notifies := []entity.Notify{n1, n2}

		db.On("GetReadyNotifies", ctx).Return(notifies, nil).Once()
		producer.On("Send", ctx, n1).Return(nil).Once()
		db.On("UpdateNotifyStatus", ctx, n1.ID, entity.StatusQueued).Return(assert.AnError).Once()

		err := s.ScheduleReadyNotifies(ctx)

		assert.Error(t, err)
		db.AssertCalled(t, "GetReadyNotifies", ctx)
		producer.AssertCalled(t, "Send", ctx, n1)
		producer.AssertNotCalled(t, "Send", ctx, n2)
		db.AssertCalled(t, "UpdateNotifyStatus", ctx, n1.ID, entity.StatusQueued)
		db.AssertNotCalled(t, "UpdateNotifyStatus", ctx, n2.ID, entity.StatusQueued)
	})
}
