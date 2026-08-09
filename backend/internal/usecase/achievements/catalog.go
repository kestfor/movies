package achievements

//go:generate go run ./cmd/cataloggen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
)

type MetricCode string
type Category string

const (
	CategoryRatings   Category = "ratings"
	CategoryTaste     Category = "taste"
	CategorySocial    Category = "social"
	CategoryComments  Category = "comments"
	CategoryWatchlist Category = "watchlist"
)

const (
	MetricRatingsTotal                 MetricCode = "ratings_total"
	MetricFullRatingsTotal             MetricCode = "full_ratings_total"
	MetricRatingsSameGenreMax          MetricCode = "ratings_same_genre_max"
	MetricRatedReleaseDecadesTotal     MetricCode = "rated_release_decades_total"
	MetricRatingScoreContrast          MetricCode = "rating_score_contrast"
	MetricPerfectSevenRatingExists     MetricCode = "perfect_seven_rating_exists"
	MetricRatedMediaBalance            MetricCode = "rated_media_balance"
	MetricRatedMoviesTotal             MetricCode = "rated_movies_total"
	MetricRatedTVTotal                 MetricCode = "rated_tv_total"
	MetricRatedGenresTotal             MetricCode = "rated_genres_total"
	MetricRatingDayStreak              MetricCode = "rating_day_streak"
	MetricFriendsHighWater             MetricCode = "friends_high_water"
	MetricSharedTitlesTotal            MetricCode = "shared_titles_total"
	MetricFriendCloseRatingsMax        MetricCode = "friend_close_ratings_max"
	MetricFriendExactFullRatingExists  MetricCode = "friend_exact_full_rating_exists"
	MetricCommentsTotal                MetricCode = "comments_total"
	MetricCommentedTitlesTotal         MetricCode = "commented_titles_total"
	MetricRepliesAuthoredTotal         MetricCode = "replies_authored_total"
	MetricReceivedReplyExists          MetricCode = "received_reply_exists"
	MetricWatchlistHighWater           MetricCode = "watchlist_high_water"
	MetricRatedBefore1980Total         MetricCode = "rated_before_1980_total"
	MetricRatedBefore2000Total         MetricCode = "rated_before_2000_total"
	MetricFullRatingDistinctScoreMax   MetricCode = "full_rating_distinct_score_max"
	MetricFullRatingSameScoreExists    MetricCode = "full_rating_same_score_exists"
	MetricRatingSameDayMax             MetricCode = "rating_same_day_max"
	MetricFriendEarlierTitlesTotal     MetricCode = "friend_earlier_titles_total"
	MetricUserEarlierTitlesTotal       MetricCode = "user_earlier_titles_total"
	MetricTitleFriendRatersMax         MetricCode = "title_friend_raters_max"
	MetricFriendsWithSharedTitleTotal  MetricCode = "friends_with_shared_title_total"
	MetricFriendFarRatingsMax          MetricCode = "friend_far_ratings_max"
	MetricTitleSameAvgFriendCountMax   MetricCode = "title_same_avg_friend_count_max"
	MetricRatedAndCommentedTitlesTotal MetricCode = "rated_and_commented_titles_total"
	MetricDistinctRepliersTotal        MetricCode = "distinct_repliers_total"
	MetricRootDirectRepliesMax         MetricCode = "root_direct_replies_max"
	MetricFriendWatchlistMatchExists   MetricCode = "friend_watchlist_match_exists"
)

var FixedCriterionCodes = []string{
	"story", "characters", "acting", "direction", "visuals", "sound", "atmosphere",
}

