package domain

import "time"

type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)

type SearchResult struct {
	TmdbID        int64     `json:"tmdb_id"`
	MediaType     MediaType `json:"media_type"`
	Title         string    `json:"title"`
	OriginalTitle string    `json:"original_title,omitempty"`
	ReleaseYear   int       `json:"release_year,omitempty"`
	PosterPath    string    `json:"poster_path,omitempty"`
	Overview      string    `json:"overview,omitempty"`
}

type SearchPage struct {
	Page         int            `json:"page"`
	TotalPages   int            `json:"total_pages"`
	TotalResults int            `json:"total_results"`
	Results      []SearchResult `json:"results"`
}

type TitleRef struct {
	TmdbID    int64     `json:"tmdb_id"`
	MediaType MediaType `json:"media_type"`
}

type CatalogCandidate struct {
	Title      Title   `json:"title"`
	Popularity float64 `json:"-"`
	GenreIDs   []int64 `json:"-"`
}

type CatalogProviderPage struct {
	Page       int
	TotalPages int
	Results    []CatalogCandidate
}

type CatalogItem struct {
	Title       Title  `json:"title"`
	InWatchlist bool   `json:"in_watchlist"`
	Reason      string `json:"reason,omitempty"`
}

type CatalogPage struct {
	Items        []CatalogItem `json:"items"`
	NextCursor   string        `json:"next_cursor,omitempty"`
	Personalized bool          `json:"personalized,omitempty"`
	Degraded     bool          `json:"degraded"`
}

type CatalogSearchPage struct {
	Page         int           `json:"page"`
	TotalPages   int           `json:"total_pages"`
	TotalResults int           `json:"total_results"`
	Results      []CatalogItem `json:"results"`
}

type Title struct {
	ID            int64     `json:"id,omitempty"`
	TmdbID        int64     `json:"tmdb_id"`
	MediaType     MediaType `json:"media_type"`
	Title         string    `json:"title"`
	OriginalTitle string    `json:"original_title,omitempty"`
	ReleaseYear   int       `json:"release_year,omitempty"`
	PosterPath    string    `json:"poster_path,omitempty"`
	Genres        []string  `json:"genres,omitempty"`
	Overview      string    `json:"overview,omitempty"`
}

type Criterion struct {
	ID          int16  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int16  `json:"sort_order"`
}

type Rating struct {
	ID        int64          `json:"id"`
	UserID    int64          `json:"-"`
	TitleID   int64          `json:"title_id"`
	AvgScore  float64        `json:"avg_score"`
	Scores    map[string]int `json:"scores"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type RatingWithUser struct {
	User      User           `json:"user"`
	AvgScore  float64        `json:"avg_score"`
	Scores    map[string]int `json:"scores"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type FriendsAverage struct {
	Overall    float64            `json:"overall"`
	ByCriteria map[string]float64 `json:"by_criteria"`
}

type TitleCard struct {
	Title          Title            `json:"title"`
	MyRating       *RatingWithUser  `json:"my_rating"`
	FriendsRatings []RatingWithUser `json:"friends_ratings"`
	FriendsAvg     *FriendsAverage  `json:"friends_avg"`
	CommentsCount  int64            `json:"comments_count"`
	InWatchlist    bool             `json:"in_watchlist"`
}

type ProfileRating struct {
	ID        int64          `json:"-"`
	Title     Title          `json:"title"`
	AvgScore  float64        `json:"avg_score"`
	Scores    map[string]int `json:"scores"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type ProfileRatingStats struct {
	Count    int     `json:"count"`
	AvgScore float64 `json:"avg_score"`
}

type ProfileRatingsPage struct {
	User         User               `json:"user"`
	Relationship string             `json:"relationship"`
	Ratings      []ProfileRating    `json:"ratings"`
	Stats        ProfileRatingStats `json:"stats"`
	NextCursor   string             `json:"next_cursor,omitempty"`
}

type WatchlistItem struct {
	Title   Title     `json:"title"`
	AddedAt time.Time `json:"added_at"`
}

type WatchlistPage struct {
	Items      []WatchlistItem `json:"items"`
	TotalCount int64           `json:"total_count"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

type WatchlistMatchItem struct {
	Title         Title     `json:"title"`
	Users         []User    `json:"users"`
	MatchesCount  int       `json:"matches_count"`
	LatestAddedAt time.Time `json:"-"`
}

type WatchlistMatchesPage struct {
	Items      []WatchlistMatchItem `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

type RecommendationSeed struct {
	RatingID  int64
	Title     Title
	AvgScore  float64
	UpdatedAt time.Time
}

type GenreRating struct {
	AvgScore float64
	Genres   []string
}

type FeedItem struct {
	ID        int64          `json:"id"`
	User      User           `json:"user"`
	Title     Title          `json:"title"`
	AvgScore  float64        `json:"avg_score"`
	Scores    map[string]int `json:"scores"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type FeedPage struct {
	Items      []FeedItem `json:"items"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

type Comment struct {
	ID        int64     `json:"id"`
	TitleID   int64     `json:"title_id"`
	User      User      `json:"user"`
	ParentID  int64     `json:"parent_id,omitempty"`
	Body      string    `json:"body"`
	IsDeleted bool      `json:"is_deleted"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Replies   []Comment `json:"replies,omitempty"`
}
