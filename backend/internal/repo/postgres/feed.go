package postgres

import (
	"context"
	"time"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"
	usecasefeed "movies/backend/internal/usecase/feed"

	"github.com/jackc/pgx/v5/pgtype"
)

type FeedRepository struct {
	queries *gen.Queries
}

func NewFeedRepository(queries *gen.Queries) *FeedRepository {
	return &FeedRepository{queries: queries}
}

func (r *FeedRepository) List(ctx context.Context, userID int64, cursor usecasefeed.Cursor, limit int) ([]domain.FeedItem, error) {
	rows, err := r.queries.ListFeedRatings(ctx, gen.ListFeedRatingsParams{
		RequesterID: userID,
		Column2:     toNullableTimestamp(cursor.CreatedAt),
		Column3:     cursor.ID,
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, err
	}

	ordered := make([]domain.FeedItem, 0)
	byID := make(map[int64]*domain.FeedItem)
	for _, row := range rows {
		item, ok := byID[row.ID]
		if !ok {
			ordered = append(ordered, domain.FeedItem{
				ID: row.ID,
				User: domain.User{
					ID:        row.AuthorID,
					UUID:      uuidToString(row.AuthorUuid),
					TgID:      row.AuthorTgID,
					Username:  textToString(row.AuthorUsername),
					FirstName: row.AuthorFirstName,
					PhotoURL:  textToString(row.AuthorPhotoUrl),
					CreatedAt: row.AuthorCreatedAt.Time,
				},
				Title: domain.Title{
					ID:            row.TitleID_2,
					TmdbID:        row.TmdbID,
					MediaType:     domain.MediaType(row.MediaType),
					Title:         row.Title,
					OriginalTitle: textToString(row.OriginalTitle),
					ReleaseYear:   int(row.ReleaseYear.Int32),
					PosterPath:    textToString(row.PosterPath),
					Genres:        unmarshalGenres(row.Genres),
					Overview:      textToString(row.Overview),
				},
				AvgScore:  numericToFloat64(row.AvgScore),
				Scores:    make(map[string]int),
				CreatedAt: row.CreatedAt.Time,
				UpdatedAt: row.UpdatedAt.Time,
			})
			item = &ordered[len(ordered)-1]
			byID[row.ID] = item
		}
		item.Scores[row.CriterionCode] = int(row.CriterionScore)
	}
	return ordered, nil
}

func toNullableTimestamp(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}