type Definition struct {
	Code        string     `json:"code"`
	Metric      MetricCode `json:"metric"`
	Target      int64      `json:"target"`
	XP          int        `json:"xp"`
	Category    Category   `json:"category"`
	Secret      bool       `json:"secret"`
	Icon        string     `json:"icon"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	SortOrder   int        `json:"sort_order"`
}

var catalog = []Definition{
	{Code: "first_frame", Metric: MetricRatingsTotal, Target: 1, XP: 50, Category: CategoryRatings, Icon: "🎬", Title: "Первый кадр", Description: "Оценить первый тайтл", SortOrder: 1},
	{Code: "warmup", Metric: MetricRatingsTotal, Target: 5, XP: 50, Category: CategoryRatings, Icon: "🍿", Title: "Разогрев", Description: "Оценить 5 тайтлов", SortOrder: 2},
	{Code: "solid_ten", Metric: MetricRatingsTotal, Target: 10, XP: 100, Category: CategoryRatings, Icon: "🔟", Title: "Крепкая десятка", Description: "Оценить 10 тайтлов", SortOrder: 3},
	{Code: "movie_marathon", Metric: MetricRatingsTotal, Target: 25, XP: 200, Category: CategoryRatings, Icon: "🏃", Title: "Киномарафон", Description: "Оценить 25 тайтлов", SortOrder: 4},
	{Code: "full_shelf", Metric: MetricRatingsTotal, Target: 50, XP: 350, Category: CategoryRatings, Icon: "🗄️", Title: "Полная полка", Description: "Оценить 50 тайтлов", SortOrder: 5},
	{Code: "hundred_credits", Metric: MetricRatingsTotal, Target: 100, XP: 500, Category: CategoryRatings, Icon: "💯", Title: "Сотня в титрах", Description: "Оценить 100 тайтлов", SortOrder: 6},
	{Code: "thoughtful_viewer", Metric: MetricFullRatingsTotal, Target: 1, XP: 50, Category: CategoryTaste, Icon: "🧐", Title: "Вдумчивый зритель", Description: "Оценить тайтл по всем 7 критериям", SortOrder: 7},
	{Code: "broken_down", Metric: MetricFullRatingsTotal, Target: 10, XP: 200, Category: CategoryTaste, Icon: "🧩", Title: "По полочкам", Description: "Поставить 10 полных оценок по 7 критериям", SortOrder: 8},
	{Code: "genre_loyalty", Metric: MetricRatingsSameGenreMax, Target: 10, XP: 200, Category: CategoryTaste, Icon: "🎭", Title: "Верность жанру", Description: "Оценить 10 тайтлов одного жанра", SortOrder: 9},
	{Code: "time_machine", Metric: MetricRatedReleaseDecadesTotal, Target: 5, XP: 200, Category: CategoryTaste, Icon: "⏳", Title: "Машина времени", Description: "Оценить тайтлы из 5 разных десятилетий", SortOrder: 10},
	{Code: "love_and_hate", Metric: MetricRatingScoreContrast, Target: 1, XP: 100, Category: CategoryTaste, Secret: true, Icon: "❤️‍🔥", Title: "Между любовью и ненавистью", Description: "Иметь оценки не выше 3 и не ниже 9", SortOrder: 11},
	{Code: "absolute_ten", Metric: MetricPerfectSevenRatingExists, Target: 1, XP: 350, Category: CategoryTaste, Secret: true, Icon: "💎", Title: "Абсолютная десятка", Description: "Поставить по 10 по всем 7 критериям", SortOrder: 12},
	{Code: "two_screens", Metric: MetricRatedMediaBalance, Target: 5, XP: 100, Category: CategoryTaste, Icon: "🖥️", Title: "Два экрана", Description: "Оценить не менее 5 фильмов и 5 сериалов", SortOrder: 13},
	{Code: "big_screen", Metric: MetricRatedMoviesTotal, Target: 20, XP: 200, Category: CategoryTaste, Icon: "🎥", Title: "Большой экран", Description: "Оценить 20 фильмов", SortOrder: 14},
	{Code: "one_more_episode", Metric: MetricRatedTVTotal, Target: 20, XP: 200, Category: CategoryTaste, Icon: "📺", Title: "Ещё одну серию", Description: "Оценить 20 сериалов", SortOrder: 15},
	{Code: "omnivore", Metric: MetricRatedGenresTotal, Target: 10, XP: 200, Category: CategoryTaste, Icon: "🌈", Title: "Всеядный", Description: "Охватить 10 разных жанров", SortOrder: 16},
	{Code: "three_evenings", Metric: MetricRatingDayStreak, Target: 3, XP: 100, Category: CategoryRatings, Icon: "🌙", Title: "Три вечера подряд", Description: "Оценивать тайтлы 3 календарных дня подряд", SortOrder: 17},
	{Code: "week_in_cinema", Metric: MetricRatingDayStreak, Target: 7, XP: 200, Category: CategoryRatings, Icon: "📅", Title: "Неделя в кино", Description: "Оценивать тайтлы 7 календарных дней подряд", SortOrder: 18},
	{Code: "first_in_circle", Metric: MetricFriendsHighWater, Target: 1, XP: 50, Category: CategorySocial, Icon: "🤝", Title: "Первый в круге", Description: "Завести первого друга", SortOrder: 19},
	{Code: "own_company", Metric: MetricFriendsHighWater, Target: 5, XP: 200, Category: CategorySocial, Icon: "🥳", Title: "Своя компания", Description: "Достичь 5 друзей", SortOrder: 20},
	{Code: "shared_screen", Metric: MetricSharedTitlesTotal, Target: 1, XP: 50, Category: CategorySocial, Icon: "👀", Title: "Общий сеанс", Description: "Оценить хотя бы один тайтл, оценённый другом", SortOrder: 21},
	{Code: "cinema_club", Metric: MetricSharedTitlesTotal, Target: 10, XP: 200, Category: CategorySocial, Icon: "🎟️", Title: "Киноклуб", Description: "Иметь 10 общих оценённых тайтлов с друзьями", SortOrder: 22},
	{Code: "same_wavelength", Metric: MetricFriendCloseRatingsMax, Target: 5, XP: 200, Category: CategorySocial, Icon: "🌊", Title: "На одной волне", Description: "На 5 общих тайтлах отличаться с одним другом не больше чем на 0,5", SortOrder: 23},
	{Code: "synchronized_take", Metric: MetricFriendExactFullRatingExists, Target: 1, XP: 350, Category: CategorySocial, Secret: true, Icon: "🪞", Title: "Синхронный дубль", Description: "Полностью совпасть с другом по всем 7 критериям одного тайтла", SortOrder: 24},
	{Code: "voice_from_the_audience", Metric: MetricCommentsTotal, Target: 1, XP: 50, Category: CategoryComments, Icon: "💬", Title: "Реплика из зала", Description: "Оставить первый комментарий", SortOrder: 25},
	{Code: "something_to_say", Metric: MetricCommentedTitlesTotal, Target: 5, XP: 100, Category: CategoryComments, Icon: "🗣️", Title: "Есть что сказать", Description: "Комментировать 5 разных тайтлов", SortOrder: 26},
	{Code: "in_dialogue", Metric: MetricRepliesAuthoredTotal, Target: 5, XP: 100, Category: CategoryComments, Icon: "↩️", Title: "В диалоге", Description: "Написать 5 ответов на комментарии", SortOrder: 27},
	{Code: "conversation_started", Metric: MetricReceivedReplyExists, Target: 1, XP: 100, Category: CategoryComments, Icon: "🔔", Title: "Завязалась беседа", Description: "Получить ответ другого пользователя на свой комментарий", SortOrder: 28},
	{Code: "noted", Metric: MetricWatchlistHighWater, Target: 1, XP: 50, Category: CategoryWatchlist, Icon: "🔖", Title: "На заметку", Description: "Добавить первый тайтл в список «Хочу посмотреть»", SortOrder: 29},
	{Code: "evening_supply", Metric: MetricWatchlistHighWater, Target: 10, XP: 100, Category: CategoryWatchlist, Icon: "🧺", Title: "Запас на вечер", Description: "Одновременно собрать 10 тайтлов в списке «Хочу посмотреть»", SortOrder: 30},
	{Code: "archivist", Metric: MetricRatedBefore1980Total, Target: 5, XP: 100, Category: CategoryTaste, Icon: "📼", Title: "Архивариус", Description: "Оценить 5 тайтлов, выпущенных до 1980 года", SortOrder: 31},
	{Code: "classics_never_age", Metric: MetricRatedBefore2000Total, Target: 15, XP: 200, Category: CategoryTaste, Icon: "🏛️", Title: "Классика не стареет", Description: "Оценить 15 тайтлов, выпущенных до 2000 года", SortOrder: 32},
	{Code: "genre_atlas", Metric: MetricRatedGenresTotal, Target: 15, XP: 350, Category: CategoryTaste, Icon: "🗺️", Title: "Жанровый атлас", Description: "Охватить 15 разных жанров", SortOrder: 33},
	{Code: "seven_facets", Metric: MetricFullRatingsTotal, Target: 25, XP: 350, Category: CategoryTaste, Icon: "🔬", Title: "Семь граней", Description: "Поставить 25 полных оценок по 7 критериям", SortOrder: 34},
	{Code: "second_hundred", Metric: MetricRatingsTotal, Target: 200, XP: 500, Category: CategoryRatings, Icon: "🎞️", Title: "Вторая сотня", Description: "Оценить 200 уникальных тайтлов", SortOrder: 35},
	{Code: "spectrum", Metric: MetricFullRatingDistinctScoreMax, Target: 7, XP: 200, Category: CategoryTaste, Secret: true, Icon: "🎨", Title: "Спектр", Description: "В одной полной оценке использовать 7 разных значений", SortOrder: 36},
	{Code: "unanimity", Metric: MetricFullRatingSameScoreExists, Target: 1, XP: 100, Category: CategoryTaste, Secret: true, Icon: "🎯", Title: "Единодушие", Description: "В одной полной оценке поставить одинаковый балл по всем критериям", SortOrder: 37},
	{Code: "triple_screening", Metric: MetricRatingSameDayMax, Target: 3, XP: 100, Category: CategoryRatings, Secret: true, Icon: "🎦", Title: "Тройной сеанс", Description: "Оценить 3 разных тайтла за один календарный день", SortOrder: 38},
	{Code: "following_a_friend", Metric: MetricFriendEarlierTitlesTotal, Target: 5, XP: 200, Category: CategorySocial, Icon: "👣", Title: "По следам друга", Description: "Оценить 5 тайтлов после того, как их оценил текущий друг", SortOrder: 39},
	{Code: "pioneer", Metric: MetricUserEarlierTitlesTotal, Target: 5, XP: 200, Category: CategorySocial, Icon: "🧭", Title: "Первооткрыватель", Description: "Первым оценить 5 тайтлов, которые позже оценили текущие друзья", SortOrder: 40},
	{Code: "three_seats", Metric: MetricTitleFriendRatersMax, Target: 2, XP: 100, Category: CategorySocial, Icon: "🪑", Title: "Три места рядом", Description: "Найти тайтл, оценённый вами и двумя друзьями", SortOrder: 41},
	{Code: "full_house", Metric: MetricTitleFriendRatersMax, Target: 4, XP: 350, Category: CategorySocial, Icon: "🎪", Title: "Полный зал", Description: "Найти тайтл, оценённый вами и четырьмя друзьями", SortOrder: 42},
	{Code: "wide_circle", Metric: MetricFriendsWithSharedTitleTotal, Target: 3, XP: 200, Category: CategorySocial, Icon: "🫂", Title: "Широкий круг", Description: "Иметь общий оценённый тайтл с тремя разными друзьями", SortOrder: 43},
	{Code: "opposites_attract", Metric: MetricFriendFarRatingsMax, Target: 5, XP: 200, Category: CategorySocial, Secret: true, Icon: "🧲", Title: "Противоположности притягиваются", Description: "На 5 общих тайтлах отличаться с одним другом не меньше чем на 3", SortOrder: 44},
	{Code: "choir", Metric: MetricTitleSameAvgFriendCountMax, Target: 2, XP: 200, Category: CategorySocial, Secret: true, Icon: "🎶", Title: "Хор", Description: "Получить одинаковую итоговую оценку одного тайтла у себя и двух друзей", SortOrder: 45},
	{Code: "discussion_regular", Metric: MetricCommentedTitlesTotal, Target: 10, XP: 200, Category: CategoryComments, Icon: "🛋️", Title: "Завсегдатай обсуждений", Description: "Комментировать 10 разных тайтлов", SortOrder: 46},
	{Code: "critic_with_explanation", Metric: MetricRatedAndCommentedTitlesTotal, Target: 10, XP: 200, Category: CategoryComments, Icon: "✍️", Title: "Критик с пояснением", Description: "И оценить, и прокомментировать 10 одинаковых тайтлов", SortOrder: 47},
	{Code: "echo", Metric: MetricDistinctRepliersTotal, Target: 3, XP: 200, Category: CategoryComments, Icon: "🗯️", Title: "Эхо", Description: "Получить ответы от трёх разных пользователей", SortOrder: 48},
	{Code: "hot_thread", Metric: MetricRootDirectRepliesMax, Target: 5, XP: 350, Category: CategoryComments, Secret: true, Icon: "🔥", Title: "Горячая ветка", Description: "Получить под своим корневым комментарием 5 прямых ответов хотя бы от двух пользователей", SortOrder: 49},
	{Code: "matching_plans", Metric: MetricFriendWatchlistMatchExists, Target: 1, XP: 100, Category: CategoryWatchlist, Secret: true, Icon: "🔗", Title: "Планы совпали", Description: "Одновременно иметь один и тот же тайтл в списке «Хочу посмотреть» с другом", SortOrder: 50},
}

func Definitions() []Definition {
	result := append([]Definition(nil), catalog...)
	sort.Slice(result, func(i, j int) bool { return result[i].SortOrder < result[j].SortOrder })
	return result
}

func ValidateCatalog(definitions []Definition) error {
	if len(definitions) == 0 {
		return errors.New("achievement catalog is empty")
	}
	allowedXP := map[int]bool{50: true, 100: true, 200: true, 350: true, 500: true}
	knownCategories := map[Category]bool{
		CategoryRatings: true, CategoryTaste: true, CategorySocial: true,
		CategoryComments: true, CategoryWatchlist: true,
	}
	knownMetrics := make(map[MetricCode]bool)
	for _, metric := range allMetrics() {
		knownMetrics[metric] = true
	}
	codes := make(map[string]bool)
	pairs := make(map[string]bool)
	orders := make(map[int]bool)
	for _, definition := range definitions {
		if definition.Code == "" || codes[definition.Code] {
			return fmt.Errorf("duplicate or empty achievement code %q", definition.Code)
		}
		codes[definition.Code] = true
		pair := fmt.Sprintf("%s:%d", definition.Metric, definition.Target)
		if pairs[pair] {
			return fmt.Errorf("duplicate achievement metric target %s", pair)
		}
		pairs[pair] = true
		if definition.SortOrder <= 0 || orders[definition.SortOrder] {
			return fmt.Errorf("invalid achievement sort order %d", definition.SortOrder)
		}
		orders[definition.SortOrder] = true
		if !knownMetrics[definition.Metric] || !knownCategories[definition.Category] {
			return fmt.Errorf("unknown achievement metric or category for %s", definition.Code)
		}
		if definition.Target <= 0 || !allowedXP[definition.XP] || definition.Icon == "" || definition.Title == "" || definition.Description == "" {
			return fmt.Errorf("invalid achievement definition %s", definition.Code)
		}
	}
	return nil
}

func CatalogFingerprint(definitions []Definition, evaluatorVersion int) (string, error) {
	if err := ValidateCatalog(definitions); err != nil {
		return "", err
	}
	ordered := append([]Definition(nil), definitions...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Code < ordered[j].Code })
	payload, err := json.Marshal(struct {
		EvaluatorVersion int          `json:"evaluator_version"`
		Definitions      []Definition `json:"definitions"`
	}{EvaluatorVersion: evaluatorVersion, Definitions: ordered})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func Level(totalXP int) int {
	if totalXP <= 0 {
		return 1
	}
	return 1 + int(math.Floor(math.Sqrt(float64(totalXP)/50)))
}

func RequiredXP(level int) int {
	if level <= 1 {
		return 0
	}
	return 50 * (level - 1) * (level - 1)
}

func RankTitle(level int) string {
	switch {
	case level >= 15:
		return "Алмаз КиноКруга"
	case level >= 13:
		return "Легенда КиноКруга"
	case level >= 10:
		return "Куратор"
	case level >= 7:
		return "Знаток"
	case level >= 4:
		return "Киноман"
	default:
		return "Зритель"
	}
}

func allMetrics() []MetricCode {
	return []MetricCode{
		MetricRatingsTotal, MetricFullRatingsTotal, MetricRatingsSameGenreMax,
		MetricRatedReleaseDecadesTotal, MetricRatingScoreContrast, MetricPerfectSevenRatingExists,
		MetricRatedMediaBalance, MetricRatedMoviesTotal, MetricRatedTVTotal, MetricRatedGenresTotal,
		MetricRatingDayStreak, MetricFriendsHighWater, MetricSharedTitlesTotal,
		MetricFriendCloseRatingsMax, MetricFriendExactFullRatingExists, MetricCommentsTotal,
		MetricCommentedTitlesTotal, MetricRepliesAuthoredTotal, MetricReceivedReplyExists,
		MetricWatchlistHighWater, MetricRatedBefore1980Total, MetricRatedBefore2000Total,
		MetricFullRatingDistinctScoreMax, MetricFullRatingSameScoreExists, MetricRatingSameDayMax,
		MetricFriendEarlierTitlesTotal, MetricUserEarlierTitlesTotal, MetricTitleFriendRatersMax,
		MetricFriendsWithSharedTitleTotal, MetricFriendFarRatingsMax, MetricTitleSameAvgFriendCountMax,
		MetricRatedAndCommentedTitlesTotal, MetricDistinctRepliersTotal, MetricRootDirectRepliesMax,
		MetricFriendWatchlistMatchExists,
	}
}
