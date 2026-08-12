package achievements

import (
	"testing"
	"time"

	"movies/backend/internal/domain"
)

func TestProspectiveMetrics(t *testing.T) {
	base := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	fullHigh := scores(8, 8, 8, 8, 8, 8, 8)
	fullLow := scores(5, 5, 5, 5, 5, 5, 5)
	friends := []FriendFact{
		{User: domain.User{ID: 2}, AcceptedAt: base.Add(-time.Hour)},
		{User: domain.User{ID: 3}, AcceptedAt: base.Add(-time.Hour)},
	}

	tests := []struct {
		name     string
		metric   MetricCode
		target   int64
		snapshot Snapshot
	}{
		{"no weak links", MetricNoWeakLinksTotal, 3, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 8, fullHigh), rating(2, 1, 2, base.Add(time.Hour), 8, fullHigh), rating(3, 1, 3, base.Add(2*time.Hour), 8, fullHigh))},
		{"no concessions", MetricNoConcessionsTotal, 3, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 5, fullLow), rating(2, 1, 2, base.Add(time.Hour), 5, fullLow), rating(3, 1, 3, base.Add(2*time.Hour), 5, fullLow))},
		{"contrast cut", MetricContrastCutExists, 1, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 5.5, scores(1, 10, 4, 5, 6, 7, 5)))},
		{"signature touch", MetricSignatureTouchMax, 3, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 7, scores(10, 7, 7, 7, 7, 7, 7)), rating(2, 1, 2, base.Add(time.Hour), 7, scores(10, 6, 7, 7, 7, 7, 7)), rating(3, 1, 3, base.Add(2*time.Hour), 7, scores(10, 5, 6, 7, 7, 7, 7)))},
		{"three eras", MetricThreeErasSameDayMax, 3, snapshotWithFacts(base, friends,
			ratingMeta(1, 1, 1, base, 7, fullHigh, 1984, domain.MediaTypeMovie, "драма"), ratingMeta(2, 1, 2, base.Add(time.Hour), 7, fullHigh, 1994, domain.MediaTypeMovie, "драма"), ratingMeta(3, 1, 3, base.Add(2*time.Hour), 7, fullHigh, 2004, domain.MediaTypeMovie, "драма"))},
		{"genre decades", MetricGenreDecadesMax, 4, snapshotWithFacts(base, friends,
			ratingMeta(1, 1, 1, base, 7, fullHigh, 1974, domain.MediaTypeMovie, "драма"), ratingMeta(2, 1, 2, base.Add(time.Hour), 7, fullHigh, 1984, domain.MediaTypeMovie, "драма"), ratingMeta(3, 1, 3, base.Add(2*time.Hour), 7, fullHigh, 1994, domain.MediaTypeMovie, "драма"), ratingMeta(4, 1, 4, base.Add(3*time.Hour), 7, fullHigh, 2004, domain.MediaTypeMovie, "драма"))},
		{"parallel years", MetricParallelYearsTotal, 2, snapshotWithFacts(base, friends,
			ratingMeta(1, 1, 1, base, 7, fullHigh, 2020, domain.MediaTypeMovie, "драма"), ratingMeta(2, 1, 2, base.Add(time.Hour), 7, fullHigh, 2020, domain.MediaTypeTV, "драма"), ratingMeta(3, 1, 3, base.Add(2*time.Hour), 7, fullHigh, 2021, domain.MediaTypeMovie, "драма"), ratingMeta(4, 1, 4, base.Add(3*time.Hour), 7, fullHigh, 2021, domain.MediaTypeTV, "драма"))},
		{"five notches", MetricFiveNotchesTotal, 5, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 2, fullLow), rating(2, 1, 2, base.Add(time.Hour), 4, fullLow), rating(3, 1, 3, base.Add(2*time.Hour), 6, fullLow), rating(4, 1, 4, base.Add(3*time.Hour), 8, fullHigh), rating(5, 1, 5, base.Add(4*time.Hour), 9, fullHigh))},
		{"middle ground", MetricMiddleGroundExists, 1, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 5, fullLow), rating(2, 2, 1, base.Add(time.Hour), 2, fullLow), rating(3, 3, 1, base.Add(2*time.Hour), 8, fullHigh))},
		{"lone dissenter", MetricLoneDissenterExists, 1, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 2, fullLow), rating(2, 2, 1, base.Add(time.Hour), 8, fullHigh), rating(3, 3, 1, base.Add(2*time.Hour), 8.5, fullHigh))},
		{"together and apart", MetricTogetherAndApartMax, 2, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 7, fullLow), rating(2, 2, 1, base, 7.2, fullLow), rating(3, 1, 2, base.Add(time.Hour), 8, fullHigh), rating(4, 2, 2, base.Add(time.Hour), 8.3, fullHigh),
			rating(5, 1, 3, base.Add(2*time.Hour), 2, fullLow), rating(6, 2, 3, base.Add(2*time.Hour), 7, fullHigh), rating(7, 1, 4, base.Add(3*time.Hour), 9, fullHigh), rating(8, 2, 4, base.Add(3*time.Hour), 4, fullLow))},
		{"rated round table", MetricRatedRoundTableExists, 1, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 7, fullHigh), rating(2, 2, 1, base, 7, fullHigh), rating(3, 3, 1, base, 7, fullHigh), comment(4, 1, 1, 101, 0, base.Add(time.Hour)), comment(5, 2, 1, 102, 101, base.Add(time.Hour)), comment(6, 3, 1, 103, 101, base.Add(time.Hour)))},
		{"critic duet", MetricCriticDuetTotal, 2, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 7, fullHigh), rating(2, 2, 1, base, 7, fullHigh), comment(3, 1, 1, 101, 0, base), comment(4, 2, 1, 102, 101, base),
			rating(5, 1, 2, base.Add(time.Hour), 7, fullHigh), rating(6, 2, 2, base.Add(time.Hour), 7, fullHigh), comment(7, 1, 2, 103, 0, base.Add(time.Hour)), comment(8, 2, 2, 104, 103, base.Add(time.Hour)))},
		{"after credits", MetricAfterCreditsTotal, 2, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 7, fullHigh), comment(2, 1, 1, 101, 0, base.Add(49*time.Hour)), rating(3, 1, 2, base.Add(time.Hour), 7, fullHigh), comment(4, 1, 2, 102, 0, base.Add(50*time.Hour)))},
		{"council watchlist", MetricCouncilWatchlistMax, 2, Snapshot{UserID: 1, Friends: []FriendFact{{User: domain.User{ID: 2}, AcceptedAt: base.Add(-time.Hour), Watchlist: []WatchlistFact{{TitleID: 1, CreatedAt: base.Add(time.Hour)}}}, {User: domain.User{ID: 3}, AcceptedAt: base.Add(-time.Hour), Watchlist: []WatchlistFact{{TitleID: 1, CreatedAt: base.Add(2 * time.Hour)}}}}, Watchlist: []WatchlistFact{{TitleID: 1, CreatedAt: base}}}},
		{"opening night", MetricOpeningNightExists, 1, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 7, fullHigh), rating(2, 2, 1, base.Add(3*time.Hour), 7, fullHigh))},
		{"chain reaction", MetricChainReactionMax, 2, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 7, fullHigh), rating(2, 2, 1, base.Add(24*time.Hour), 7, fullHigh), rating(3, 3, 1, base.Add(48*time.Hour), 7, fullHigh))},
		{"trusted recommendation", MetricTrustedRecommendationExists, 1, snapshotWithFacts(base, friends,
			rating(1, 2, 1, base, 9, fullHigh), watch(2, 1, 1, base.Add(time.Hour)), rating(3, 1, 1, base.Add(2*24*time.Hour), 8, fullHigh))},
		{"changed mind", MetricChangedMindExists, 1, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 5, fullLow), comment(2, 1, 1, 101, 0, base.Add(time.Hour)), comment(3, 2, 1, 102, 101, base.Add(2*time.Hour)), ratingUpdate(4, 1, 1, base.Add(3*time.Hour), 7, 5, fullHigh, fullLow))},
		{"patient ticket", MetricPatientTicketExists, 1, snapshotWithFacts(base, friends,
			watch(1, 1, 1, base), rating(2, 1, 1, base.Add(8*24*time.Hour), 7, fullHigh))},
		{"clear queue", MetricClearQueueMax, 2, snapshotWithFacts(base, friends,
			watch(1, 1, 1, base), watch(2, 1, 2, base), rating(3, 1, 1, base.Add(time.Hour), 7, fullHigh), rating(4, 1, 2, base.Add(24*time.Hour), 7, fullHigh))},
		{"agreed session", MetricAgreedSessionExists, 1, snapshotWithFacts(base, friends,
			watch(1, 1, 1, base), watch(2, 2, 1, base.Add(time.Hour)), rating(3, 1, 1, base.Add(24*time.Hour), 7, fullHigh), rating(4, 2, 1, base.Add(48*time.Hour), 7, fullHigh))},
		{"relay", MetricRelayExists, 1, snapshotWithFacts(base, friends,
			rating(1, 2, 1, base, 7, fullHigh), rating(2, 1, 1, base.Add(24*time.Hour), 7, fullHigh), rating(3, 3, 1, base.Add(48*time.Hour), 7, fullHigh))},
		{"word for word", MetricWordForWordMax, 4, snapshotWithFacts(base, friends,
			comment(1, 1, 1, 101, 0, base), comment(2, 2, 1, 102, 101, base.Add(time.Hour)), comment(3, 1, 1, 103, 102, base.Add(2*time.Hour)), comment(4, 2, 1, 104, 103, base.Add(3*time.Hour)))},
		{"discuss then rate", MetricDiscussThenRateExists, 1, snapshotWithFacts(base, friends,
			comment(1, 1, 1, 101, 0, base), comment(2, 2, 1, 102, 101, base.Add(time.Hour)), rating(3, 1, 1, base.Add(2*time.Hour), 7, fullHigh), rating(4, 2, 1, base.Add(3*time.Hour), 7.5, fullHigh))},
		{"thread resurrection", MetricThreadResurrectionExists, 1, snapshotWithFacts(base, friends,
			comment(1, 2, 1, 101, 0, base), comment(2, 1, 1, 102, 101, base.Add(15*24*time.Hour)), comment(3, 2, 1, 103, 102, base.Add(15*24*time.Hour+time.Hour)))},
		{"two good tips", MetricGoodTipsMax, 2, snapshotWithFacts(base, friends,
			rating(1, 2, 1, base, 9, fullHigh), rating(2, 1, 1, base.Add(24*time.Hour), 8, fullHigh), rating(3, 3, 2, base.Add(2*time.Hour), 9, fullHigh), rating(4, 1, 2, base.Add(48*time.Hour), 8, fullHigh))},
		{"mood arc", MetricMoodArcExists, 1, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 4, fullLow), rating(2, 1, 2, base.Add(time.Hour), 6, fullLow), rating(3, 1, 3, base.Add(2*time.Hour), 9, fullHigh))},
		{"deliberate rating", MetricDeliberateRatingExists, 1, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 7, map[string]int{"story": 7, "acting": 7}), ratingUpdate(2, 1, 1, base.Add(13*time.Hour), 8, 7, fullHigh, map[string]int{"story": 7, "acting": 7}))},
		{"shared finale", MetricSharedFinaleExists, 1, snapshotWithFacts(base, friends,
			rating(1, 1, 1, base, 7, fullHigh), rating(2, 2, 1, base.Add(time.Hour), 8, fullHigh), rating(3, 3, 1, base.Add(2*time.Hour), 8.5, fullHigh))},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			evaluator, err := NewEvaluator(Definitions())
			if err != nil {
				t.Fatal(err)
			}
			introduced := introducedAt(base)
			evaluation := evaluator.EvaluateWithIntroduced(test.snapshot, base.Add(60*24*time.Hour), introduced)
			if got := evaluation.Metrics[test.metric].Value; got < test.target {
				t.Fatalf("metric %s = %d, want at least %d", test.metric, got, test.target)
			}
		})
	}
}

