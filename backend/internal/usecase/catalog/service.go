package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"movies/backend/internal/domain"
)

const (
	defaultLimit = 20
	maxLimit     = 50
)

var (
	ErrValidation = errors.New("validation_failed")
	ErrUpstream   = errors.New("upstream_error")
)

type Provider interface {
	Discover(ctx context.Context, mediaType domain.MediaType, page int) (domain.CatalogProviderPage, error)
	Recommendations(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (domain.CatalogProviderPage, error)
}

type Repository interface {
	ListRatedRefs(ctx context.Context, userID int64) ([]domain.TitleRef, error)
	ListRefs(ctx context.Context, userID int64) ([]domain.TitleRef, error)
	CountRatings(ctx context.Context, userID int64) (int64, error)
	ListRecommendationSeeds(ctx context.Context, userID int64) ([]domain.RecommendationSeed, error)
	ListGenreRatings(ctx context.Context, userID int64) ([]domain.GenreRating, error)
}

type Service struct {
	provider Provider
	repo     Repository
}

func NewService(provider Provider, repo Repository) *Service {
	return &Service{provider: provider, repo: repo}
}

type discoverCursor struct {
	Version int    `json:"v"`
	Type    string `json:"type"`
	Page    int    `json:"page"`
	Offset  int    `json:"offset"`
}

type recommendationCursor struct {
	Version     int    `json:"v"`
	Fingerprint string `json:"fingerprint"`
	Offset      int    `json:"offset"`
}

func (s *Service) Discover(ctx context.Context, userID int64, mediaFilter, rawCursor string, limit int) (domain.CatalogPage, error) {
	if userID == 0 {
		return domain.CatalogPage{}, ErrValidation
	}
	mediaFilter = normalizeMediaFilter(mediaFilter)
	if mediaFilter == "" {
		return domain.CatalogPage{}, ErrValidation
	}
	limit = normalizeLimit(limit)
	cursor, err := decodeDiscoverCursor(rawCursor, mediaFilter)
	if err != nil {
		return domain.CatalogPage{}, err
	}
	rated, err := s.repo.ListRatedRefs(ctx, userID)
	if err != nil {
		return domain.CatalogPage{}, err
	}
	watchlisted, err := s.repo.ListRefs(ctx, userID)
	if err != nil {
		return domain.CatalogPage{}, err
	}
	items, next, degraded, err := s.collectDiscover(ctx, mediaFilter, cursor, limit, refSet(rated), refSet(watchlisted))
	if err != nil {
		return domain.CatalogPage{}, err
	}
	page := domain.CatalogPage{Items: items, Degraded: degraded}
	if next != nil {
		page.NextCursor = encodeJSON(next)
	}
	return page, nil
}

func (s *Service) Recommendations(ctx context.Context, userID int64, rawCursor string, limit int) (domain.CatalogPage, error) {
	if userID == 0 {
		return domain.CatalogPage{}, ErrValidation
	}
	limit = normalizeLimit(limit)
	count, err := s.repo.CountRatings(ctx, userID)
	if err != nil {
		return domain.CatalogPage{}, err
	}
	seeds, err := s.repo.ListRecommendationSeeds(ctx, userID)
	if err != nil {
		return domain.CatalogPage{}, err
	}
	genreRatings, err := s.repo.ListGenreRatings(ctx, userID)
	if err != nil {
		return domain.CatalogPage{}, err
	}
	fingerprint := recommendationFingerprint(count, seeds, genreRatings)
	cursor, err := decodeRecommendationCursor(rawCursor, fingerprint)
	if err != nil {
		return domain.CatalogPage{}, err
	}
	rated, err := s.repo.ListRatedRefs(ctx, userID)
	if err != nil {
		return domain.CatalogPage{}, err
	}
	watchlisted, err := s.repo.ListRefs(ctx, userID)
	if err != nil {
		return domain.CatalogPage{}, err
	}
	excluded := refSet(append(append([]domain.TitleRef{}, rated...), watchlisted...))
	needed := cursor.Offset + limit + 1
	personalized := count >= 3 && len(seeds) > 0
	items := make([]domain.CatalogItem, 0, needed)
	degraded := false
	if personalized {
		var partial bool
		items, partial = s.rankedRecommendations(ctx, seeds, genreRatings, excluded)
		degraded = partial
	}
	if len(items) < needed {
		fallback, _, partial, fallbackErr := s.collectDiscover(
			ctx, "all", discoverCursor{Version: 1, Type: "all", Page: 1}, needed-len(items), excludedWithItems(excluded, items), nil,
		)
		degraded = degraded || partial
		if fallbackErr != nil && len(items) == 0 {
			return domain.CatalogPage{}, fallbackErr
		}
		for i := range fallback {
			if personalized {
				fallback[i].Reason = "Популярно сейчас"
			} else {
				fallback[i].Reason = "Оцените несколько тайтлов — рекомендации станут точнее"
			}
		}
		items = append(items, fallback...)
	}
	if cursor.Offset > len(items) {
		return domain.CatalogPage{}, ErrValidation
	}
	end := min(cursor.Offset+limit, len(items))
	page := domain.CatalogPage{Items: items[cursor.Offset:end], Personalized: personalized, Degraded: degraded}
	if end < len(items) {
		page.NextCursor = encodeJSON(recommendationCursor{Version: 1, Fingerprint: fingerprint, Offset: end})
	}
	return page, nil
}

func (s *Service) collectDiscover(ctx context.Context, mediaFilter string, cursor discoverCursor, limit int, excluded, watchlisted map[domain.TitleRef]bool) ([]domain.CatalogItem, *discoverCursor, bool, error) {
	items := make([]domain.CatalogItem, 0, limit)
	degraded := false
	for len(items) < limit && cursor.Page <= 500 {
		candidates, totalPages, partial, err := s.discoverWindow(ctx, mediaFilter, cursor.Page)
		degraded = degraded || partial
		if err != nil {
			if len(items) == 0 {
				return nil, nil, degraded, ErrUpstream
			}
			break
		}
		if cursor.Offset > len(candidates) {
			return nil, nil, degraded, ErrValidation
		}
		for cursor.Offset < len(candidates) && len(items) < limit {
			candidate := candidates[cursor.Offset]
			cursor.Offset++
			ref := titleRef(candidate.Title)
			if excluded[ref] {
				continue
			}
			items = append(items, domain.CatalogItem{Title: candidate.Title, InWatchlist: watchlisted != nil && watchlisted[ref]})
		}
		if len(items) == limit {
			hasRemaining := cursor.Page < totalPages
			for index := cursor.Offset; index < len(candidates) && !hasRemaining; index++ {
				hasRemaining = !excluded[titleRef(candidates[index].Title)]
			}
			if hasRemaining {
				return items, &cursor, degraded, nil
			}
			return items, nil, degraded, nil
		}
		if cursor.Offset >= len(candidates) {
			if cursor.Page >= totalPages {
				break
			}
			cursor.Page++
			cursor.Offset = 0
		}
	}
	return items, nil, degraded, nil
}

func (s *Service) discoverWindow(ctx context.Context, mediaFilter string, page int) ([]domain.CatalogCandidate, int, bool, error) {
	if mediaFilter == "movie" || mediaFilter == "tv" {
		result, err := s.provider.Discover(ctx, domain.MediaType(mediaFilter), page)
		if err != nil {
			return nil, 0, false, err
		}
		return result.Results, result.TotalPages, false, nil
	}
	movies, movieErr := s.provider.Discover(ctx, domain.MediaTypeMovie, page)
	series, seriesErr := s.provider.Discover(ctx, domain.MediaTypeTV, page)
	if movieErr != nil && seriesErr != nil {
		return nil, 0, false, ErrUpstream
	}
	items := append(append([]domain.CatalogCandidate{}, movies.Results...), series.Results...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Popularity != items[j].Popularity {
			return items[i].Popularity > items[j].Popularity
		}
		return refKey(titleRef(items[i].Title)) < refKey(titleRef(items[j].Title))
	})
	totalPages := max(movies.TotalPages, series.TotalPages)
	return items, totalPages, movieErr != nil || seriesErr != nil, nil
}

