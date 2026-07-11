package domain

import "time"

type User struct {
	ID        int64     `json:"id"`
	TgID      int64     `json:"tg_id"`
	Username  string    `json:"username,omitempty"`
	FirstName string    `json:"first_name"`
	PhotoURL  string    `json:"photo_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
