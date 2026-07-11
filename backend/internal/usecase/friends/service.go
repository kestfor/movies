package friends

import (
	"context"
	"errors"

	"movies/backend/internal/domain"
)

var (
	ErrValidation = errors.New("validation_failed")
	ErrConflict   = errors.New("conflict")
	ErrNotFound   = errors.New("not_found")
	ErrForbidden  = errors.New("forbidden")
)

type Repository interface {
	ListFriends(ctx context.Context, userID int64) ([]domain.User, error)
	ListIncomingRequests(ctx context.Context, userID int64) ([]domain.FriendRequest, error)
	GetBetween(ctx context.Context, userID, otherUserID int64) (domain.Friendship, bool, error)
	CreateRequest(ctx context.Context, requesterID, addresseeID int64) (domain.Friendship, error)
	AcceptRequest(ctx context.Context, requesterID, addresseeID int64) (domain.Friendship, bool, error)
	DeletePendingRequest(ctx context.Context, userID, otherUserID int64) (bool, error)
	DeleteFriend(ctx context.Context, userID, otherUserID int64) (bool, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListFriends(ctx context.Context, userID int64) ([]domain.User, error) {
	if userID == 0 {
		return nil, ErrValidation
	}
	return s.repo.ListFriends(ctx, userID)
}

func (s *Service) ListIncomingRequests(ctx context.Context, userID int64) ([]domain.FriendRequest, error) {
	if userID == 0 {
		return nil, ErrValidation
	}
	return s.repo.ListIncomingRequests(ctx, userID)
}

func (s *Service) CreateRequest(ctx context.Context, requesterID, addresseeID int64) (domain.Friendship, error) {
	if requesterID == 0 || addresseeID == 0 || requesterID == addresseeID {
		return domain.Friendship{}, ErrValidation
	}

	existing, ok, err := s.repo.GetBetween(ctx, requesterID, addresseeID)
	if err != nil {
		return domain.Friendship{}, err
	}
	if ok {
		if existing.Status == domain.FriendshipStatusAccepted {
			return domain.Friendship{}, ErrConflict
		}
		if existing.RequesterID == requesterID {
			return domain.Friendship{}, ErrConflict
		}
		accepted, ok, err := s.repo.AcceptRequest(ctx, existing.RequesterID, existing.AddresseeID)
		if err != nil {
			return domain.Friendship{}, err
		}
		if !ok {
			return domain.Friendship{}, ErrConflict
		}
		return accepted, nil
	}

	return s.repo.CreateRequest(ctx, requesterID, addresseeID)
}

func (s *Service) AcceptRequest(ctx context.Context, currentUserID, requesterID int64) (domain.Friendship, error) {
	if currentUserID == 0 || requesterID == 0 || currentUserID == requesterID {
		return domain.Friendship{}, ErrValidation
	}

	friendship, ok, err := s.repo.AcceptRequest(ctx, requesterID, currentUserID)
	if err != nil {
		return domain.Friendship{}, err
	}
	if !ok {
		return domain.Friendship{}, ErrNotFound
	}
	return friendship, nil
}

func (s *Service) DeleteRequest(ctx context.Context, currentUserID, otherUserID int64) error {
	if currentUserID == 0 || otherUserID == 0 || currentUserID == otherUserID {
		return ErrValidation
	}

	deleted, err := s.repo.DeletePendingRequest(ctx, currentUserID, otherUserID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

func (s *Service) DeleteFriend(ctx context.Context, currentUserID, friendID int64) error {
	if currentUserID == 0 || friendID == 0 || currentUserID == friendID {
		return ErrValidation
	}

	deleted, err := s.repo.DeleteFriend(ctx, currentUserID, friendID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}
