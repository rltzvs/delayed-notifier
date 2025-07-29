package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"delayed-notifier/internal/controller"
	"delayed-notifier/internal/entity"
)

func writeError(w http.ResponseWriter, message string, statusCode int, log *slog.Logger) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": message}); err != nil {
		log.Error("failed to encode error", slog.Any("error", err))
	}
}

type NotifyHandler struct {
	service controller.NotifyService
	logger  *slog.Logger
}

func NewNotifyHandler(service controller.NotifyService, logger *slog.Logger) *NotifyHandler {
	return &NotifyHandler{
		service: service,
		logger:  logger,
	}
}

func (h *NotifyHandler) CreateNotify(w http.ResponseWriter, r *http.Request) {
	var input entity.Notify
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.logger.Error("invalid request body", slog.Any("error", err))
		writeError(w, "invalid request body", http.StatusBadRequest, h.logger)
		return
	}

	if err := input.Validate(); err != nil {
		h.logger.Error("validation error", slog.Any("error", err))
		writeError(w, err.Error(), http.StatusBadRequest, h.logger)
		return
	}

	input.Status = entity.StatusScheduled
	created, err := h.service.CreateNotify(r.Context(), input)
	if err != nil {
		h.logger.Error("failed to create notify", slog.Any("error", err))
		writeError(w, "failed to create notify", http.StatusInternalServerError, h.logger)
		return
	}

	h.logger.Info("notify created", slog.String("id", created.ID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(created); err != nil {
		h.logger.Error("failed to encode notify", slog.Any("error", err))
		writeError(w, "failed to encode notify", http.StatusInternalServerError, h.logger)
		return
	}
}

func (h *NotifyHandler) GetNotify(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "notifyID")
	if id == "" {
		writeError(w, "notifyID is required", http.StatusBadRequest, h.logger)
		return
	}

	notify, err := h.service.GetNotify(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrNotifyNotFound) {
			h.logger.Info("notify not found", slog.String("id", id))
			writeError(w, "notify not found", http.StatusNotFound, h.logger)
			return
		}

		h.logger.Error("failed to get notify", slog.Any("error", err))
		writeError(w, "internal server error", http.StatusInternalServerError, h.logger)
		return
	}

	h.logger.Info("notify fetched", slog.String("id", notify.ID))
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(notify); err != nil {
		h.logger.Error("failed to encode notify", slog.Any("error", err))
		writeError(w, "failed to encode notify", http.StatusInternalServerError, h.logger)
		return
	}
}

func (h *NotifyHandler) DeleteNotify(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "notifyID")
	if id == "" {
		writeError(w, "notifyID is required", http.StatusBadRequest, h.logger)
		return
	}

	err := h.service.DeleteNotify(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to delete notify", slog.Any("error", err), slog.String("id", id))
		writeError(w, "failed to delete notify", http.StatusInternalServerError, h.logger)
		return
	}

	h.logger.Info("notify deleted", slog.String("id", id))
	w.WriteHeader(http.StatusNoContent)
}