func TestProspectiveMetricsRespectIntroductionBoundary(t *testing.T) {
	base := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	evaluator, err := NewEvaluator(Definitions())
	if err != nil {
		t.Fatal(err)
	}
	full := scores(8, 8, 8, 8, 8, 8, 8)
	snapshot := snapshotWithFacts(base, nil,
		rating(1, 1, 1, base.Add(-time.Nanosecond), 8, full),
		rating(2, 1, 2, base, 8, full),
	)
	evaluation := evaluator.EvaluateWithIntroduced(snapshot, base.Add(time.Hour), introducedAt(base))
	if got := evaluation.Metrics[MetricNoWeakLinksTotal].Value; got != 1 {
		t.Fatalf("metric at introduction boundary = %d, want 1", got)
	}
	withoutIntroduction := evaluator.Evaluate(snapshot, base.Add(time.Hour))
	if got := withoutIntroduction.Metrics[MetricNoWeakLinksTotal].Value; got != 0 {
		t.Fatalf("prospective metric without introduction = %d, want 0", got)
	}
}

func TestBackfillExcludesProspectiveCandidates(t *testing.T) {
	candidates := []Candidate{
		{Definition: Definition{Code: "legacy", AwardPolicy: AwardPolicyLifetime}},
		{Definition: Definition{Code: "new", AwardPolicy: AwardPolicySinceIntroduction}},
	}
	filtered := candidatesForSource(candidates, AwardSourceBackfill)
	if len(filtered) != 1 || filtered[0].Definition.Code != "legacy" {
		t.Fatalf("backfill candidates = %#v, want only legacy", filtered)
	}
	if got := candidatesForSource(candidates, AwardSourceReconcile); len(got) != 2 {
		t.Fatalf("reconcile candidates = %d, want 2", len(got))
	}
}

