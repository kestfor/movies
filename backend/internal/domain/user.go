package domain

import "time"

type User struct {
	ID        int64     `json:"-"`
	UUID      string    `json:"uuid"`
	TgID      int64     `json:"-"`
	Username  string    `json:"username,omitempty"`
	FirstName string    `json:"first_name"`
	PhotoURL  string    `json:"photo_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type UserSearchResult struct {
	User             User   `json:"user"`
	Relationship     string `json:"relationship"`
	CanSendRequest   bool   `json:"can_send_request"`
	CanOpenProfile   bool   `json:"can_open_profile"`
	CanAcceptRequest bool   `json:"can_accept_request"`
}
