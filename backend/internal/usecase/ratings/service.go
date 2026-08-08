package ratings

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"movies/backend/internal/domain"
)

const (
	SortRecent   = "recent"
	SortScore    = "score"
	SortTitle    = "title"
	OrderAsc     = "asc"
	OrderDesc    = "desc"
	defaultLimit = 20
	maxLimit     = 50
)

var (
	ErrValidation = errors.New("validation_failed")
	ErrUpstream   = errors.New("upstream_error")
)

type Provider interface {
	Get(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (domain.Title, error)
}

type AchievementObserver interface {
	ObserveCircle(ctx context.Context, userID int64)
}

type Repository interface {
	GetUserByUUID(ctx context.Context, uuid string) (domain.User, bool, error)
	GetUserRelationship(ctx context.Context, viewerID, userID int64) (string, error)
	GetTitleID(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (int64, bool, error)
	ListCriteriaByCodes(ctx context.Context, codes []string) (map[string]domain.Criterion, error)
	Upsert(ctx context.Context, params UpsertRatingParams) (domain.Rating, error)
	Delete(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) error
	UserCanSeeRatings(ctx context.Context, viewerID, userID int64) (bool, error)
	ListUserRatings(ctx context.Context, userID int64, query ListQuery) ([]domain.ProfileRating, error)
	GetProfileStats(ctx context.Context, userID int64) (domain.ProfileRatingStats, error)
}

type Cursor struct {
	ID     int64
	Recent time.Time
	Score  float64
	Title  string
}

type ListQuery struct {
	Sort   string
	Order  string
	Cursor Cursor
	Limit  int
}

type UpsertRatingParams struct {
	UserID    int64
	Title     domain.Title
	AvgTenths int
	Scores    map[string]int
	Criteria  map[string]domain.Criterion
}

type Service struct {
	repo         Repository
	provider     Provider
	achievements AchievementObserver
}

func NewService(repo Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
}

func (s *Service) SetAchievementObserver(observer AchievementObserver) {
	s.achievements = observer
}

func (s *Service) Upsert(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64, scores map[string]int) (domain.Rating, error) {
	if userID == 0 || !validTitleRef(mediaType, tmdbID) {
		return domain.Rating{}, ErrValidation
	}
	if len(scores) == 0 {
		return domain.Rating{}, ErrValidation
	}

	codes := make([]string, 0, len(scores))
	total := 0
	for code, score := range scores {
		if code == "" || score < 1 || score > 10 {
			return domain.Rating{}, ErrValidation
		}
		codes = append(codes, code)
		total += score
	}

	criteria, err := s.repo.ListCriteriaByCodes(ctx, codes)
	if err != nil {
		return domain.Rating{}, err
	}
	if len(criteria) != len(scores) {
		return domain.Rating{}, ErrValidation
	}

	titleID, exists, err := s.repo.GetTitleID(ctx, mediaType, tmdbID)
	if err != nil {
		return domain.Rating{}, err
	}
	title := domain.Title{ID: titleID, TmdbID: tmdbID, MediaType: mediaType}
	if !exists {
		title, err = s.provider.Get(ctx, mediaType, tmdbID)
		if err != nil {
			return domain.Rating{}, err
		}
	}

	avgTenths := int(math.Round(float64(total) / float64(len(scores)) * 10))
	rating, err := s.repo.Upsert(ctx, UpsertRatingParams{
		UserID:    userID,
		Title:     title,
		AvgTenths: avgTenths,
		Scores:    scores,
		Criteria:  criteria,
	})
	if err != nil {
		return domain.Rating{}, err
	}
	if s.achievements != nil {
		s.achievements.ObserveCircle(ctx, userID)
	}
	return rating, nil
}

func (s *Service) Delete(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) error {
	if userID == 0 || !validTitleRef(mediaType, tmdbID) {
		return ErrValidation
	}
	return s.repo.Delete(ctx, userID, mediaType, tmdbID)
}

func (s *Service) ListUserRatings(ctx context.Context, viewerID, userID int64, sortBy, order, rawCursor string, limit int) (domain.ProfileRatingsPage, error) {
	if viewerID == 0 || userID == 0 {
		return domain.ProfileRatingsPage{}, ErrValidation
	}

	ok, err := s.repo.UserCanSeeRatings(ctx, viewerID, userID)
	if err != nil {
		return domain.ProfileRatingsPage{}, err
	}
	if !ok {
		return domain.ProfileRatingsPage{Ratings: []domain.ProfileRating{}}, nil
	}
	query, err := parseListQuery(sortBy, order, rawCursor, limit)
	if err != nil {
		return domain.ProfileRatingsPage{}, err
	}
	ratings, err := s.repo.ListUserRatings(ctx, userID, ListQuery{
		Sort: query.Sort, Order: query.Order, Cursor: query.Cursor, Limit: query.Limit + 1,
	})
	if err != nil {
		return domain.ProfileRatingsPage{}, err
	}
	stats, err := s.repo.GetProfileStats(ctx, userID)
	if err != nil {
		return domain.ProfileRatingsPage{}, err
	}
	page := domain.ProfileRatingsPage{
		User: domain.User{ID: userID}, Ratings: ratings, Stats: stats,
	}
	if len(ratings) > query.Limit {
		page.Ratings = ratings[:query.Limit]
		page.NextCursor = encodeListCursor(query, page.Ratings[len(page.Ratings)-1])
	}
	return page, nil
}

func (s *Service) ListUserRatingsByUUID(ctx context.Context, viewerID int64, userUUID, sortBy, order, rawCursor string, limit int) (domain.ProfileRatingsPage, error) {
	if viewerID == 0 || userUUID == "" {
		return domain.ProfileRatingsPage{}, ErrValidation
	}

	target, ok, err := s.repo.GetUserByUUID(ctx, userUUID)
	if err != nil {
		return domain.ProfileRatingsPage{}, err
	}
	if !ok {
		return domain.ProfileRatingsPage{}, ErrValidation
	}

	page, err := s.ListUserRatings(ctx, viewerID, target.ID, sortBy, order, rawCursor, limit)
	if err != nil {
		return domain.ProfileRatingsPage{}, err
	}
	relationship, err := s.repo.GetUserRelationship(ctx, viewerID, target.ID)
	if err != nil {
		return domain.ProfileRatingsPage{}, err
	}
	page.User = target
	page.Relationship = relationship
	return page, nil
}

type parsedListQuery struct {
	Sort   string
	Order  string
	Cursor Cursor
	Limit  int
}

type listCursorPayload struct {
	Version int       `json:"v"`
	Sort    string    `json:"sort"`
	Order   string    `json:"order"`
	ID      int64     `json:"id"`
	Recent  time.Time `json:"recent,omitempty"`
	Score   float64   `json:"score,omitempty"`
	Title   string    `json:"title,omitempty"`
}

func parseListQuery(sortBy, order, rawCursor string, limit int) (parsedListQuery, error) {
	if sortBy == "" {
		sortBy = SortRecent
	}
	if order == "" {
		order = OrderDesc
	}
	if sortBy != SortRecent && sortBy != SortScore && sortBy != SortTitle {
		return parsedListQuery{}, ErrValidation
	}
	if order != OrderAsc && order != OrderDesc {
		return parsedListQuery{}, ErrValidation
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	query := parsedListQuery{Sort: sortBy, Order: order, Limit: limit}
	if rawCursor == "" {
		return query, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(rawCursor)
	if err != nil {
		return parsedListQuery{}, ErrValidation
	}
	var payload listCursorPayload
	if json.Unmarshal(data, &payload) != nil || payload.Version != 1 || payload.Sort != sortBy || payload.Order != order || payload.ID <= 0 {
		return parsedListQuery{}, ErrValidation
	}
	query.Cursor = Cursor{ID: payload.ID, Recent: payload.Recent, Score: payload.Score, Title: payload.Title}
	switch sortBy {
	case SortRecent:
		if payload.Recent.IsZero() {
			return parsedListQuery{}, ErrValidation
		}
	case SortScore:
		if payload.Score < 1 || payload.Score > 10 {
			return parsedListQuery{}, ErrValidation
		}
	case SortTitle:
		if strings.TrimSpace(payload.Title) == "" {
			return parsedListQuery{}, ErrValidation
		}
	}
	return query, nil
}

func encodeListCursor(query parsedListQuery, rating domain.ProfileRating) string {
	payload := listCursorPayload{Version: 1, Sort: query.Sort, Order: query.Order, ID: rating.ID}
	switch query.Sort {
	case SortRecent:
		payload.Recent = rating.UpdatedAt
	case SortScore:
		payload.Score = rating.AvgScore
	case SortTitle:
		payload.Title = strings.ToLower(rating.Title.Title)
	}
	data, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString(data)
}

func validTitleRef(mediaType domain.MediaType, tmdbID int64) bool {
	return (mediaType == domain.MediaTypeMovie || mediaType == domain.MediaTypeTV) && tmdbID > 0
}

func profileStats(ratings []domain.ProfileRating) domain.ProfileRatingStats {
	if len(ratings) == 0 {
		return domain.ProfileRatingStats{}
	}

	total := 0.0
	for _, rating := range ratings {
		total += rating.AvgScore
	}
	return domain.ProfileRatingStats{
		Count:    len(ratings),
		AvgScore: math.Round(total/float64(len(ratings))*10) / 10,
	}
}
