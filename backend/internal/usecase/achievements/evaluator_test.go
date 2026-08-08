package achievements

import (
	"testing"
	"time"

	"movies/backend/internal/domain"
)

func TestEvaluatorCoreMetrics(t *testing.T) {
	evaluator, err := NewEvaluator(Definitions())
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 1, 17, 30, 0, 0, time.UTC) // 2026-08-02 in Novosibirsk.
	full := map[string]int{
		"story": 10, "characters": 10, "acting": 10, "direction": 10,
		"visuals": 10, "sound": 10, "atmosphere": 10,
	}
	snapshot := Snapshot{
		UserID: 1,
		RatingHistory: []RatingFact{
			{TitleID: 1, MediaType: domain.MediaTypeMovie, ReleaseYear: 1979, Genres: []string{"Драма"}, CreatedAt: base},
			{TitleID: 2, MediaType: domain.MediaTypeTV, ReleaseYear: 1999, Genres: []string{"Комедия"}, CreatedAt: base.Add(24 * time.Hour)},
			{TitleID: 3, MediaType: domain.MediaTypeMovie, ReleaseYear: 2020, Genres: []string{"Драма", "Фантастика"}, CreatedAt: base.Add(48 * time.Hour)},
		},
		Ratings: []RatingFact{
			{TitleID: 1, MediaType: domain.MediaTypeMovie, AvgScore: 10, Scores: full, CreatedAt: base, UpdatedAt: base},
			{TitleID: 2, MediaType: domain.MediaTypeTV, AvgScore: 2, Scores: map[string]int{"story": 2}, CreatedAt: base.Add(24 * time.Hour), UpdatedAt: base.Add(24 * time.Hour)},
			{TitleID: 3, MediaType: domain.MediaTypeMovie, AvgScore: 9, Scores: map[string]int{"story": 9}, CreatedAt: base.Add(48 * time.Hour), UpdatedAt: base.Add(48 * time.Hour)},
		},
	}
	evaluation := evaluator.Evaluate(snapshot, base.Add(72*time.Hour))
	assertMetric(t, evaluation, MetricRatingsTotal, 3)
	assertMetric(t, evaluation, MetricFullRatingsTotal, 1)
	assertMetric(t, evaluation, MetricRatingDayStreak, 3)
	assertMetric(t, evaluation, MetricRatingScoreContrast, 1)
	assertMetric(t, evaluation, MetricPerfectSevenRatingExists, 1)
	assertMetric(t, evaluation, MetricRatedGenresTotal, 3)
}

func TestEvaluatorSocialCommentsAndWatchlist(t *testing.T) {
	evaluator, err := NewEvaluator(Definitions())
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	full := map[string]int{"story": 7, "characters": 7, "acting": 7, "direction": 7, "visuals": 7, "sound": 7, "atmosphere": 7}
	snapshot := Snapshot{
		UserID:        1,
		RatingHistory: []RatingFact{{TitleID: 1, CreatedAt: base}},
		Ratings:       []RatingFact{{TitleID: 1, AvgScore: 7, Scores: full, CreatedAt: base, UpdatedAt: base}},
		Friends: []FriendFact{{
			User: domain.User{ID: 2}, AcceptedAt: base.Add(time.Hour),
			Ratings:   []RatingFact{{TitleID: 1, AvgScore: 7, Scores: full, CreatedAt: base.Add(2 * time.Hour), UpdatedAt: base.Add(2 * time.Hour)}},
			Watchlist: []WatchlistFact{{TitleID: 10, CreatedAt: base.Add(3 * time.Hour)}},
		}},
		Comments: []CommentFact{
			{ID: 1, TitleID: 1, UserID: 1, CreatedAt: base},
			{ID: 2, TitleID: 1, UserID: 2, ParentID: 1, CreatedAt: base.Add(time.Hour)},
		},
		Watchlist: []WatchlistFact{{TitleID: 10, CreatedAt: base}},
	}
	evaluation := evaluator.Evaluate(snapshot, base.Add(24*time.Hour))
	assertMetric(t, evaluation, MetricFriendsHighWater, 1)
	assertMetric(t, evaluation, MetricSharedTitlesTotal, 1)
	assertMetric(t, evaluation, MetricFriendExactFullRatingExists, 1)
	assertMetric(t, evaluation, MetricReceivedReplyExists, 1)
	assertMetric(t, evaluation, MetricFriendWatchlistMatchExists, 1)
}

func TestHotThreadNeedsTwoOtherAuthors(t *testing.T) {
	evaluator, err := NewEvaluator(Definitions())
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	comments := []CommentFact{{ID: 1, TitleID: 1, UserID: 1, CreatedAt: base}}
	for index := 0; index < 5; index++ {
		comments = append(comments, CommentFact{ID: int64(index + 2), TitleID: 1, UserID: 2, ParentID: 1, CreatedAt: base.Add(time.Duration(index+1) * time.Hour)})
	}
	evaluation := evaluator.Evaluate(Snapshot{UserID: 1, Comments: comments}, base.Add(24*time.Hour))
	assertMetric(t, evaluation, MetricRootDirectRepliesMax, 0)
	comments = append(comments, CommentFact{ID: 7, TitleID: 1, UserID: 3, ParentID: 1, CreatedAt: base.Add(6 * time.Hour)})
	evaluation = evaluator.Evaluate(Snapshot{UserID: 1, Comments: comments}, base.Add(24*time.Hour))
	assertMetric(t, evaluation, MetricRootDirectRepliesMax, 6)
}

func assertMetric(t *testing.T, evaluation Evaluation, code MetricCode, want int64) {
	t.Helper()
	if got := evaluation.Metrics[code].Value; got != want {
		t.Fatalf("metric %s = %d, want %d", code, got, want)
	}
}
