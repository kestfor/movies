package postgres

import (
	"context"
	"errors"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"

	"github.com/jackc/pgx/v5"
)

type FriendRepository struct {
	queries *gen.Queries
}

func NewFriendRepository(queries *gen.Queries) *FriendRepository {
	return &FriendRepository{queries: queries}
}

func (r *FriendRepository) GetUserByUUID(ctx context.Context, rawUUID string) (domain.User, bool, error) {
	uuid, ok := uuidFromString(rawUUID)
	if !ok {
		return domain.User{}, false, nil
	}
	user, err := r.queries.GetUserByUUID(ctx, uuid)
	if err == nil {
		return toDomainUser(user), true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, false, nil
	}
	return domain.User{}, false, err
}

func (r *FriendRepository) ListFriends(ctx context.Context, userID int64) ([]domain.User, error) {
	users, err := r.queries.ListAcceptedFriends(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.User, 0, len(users))
	for _, user := range users {
		result = append(result, toDomainUser(user))
	}
	return result, nil
}

func (r *FriendRepository) ListIncomingRequests(ctx context.Context, userID int64) ([]domain.FriendRequest, error) {
	rows, err := r.queries.ListIncomingFriendRequests(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.FriendRequest, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.FriendRequest{
			User: domain.User{
				ID:        row.ID,
				UUID:      uuidToString(row.Uuid),
				TgID:      row.TgID,
				Username:  textToString(row.Username),
				FirstName: row.FirstName,
				PhotoURL:  textToString(row.PhotoUrl),
				CreatedAt: row.CreatedAt.Time,
			},
			RequestedAt: row.RequestedAt.Time,
		})
	}
	return result, nil
}

func (r *FriendRepository) GetBetween(ctx context.Context, userID, otherUserID int64) (domain.Friendship, bool, error) {
	friendship, err := r.queries.GetFriendshipBetweenUsers(ctx, gen.GetFriendshipBetweenUsersParams{
		RequesterID: userID,
		AddresseeID: otherUserID,
	})
	if err == nil {
		return toDomainFriendship(friendship), true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Friendship{}, false, nil
	}
	return domain.Friendship{}, false, err
}

func (r *FriendRepository) CreateRequest(ctx context.Context, requesterID, addresseeID int64) (domain.Friendship, error) {
	friendship, err := r.queries.CreateFriendRequest(ctx, gen.CreateFriendRequestParams{
		RequesterID: requesterID,
		AddresseeID: addresseeID,
	})
	if err != nil {
		return domain.Friendship{}, err
	}
	return toDomainFriendship(friendship), nil
}

func (r *FriendRepository) AcceptRequest(ctx context.Context, requesterID, addresseeID int64) (domain.Friendship, bool, error) {
	friendship, err := r.queries.AcceptFriendRequest(ctx, gen.AcceptFriendRequestParams{
		RequesterID: requesterID,
		AddresseeID: addresseeID,
	})
	if err == nil {
		return toDomainFriendship(friendship), true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Friendship{}, false, nil
	}
	return domain.Friendship{}, false, err
}

func (r *FriendRepository) DeletePendingRequest(ctx context.Context, userID, otherUserID int64) (bool, error) {
	rows, err := r.queries.DeletePendingFriendRequestBetweenUsers(ctx, gen.DeletePendingFriendRequestBetweenUsersParams{
		RequesterID: userID,
		AddresseeID: otherUserID,
	})
	return rows > 0, err
}

func (r *FriendRepository) DeleteFriend(ctx context.Context, userID, otherUserID int64) (bool, error) {
	rows, err := r.queries.DeleteAcceptedFriendshipBetweenUsers(ctx, gen.DeleteAcceptedFriendshipBetweenUsersParams{
		RequesterID: userID,
		AddresseeID: otherUserID,
	})
	return rows > 0, err
}

func toDomainFriendship(friendship gen.Friendship) domain.Friendship {
	result := domain.Friendship{
		RequesterID: friendship.RequesterID,
		AddresseeID: friendship.AddresseeID,
		Status:      domain.FriendshipStatus(friendship.Status),
		CreatedAt:   friendship.CreatedAt.Time,
	}
	if friendship.RespondedAt.Valid {
		result.RespondedAt = friendship.RespondedAt.Time
	}
	return result
}
