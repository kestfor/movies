package watchlist

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
	ErrConflict   = errors.New("conflict")
	ErrUpstream   = errors.New("upstream_error")
)

type Provider interface {
	Get(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (domain.Title, error)
}

type Cursor struct {
	AddedAt time.Time
	TitleID int64
}

type Repository interface {
	GetUserByUUID(ctx context.Context, uuid string) (domain.User, bool, error)
	UserCanSeeWatchlist(ctx context.Context, viewerID, userID int64) (bool, error)
	GetTitleID(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (int64, bool, error)
	Add(ctx context.Context, userID int64, title domain.Title) (bool, error)
	Remove(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) error
	List(ctx context.Context, userID int64, cursor Cursor, limit int) ([]domain.WatchlistItem, error)
	Count(ctx context.Context, userID int64) (int64, error)
	IsInWatchlist(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) (bool, error)
	ListRefs(ctx context.Context, userID int64) ([]domain.TitleRef, error)
}

type Service struct {
	repo     Repository
	provider Provider
}

func NewService(repo Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
}

func (s *Service) Add(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) error {
	if userID == 0 || !validTitleRef(mediaType, tmdbID) {
		return ErrValidation
	}
	titleID, exists, err := s.repo.GetTitleID(ctx, mediaType, tmdbID)
	if err != nil {
		return err
	}
	title := domain.Title{ID: titleID, TmdbID: tmdbID, MediaType: mediaType}
	if !exists {
		title, err = s.provider.Get(ctx, mediaType, tmdbID)
		if err != nil {
			return err
		}
	}
	allowed, err := s.repo.Add(ctx, userID, title)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrConflict
	}
	return nil
}

func (s *Service) Remove(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) error {
	if userID == 0 || !validTitleRef(mediaType, tmdbID) {
		return ErrValidation
	}
	return s.repo.Remove(ctx, userID, mediaType, tmdbID)
}

func (s *Service) IsInWatchlist(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) (bool, error) {
	if userID == 0 || !validTitleRef(mediaType, tmdbID) {
		return false, ErrValidation
	}
	return s.repo.IsInWatchlist(ctx, userID, mediaType, tmdbID)
}

func (s *Service) Statuses(ctx context.Context, userID int64, refs []domain.TitleRef) (map[domain.TitleRef]bool, error) {
	if userID == 0 {
		return nil, ErrValidation
	}
	stored, err := s.repo.ListRefs(ctx, userID)
	if err != nil {
		return nil, err
	}
	set := make(map[domain.TitleRef]bool, len(stored))
	for _, ref := range stored {
		set[ref] = true
	}
	result := make(map[domain.TitleRef]bool, len(refs))
	for _, ref := range refs {
		result[ref] = set[ref]
	}
	return result, nil
}

func (s *Service) ListByUUID(ctx context.Context, viewerID int64, userUUID, rawCursor string, limit int) (domain.WatchlistPage, error) {
	if viewerID == 0 || userUUID == "" {
		return domain.WatchlistPage{}, ErrValidation
	}
	target, ok, err := s.repo.GetUserByUUID(ctx, userUUID)
	if err != nil {
		return domain.WatchlistPage{}, err
	}
	if !ok {
		return domain.WatchlistPage{}, ErrValidation
	}
	visible, err := s.repo.UserCanSeeWatchlist(ctx, viewerID, target.ID)
	if err != nil {
		return domain.WatchlistPage{}, err
	}
	if !visible {
		return domain.WatchlistPage{Items: []domain.WatchlistItem{}}, nil
	}
	cursor, err := decodeCursor(rawCursor)
	if err != nil {
		return domain.WatchlistPage{}, err
	}
	limit = normalizeLimit(limit)
	items, err := s.repo.List(ctx, target.ID, cursor, limit+1)
	if err != nil {
		return domain.WatchlistPage{}, err
	}
	count, err := s.repo.Count(ctx, target.ID)
	if err != nil {
		return domain.WatchlistPage{}, err
	}
	page := domain.WatchlistPage{Items: items, TotalCount: count}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(Cursor{AddedAt: last.AddedAt, TitleID: last.Title.ID})
	}
	return page, nil
}

func validTitleRef(mediaType domain.MediaType, tmdbID int64) bool {
	return (mediaType == domain.MediaTypeMovie || mediaType == domain.MediaTypeTV) && tmdbID > 0
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
		Version int       `json:"v"`
		AddedAt time.Time `json:"added_at"`
		TitleID int64     `json:"title_id"`
	}
	if json.Unmarshal(data, &payload) != nil || payload.Version != 1 || payload.AddedAt.IsZero() || payload.TitleID <= 0 {
		return Cursor{}, ErrValidation
	}
	return Cursor{AddedAt: payload.AddedAt, TitleID: payload.TitleID}, nil
}

func encodeCursor(cursor Cursor) string {
	data, _ := json.Marshal(struct {
		Version int       `json:"v"`
		AddedAt time.Time `json:"added_at"`
		TitleID int64     `json:"title_id"`
	}{Version: 1, AddedAt: cursor.AddedAt, TitleID: cursor.TitleID})
	return base64.RawURLEncoding.EncodeToString(data)
}
