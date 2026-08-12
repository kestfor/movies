package achievements

import (
	"math"
	"sort"
	"strings"
	"time"

	"movies/backend/internal/domain"
)

const AchievementTimezone = "Asia/Novosibirsk"

type RatingFact struct {
	TitleID     int64
	MediaType   domain.MediaType
	ReleaseYear int
	Genres      []string
	AvgScore    float64
	Scores      map[string]int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FriendFact struct {
	User       domain.User
	AcceptedAt time.Time
	Ratings    []RatingFact
	Watchlist  []WatchlistFact
}

type CommentFact struct {
	ID        int64
	TitleID   int64
	UserID    int64
	ParentID  int64
	CreatedAt time.Time
}

type WatchlistFact struct {
	TitleID   int64
	CreatedAt time.Time
}

type ActionFactKind string

const (
	ActionFactRatingCreated  ActionFactKind = "rating_created"
	ActionFactRatingUpdated  ActionFactKind = "rating_updated"
	ActionFactCommentCreated ActionFactKind = "comment_created"
	ActionFactWatchlistAdded ActionFactKind = "watchlist_added"
)

type ActionFact struct {
	ID               int64
	Kind             ActionFactKind
	ActorID          int64
	TitleID          int64
	EntityID         int64
	ParentEntityID   int64
	MediaType        domain.MediaType
	ReleaseYear      int
	Genres           []string
	AvgScore         float64
	PreviousAvgScore float64
	Scores           map[string]int
	PreviousScores   map[string]int
	OccurredAt       time.Time
}

type Snapshot struct {
	UserID        int64
	RatingHistory []RatingFact
	Ratings       []RatingFact
	Friends       []FriendFact
	Comments      []CommentFact
	Watchlist     []WatchlistFact
	ActionFacts   []ActionFact
}

type MetricResult struct {
	Value     int64
	ReachedAt map[int64]time.Time
}

type Candidate struct {
	Definition Definition
	EarnedAt   time.Time
}

type Evaluation struct {
	Metrics    map[MetricCode]MetricResult
	Candidates []Candidate
}

type Evaluator struct {
	definitions []Definition
	location    *time.Location
}

func NewEvaluator(definitions []Definition) (*Evaluator, error) {
	if err := ValidateCatalog(definitions); err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(AchievementTimezone)
	if err != nil {
		return nil, err
	}
	return &Evaluator{definitions: append([]Definition(nil), definitions...), location: location}, nil
}

func (e *Evaluator) Evaluate(snapshot Snapshot, fallback time.Time) Evaluation {
	return e.evaluate(snapshot, fallback, nil)
}

func (e *Evaluator) EvaluateWithIntroduced(snapshot Snapshot, fallback time.Time, introduced map[string]time.Time) Evaluation {
	return e.evaluate(snapshot, fallback, introduced)
}

func (e *Evaluator) evaluate(snapshot Snapshot, fallback time.Time, introduced map[string]time.Time) Evaluation {
	metrics := make(map[MetricCode]MetricResult, len(allMetrics()))
	e.ratingMetrics(snapshot, metrics)
	e.socialMetrics(snapshot, metrics)
	e.commentMetrics(snapshot, metrics)
	e.watchlistMetrics(snapshot, metrics)
	e.prospectiveMetrics(snapshot, metrics, introduced)

	candidates := make([]Candidate, 0)
	for _, definition := range e.definitions {
		if definition.AwardPolicy == AwardPolicySinceIntroduction && introduced[definition.Code].IsZero() {
			continue
		}
		result := metrics[definition.Metric]
		if result.Value < definition.Target {
			continue
		}
		earnedAt := result.ReachedAt[definition.Target]
		if earnedAt.IsZero() {
			earnedAt = fallback
		}
		candidates = append(candidates, Candidate{Definition: definition, EarnedAt: earnedAt})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].EarnedAt.Equal(candidates[j].EarnedAt) {
			return candidates[i].Definition.SortOrder < candidates[j].Definition.SortOrder
		}
		return candidates[i].EarnedAt.Before(candidates[j].EarnedAt)
	})
	return Evaluation{Metrics: metrics, Candidates: candidates}
}

