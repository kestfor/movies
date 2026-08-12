package achievements

import (
	"math"
	"sort"
	"time"
)

var prospectiveMetricCodes = []MetricCode{
	MetricNoWeakLinksTotal, MetricNoConcessionsTotal, MetricContrastCutExists,
	MetricSignatureTouchMax, MetricThreeErasSameDayMax, MetricGenreDecadesMax,
	MetricParallelYearsTotal, MetricFiveNotchesTotal, MetricMiddleGroundExists,
	MetricLoneDissenterExists, MetricTogetherAndApartMax, MetricRatedRoundTableExists,
	MetricCriticDuetTotal, MetricAfterCreditsTotal, MetricCouncilWatchlistMax,
	MetricOpeningNightExists, MetricChainReactionMax, MetricTrustedRecommendationExists,
	MetricChangedMindExists, MetricPatientTicketExists, MetricClearQueueMax,
	MetricAgreedSessionExists, MetricRelayExists, MetricWordForWordMax,
	MetricDiscussThenRateExists, MetricThreadResurrectionExists, MetricGoodTipsMax,
	MetricMoodArcExists, MetricDeliberateRatingExists, MetricSharedFinaleExists,
}

type prospectiveContext struct {
	ownerID            int64
	location           *time.Location
	friends            map[int64]time.Time
	facts              []ActionFact
	watchlists         map[int64][]WatchlistFact
	introducedByMetric map[MetricCode]time.Time
}

func (e *Evaluator) prospectiveMetrics(snapshot Snapshot, metrics map[MetricCode]MetricResult, introduced map[string]time.Time) {
	for _, code := range prospectiveMetricCodes {
		metrics[code] = zeroResult()
	}
	if len(introduced) == 0 {
		return
	}
	ctx := prospectiveContext{
		ownerID:            snapshot.UserID,
		location:           e.location,
		friends:            make(map[int64]time.Time, len(snapshot.Friends)),
		facts:              append([]ActionFact(nil), snapshot.ActionFacts...),
		watchlists:         map[int64][]WatchlistFact{snapshot.UserID: snapshot.Watchlist},
		introducedByMetric: make(map[MetricCode]time.Time),
	}
	for _, friend := range snapshot.Friends {
		ctx.friends[friend.User.ID] = friend.AcceptedAt
		ctx.watchlists[friend.User.ID] = friend.Watchlist
	}
	for _, definition := range e.definitions {
		if definition.AwardPolicy == AwardPolicySinceIntroduction {
			ctx.introducedByMetric[definition.Metric] = introduced[definition.Code]
		}
	}
	sort.Slice(ctx.facts, func(i, j int) bool {
		if ctx.facts[i].OccurredAt.Equal(ctx.facts[j].OccurredAt) {
			return ctx.facts[i].ID < ctx.facts[j].ID
		}
		return ctx.facts[i].OccurredAt.Before(ctx.facts[j].OccurredAt)
	})

	metrics[MetricNoWeakLinksTotal] = ctx.fullRatingCount(MetricNoWeakLinksTotal, func(f ActionFact) bool { return allScoresAtLeast(f.Scores, 8) })
	metrics[MetricNoConcessionsTotal] = ctx.fullRatingCount(MetricNoConcessionsTotal, func(f ActionFact) bool { return allScoresAtMost(f.Scores, 5) })
	metrics[MetricContrastCutExists] = ctx.contrastCut()
	metrics[MetricSignatureTouchMax] = ctx.signatureTouch()
	metrics[MetricThreeErasSameDayMax] = ctx.threeErasSameDay()
	metrics[MetricGenreDecadesMax] = ctx.genreDecades()
	metrics[MetricParallelYearsTotal] = ctx.parallelYears()
	metrics[MetricFiveNotchesTotal] = ctx.fiveNotches()
	metrics[MetricMiddleGroundExists] = ctx.middleGround()
	metrics[MetricLoneDissenterExists] = ctx.loneDissenter()
	metrics[MetricTogetherAndApartMax] = ctx.togetherAndApart()
	metrics[MetricRatedRoundTableExists] = ctx.ratedRoundTable()
	metrics[MetricCriticDuetTotal] = ctx.criticDuet()
	metrics[MetricAfterCreditsTotal] = ctx.afterCredits()
	metrics[MetricCouncilWatchlistMax] = ctx.councilWatchlist()
	metrics[MetricOpeningNightExists] = ctx.openingNight()
	metrics[MetricChainReactionMax] = ctx.chainReaction()
	metrics[MetricTrustedRecommendationExists] = ctx.trustedRecommendation()
	metrics[MetricChangedMindExists] = ctx.changedMind()
	metrics[MetricPatientTicketExists] = ctx.patientTicket()
	metrics[MetricClearQueueMax] = ctx.clearQueue()
	metrics[MetricAgreedSessionExists] = ctx.agreedSession()
	metrics[MetricRelayExists] = ctx.relay()
	metrics[MetricWordForWordMax] = ctx.wordForWord()
	metrics[MetricDiscussThenRateExists] = ctx.discussThenRate()
	metrics[MetricThreadResurrectionExists] = ctx.threadResurrection()
	metrics[MetricGoodTipsMax] = ctx.goodTips()
	metrics[MetricMoodArcExists] = ctx.moodArc()
	metrics[MetricDeliberateRatingExists] = ctx.deliberateRating()
	metrics[MetricSharedFinaleExists] = ctx.sharedFinale()
}

