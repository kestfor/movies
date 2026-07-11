package postgres

import (
	"context"
	"errors"
	"strings"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"
	"movies/backend/internal/usecase/auth"

	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	queries *gen.Queries
}

func NewUserRepository(queries *gen.Queries) *UserRepository {
	return &UserRepository{queries: queries}
}

func (r *UserRepository) UpsertTelegramUser(ctx context.Context, params auth.UpsertTelegramUserParams) (domain.User, error) {
	user, err := r.queries.UpsertUserByTelegramID(ctx, gen.UpsertUserByTelegramIDParams{
		TgID:      params.TgID,
		Username:  toNullText(params.Username),
		FirstName: params.FirstName,
		PhotoUrl:  toNullText(params.PhotoURL),
	})
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(user), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (domain.User, error) {
	user, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(user), nil
}

func (r *UserRepository) GetByUUID(ctx context.Context, rawUUID string) (domain.User, bool, error) {
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

func (r *UserRepository) SearchByUsernamePrefix(ctx context.Context, currentUserID int64, query string, limit int32) ([]domain.UserSearchResult, error) {
	query = strings.TrimPrefix(strings.TrimSpace(query), "@")
	if query == "" {
		return []domain.UserSearchResult{}, nil
	}
	rows, err := r.queries.SearchUsersByUsernamePrefix(ctx, gen.SearchUsersByUsernamePrefixParams{
		ID:    currentUserID,
		Lower: query,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	result := make([]domain.UserSearchResult, 0, len(rows))
	for _, row := range rows {
		relationship := row.Relationship
		result = append(result, domain.UserSearchResult{
			User: domain.User{
				ID:        row.ID,
				UUID:      uuidToString(row.Uuid),
				TgID:      row.TgID,
				Username:  textToString(row.Username),
				FirstName: row.FirstName,
				PhotoURL:  textToString(row.PhotoUrl),
				CreatedAt: row.CreatedAt.Time,
			},
			Relationship:     relationship,
			CanSendRequest:   relationship == "none",
			CanOpenProfile:   relationship == "friend" || relationship == "self",
			CanAcceptRequest: relationship == "incoming",
		})
	}
	return result, nil
}