func (e *Evaluator) ratingMetrics(snapshot Snapshot, metrics map[MetricCode]MetricResult) {
	history := uniqueRatings(snapshot.RatingHistory)
	current := uniqueRatings(snapshot.Ratings)

	metrics[MetricRatingsTotal] = countResult(ratingDates(history, false))
	metrics[MetricRatedMoviesTotal] = countResult(filterRatingDates(history, func(item RatingFact) bool {
		return item.MediaType == domain.MediaTypeMovie
	}))
	metrics[MetricRatedTVTotal] = countResult(filterRatingDates(history, func(item RatingFact) bool {
		return item.MediaType == domain.MediaTypeTV
	}))
	metrics[MetricRatedBefore1980Total] = countResult(filterRatingDates(history, func(item RatingFact) bool {
		return item.ReleaseYear > 0 && item.ReleaseYear < 1980
	}))
	metrics[MetricRatedBefore2000Total] = countResult(filterRatingDates(history, func(item RatingFact) bool {
		return item.ReleaseYear > 0 && item.ReleaseYear < 2000
	}))

	movieDates := filterRatingDates(history, func(item RatingFact) bool { return item.MediaType == domain.MediaTypeMovie })
	tvDates := filterRatingDates(history, func(item RatingFact) bool { return item.MediaType == domain.MediaTypeTV })
	metrics[MetricRatedMediaBalance] = pairedCountResult(movieDates, tvDates)

	fullDates := make([]time.Time, 0)
	genreDates := make(map[string][]time.Time)
	genreFirst := make(map[string]time.Time)
	decadeFirst := make(map[int]time.Time)
	perfectAt := time.Time{}
	sameScoreAt := time.Time{}
	maxDistinct := int64(0)
	distinctReached := make(map[int64]time.Time)
	lowAt, highAt := time.Time{}, time.Time{}
	for _, rating := range current {
		updatedAt := effectiveUpdatedAt(rating)
		if isFullRating(rating.Scores) {
			fullDates = append(fullDates, updatedAt)
			distinct := distinctScoreCount(rating.Scores)
			for value := int64(1); value <= int64(distinct); value++ {
				if existing := distinctReached[value]; existing.IsZero() || updatedAt.Before(existing) {
					distinctReached[value] = updatedAt
				}
			}
			if int64(distinct) > maxDistinct {
				maxDistinct = int64(distinct)
			}
			if allScoresEqual(rating.Scores) && (sameScoreAt.IsZero() || updatedAt.Before(sameScoreAt)) {
				sameScoreAt = updatedAt
			}
			if allScoresEqualTo(rating.Scores, 10) && (perfectAt.IsZero() || updatedAt.Before(perfectAt)) {
				perfectAt = updatedAt
			}
		}
		if rating.AvgScore <= 3 && (lowAt.IsZero() || updatedAt.Before(lowAt)) {
			lowAt = updatedAt
		}
		if rating.AvgScore >= 9 && (highAt.IsZero() || updatedAt.Before(highAt)) {
			highAt = updatedAt
		}
	}
	for _, rating := range history {
		for _, genre := range normalizedGenres(rating.Genres) {
			genreDates[genre] = append(genreDates[genre], rating.CreatedAt)
			if genreFirst[genre].IsZero() || rating.CreatedAt.Before(genreFirst[genre]) {
				genreFirst[genre] = rating.CreatedAt
			}
		}
		if rating.ReleaseYear > 0 {
			decade := rating.ReleaseYear / 10 * 10
			if decadeFirst[decade].IsZero() || rating.CreatedAt.Before(decadeFirst[decade]) {
				decadeFirst[decade] = rating.CreatedAt
			}
		}
	}

	metrics[MetricFullRatingsTotal] = countResult(fullDates)
	metrics[MetricRatingsSameGenreMax] = maxGroupResult(genreDates)
	metrics[MetricRatedGenresTotal] = distinctSetResult(genreFirst)
	metrics[MetricRatedReleaseDecadesTotal] = distinctIntSetResult(decadeFirst)
	metrics[MetricFullRatingDistinctScoreMax] = MetricResult{Value: maxDistinct, ReachedAt: distinctReached}
	metrics[MetricFullRatingSameScoreExists] = boolResult(sameScoreAt)
	metrics[MetricPerfectSevenRatingExists] = boolResult(perfectAt)
	if !lowAt.IsZero() && !highAt.IsZero() {
		metrics[MetricRatingScoreContrast] = boolResult(maxTime(lowAt, highAt))
	} else {
		metrics[MetricRatingScoreContrast] = zeroResult()
	}
	metrics[MetricRatingDayStreak] = e.streakResult(history)
	metrics[MetricRatingSameDayMax] = e.sameDayResult(history)
}