func (c prospectiveContext) after(metric MetricCode) []ActionFact {
	introducedAt := c.introducedByMetric[metric]
	if introducedAt.IsZero() {
		return nil
	}
	result := make([]ActionFact, 0)
	for _, fact := range c.facts {
		if !fact.OccurredAt.Before(introducedAt) {
			result = append(result, fact)
		}
	}
	return result
}

func (c prospectiveContext) friendAt(userID int64, _ time.Time) bool {
	_, ok := c.friends[userID]
	return ok
}

func (c prospectiveContext) friendCompletion(userID int64, at time.Time) time.Time {
	return maxTime(at, c.friends[userID])
}

type actorTitleKey struct{ actorID, titleID int64 }

func firstRatingFacts(facts []ActionFact) map[actorTitleKey]ActionFact {
	result := make(map[actorTitleKey]ActionFact)
	for _, fact := range facts {
		if fact.Kind != ActionFactRatingCreated {
			continue
		}
		key := actorTitleKey{fact.ActorID, fact.TitleID}
		if _, exists := result[key]; !exists {
			result[key] = fact
		}
	}
	return result
}

func latestRatingFacts(facts []ActionFact) map[actorTitleKey]ActionFact {
	result := make(map[actorTitleKey]ActionFact)
	for _, fact := range facts {
		if fact.Kind == ActionFactRatingCreated || fact.Kind == ActionFactRatingUpdated {
			result[actorTitleKey{fact.ActorID, fact.TitleID}] = fact
		}
	}
	return result
}

func firstFullRatingFacts(facts []ActionFact) map[actorTitleKey]ActionFact {
	result := make(map[actorTitleKey]ActionFact)
	for _, fact := range facts {
		if (fact.Kind != ActionFactRatingCreated && fact.Kind != ActionFactRatingUpdated) || !isFullRating(fact.Scores) {
			continue
		}
		key := actorTitleKey{fact.ActorID, fact.TitleID}
		if _, exists := result[key]; !exists {
			result[key] = fact
		}
	}
	return result
}

func commentFacts(facts []ActionFact) []ActionFact {
	result := make([]ActionFact, 0)
	for _, fact := range facts {
		if fact.Kind == ActionFactCommentCreated {
			result = append(result, fact)
		}
	}
	return result
}

func watchlistFacts(facts []ActionFact) map[actorTitleKey]ActionFact {
	result := make(map[actorTitleKey]ActionFact)
	for _, fact := range facts {
		if fact.Kind != ActionFactWatchlistAdded {
			continue
		}
		key := actorTitleKey{fact.ActorID, fact.TitleID}
		if _, exists := result[key]; !exists {
			result[key] = fact
		}
	}
	return result
}

func (c prospectiveContext) fullRatingCount(metric MetricCode, include func(ActionFact) bool) MetricResult {
	times := make([]time.Time, 0)
	for key, fact := range firstFullRatingFacts(c.after(metric)) {
		if key.actorID == c.ownerID && include(fact) {
			times = append(times, fact.OccurredAt)
		}
	}
	return countResult(times)
}