type rankedCandidate struct {
	candidate domain.CatalogCandidate
	score     float64
	reason    string
}

func (s *Service) rankedRecommendations(ctx context.Context, seeds []domain.RecommendationSeed, ratings []domain.GenreRating, excluded map[domain.TitleRef]bool) ([]domain.CatalogItem, bool) {
	affinities := genreAffinities(ratings)
	byRef := make(map[domain.TitleRef]*rankedCandidate)
	degraded := false
	for _, seed := range seeds {
		page, err := s.provider.Recommendations(ctx, seed.Title.MediaType, seed.Title.TmdbID)
		if err != nil {
			degraded = true
			continue
		}
		for index, candidate := range page.Results {
			ref := titleRef(candidate.Title)
			if excluded[ref] {
				continue
			}
			item := byRef[ref]
			if item == nil {
				item = &rankedCandidate{candidate: candidate, reason: "Похоже на «" + seed.Title.Title + "»"}
				byRef[ref] = item
			}
			rank := min(index+1, 20)
			item.score += seed.AvgScore * float64(21-rank) / 20
		}
	}
	ordered := make([]rankedCandidate, 0, len(byRef))
	for _, item := range byRef {
		for _, genre := range item.candidate.Title.Genres {
			item.score += 0.25 * math.Max(affinities[genre]-5, 0)
		}
		ordered = append(ordered, *item)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].score != ordered[j].score {
			return ordered[i].score > ordered[j].score
		}
		if ordered[i].candidate.Popularity != ordered[j].candidate.Popularity {
			return ordered[i].candidate.Popularity > ordered[j].candidate.Popularity
		}
		return refKey(titleRef(ordered[i].candidate.Title)) < refKey(titleRef(ordered[j].candidate.Title))
	})
	items := make([]domain.CatalogItem, 0, len(ordered))
	for _, item := range ordered {
		items = append(items, domain.CatalogItem{Title: item.candidate.Title, Reason: item.reason})
	}
	return items, degraded
}

