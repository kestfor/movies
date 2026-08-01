package catalog

import (
	"context"
	"testing"

	"movies/backend/internal/domain"
)

type fakeProvider struct {
	discover        map[domain.MediaType]domain.CatalogProviderPage
	recommendations map[domain.TitleRef]domain.CatalogProviderPage
}

func (f fakeProvider) Discover(_ context.Context, mediaType domain.MediaType, _ int) (domain.CatalogProviderPage, error) {
	return f.discover[mediaType], nil
}
func (f fakeProvider) Recommendations(_ context.Context, mediaType domain.MediaType, tmdbID int64) (domain.CatalogProviderPage, error) {
	return f.recommendations[domain.TitleRef{MediaType: mediaType, TmdbID: tmdbID}], nil
}

type fakeRepository struct {
	rated        []domain.TitleRef
	watchlisted  []domain.TitleRef
	count        int64
	seeds        []domain.RecommendationSeed
	genreRatings []domain.GenreRating
}

func (f fakeRepository) ListRatedRefs(context.Context, int64) ([]domain.TitleRef, error) {
	return f.rated, nil
}
func (f fakeRepository) ListRefs(context.Context, int64) ([]domain.TitleRef, error) {
	return f.watchlisted, nil
}
func (f fakeRepository) CountRatings(context.Context, int64) (int64, error) { return f.count, nil }
func (f fakeRepository) ListRecommendationSeeds(context.Context, int64) ([]domain.RecommendationSeed, error) {
	return f.seeds, nil
}
func (f fakeRepository) ListGenreRatings(context.Context, int64) ([]domain.GenreRating, error) {
	return f.genreRatings, nil
}

func TestDiscoverExcludesRatedKeepsWatchlistAndDoesNotSkipCursor(t *testing.T) {
	a := candidate(domain.MediaTypeMovie, 1, 30)
	b := candidate(domain.MediaTypeMovie, 2, 20)
	c := candidate(domain.MediaTypeMovie, 3, 10)
	repo := fakeRepository{
		rated:       []domain.TitleRef{{MediaType: domain.MediaTypeMovie, TmdbID: 1}},
		watchlisted: []domain.TitleRef{{MediaType: domain.MediaTypeMovie, TmdbID: 2}},
	}
	service := NewService(fakeProvider{discover: map[domain.MediaType]domain.CatalogProviderPage{
		domain.MediaTypeMovie: {Page: 1, TotalPages: 1, Results: []domain.CatalogCandidate{a, b, c}},
	}}, repo)
	page, err := service.Discover(context.Background(), 1, "movie", "", 1)
	if err != nil || len(page.Items) != 1 || page.Items[0].Title.TmdbID != 2 || !page.Items[0].InWatchlist || page.NextCursor == "" {
		t.Fatalf("unexpected first page=%#v err=%v", page, err)
	}
	next, err := service.Discover(context.Background(), 1, "movie", page.NextCursor, 1)
	if err != nil || len(next.Items) != 1 || next.Items[0].Title.TmdbID != 3 {
		t.Fatalf("cursor skipped item: page=%#v err=%v", next, err)
	}
}

func TestRecommendationsAreRankedDeduplicatedAndExcluded(t *testing.T) {
	seedA := domain.RecommendationSeed{RatingID: 1, Title: domain.Title{TmdbID: 10, MediaType: domain.MediaTypeMovie, Title: "Seed A"}, AvgScore: 9}
	seedB := domain.RecommendationSeed{RatingID: 2, Title: domain.Title{TmdbID: 20, MediaType: domain.MediaTypeMovie, Title: "Seed B"}, AvgScore: 8}
	shared := candidate(domain.MediaTypeMovie, 100, 10)
	shared.Title.Genres = []string{"Фантастика"}
	excluded := candidate(domain.MediaTypeMovie, 200, 100)
	repo := fakeRepository{
		count: 3, seeds: []domain.RecommendationSeed{seedA, seedB},
		genreRatings: []domain.GenreRating{{AvgScore: 9, Genres: []string{"Фантастика"}}},
		rated:        []domain.TitleRef{{MediaType: domain.MediaTypeMovie, TmdbID: 200}},
	}
	provider := fakeProvider{
		discover: map[domain.MediaType]domain.CatalogProviderPage{
			domain.MediaTypeMovie: {TotalPages: 1}, domain.MediaTypeTV: {TotalPages: 1},
		},
		recommendations: map[domain.TitleRef]domain.CatalogProviderPage{
			{MediaType: domain.MediaTypeMovie, TmdbID: 10}: {Results: []domain.CatalogCandidate{shared, excluded}},
			{MediaType: domain.MediaTypeMovie, TmdbID: 20}: {Results: []domain.CatalogCandidate{shared}},
		},
	}
	page, err := NewService(provider, repo).Recommendations(context.Background(), 1, "", 20)
	if err != nil || !page.Personalized || len(page.Items) != 1 || page.Items[0].Title.TmdbID != 100 {
		t.Fatalf("unexpected recommendations=%#v err=%v", page, err)
	}
}

func candidate(mediaType domain.MediaType, id int64, popularity float64) domain.CatalogCandidate {
	return domain.CatalogCandidate{Title: domain.Title{TmdbID: id, MediaType: mediaType, Title: "Title"}, Popularity: popularity}
}
