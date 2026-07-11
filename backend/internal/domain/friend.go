package domain

import "time"

type FriendshipStatus string

const (
	FriendshipStatusPending  FriendshipStatus = "pending"
	FriendshipStatusAccepted FriendshipStatus = "accepted"
)

type Friendship struct {
	RequesterID int64            `json:"-"`
	AddresseeID int64            `json:"-"`
	Status      FriendshipStatus `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	RespondedAt time.Time        `json:"responded_at,omitempty"`
}

type FriendRequest struct {
	User        User      `json:"user"`
	RequestedAt time.Time `json:"requested_at"`
}