func (e *Evaluator) socialMetrics(snapshot Snapshot, metrics map[MetricCode]MetricResult) {
	friendDates := make([]time.Time, 0, len(snapshot.Friends))
	selfRatings := ratingMap(snapshot.Ratings)
	sharedDates := make(map[int64]time.Time)
	friendSharedDates := make(map[int64]time.Time)
	followingDates := make(map[int64]time.Time)
	pioneerDates := make(map[int64]time.Time)
	closeGroups := make(map[string][]time.Time)
	farGroups := make(map[string][]time.Time)
	titleFriendDates := make(map[int64][]time.Time)
	titleSameAvgDates := make(map[int64][]time.Time)
	exactAt := time.Time{}

	for _, friend := range snapshot.Friends {
		friendDates = append(friendDates, friend.AcceptedAt)
		friendRatings := ratingMap(friend.Ratings)
		for titleID, own := range selfRatings {
			other, ok := friendRatings[titleID]
			if !ok {
				continue
			}
			eligibleAt := maxTime(friend.AcceptedAt, own.CreatedAt, other.CreatedAt)
			if sharedDates[titleID].IsZero() || eligibleAt.Before(sharedDates[titleID]) {
				sharedDates[titleID] = eligibleAt
			}
			if friendSharedDates[friend.User.ID].IsZero() || eligibleAt.Before(friendSharedDates[friend.User.ID]) {
				friendSharedDates[friend.User.ID] = eligibleAt
			}
			titleFriendDates[titleID] = append(titleFriendDates[titleID], eligibleAt)
			difference := math.Abs(own.AvgScore - other.AvgScore)
			groupKey := stringKey(friend.User.ID)
			if difference <= 0.5+1e-9 {
				closeGroups[groupKey] = append(closeGroups[groupKey], maxTime(friend.AcceptedAt, effectiveUpdatedAt(own), effectiveUpdatedAt(other)))
			}
			if difference >= 3-1e-9 {
				farGroups[groupKey] = append(farGroups[groupKey], maxTime(friend.AcceptedAt, effectiveUpdatedAt(own), effectiveUpdatedAt(other)))
			}
			if math.Abs(difference) < 1e-9 {
				titleSameAvgDates[titleID] = append(titleSameAvgDates[titleID], maxTime(friend.AcceptedAt, effectiveUpdatedAt(own), effectiveUpdatedAt(other)))
			}
			if other.CreatedAt.Before(own.CreatedAt) {
				if followingDates[titleID].IsZero() || eligibleAt.Before(followingDates[titleID]) {
					followingDates[titleID] = eligibleAt
				}
			}
			if own.CreatedAt.Before(other.CreatedAt) {
				if pioneerDates[titleID].IsZero() || eligibleAt.Before(pioneerDates[titleID]) {
					pioneerDates[titleID] = eligibleAt
				}
			}
			if exactScoreMaps(own.Scores, other.Scores) {
				candidateAt := maxTime(friend.AcceptedAt, effectiveUpdatedAt(own), effectiveUpdatedAt(other))
				if exactAt.IsZero() || candidateAt.Before(exactAt) {
					exactAt = candidateAt
				}
			}
		}
	}

	metrics[MetricFriendsHighWater] = countResult(friendDates)
	metrics[MetricSharedTitlesTotal] = distinctIDResult(sharedDates)
	metrics[MetricFriendsWithSharedTitleTotal] = distinctIDResult(friendSharedDates)
	metrics[MetricFriendEarlierTitlesTotal] = distinctIDResult(followingDates)
	metrics[MetricUserEarlierTitlesTotal] = distinctIDResult(pioneerDates)
	metrics[MetricFriendCloseRatingsMax] = maxGroupResult(closeGroups)
	metrics[MetricFriendFarRatingsMax] = maxGroupResult(farGroups)
	metrics[MetricTitleFriendRatersMax] = maxGroupResultByID(titleFriendDates)
	metrics[MetricTitleSameAvgFriendCountMax] = maxGroupResultByID(titleSameAvgDates)
	metrics[MetricFriendExactFullRatingExists] = boolResult(exactAt)
}