func snapshotWithFacts(_ time.Time, friends []FriendFact, facts ...ActionFact) Snapshot {
	return Snapshot{UserID: 1, Friends: friends, ActionFacts: facts}
}

func introducedAt(at time.Time) map[string]time.Time {
	result := make(map[string]time.Time)
	for _, definition := range Definitions() {
		if definition.AwardPolicy == AwardPolicySinceIntroduction {
			result[definition.Code] = at
		}
	}
	return result
}

func rating(id, actorID, titleID int64, at time.Time, avg float64, values map[string]int) ActionFact {
	return ratingMeta(id, actorID, titleID, at, avg, values, 2020, domain.MediaTypeMovie, "драма")
}

func ratingMeta(id, actorID, titleID int64, at time.Time, avg float64, values map[string]int, year int, mediaType domain.MediaType, genres ...string) ActionFact {
	return ActionFact{ID: id, Kind: ActionFactRatingCreated, ActorID: actorID, TitleID: titleID, EntityID: id, AvgScore: avg, Scores: values, OccurredAt: at, ReleaseYear: year, MediaType: mediaType, Genres: genres}
}

func ratingUpdate(id, actorID, titleID int64, at time.Time, avg, previous float64, values, previousValues map[string]int) ActionFact {
	return ActionFact{ID: id, Kind: ActionFactRatingUpdated, ActorID: actorID, TitleID: titleID, EntityID: titleID, AvgScore: avg, PreviousAvgScore: previous, Scores: values, PreviousScores: previousValues, OccurredAt: at, ReleaseYear: 2020, MediaType: domain.MediaTypeMovie, Genres: []string{"драма"}}
}

func comment(id, actorID, titleID, entityID, parentID int64, at time.Time) ActionFact {
	return ActionFact{ID: id, Kind: ActionFactCommentCreated, ActorID: actorID, TitleID: titleID, EntityID: entityID, ParentEntityID: parentID, OccurredAt: at}
}

func watch(id, actorID, titleID int64, at time.Time) ActionFact {
	return ActionFact{ID: id, Kind: ActionFactWatchlistAdded, ActorID: actorID, TitleID: titleID, OccurredAt: at}
}

func scores(story, characters, acting, direction, visuals, sound, atmosphere int) map[string]int {
	return map[string]int{
		"story": story, "characters": characters, "acting": acting, "direction": direction,
		"visuals": visuals, "sound": sound, "atmosphere": atmosphere,
	}
}
