package titles

import (
	"context"
	"errors"
	"math"
	"strings"

	"movies/backend/internal/domain"
)

var (
	ErrValidation = errors.New("validation_failed")
	ErrNotFound   = errors.New("not_found")
	ErrUpstream   = errors.New("upstream_error")
)

type Provider interface {
	Search(ctx context.Context, query string, page int) (domain.SearchPage, error)
	Get(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (domain.Title, error)
}

type SocialRepository interface {
	GetTitleID(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (int64, bool, error)
	GetRatingByUserTitle(ctx context.Context, userID, titleID int64) (*domain.RatingWithUser, error)
	ListFriendRatingsByTitle(ctx context.Context, userID, titleID int64) ([]domain.RatingWithUser, error)
	CountCommentsByTitle(ctx context.Context, titleID int64) (int64, error)
}

type Service struct {
	provider Provider
	social   SocialRepository
}

func NewService(provider Provider, social SocialRepository) *Service {
	return &Service{provider: provider, social: social}
}

func (s *Service) Search(ctx context.Context, query string, page int) (domain.SearchPage, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return domain.SearchPage{}, ErrValidation
	}
	if page < 1 {
		page = 1
	}
	return s.provider.Search(ctx, query, page)
}

func (s *Service) Get(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (domain.Title, error) {
	if mediaType != domain.MediaTypeMovie && mediaType != domain.MediaTypeTV {
		return domain.Title{}, ErrValidation
	}
	if tmdbID <= 0 {
		return domain.Title{}, ErrValidation
	}
	return s.provider.Get(ctx, mediaType, tmdbID)
}

func (s *Service) GetCard(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) (domain.TitleCard, error) {
	if userID == 0 {
		return domain.TitleCard{}, ErrValidation
	}

	title, err := s.Get(ctx, mediaType, tmdbID)
	if err != nil {
		return domain.TitleCard{}, err
	}

	card := domain.TitleCard{
		Title:          title,
		FriendsRatings: []domain.RatingWithUser{},
	}
	if s.social == nil {
		return card, nil
	}

	titleID, ok, err := s.social.GetTitleID(ctx, mediaType, tmdbID)
	if err != nil {
		return domain.TitleCard{}, err
	}
	if !ok {
		return card, nil
	}
	card.Title.ID = titleID

	myRating, err := s.social.GetRatingByUserTitle(ctx, userID, titleID)
	if err != nil {
		return domain.TitleCard{}, err
	}
	card.MyRating = myRating

	friendsRatings, err := s.social.ListFriendRatingsByTitle(ctx, userID, titleID)
	if err != nil {
		return domain.TitleCard{}, err
	}
	card.FriendsRatings = friendsRatings
	card.FriendsAvg = averageRatings(myRating, friendsRatings)

	commentsCount, err := s.social.CountCommentsByTitle(ctx, titleID)
	if err != nil {
		return domain.TitleCard{}, err
	}
	card.CommentsCount = commentsCount

	return card, nil
}

func averageRatings(myRating *domain.RatingWithUser, friendsRatings []domain.RatingWithUser) *domain.FriendsAverage {
	totalOverall := 0.0
	countOverall := 0
	criteriaTotals := make(map[string]int)
	criteriaCounts := make(map[string]int)

	add := func(rating domain.RatingWithUser) {
		totalOverall += rating.AvgScore
		countOverall++
		for code, score := range rating.Scores {
			criteriaTotals[code] += score
			criteriaCounts[code]++
		}
	}

	if myRating != nil {
		add(*myRating)
	}
	for _, rating := range friendsRatings {
		add(rating)
	}
	if countOverall == 0 {
		return nil
	}

	byCriteria := make(map[string]float64, len(criteriaTotals))
	for code, total := range criteriaTotals {
		byCriteria[code] = round1(float64(total) / float64(criteriaCounts[code]))
	}

	return &domain.FriendsAverage{
		Overall:    round1(totalOverall / float64(countOverall)),
		ByCriteria: byCriteria,
	}
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}
