package postgres

import (
	"context"
	"errors"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"
	usecasecomments "movies/backend/internal/usecase/comments"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	pool    *pgxpool.Pool
	queries *gen.Queries
}

func NewCommentRepository(pool *pgxpool.Pool, queries *gen.Queries) *CommentRepository {
	return &CommentRepository{pool: pool, queries: queries}
}

func (r *CommentRepository) GetTitleID(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (int64, bool, error) {
	titleID, err := r.queries.GetTitleIDByTMDB(ctx, gen.GetTitleIDByTMDBParams{
		TmdbID:    tmdbID,
		MediaType: toGenMediaType(mediaType),
	})
	if err == nil {
		return titleID, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return 0, false, err
}

func (r *CommentRepository) Get(ctx context.Context, id int64) (domain.Comment, bool, error) {
	comment, err := r.queries.GetCommentForValidation(ctx, id)
	if err == nil {
		return toDomainComment(comment), true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Comment{}, false, nil
	}
	return domain.Comment{}, false, err
}

func (r *CommentRepository) Create(ctx context.Context, params usecasecomments.CreateCommentParams) (domain.Comment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Comment{}, err
	}
	defer tx.Rollback(ctx)

	q := r.queries.WithTx(tx)
	titleID := params.Title.ID
	if titleID == 0 {
		titleID, err = upsertTitle(ctx, q, params.Title)
		if err != nil {
			return domain.Comment{}, err
		}
	}

	comment, err := q.InsertComment(ctx, gen.InsertCommentParams{
		TitleID:  titleID,
		UserID:   params.UserID,
		ParentID: toNullInt8(params.ParentID),
		Body:     params.Body,
	})
	if err != nil {
		return domain.Comment{}, err
	}
	result := toDomainComment(comment)
	if err := attachCommentUser(ctx, q, &result); err != nil {
		return domain.Comment{}, err
	}

	eventID, err := q.CreateCommentActivityEvent(ctx, gen.CreateCommentActivityEventParams{
		ActorID:   params.UserID,
		TitleID:   toNullInt8(titleID),
		CommentID: toNullInt8(comment.ID),
	})
	if err != nil {
		return domain.Comment{}, err
	}
	if err := q.DeliverActivityEventToFriends(ctx, gen.DeliverActivityEventToFriendsParams{
		RequesterID: params.UserID,
		EventID:     eventID,
	}); err != nil {
		return domain.Comment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Comment{}, err
	}

	return result, nil
}

func (r *CommentRepository) ListByTitle(ctx context.Context, titleID int64) ([]domain.Comment, error) {
	rows, err := r.queries.ListCommentsByTitle(ctx, titleID)
	if err != nil {
		return nil, err
	}

	comments := make([]domain.Comment, 0, len(rows))
	for _, row := range rows {
		comments = append(comments, domain.Comment{
			ID:        row.ID,
			TitleID:   row.TitleID,
			ParentID:  int8ToInt64(row.ParentID),
			Body:      row.Body,
			IsDeleted: row.IsDeleted,
			CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
			User: domain.User{
				ID:        row.AuthorID,
				UUID:      uuidToString(row.AuthorUuid),
				TgID:      row.AuthorTgID,
				Username:  textToString(row.AuthorUsername),
				FirstName: row.AuthorFirstName,
				PhotoURL:  textToString(row.AuthorPhotoUrl),
				CreatedAt: row.AuthorCreatedAt.Time,
			},
		})
	}
	return comments, nil
}

func (r *CommentRepository) UpdateBody(ctx context.Context, id, userID int64, body string) (domain.Comment, error) {
	comment, err := r.queries.UpdateCommentBody(ctx, gen.UpdateCommentBodyParams{
		ID:     id,
		Body:   body,
		UserID: userID,
	})
	if err != nil {
		return domain.Comment{}, err
	}
	result := toDomainComment(comment)
	if err := attachCommentUser(ctx, r.queries, &result); err != nil {
		return domain.Comment{}, err
	}
	return result, nil
}

func (r *CommentRepository) SoftDelete(ctx context.Context, id, userID int64) (domain.Comment, error) {
	comment, err := r.queries.SoftDeleteComment(ctx, gen.SoftDeleteCommentParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return domain.Comment{}, err
	}
	result := toDomainComment(comment)
	if err := attachCommentUser(ctx, r.queries, &result); err != nil {
		return domain.Comment{}, err
	}
	return result, nil
}

func toDomainComment(comment gen.Comment) domain.Comment {
	return domain.Comment{
		ID:        comment.ID,
		TitleID:   comment.TitleID,
		User:      domain.User{ID: comment.UserID},
		ParentID:  int8ToInt64(comment.ParentID),
		Body:      comment.Body,
		IsDeleted: comment.IsDeleted,
		CreatedAt: comment.CreatedAt.Time,
		UpdatedAt: comment.UpdatedAt.Time,
	}
}

func attachCommentUser(ctx context.Context, q *gen.Queries, comment *domain.Comment) error {
	user, err := q.GetUserByID(ctx, comment.User.ID)
	if err != nil {
		return err
	}
	comment.User = toDomainUserFields(user.ID, user.Uuid, user.TgID, user.Username, user.FirstName, user.PhotoUrl, user.CreatedAt)
	return nil
}
