package domain

import "time"

type ActivityEventKind string

const (
	ActivityEventKindRatingCreated  ActivityEventKind = "rating_created"
	ActivityEventKindCommentCreated ActivityEventKind = "comment_created"
)

type NotificationComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

type NotificationRating struct {
	ID       int64   `json:"id"`
	AvgScore float64 `json:"avg_score"`
}

type Notification struct {
	EventID   int64                `json:"event_id"`
	Kind      ActivityEventKind    `json:"kind"`
	Actor     User                 `json:"actor"`
	Title     Title                `json:"title"`
	Rating    *NotificationRating  `json:"rating,omitempty"`
	Comment   *NotificationComment `json:"comment,omitempty"`
	ReadAt    *time.Time           `json:"read_at,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
	DeepLink  string               `json:"deep_link"`
}

type NotificationsPage struct {
	Items      []Notification `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}
