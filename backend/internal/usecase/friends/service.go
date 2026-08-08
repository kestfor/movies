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
	GetUserByUUID(ctx context.Context, uuid string) (domain.User, bool, error)
	ListFriends(ctx context.Context, userID int64) ([]domain.User, error)
	ListIncomingRequests(ctx context.Context, userID int64) ([]domain.FriendRequest, error)
	GetBetween(ctx context.Context, userID, otherUserID int64) (domain.Friendship, bool, error)
	CreateRequest(ctx context.Context, requesterID, addresseeID int64) (domain.Friendship, error)
	AcceptRequest(ctx context.Context, requesterID, addresseeID int64) (domain.Friendship, bool, error)
	DeletePendingRequest(ctx context.Context, userID, otherUserID int64) (bool, error)
	DeleteFriend(ctx context.Context, userID, otherUserID int64) (bool, error)
}

type AchievementObserver interface {
	ObserveCircle(ctx context.Context, userID int64)
}

type Service struct {
	repo         Repository
	achievements AchievementObserver
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetAchievementObserver(observer AchievementObserver) {
	s.achievements = observer
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
		if s.achievements != nil {
			s.achievements.ObserveCircle(ctx, requesterID)
		}
		return accepted, nil
	}

	return s.repo.CreateRequest(ctx, requesterID, addresseeID)
}

func (s *Service) CreateRequestByUUID(ctx context.Context, requesterID int64, addresseeUUID string) (domain.Friendship, error) {
	if requesterID == 0 || addresseeUUID == "" {
		return domain.Friendship{}, ErrValidation
	}
	addressee, ok, err := s.repo.GetUserByUUID(ctx, addresseeUUID)
	if err != nil {
		return domain.Friendship{}, err
	}
	if !ok {
		return domain.Friendship{}, ErrNotFound
	}
	return s.CreateRequest(ctx, requesterID, addressee.ID)
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
	if s.achievements != nil {
		s.achievements.ObserveCircle(ctx, currentUserID)
	}
	return friendship, nil
}

func (s *Service) AcceptRequestByUUID(ctx context.Context, currentUserID int64, requesterUUID string) (domain.Friendship, error) {
	if currentUserID == 0 || requesterUUID == "" {
		return domain.Friendship{}, ErrValidation
	}
	requester, ok, err := s.repo.GetUserByUUID(ctx, requesterUUID)
	if err != nil {
		return domain.Friendship{}, err
	}
	if !ok {
		return domain.Friendship{}, ErrNotFound
	}
	return s.AcceptRequest(ctx, currentUserID, requester.ID)
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

func (s *Service) DeleteRequestByUUID(ctx context.Context, currentUserID int64, otherUserUUID string) error {
	if currentUserID == 0 || otherUserUUID == "" {
		return ErrValidation
	}
	otherUser, ok, err := s.repo.GetUserByUUID(ctx, otherUserUUID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return s.DeleteRequest(ctx, currentUserID, otherUser.ID)
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

func (s *Service) DeleteFriendByUUID(ctx context.Context, currentUserID int64, friendUUID string) error {
	if currentUserID == 0 || friendUUID == "" {
		return ErrValidation
	}
	friend, ok, err := s.repo.GetUserByUUID(ctx, friendUUID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return s.DeleteFriend(ctx, currentUserID, friend.ID)
}
