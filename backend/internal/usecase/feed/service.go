package feed

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

var ErrValidation = errors.New("validation_failed")

type Cursor struct {
	CreatedAt time.Time
	ID        int64
}

type Repository interface {
	List(ctx context.Context, userID int64, cursor Cursor, limit int) ([]domain.FeedItem, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID int64, rawCursor string, limit int) (domain.FeedPage, error) {
	if userID == 0 {
		return domain.FeedPage{}, ErrValidation
	}
	cursor, err := decodeCursor(rawCursor)
	if err != nil {
		return domain.FeedPage{}, err
	}
	limit = normalizeLimit(limit)

	items, err := s.repo.List(ctx, userID, cursor, limit+1)
	if err != nil {
		return domain.FeedPage{}, err
	}

	page := domain.FeedPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
	}
	return page, nil
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
