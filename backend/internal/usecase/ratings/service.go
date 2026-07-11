package ratings

import (
	"context"
	"errors"
	"math"

	"movies/backend/internal/domain"
)

var (
	ErrValidation = errors.New("validation_failed")
	ErrUpstream   = errors.New("upstream_error")
)

type Provider interface {
	Get(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (domain.Title, error)
}

type Repository interface {
	GetUserByUUID(ctx context.Context, uuid string) (domain.User, bool, error)
	GetUserRelationship(ctx context.Context, viewerID, userID int64) (string, error)
	GetTitleID(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (int64, bool, error)
	ListCriteriaByCodes(ctx context.Context, codes []string) (map[string]domain.Criterion, error)
	Upsert(ctx context.Context, params UpsertRatingParams) (domain.Rating, error)
	Delete(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) error
	UserCanSeeRatings(ctx context.Context, viewerID, userID int64) (bool, error)
	ListUserRatings(ctx context.Context, userID int64) ([]domain.ProfileRating, error)
}

type UpsertRatingParams struct {
	UserID    int64
	Title     domain.Title
	AvgTenths int
	Scores    map[string]int
	Criteria  map[string]domain.Criterion
}

type Service struct {
	repo     Repository
	provider Provider
}

func NewService(repo Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
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
	return s.repo.Upsert(ctx, UpsertRatingParams{
		UserID:    userID,
		Title:     title,
		AvgTenths: avgTenths,
		Scores:    scores,
		Criteria:  criteria,
	})
}

func (s *Service) Delete(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) error {
	if userID == 0 || !validTitleRef(mediaType, tmdbID) {
		return ErrValidation
	}
	return s.repo.Delete(ctx, userID, mediaType, tmdbID)
}

func (s *Service) ListUserRatings(ctx context.Context, viewerID, userID int64) (domain.ProfileRatingsPage, error) {
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
	ratings, err := s.repo.ListUserRatings(ctx, userID)
	if err != nil {
		return domain.ProfileRatingsPage{}, err
	}
	return domain.ProfileRatingsPage{
		User:         domain.User{ID: userID},
		Relationship: "",
		Ratings:      ratings,
		Stats:        profileStats(ratings),
	}, nil
}

func (s *Service) ListUserRatingsByUUID(ctx context.Context, viewerID int64, userUUID string) (domain.ProfileRatingsPage, error) {
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

	page, err := s.ListUserRatings(ctx, viewerID, target.ID)
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
