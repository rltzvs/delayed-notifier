package http

import (
	"delayed-notifier/internal/controller"
	"delayed-notifier/internal/entity"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := input.Validate(); err != nil {
		h.logger.Error("validation error", slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created, err := h.service.CreateNotify(r.Context(), input)
	if err != nil {
		h.logger.Error("failed to create notify", slog.Any("error", err))
		http.Error(w, "failed to create notify", http.StatusInternalServerError)
		return
	}
	h.logger.Info("notify created", slog.String("id", created.ID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *NotifyHandler) GetNotify(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "notifyID")
	if id == "" {
		http.Error(w, "notifyID is required", http.StatusBadRequest)
		return
	}
	notify, err := h.service.GetNotify(r.Context(), id)
	if err != nil {
		if errors.Is(err, entity.ErrNotifyNotFound) {
			h.logger.Info("notify not found", slog.String("id", id))
			http.Error(w, "notify not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get notify", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("notify fetched", slog.String("id", notify.ID))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notify)
}

func (h *NotifyHandler) DeleteNotify(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "notifyID")
	if id == "" {
		http.Error(w, "notifyID is required", http.StatusBadRequest)
		return
	}
	err := h.service.DeleteNotify(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to delete notify", slog.Any("error", err), slog.String("id", id))
		http.Error(w, "failed to delete notify", http.StatusInternalServerError)
		return
	}
	h.logger.Info("notify deleted", slog.String("id", id))
	w.WriteHeader(http.StatusNoContent)
}
