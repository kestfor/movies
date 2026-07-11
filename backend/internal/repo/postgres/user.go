package postgres

import (
	"context"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"
	"movies/backend/internal/usecase/auth"
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