func genreAffinities(ratings []domain.GenreRating) map[string]float64 {
	totals := make(map[string]float64)
	counts := make(map[string]int)
	for _, rating := range ratings {
		for _, genre := range rating.Genres {
			totals[genre] += rating.AvgScore
			counts[genre]++
		}
	}
	result := make(map[string]float64, len(totals))
	for genre, total := range totals {
		result[genre] = total / float64(counts[genre])
	}
	return result
}

func recommendationFingerprint(count int64, seeds []domain.RecommendationSeed, ratings []domain.GenreRating) string {
	data, _ := json.Marshal(struct {
		Count   int64                       `json:"count"`
		Seeds   []domain.RecommendationSeed `json:"seeds"`
		Ratings []domain.GenreRating        `json:"ratings"`
	}{count, seeds, ratings})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

func decodeDiscoverCursor(raw, mediaFilter string) (discoverCursor, error) {
	cursor := discoverCursor{Version: 1, Type: mediaFilter, Page: 1}
	if raw == "" {
		return cursor, nil
	}
	if err := decodeJSON(raw, &cursor); err != nil || cursor.Version != 1 || cursor.Type != mediaFilter || cursor.Page < 1 || cursor.Offset < 0 {
		return discoverCursor{}, ErrValidation
	}
	return cursor, nil
}

func decodeRecommendationCursor(raw, fingerprint string) (recommendationCursor, error) {
	cursor := recommendationCursor{Version: 1, Fingerprint: fingerprint}
	if raw == "" {
		return cursor, nil
	}
	if err := decodeJSON(raw, &cursor); err != nil || cursor.Version != 1 || cursor.Fingerprint != fingerprint || cursor.Offset < 0 {
		return recommendationCursor{}, ErrValidation
	}
	return cursor, nil
}

func encodeJSON(value any) string {
	data, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeJSON(raw string, value any) error {
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func normalizeMediaFilter(value string) string {
	if value == "" {
		return "all"
	}
	value = strings.ToLower(value)
	if value != "all" && value != "movie" && value != "tv" {
		return ""
	}
	return value
}

func normalizeLimit(value int) int {
	if value <= 0 {
		return defaultLimit
	}
	if value > maxLimit {
		return maxLimit
	}
	return value
}

func titleRef(title domain.Title) domain.TitleRef {
	return domain.TitleRef{TmdbID: title.TmdbID, MediaType: title.MediaType}
}

func refSet(refs []domain.TitleRef) map[domain.TitleRef]bool {
	set := make(map[domain.TitleRef]bool, len(refs))
	for _, ref := range refs {
		set[ref] = true
	}
	return set
}

func excludedWithItems(base map[domain.TitleRef]bool, items []domain.CatalogItem) map[domain.TitleRef]bool {
	result := make(map[domain.TitleRef]bool, len(base)+len(items))
	for ref, value := range base {
		result[ref] = value
	}
	for _, item := range items {
		result[titleRef(item.Title)] = true
	}
	return result
}

func refKey(ref domain.TitleRef) string {
	return fmt.Sprintf("%s:%020d", ref.MediaType, ref.TmdbID)
}
