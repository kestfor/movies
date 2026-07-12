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
}

type ProfileRating struct {
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