func (c prospectiveContext) contrastCut() MetricResult {
	best := time.Time{}
	for key, fact := range firstFullRatingFacts(c.after(MetricContrastCutExists)) {
		if key.actorID == c.ownerID && fact.AvgScore >= 4 && fact.AvgScore <= 7 && containsScore(fact.Scores, 1) && containsScore(fact.Scores, 10) {
			best = firstTime(best, fact.OccurredAt)
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) signatureTouch() MetricResult {
	groups := make(map[string][]time.Time)
	for key, fact := range firstFullRatingFacts(c.after(MetricSignatureTouchMax)) {
		if key.actorID != c.ownerID {
			continue
		}
		if code := uniqueExtremeCriterion(fact.Scores, true); code != "" {
			groups[code] = append(groups[code], fact.OccurredAt)
		}
	}
	return maxGroupResult(groups)
}

func (c prospectiveContext) threeErasSameDay() MetricResult {
	byDay := make(map[string]map[int]time.Time)
	for key, fact := range firstRatingFacts(c.after(MetricThreeErasSameDayMax)) {
		if key.actorID != c.ownerID || fact.ReleaseYear <= 0 {
			continue
		}
		day := fact.OccurredAt.In(c.location).Format("2006-01-02")
		if byDay[day] == nil {
			byDay[day] = make(map[int]time.Time)
		}
		decade := fact.ReleaseYear / 10 * 10
		if byDay[day][decade].IsZero() {
			byDay[day][decade] = fact.OccurredAt
		}
	}
	groups := make(map[string][]time.Time)
	for day, decades := range byDay {
		for _, at := range decades {
			groups[day] = append(groups[day], at)
		}
	}
	return maxGroupResult(groups)
}

func (c prospectiveContext) genreDecades() MetricResult {
	sets := make(map[string]map[int]time.Time)
	for key, fact := range firstRatingFacts(c.after(MetricGenreDecadesMax)) {
		if key.actorID != c.ownerID || fact.ReleaseYear <= 0 {
			continue
		}
		for _, genre := range normalizedGenres(fact.Genres) {
			if sets[genre] == nil {
				sets[genre] = make(map[int]time.Time)
			}
			decade := fact.ReleaseYear / 10 * 10
			if sets[genre][decade].IsZero() {
				sets[genre][decade] = fact.OccurredAt
			}
		}
	}
	groups := make(map[string][]time.Time)
	for genre, decades := range sets {
		for _, at := range decades {
			groups[genre] = append(groups[genre], at)
		}
	}
	return maxGroupResult(groups)
}

func (c prospectiveContext) parallelYears() MetricResult {
	type mediaTimes struct{ movie, tv time.Time }
	years := make(map[int]mediaTimes)
	for key, fact := range firstRatingFacts(c.after(MetricParallelYearsTotal)) {
		if key.actorID != c.ownerID || fact.ReleaseYear <= 0 {
			continue
		}
		item := years[fact.ReleaseYear]
		switch fact.MediaType {
		case "movie":
			item.movie = firstTime(item.movie, fact.OccurredAt)
		case "tv":
			item.tv = firstTime(item.tv, fact.OccurredAt)
		}
		years[fact.ReleaseYear] = item
	}
	times := make([]time.Time, 0)
	for _, item := range years {
		if !item.movie.IsZero() && !item.tv.IsZero() {
			times = append(times, maxTime(item.movie, item.tv))
		}
	}
	return countResult(times)
}

func (c prospectiveContext) fiveNotches() MetricResult {
	facts := firstFullRatingFacts(c.after(MetricFiveNotchesTotal))
	ordered := make([]ActionFact, 0)
	for key, fact := range facts {
		if key.actorID == c.ownerID {
			ordered = append(ordered, fact)
		}
	}
	sortFacts(ordered)
	buckets := make(map[int]bool)
	low, high := false, false
	result := zeroResult()
	for _, fact := range ordered {
		bucket := int(math.Floor(fact.AvgScore + 0.5))
		if bucket < 1 {
			bucket = 1
		}
		if bucket > 10 {
			bucket = 10
		}
		buckets[bucket] = true
		low = low || bucket <= 3
		high = high || bucket >= 9
		value := int64(len(buckets))
		if (!low || !high) && value > 4 {
			value = 4
		}
		advanceResult(&result, value, fact.OccurredAt)
	}
	return result
}

func (c prospectiveContext) middleGround() MetricResult {
	return c.threeRatingPattern(MetricMiddleGroundExists, func(own, left, right float64) bool {
		return math.Abs(left-right) >= 4-1e-9 && math.Abs(own-(left+right)/2) <= 0.5+1e-9
	})
}

func (c prospectiveContext) loneDissenter() MetricResult {
	return c.threeRatingPattern(MetricLoneDissenterExists, func(own, left, right float64) bool {
		return math.Abs(left-right) <= 0.5+1e-9 && math.Abs(own-left) >= 3-1e-9 && math.Abs(own-right) >= 3-1e-9
	})
}

func (c prospectiveContext) threeRatingPattern(metric MetricCode, matches func(float64, float64, float64) bool) MetricResult {
	ratings := latestRatingFacts(c.after(metric))
	best := time.Time{}
	for ownKey, own := range ratings {
		if ownKey.actorID != c.ownerID {
			continue
		}
		friends := make([]ActionFact, 0)
		for key, rating := range ratings {
			if key.titleID == ownKey.titleID && key.actorID != c.ownerID && c.friendAt(key.actorID, maxTime(own.OccurredAt, rating.OccurredAt)) {
				friends = append(friends, rating)
			}
		}
		for i := 0; i < len(friends); i++ {
			for j := i + 1; j < len(friends); j++ {
				at := maxTime(own.OccurredAt, friends[i].OccurredAt, friends[j].OccurredAt)
				at = c.friendCompletion(friends[i].ActorID, c.friendCompletion(friends[j].ActorID, at))
				if c.friendAt(friends[i].ActorID, at) && c.friendAt(friends[j].ActorID, at) && matches(own.AvgScore, friends[i].AvgScore, friends[j].AvgScore) {
					best = firstTime(best, at)
				}
			}
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) togetherAndApart() MetricResult {
	ratings := latestRatingFacts(c.after(MetricTogetherAndApartMax))
	type pairTimes struct{ close, far []time.Time }
	groups := make(map[int64]pairTimes)
	for key, own := range ratings {
		if key.actorID != c.ownerID {
			continue
		}
		for friendID := range c.friends {
			other, ok := ratings[actorTitleKey{friendID, key.titleID}]
			at := c.friendCompletion(friendID, maxTime(own.OccurredAt, other.OccurredAt))
			if !ok || !c.friendAt(friendID, at) {
				continue
			}
			item := groups[friendID]
			difference := math.Abs(own.AvgScore - other.AvgScore)
			if difference <= 0.5+1e-9 {
				item.close = append(item.close, at)
			}
			if difference >= 3-1e-9 {
				item.far = append(item.far, at)
			}
			groups[friendID] = item
		}
	}
	result := zeroResult()
	for _, item := range groups {
		paired := pairedCountResult(item.close, item.far)
		mergeMaxResult(&result, paired)
	}
	return result
}

func (c prospectiveContext) ratedRoundTable() MetricResult {
	facts := c.after(MetricRatedRoundTableExists)
	ratings := latestRatingFacts(facts)
	comments := actorTitleCommentTimes(commentFacts(facts))
	best := time.Time{}
	for key, ownRating := range ratings {
		if key.actorID != c.ownerID || comments[key].IsZero() {
			continue
		}
		friends := make([]time.Time, 0)
		for friendID := range c.friends {
			friendKey := actorTitleKey{friendID, key.titleID}
			friendRating, rated := ratings[friendKey]
			commentAt := comments[friendKey]
			at := c.friendCompletion(friendID, maxTime(ownRating.OccurredAt, comments[key], friendRating.OccurredAt, commentAt))
			if rated && !commentAt.IsZero() && c.friendAt(friendID, at) {
				friends = append(friends, at)
			}
		}
		if len(friends) >= 2 {
			sort.Slice(friends, func(i, j int) bool { return friends[i].Before(friends[j]) })
			best = firstTime(best, friends[1])
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) criticDuet() MetricResult {
	facts := c.after(MetricCriticDuetTotal)
	ratings := latestRatingFacts(facts)
	comments := actorTitleCommentTimes(commentFacts(facts))
	groups := make(map[string][]time.Time)
	for key, ownRating := range ratings {
		if key.actorID != c.ownerID || comments[key].IsZero() {
			continue
		}
		for friendID := range c.friends {
			friendKey := actorTitleKey{friendID, key.titleID}
			friendRating, rated := ratings[friendKey]
			commentAt := comments[friendKey]
			at := c.friendCompletion(friendID, maxTime(ownRating.OccurredAt, comments[key], friendRating.OccurredAt, commentAt))
			if rated && !commentAt.IsZero() && c.friendAt(friendID, at) {
				groups[stringKey(friendID)] = append(groups[stringKey(friendID)], at)
			}
		}
	}
	return maxGroupResult(groups)
}

func (c prospectiveContext) afterCredits() MetricResult {
	facts := c.after(MetricAfterCreditsTotal)
	ratings := firstRatingFacts(facts)
	seen := make(map[int64]time.Time)
	for _, comment := range commentFacts(facts) {
		if comment.ActorID != c.ownerID {
			continue
		}
		rating, ok := ratings[actorTitleKey{c.ownerID, comment.TitleID}]
		if ok && !comment.OccurredAt.Before(rating.OccurredAt.Add(48*time.Hour)) {
			seen[comment.TitleID] = firstTime(seen[comment.TitleID], comment.OccurredAt)
		}
	}
	return distinctIDResult(seen)
}

func (c prospectiveContext) councilWatchlist() MetricResult {
	introducedAt := c.introducedByMetric[MetricCouncilWatchlistMax]
	if introducedAt.IsZero() {
		return zeroResult()
	}
	own := make(map[int64]time.Time)
	for _, item := range c.watchlists[c.ownerID] {
		if !item.CreatedAt.Before(introducedAt) {
			own[item.TitleID] = item.CreatedAt
		}
	}
	groups := make(map[int64][]time.Time)
	for friendID, items := range c.watchlists {
		if friendID == c.ownerID {
			continue
		}
		for _, item := range items {
			ownAt, ok := own[item.TitleID]
			at := c.friendCompletion(friendID, maxTime(ownAt, item.CreatedAt))
			if ok && !item.CreatedAt.Before(introducedAt) && c.friendAt(friendID, at) {
				groups[item.TitleID] = append(groups[item.TitleID], at)
			}
		}
	}
	return maxGroupResultByID(groups)
}

func (c prospectiveContext) openingNight() MetricResult {
	ratings := firstRatingFacts(c.after(MetricOpeningNightExists))
	best := time.Time{}
	for key, own := range ratings {
		if key.actorID != c.ownerID {
			continue
		}
		for friendID := range c.friends {
			other, ok := ratings[actorTitleKey{friendID, key.titleID}]
			at := c.friendCompletion(friendID, maxTime(own.OccurredAt, other.OccurredAt))
			if ok && durationAbs(own.OccurredAt.Sub(other.OccurredAt)) <= 4*time.Hour && c.friendAt(friendID, at) {
				best = firstTime(best, at)
			}
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) chainReaction() MetricResult {
	ratings := firstRatingFacts(c.after(MetricChainReactionMax))
	groups := make(map[int64][]time.Time)
	for key, own := range ratings {
		if key.actorID != c.ownerID {
			continue
		}
		for friendID := range c.friends {
			other, ok := ratings[actorTitleKey{friendID, key.titleID}]
			if ok && !other.OccurredAt.Before(own.OccurredAt) && other.OccurredAt.Sub(own.OccurredAt) <= 72*time.Hour && c.friendAt(friendID, other.OccurredAt) {
				groups[key.titleID] = append(groups[key.titleID], c.friendCompletion(friendID, other.OccurredAt))
			}
		}
	}
	return maxGroupResultByID(groups)
}

func (c prospectiveContext) trustedRecommendation() MetricResult {
	facts := c.after(MetricTrustedRecommendationExists)
	ratings := firstRatingFacts(facts)
	watch := watchlistFacts(facts)
	best := time.Time{}
	for key, ownRating := range ratings {
		if key.actorID != c.ownerID || ownRating.AvgScore < 8 {
			continue
		}
		add, ok := watch[actorTitleKey{c.ownerID, key.titleID}]
		if !ok || ownRating.OccurredAt.Before(add.OccurredAt) || ownRating.OccurredAt.Sub(add.OccurredAt) > 14*24*time.Hour {
			continue
		}
		for friendID := range c.friends {
			advice, ok := ratings[actorTitleKey{friendID, key.titleID}]
			if ok && advice.AvgScore >= 9 && !add.OccurredAt.Before(advice.OccurredAt) && add.OccurredAt.Sub(advice.OccurredAt) <= 48*time.Hour && c.friendAt(friendID, ownRating.OccurredAt) {
				best = firstTime(best, c.friendCompletion(friendID, ownRating.OccurredAt))
			}
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) changedMind() MetricResult {
	facts := c.after(MetricChangedMindExists)
	comments := commentFacts(facts)
	byID := make(map[int64]ActionFact, len(comments))
	for _, comment := range comments {
		byID[comment.EntityID] = comment
	}
	best := time.Time{}
	for _, updated := range facts {
		if updated.Kind != ActionFactRatingUpdated || updated.ActorID != c.ownerID || updated.AvgScore < 7 || updated.PreviousAvgScore > 5 {
			continue
		}
		for _, reply := range comments {
			parent := byID[reply.ParentEntityID]
			if reply.TitleID != updated.TitleID || reply.ActorID == c.ownerID || parent.ActorID != c.ownerID || parent.TitleID != updated.TitleID {
				continue
			}
			lowBeforeComment := false
			for _, candidate := range facts {
				if candidate.ActorID == c.ownerID && candidate.TitleID == updated.TitleID &&
					(candidate.Kind == ActionFactRatingCreated || candidate.Kind == ActionFactRatingUpdated) &&
					candidate.AvgScore <= 5 && !candidate.OccurredAt.After(parent.OccurredAt) {
					lowBeforeComment = true
					break
				}
			}
			if lowBeforeComment && !reply.OccurredAt.Before(parent.OccurredAt) && !updated.OccurredAt.Before(reply.OccurredAt) && updated.OccurredAt.Sub(reply.OccurredAt) <= 7*24*time.Hour && c.friendAt(reply.ActorID, updated.OccurredAt) {
				best = firstTime(best, c.friendCompletion(reply.ActorID, updated.OccurredAt))
			}
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) patientTicket() MetricResult {
	facts := c.after(MetricPatientTicketExists)
	ratings := firstRatingFacts(facts)
	watch := watchlistFacts(facts)
	best := time.Time{}
	for key, rating := range ratings {
		if key.actorID != c.ownerID {
			continue
		}
		add, ok := watch[key]
		delta := rating.OccurredAt.Sub(add.OccurredAt)
		if ok && delta >= 7*24*time.Hour && delta <= 30*24*time.Hour {
			best = firstTime(best, rating.OccurredAt)
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) clearQueue() MetricResult {
	facts := c.after(MetricClearQueueMax)
	ratings := firstRatingFacts(facts)
	watch := watchlistFacts(facts)
	times := make([]time.Time, 0)
	for key, rating := range ratings {
		add, ok := watch[key]
		if key.actorID == c.ownerID && ok && !rating.OccurredAt.Before(add.OccurredAt) {
			times = append(times, rating.OccurredAt)
		}
	}
	return rollingCountResult(times, 48*time.Hour)
}

func (c prospectiveContext) agreedSession() MetricResult {
	facts := c.after(MetricAgreedSessionExists)
	ratings := firstRatingFacts(facts)
	watch := watchlistFacts(facts)
	best := time.Time{}
	for key, ownAdd := range watch {
		if key.actorID != c.ownerID {
			continue
		}
		ownRating, ownRated := ratings[key]
		for friendID := range c.friends {
			friendKey := actorTitleKey{friendID, key.titleID}
			friendAdd, added := watch[friendKey]
			friendRating, friendRated := ratings[friendKey]
			firstAdd := minTime(ownAdd.OccurredAt, friendAdd.OccurredAt)
			actionCompletion := maxTime(ownRating.OccurredAt, friendRating.OccurredAt)
			completion := c.friendCompletion(friendID, actionCompletion)
			if added && ownRated && friendRated && durationAbs(ownAdd.OccurredAt.Sub(friendAdd.OccurredAt)) <= 24*time.Hour &&
				!ownRating.OccurredAt.Before(ownAdd.OccurredAt) && !friendRating.OccurredAt.Before(friendAdd.OccurredAt) &&
				actionCompletion.Sub(firstAdd) <= 14*24*time.Hour && c.friendAt(friendID, completion) {
				best = firstTime(best, completion)
			}
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) relay() MetricResult {
	ratings := firstRatingFacts(c.after(MetricRelayExists))
	byTitle := ratingFactsByTitle(ratings)
	best := time.Time{}
	for _, items := range byTitle {
		sortFacts(items)
		for middleIndex, middle := range items {
			if middle.ActorID != c.ownerID {
				continue
			}
			for left := 0; left < middleIndex; left++ {
				if items[left].ActorID == c.ownerID || middle.OccurredAt.Sub(items[left].OccurredAt) > 48*time.Hour {
					continue
				}
				for right := middleIndex + 1; right < len(items); right++ {
					if items[right].ActorID == c.ownerID || items[right].ActorID == items[left].ActorID || items[right].OccurredAt.Sub(middle.OccurredAt) > 48*time.Hour {
						continue
					}
					if c.friendAt(items[left].ActorID, items[right].OccurredAt) && c.friendAt(items[right].ActorID, items[right].OccurredAt) {
						completion := c.friendCompletion(items[left].ActorID, c.friendCompletion(items[right].ActorID, items[right].OccurredAt))
						best = firstTime(best, completion)
					}
				}
			}
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) wordForWord() MetricResult {
	comments := commentFacts(c.after(MetricWordForWordMax))
	byID := make(map[int64]ActionFact, len(comments))
	for _, comment := range comments {
		byID[comment.EntityID] = comment
	}
	groups := make(map[string][]ActionFact)
	for _, comment := range comments {
		root := commentRoot(comment, byID)
		if root == 0 {
			continue
		}
		for friendID := range c.friends {
			if comment.ActorID == c.ownerID || comment.ActorID == friendID {
				key := stringKey(root) + ":" + stringKey(friendID)
				groups[key] = append(groups[key], comment)
			}
		}
	}
	result := zeroResult()
	for _, items := range groups {
		sortFacts(items)
		for start := range items {
			count := int64(1)
			previous := items[start].ActorID
			for end := start + 1; end < len(items) && items[end].OccurredAt.Sub(items[start].OccurredAt) <= 48*time.Hour; end++ {
				if items[end].ActorID == previous {
					break
				}
				count++
				previous = items[end].ActorID
				advanceResult(&result, count, items[end].OccurredAt)
			}
		}
	}
	return result
}

func (c prospectiveContext) discussThenRate() MetricResult {
	facts := c.after(MetricDiscussThenRateExists)
	ratings := firstRatingFacts(facts)
	comments := actorTitleCommentTimes(commentFacts(facts))
	best := time.Time{}
	for key, ownRating := range ratings {
		if key.actorID != c.ownerID {
			continue
		}
		ownComment := comments[key]
		for friendID := range c.friends {
			friendKey := actorTitleKey{friendID, key.titleID}
			friendRating, rated := ratings[friendKey]
			friendComment := comments[friendKey]
			lastComment := maxTime(ownComment, friendComment)
			actionCompletion := maxTime(ownRating.OccurredAt, friendRating.OccurredAt)
			completion := c.friendCompletion(friendID, actionCompletion)
			if rated && !ownComment.IsZero() && !friendComment.IsZero() && !actionCompletion.Before(lastComment) && actionCompletion.Sub(lastComment) <= 48*time.Hour &&
				!ownRating.OccurredAt.Before(ownComment) && !friendRating.OccurredAt.Before(friendComment) && math.Abs(ownRating.AvgScore-friendRating.AvgScore) <= 1+1e-9 && c.friendAt(friendID, completion) {
				best = firstTime(best, completion)
			}
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) threadResurrection() MetricResult {
	comments := commentFacts(c.after(MetricThreadResurrectionExists))
	byID := make(map[int64]ActionFact, len(comments))
	for _, comment := range comments {
		byID[comment.EntityID] = comment
	}
	best := time.Time{}
	for _, ownReply := range comments {
		parent := byID[ownReply.ParentEntityID]
		if ownReply.ActorID != c.ownerID || parent.EntityID == 0 || parent.ActorID == c.ownerID || ownReply.OccurredAt.Sub(parent.OccurredAt) < 14*24*time.Hour {
			continue
		}
		for _, answer := range comments {
			if answer.ParentEntityID == ownReply.EntityID && answer.ActorID == parent.ActorID && !answer.OccurredAt.Before(ownReply.OccurredAt) && answer.OccurredAt.Sub(ownReply.OccurredAt) <= 48*time.Hour {
				best = firstTime(best, answer.OccurredAt)
			}
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) goodTips() MetricResult {
	ratings := firstRatingFacts(c.after(MetricGoodTipsMax))
	type tip struct {
		titleID      int64
		actionAt     time.Time
		completionAt time.Time
		friendID     int64
	}
	tips := make([]tip, 0)
	for key, own := range ratings {
		if key.actorID != c.ownerID || own.AvgScore < 8 {
			continue
		}
		for friendID := range c.friends {
			advice, ok := ratings[actorTitleKey{friendID, key.titleID}]
			if ok && advice.AvgScore >= 9 && !own.OccurredAt.Before(advice.OccurredAt) && own.OccurredAt.Sub(advice.OccurredAt) <= 14*24*time.Hour && c.friendAt(friendID, own.OccurredAt) {
				tips = append(tips, tip{key.titleID, own.OccurredAt, c.friendCompletion(friendID, own.OccurredAt), friendID})
			}
		}
	}
	sort.Slice(tips, func(i, j int) bool { return tips[i].actionAt.Before(tips[j].actionAt) })
	result := zeroResult()
	for index, current := range tips {
		advanceResult(&result, 1, current.completionAt)
		for otherIndex := index + 1; otherIndex < len(tips); otherIndex++ {
			other := tips[otherIndex]
			if other.actionAt.Sub(current.actionAt) > 14*24*time.Hour {
				break
			}
			if current.titleID != other.titleID && current.friendID != other.friendID {
				advanceResult(&result, 2, maxTime(current.completionAt, other.completionAt))
			}
		}
	}
	return result
}

func (c prospectiveContext) moodArc() MetricResult {
	ratings := make([]ActionFact, 0)
	for key, fact := range firstRatingFacts(c.after(MetricMoodArcExists)) {
		if key.actorID == c.ownerID {
			ratings = append(ratings, fact)
		}
	}
	sortFacts(ratings)
	best := time.Time{}
	for i := 0; i < len(ratings); i++ {
		if ratings[i].AvgScore > 4 {
			continue
		}
		for j := i + 1; j < len(ratings); j++ {
			if ratings[j].OccurredAt.Sub(ratings[i].OccurredAt) > 48*time.Hour || ratings[j].AvgScore < 5 || ratings[j].AvgScore > 7 {
				continue
			}
			for k := j + 1; k < len(ratings); k++ {
				if ratings[k].OccurredAt.Sub(ratings[i].OccurredAt) > 48*time.Hour {
					break
				}
				if ratings[k].AvgScore >= 9 {
					best = firstTime(best, ratings[k].OccurredAt)
				}
			}
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) deliberateRating() MetricResult {
	facts := c.after(MetricDeliberateRatingExists)
	created := make(map[actorTitleKey]ActionFact)
	firstUpdateSeen := make(map[actorTitleKey]bool)
	best := time.Time{}
	for _, fact := range facts {
		if fact.ActorID != c.ownerID || (fact.Kind != ActionFactRatingCreated && fact.Kind != ActionFactRatingUpdated) {
			continue
		}
		key := actorTitleKey{fact.ActorID, fact.TitleID}
		if fact.Kind == ActionFactRatingCreated {
			created[key] = fact
			continue
		}
		if firstUpdateSeen[key] {
			continue
		}
		firstUpdateSeen[key] = true
		start, ok := created[key]
		delta := fact.OccurredAt.Sub(start.OccurredAt)
		if ok && len(start.Scores) >= 1 && len(start.Scores) <= 3 && isFullRating(fact.Scores) && delta >= 12*time.Hour && delta <= 7*24*time.Hour {
			best = firstTime(best, fact.OccurredAt)
		}
	}
	return boolResult(best)
}

func (c prospectiveContext) sharedFinale() MetricResult {
	ratings := firstRatingFacts(c.after(MetricSharedFinaleExists))
	best := time.Time{}
	for key, own := range ratings {
		if key.actorID != c.ownerID {
			continue
		}
		friends := make([]ActionFact, 0)
		day := own.OccurredAt.In(c.location).Format("2006-01-02")
		for friendID := range c.friends {
			other, ok := ratings[actorTitleKey{friendID, key.titleID}]
			if ok && other.OccurredAt.In(c.location).Format("2006-01-02") == day {
				friends = append(friends, other)
			}
		}
		for i := 0; i < len(friends); i++ {
			for j := i + 1; j < len(friends); j++ {
				at := maxTime(own.OccurredAt, friends[i].OccurredAt, friends[j].OccurredAt)
				at = c.friendCompletion(friends[i].ActorID, c.friendCompletion(friends[j].ActorID, at))
				minimum := math.Min(own.AvgScore, math.Min(friends[i].AvgScore, friends[j].AvgScore))
				maximum := math.Max(own.AvgScore, math.Max(friends[i].AvgScore, friends[j].AvgScore))
				if maximum-minimum <= 1.5+1e-9 && c.friendAt(friends[i].ActorID, at) && c.friendAt(friends[j].ActorID, at) {
					best = firstTime(best, at)
				}
			}
		}
	}
	return boolResult(best)
}

func actorTitleCommentTimes(comments []ActionFact) map[actorTitleKey]time.Time {
	result := make(map[actorTitleKey]time.Time)
	for _, comment := range comments {
		key := actorTitleKey{comment.ActorID, comment.TitleID}
		result[key] = firstTime(result[key], comment.OccurredAt)
	}
	return result
}

func ratingFactsByTitle(ratings map[actorTitleKey]ActionFact) map[int64][]ActionFact {
	result := make(map[int64][]ActionFact)
	for _, rating := range ratings {
		result[rating.TitleID] = append(result[rating.TitleID], rating)
	}
	return result
}

func commentRoot(comment ActionFact, byID map[int64]ActionFact) int64 {
	seen := make(map[int64]bool)
	current := comment
	for current.ParentEntityID != 0 && !seen[current.EntityID] {
		seen[current.EntityID] = true
		parent, ok := byID[current.ParentEntityID]
		if !ok {
			break
		}
		current = parent
	}
	return current.EntityID
}

func allScoresAtLeast(scores map[string]int, minimum int) bool {
	if !isFullRating(scores) {
		return false
	}
	for _, code := range FixedCriterionCodes {
		if scores[code] < minimum {
			return false
		}
	}
	return true
}

func allScoresAtMost(scores map[string]int, maximum int) bool {
	if !isFullRating(scores) {
		return false
	}
	for _, code := range FixedCriterionCodes {
		if scores[code] > maximum {
			return false
		}
	}
	return true
}

func containsScore(scores map[string]int, target int) bool {
	for _, score := range scores {
		if score == target {
			return true
		}
	}
	return false
}

func uniqueExtremeCriterion(scores map[string]int, highest bool) string {
	if !isFullRating(scores) {
		return ""
	}
	extreme := scores[FixedCriterionCodes[0]]
	code := FixedCriterionCodes[0]
	unique := true
	for _, candidate := range FixedCriterionCodes[1:] {
		score := scores[candidate]
		if (highest && score > extreme) || (!highest && score < extreme) {
			extreme, code, unique = score, candidate, true
		} else if score == extreme {
			unique = false
		}
	}
	if !unique {
		return ""
	}
	return code
}

func rollingCountResult(times []time.Time, window time.Duration) MetricResult {
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	result := zeroResult()
	left := 0
	for right, at := range times {
		for at.Sub(times[left]) > window {
			left++
		}
		advanceResult(&result, int64(right-left+1), at)
	}
	return result
}

func advanceResult(result *MetricResult, value int64, at time.Time) {
	if value <= result.Value {
		return
	}
	for target := result.Value + 1; target <= value; target++ {
		result.ReachedAt[target] = at
	}
	result.Value = value
}

func mergeMaxResult(target *MetricResult, candidate MetricResult) {
	for value := int64(1); value <= candidate.Value; value++ {
		at := candidate.ReachedAt[value]
		if existing := target.ReachedAt[value]; existing.IsZero() || (!at.IsZero() && at.Before(existing)) {
			target.ReachedAt[value] = at
		}
	}
	if candidate.Value > target.Value {
		target.Value = candidate.Value
	}
}

func sortFacts(facts []ActionFact) {
	sort.Slice(facts, func(i, j int) bool {
		if facts[i].OccurredAt.Equal(facts[j].OccurredAt) {
			return facts[i].ID < facts[j].ID
		}
		return facts[i].OccurredAt.Before(facts[j].OccurredAt)
	})
}

func firstTime(current, candidate time.Time) time.Time {
	if current.IsZero() || (!candidate.IsZero() && candidate.Before(current)) {
		return candidate
	}
	return current
}

func minTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

func durationAbs(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}
