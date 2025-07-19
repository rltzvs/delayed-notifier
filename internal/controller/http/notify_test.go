package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"delayed-notifier/internal/entity"
	mock_service "delayed-notifier/internal/service/mocks"
)

func setupHandler() (*NotifyHandler, *mock_service.NotifyService) {
	mockService := new(mock_service.NotifyService)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewNotifyHandler(mockService, logger)
	return handler, mockService
}

func addNotifyIDToCtx(req *http.Request, notifyID string) *http.Request {
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{
			Keys:   []string{"notifyID"},
			Values: []string{notifyID},
		},
	})
	return req.WithContext(ctx)
}

func mustEncode(t *testing.T, v any) (*bytes.Buffer, string) {
	t.Helper()
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(v)
	require.NoError(t, err)
	return buf, "application/json"
}

func TestCreateNotify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, mockNotifyService := setupHandler()

		input := entity.Notify{
			SendAt:  time.Now().Add(time.Minute),
			Message: "test message",
		}
		expected := input
		expected.Status = entity.StatusScheduled
		expected.ID = "test-id"

		mockNotifyService.
			On("CreateNotify", mock.Anything, mock.MatchedBy(func(n entity.Notify) bool {
				return n.Message == "test message" &&
					n.Status == entity.StatusScheduled
			})).Return(expected, nil).Once()

		body, contentType := mustEncode(t, input)
		req := httptest.NewRequest(http.MethodPost, "/notify", body)
		req.Header.Set("Content-Type", contentType)

		rec := httptest.NewRecorder()
		handler.CreateNotify(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var actual entity.Notify
		err := json.NewDecoder(rec.Body).Decode(&actual)
		require.NoError(t, err)

		assert.WithinDuration(t, expected.SendAt, actual.SendAt, time.Millisecond)
		assert.Equal(t, expected.ID, actual.ID)
		assert.Equal(t, expected.Message, actual.Message)
		assert.Equal(t, expected.Status, actual.Status)
		mockNotifyService.AssertExpectations(t)
	})

	t.Run("invalid json", func(t *testing.T) {
		handler, _ := setupHandler()

		req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString("{invalid}"))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.CreateNotify(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid request body")
	})

	t.Run("validation error", func(t *testing.T) {
		handler, _ := setupHandler()

		input := entity.Notify{
			SendAt: time.Now().Add(time.Minute),
		}

		body, contentType := mustEncode(t, input)
		req := httptest.NewRequest(http.MethodPost, "/notify", body)
		req.Header.Set("Content-Type", contentType)

		rec := httptest.NewRecorder()
		handler.CreateNotify(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "message is required")
	})

	t.Run("internal error", func(t *testing.T) {
		handler, mockNotifyService := setupHandler()

		input := entity.Notify{
			SendAt:  time.Now().Add(time.Minute),
			Message: "test message",
		}

		mockNotifyService.
			On("CreateNotify", mock.Anything, mock.Anything).
			Return(entity.Notify{}, assert.AnError).Once()

		body, contentType := mustEncode(t, input)
		req := httptest.NewRequest(http.MethodPost, "/notify", body)
		req.Header.Set("Content-Type", contentType)

		rec := httptest.NewRecorder()
		handler.CreateNotify(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "failed to create notify")
		mockNotifyService.AssertExpectations(t)
	})
}

func TestGetNotify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, mockService := setupHandler()

		expected := entity.Notify{
			ID:      "123",
			Message: "reminder",
			Status:  entity.StatusScheduled,
			SendAt:  time.Now().Add(time.Minute),
		}

		mockService.
			On("GetNotify", mock.Anything, "123").
			Return(expected, nil).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/notify/123", nil)
		req = addNotifyIDToCtx(req, "123")

		rec := httptest.NewRecorder()
		handler.GetNotify(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var actual entity.Notify
		err := json.NewDecoder(rec.Body).Decode(&actual)
		require.NoError(t, err)

		assert.Equal(t, expected.ID, actual.ID)
		assert.Equal(t, expected.Message, actual.Message)
		assert.Equal(t, expected.Status, actual.Status)
		mockService.AssertExpectations(t)
	})

	t.Run("missing notifyID", func(t *testing.T) {
		handler, _ := setupHandler()

		req := httptest.NewRequest(http.MethodGet, "/notify/", nil)
		rec := httptest.NewRecorder()
		handler.GetNotify(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "notifyID is required")
	})

	t.Run("not found", func(t *testing.T) {
		handler, mockService := setupHandler()

		mockService.
			On("GetNotify", mock.Anything, "not-exist").
			Return(entity.Notify{}, entity.ErrNotifyNotFound).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/notify/not-exist", nil)
		req = addNotifyIDToCtx(req, "not-exist")

		rec := httptest.NewRecorder()
		handler.GetNotify(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "notify not found")
		mockService.AssertExpectations(t)
	})

	t.Run("internal error", func(t *testing.T) {
		handler, mockService := setupHandler()

		mockService.
			On("GetNotify", mock.Anything, "123").
			Return(entity.Notify{}, assert.AnError).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/notify/123", nil)
		req = addNotifyIDToCtx(req, "123")

		rec := httptest.NewRecorder()
		handler.GetNotify(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "internal server error")
		mockService.AssertExpectations(t)
	})
}

func TestDeleteNotify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler, mockService := setupHandler()

		mockService.
			On("DeleteNotify", mock.Anything, "123").
			Return(nil).
			Once()

		req := httptest.NewRequest(http.MethodDelete, "/notify/123", nil)
		req = addNotifyIDToCtx(req, "123")

		rec := httptest.NewRecorder()
		handler.DeleteNotify(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("missing notifyID", func(t *testing.T) {
		handler, _ := setupHandler()

		req := httptest.NewRequest(http.MethodDelete, "/notify/", nil)
		rec := httptest.NewRecorder()
		handler.DeleteNotify(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "notifyID is required")
	})

	t.Run("internal error", func(t *testing.T) {
		handler, mockService := setupHandler()

		mockService.
			On("DeleteNotify", mock.Anything, "123").
			Return(assert.AnError).
			Once()

		req := httptest.NewRequest(http.MethodDelete, "/notify/123", nil)
		req = addNotifyIDToCtx(req, "123")

		rec := httptest.NewRecorder()
		handler.DeleteNotify(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "failed to delete notify")
		mockService.AssertExpectations(t)
	})
}
