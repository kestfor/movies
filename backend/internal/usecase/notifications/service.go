package notifications

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"movies/backend/internal/domain"
)

const (
	defaultLimit = 20
	maxLimit     = 50
)

var (
	ErrValidation = errors.New("validation_failed")
	ErrNotFound   = errors.New("not_found")
)

type Cursor struct {
	CreatedAt time.Time
	ID        int64
}

type Repository interface {
	List(ctx context.Context, userID int64, cursor Cursor, limit int) ([]domain.Notification, error)
	CountUnread(ctx context.Context, userID int64) (int64, error)
	MarkRead(ctx context.Context, userID, eventID int64) (bool, error)
	MarkAllRead(ctx context.Context, userID int64) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID int64, rawCursor string, limit int) (domain.NotificationsPage, error) {
	if userID == 0 {
		return domain.NotificationsPage{}, ErrValidation
	}
	cursor, err := decodeCursor(rawCursor)
	if err != nil {
		return domain.NotificationsPage{}, err
	}
	limit = normalizeLimit(limit)

	items, err := s.repo.List(ctx, userID, cursor, limit+1)
	if err != nil {
		return domain.NotificationsPage{}, err
	}

	page := domain.NotificationsPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(Cursor{CreatedAt: last.CreatedAt, ID: last.EventID})
	}
	return page, nil
}

func (s *Service) CountUnread(ctx context.Context, userID int64) (int64, error) {
	if userID == 0 {
		return 0, ErrValidation
	}
	return s.repo.CountUnread(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, eventID int64) error {
	if userID == 0 || eventID <= 0 {
		return ErrValidation
	}
	ok, err := s.repo.MarkRead(ctx, userID, eventID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID int64) error {
	if userID == 0 {
		return ErrValidation
	}
	return s.repo.MarkAllRead(ctx, userID)
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func decodeCursor(raw string) (Cursor, error) {
	if raw == "" {
		return Cursor{}, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return Cursor{}, ErrValidation
	}

	var payload struct {
		CreatedAt time.Time `json:"created_at"`
		ID        int64     `json:"id"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return Cursor{}, ErrValidation
	}
	if payload.CreatedAt.IsZero() || payload.ID <= 0 {
		return Cursor{}, ErrValidation
	}
	return Cursor{CreatedAt: payload.CreatedAt, ID: payload.ID}, nil
}

func encodeCursor(cursor Cursor) string {
	payload := struct {
		CreatedAt time.Time `json:"created_at"`
		ID        int64     `json:"id"`
	}{
		CreatedAt: cursor.CreatedAt,
		ID:        cursor.ID,
	}
	data, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString(data)
}
