package entity

import (
	"errors"
	"time"
)

var (
	ErrNotifyNotFound = errors.New("notify not found")
)

const (
	StatusScheduled = "scheduled"
	StatusQueued    = "queued"
	StatusSent      = "sent"
	StatusFailed    = "failed"
)

type Notify struct {
	ID      string    `json:"id"`
	SendAt  time.Time `json:"send_at"`
	Message string    `json:"message"`
	Status  string    `json:"status,omitempty"`
}

func (n *Notify) Validate() error {
	if n.Message == "" {
		return errors.New("message is required")
	}
	if n.SendAt.IsZero() {
		return errors.New("send_at is required")
	}
	if n.SendAt.Before(time.Now()) {
		return errors.New("send_at must be in the future")
	}
	return nil
}