func (e *Evaluator) commentMetrics(snapshot Snapshot, metrics map[MetricCode]MetricResult) {
	byID := make(map[int64]CommentFact, len(snapshot.Comments))
	for _, comment := range snapshot.Comments {
		byID[comment.ID] = comment
	}
	authoredDates := make([]time.Time, 0)
	replyDates := make([]time.Time, 0)
	commentedTitles := make(map[int64]time.Time)
	receivedAt := time.Time{}
	replierFirst := make(map[int64]time.Time)
	directReplies := make(map[int64][]CommentFact)
	for _, comment := range snapshot.Comments {
		if comment.UserID == snapshot.UserID {
			authoredDates = append(authoredDates, comment.CreatedAt)
			if commentedTitles[comment.TitleID].IsZero() || comment.CreatedAt.Before(commentedTitles[comment.TitleID]) {
				commentedTitles[comment.TitleID] = comment.CreatedAt
			}
			if comment.ParentID != 0 {
				replyDates = append(replyDates, comment.CreatedAt)
			}
		}
		if comment.ParentID == 0 {
			continue
		}
		parent, ok := byID[comment.ParentID]
		if !ok || parent.UserID != snapshot.UserID || comment.UserID == snapshot.UserID {
			continue
		}
		if receivedAt.IsZero() || comment.CreatedAt.Before(receivedAt) {
			receivedAt = comment.CreatedAt
		}
		if replierFirst[comment.UserID].IsZero() || comment.CreatedAt.Before(replierFirst[comment.UserID]) {
			replierFirst[comment.UserID] = comment.CreatedAt
		}
		if parent.ParentID == 0 {
			directReplies[parent.ID] = append(directReplies[parent.ID], comment)
		}
	}

	metrics[MetricCommentsTotal] = countResult(authoredDates)
	metrics[MetricCommentedTitlesTotal] = distinctIDResult(commentedTitles)
	metrics[MetricRepliesAuthoredTotal] = countResult(replyDates)
	metrics[MetricReceivedReplyExists] = boolResult(receivedAt)
	metrics[MetricDistinctRepliersTotal] = distinctIDResult(replierFirst)
	metrics[MetricRootDirectRepliesMax] = hotThreadResult(directReplies)

	ratedTitles := make(map[int64]time.Time)
	for _, rating := range uniqueRatings(snapshot.RatingHistory) {
		ratedTitles[rating.TitleID] = rating.CreatedAt
	}
	both := make(map[int64]time.Time)
	for titleID, commentAt := range commentedTitles {
		if ratingAt, ok := ratedTitles[titleID]; ok {
			both[titleID] = maxTime(commentAt, ratingAt)
		}
	}
	metrics[MetricRatedAndCommentedTitlesTotal] = distinctIDResult(both)
}

func (e *Evaluator) watchlistMetrics(snapshot Snapshot, metrics map[MetricCode]MetricResult) {
	dates := make([]time.Time, 0, len(snapshot.Watchlist))
	items := make(map[int64]time.Time, len(snapshot.Watchlist))
	for _, item := range snapshot.Watchlist {
		dates = append(dates, item.CreatedAt)
		items[item.TitleID] = item.CreatedAt
	}
	metrics[MetricWatchlistHighWater] = countResult(dates)
	matchAt := time.Time{}
	for _, friend := range snapshot.Friends {
		for _, item := range friend.Watchlist {
			ownAt, ok := items[item.TitleID]
			if !ok {
				continue
			}
			candidateAt := maxTime(friend.AcceptedAt, ownAt, item.CreatedAt)
			if matchAt.IsZero() || candidateAt.Before(matchAt) {
				matchAt = candidateAt
			}
		}
	}
	metrics[MetricFriendWatchlistMatchExists] = boolResult(matchAt)
}

func (e *Evaluator) streakResult(ratings []RatingFact) MetricResult {
	dayAt := make(map[string]time.Time)
	for _, rating := range uniqueRatings(ratings) {
		local := rating.CreatedAt.In(e.location)
		key := local.Format("2006-01-02")
		if dayAt[key].IsZero() || rating.CreatedAt.Before(dayAt[key]) {
			dayAt[key] = rating.CreatedAt
		}
	}
	days := make([]time.Time, 0, len(dayAt))
	for _, at := range dayAt {
		days = append(days, at)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	result := zeroResult()
	current := int64(0)
	var previous time.Time
	for _, at := range days {
		localDay := dateOnly(at.In(e.location))
		if !previous.IsZero() && localDay.Equal(previous.AddDate(0, 0, 1)) {
			current++
		} else {
			current = 1
		}
		previous = localDay
		if current > result.Value {
			for value := result.Value + 1; value <= current; value++ {
				result.ReachedAt[value] = at
			}
			result.Value = current
		}
	}
	return result
}

func (e *Evaluator) sameDayResult(ratings []RatingFact) MetricResult {
	groups := make(map[string][]time.Time)
	for _, rating := range uniqueRatings(ratings) {
		key := rating.CreatedAt.In(e.location).Format("2006-01-02")
		groups[key] = append(groups[key], rating.CreatedAt)
	}
	return maxGroupResult(groups)
}

func uniqueRatings(items []RatingFact) []RatingFact {
	byTitle := make(map[int64]RatingFact, len(items))
	for _, item := range items {
		current, ok := byTitle[item.TitleID]
		if !ok || item.CreatedAt.Before(current.CreatedAt) {
			byTitle[item.TitleID] = item
			continue
		}
		if len(current.Scores) == 0 && len(item.Scores) > 0 {
			item.CreatedAt = current.CreatedAt
			byTitle[item.TitleID] = item
		}
	}
	result := make([]RatingFact, 0, len(byTitle))
	for _, item := range byTitle {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].TitleID < result[j].TitleID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

func ratingMap(items []RatingFact) map[int64]RatingFact {
	result := make(map[int64]RatingFact)
	for _, item := range uniqueRatings(items) {
		result[item.TitleID] = item
	}
	return result
}

func ratingDates(items []RatingFact, updated bool) []time.Time {
	result := make([]time.Time, 0, len(items))
	for _, item := range items {
		if updated {
			result = append(result, effectiveUpdatedAt(item))
		} else {
			result = append(result, item.CreatedAt)
		}
	}
	return result
}

func filterRatingDates(items []RatingFact, include func(RatingFact) bool) []time.Time {
	result := make([]time.Time, 0)
	for _, item := range items {
		if include(item) {
			result = append(result, item.CreatedAt)
		}
	}
	return result
}

func countResult(dates []time.Time) MetricResult {
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	result := MetricResult{Value: int64(len(dates)), ReachedAt: make(map[int64]time.Time, len(dates))}
	for index, at := range dates {
		result.ReachedAt[int64(index+1)] = at
	}
	return result
}

func pairedCountResult(left, right []time.Time) MetricResult {
	sort.Slice(left, func(i, j int) bool { return left[i].Before(left[j]) })
	sort.Slice(right, func(i, j int) bool { return right[i].Before(right[j]) })
	value := minInt(len(left), len(right))
	result := MetricResult{Value: int64(value), ReachedAt: make(map[int64]time.Time, value)}
	for index := 0; index < value; index++ {
		result.ReachedAt[int64(index+1)] = maxTime(left[index], right[index])
	}
	return result
}

func maxGroupResult(groups map[string][]time.Time) MetricResult {
	result := zeroResult()
	for _, dates := range groups {
		sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
		if int64(len(dates)) > result.Value {
			result.Value = int64(len(dates))
		}
		for index, at := range dates {
			target := int64(index + 1)
			if existing := result.ReachedAt[target]; existing.IsZero() || at.Before(existing) {
				result.ReachedAt[target] = at
			}
		}
	}
	return result
}

func maxGroupResultByID(groups map[int64][]time.Time) MetricResult {
	converted := make(map[string][]time.Time, len(groups))
	for key, dates := range groups {
		converted[stringKey(key)] = dates
	}
	return maxGroupResult(converted)
}

func distinctSetResult(values map[string]time.Time) MetricResult {
	dates := make([]time.Time, 0, len(values))
	for _, at := range values {
		dates = append(dates, at)
	}
	return countResult(dates)
}

func distinctIntSetResult(values map[int]time.Time) MetricResult {
	dates := make([]time.Time, 0, len(values))
	for _, at := range values {
		dates = append(dates, at)
	}
	return countResult(dates)
}

func distinctIDResult(values map[int64]time.Time) MetricResult {
	dates := make([]time.Time, 0, len(values))
	for _, at := range values {
		dates = append(dates, at)
	}
	return countResult(dates)
}

func hotThreadResult(groups map[int64][]CommentFact) MetricResult {
	result := zeroResult()
	for _, replies := range groups {
		sort.Slice(replies, func(i, j int) bool { return replies[i].CreatedAt.Before(replies[j].CreatedAt) })
		authors := make(map[int64]bool)
		for index, reply := range replies {
			authors[reply.UserID] = true
			count := int64(index + 1)
			if len(authors) < 2 {
				continue
			}
			if count > result.Value {
				for target := result.Value + 1; target <= count; target++ {
					result.ReachedAt[target] = reply.CreatedAt
				}
				result.Value = count
			}
		}
	}
	return result
}

func boolResult(at time.Time) MetricResult {
	if at.IsZero() {
		return zeroResult()
	}
	return MetricResult{Value: 1, ReachedAt: map[int64]time.Time{1: at}}
}

func zeroResult() MetricResult {
	return MetricResult{ReachedAt: make(map[int64]time.Time)}
}

func normalizedGenres(genres []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(genres))
	for _, genre := range genres {
		genre = strings.ToLower(strings.TrimSpace(genre))
		if genre == "" || seen[genre] {
			continue
		}
		seen[genre] = true
		result = append(result, genre)
	}
	return result
}

func isFullRating(scores map[string]int) bool {
	if len(scores) != len(FixedCriterionCodes) {
		return false
	}
	for _, code := range FixedCriterionCodes {
		if _, ok := scores[code]; !ok {
			return false
		}
	}
	return true
}

func distinctScoreCount(scores map[string]int) int {
	values := make(map[int]bool)
	for _, code := range FixedCriterionCodes {
		if value, ok := scores[code]; ok {
			values[value] = true
		}
	}
	return len(values)
}

func allScoresEqual(scores map[string]int) bool {
	if !isFullRating(scores) {
		return false
	}
	return distinctScoreCount(scores) == 1
}

func allScoresEqualTo(scores map[string]int, target int) bool {
	if !isFullRating(scores) {
		return false
	}
	for _, code := range FixedCriterionCodes {
		if scores[code] != target {
			return false
		}
	}
	return true
}

func exactScoreMaps(left, right map[string]int) bool {
	if !isFullRating(left) || !isFullRating(right) {
		return false
	}
	for _, code := range FixedCriterionCodes {
		if left[code] != right[code] {
			return false
		}
	}
	return true
}

func effectiveUpdatedAt(rating RatingFact) time.Time {
	if rating.UpdatedAt.IsZero() {
		return rating.CreatedAt
	}
	return rating.UpdatedAt
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func maxTime(values ...time.Time) time.Time {
	result := time.Time{}
	for _, value := range values {
		if value.After(result) {
			result = value
		}
	}
	return result
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func stringKey(value int64) string {
	return fmtInt64(value)
}

func fmtInt64(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		buffer[index] = '-'
	}
	return string(buffer[index:])
}
