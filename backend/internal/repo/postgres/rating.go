package postgres

import (
	"context"
	"errors"
	"sort"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"
	usecaseratings "movies/backend/internal/usecase/ratings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RatingRepository struct {
	pool    *pgxpool.Pool
	queries *gen.Queries
}

func NewRatingRepository(pool *pgxpool.Pool, queries *gen.Queries) *RatingRepository {
	return &RatingRepository{pool: pool, queries: queries}
}

func (r *RatingRepository) TitleExists(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (bool, error) {
	_, ok, err := r.GetTitleID(ctx, mediaType, tmdbID)
	return ok, err
}

func (r *RatingRepository) GetTitleID(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (int64, bool, error) {
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

func (r *RatingRepository) ListCriteriaByCodes(ctx context.Context, codes []string) (map[string]domain.Criterion, error) {
	items, err := r.queries.ListCriteriaByCodes(ctx, codes)
	if err != nil {
		return nil, err
	}

	criteria := make(map[string]domain.Criterion, len(items))
	for _, item := range items {
		criteria[item.Code] = domain.Criterion{
			ID:        item.ID,
			Code:      item.Code,
			Name:      item.Name,
			SortOrder: item.SortOrder,
		}
	}
	return criteria, nil
}

func (r *RatingRepository) Upsert(ctx context.Context, params usecaseratings.UpsertRatingParams) (domain.Rating, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Rating{}, err
	}
	defer tx.Rollback(ctx)

	q := r.queries.WithTx(tx)
	titleID := params.Title.ID
	if titleID == 0 {
		titleID, err = upsertTitle(ctx, q, params.Title)
		if err != nil {
			return domain.Rating{}, err
		}
	}

	rating, err := q.UpsertRating(ctx, gen.UpsertRatingParams{
		UserID:  params.UserID,
		TitleID: titleID,
		Column3: toNumericTenths(params.AvgTenths),
	})
	if err != nil {
		return domain.Rating{}, err
	}

	if err := q.DeleteRatingScores(ctx, rating.ID); err != nil {
		return domain.Rating{}, err
	}

	codes := make([]string, 0, len(params.Scores))
	for code := range params.Scores {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	for _, code := range codes {
		criterion := params.Criteria[code]
		if err := q.InsertRatingScore(ctx, gen.InsertRatingScoreParams{
			RatingID:    rating.ID,
			CriterionID: criterion.ID,
			Score:       int16(params.Scores[code]),
		}); err != nil {
			return domain.Rating{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Rating{}, err
	}

	return domain.Rating{
		ID:        rating.ID,
		UserID:    rating.UserID,
		TitleID:   rating.TitleID,
		AvgScore:  float64(params.AvgTenths) / 10,
		Scores:    params.Scores,
		CreatedAt: rating.CreatedAt.Time,
		UpdatedAt: rating.UpdatedAt.Time,
	}, nil
}

func (r *RatingRepository) Delete(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) error {
	titleID, err := r.queries.GetTitleIDByTMDB(ctx, gen.GetTitleIDByTMDBParams{
		TmdbID:    tmdbID,
		MediaType: toGenMediaType(mediaType),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	return r.queries.DeleteRatingForUserTitle(ctx, gen.DeleteRatingForUserTitleParams{
		UserID:  userID,
		TitleID: titleID,
	})
}

func (r *RatingRepository) GetRatingByUserTitle(ctx context.Context, userID, titleID int64) (*domain.RatingWithUser, error) {
	rows, err := r.queries.GetRatingByUserTitle(ctx, gen.GetRatingByUserTitleParams{
		UserID:  userID,
		TitleID: titleID,
	})
	if err != nil {
		return nil, err
	}
	ratings := ratingRowsToDomain(rows, getRatingByUserTitleRowData)
	if len(ratings) == 0 {
		return nil, nil
	}
	return &ratings[0], nil
}

func (r *RatingRepository) ListFriendRatingsByTitle(ctx context.Context, userID, titleID int64) ([]domain.RatingWithUser, error) {
	rows, err := r.queries.ListFriendRatingsByTitle(ctx, gen.ListFriendRatingsByTitleParams{
		RequesterID: userID,
		TitleID:     titleID,
	})
	if err != nil {
		return nil, err
	}
	return ratingRowsToDomain(rows, listFriendRatingsByTitleRowData), nil
}

func (r *RatingRepository) CountCommentsByTitle(ctx context.Context, titleID int64) (int64, error) {
	return r.queries.CountCommentsByTitle(ctx, titleID)
}

func (r *RatingRepository) UserCanSeeRatings(ctx context.Context, viewerID, userID int64) (bool, error) {
	return r.queries.UserCanSeeRatings(ctx, gen.UserCanSeeRatingsParams{
		Column1: viewerID,
		Column2: userID,
	})
}

func (r *RatingRepository) ListUserRatings(ctx context.Context, userID int64) ([]domain.ProfileRating, error) {
	rows, err := r.queries.ListUserRatings(ctx, userID)
	if err != nil {
		return nil, err
	}

	ordered := make([]domain.ProfileRating, 0)
	byID := make(map[int64]*domain.ProfileRating)
	for _, row := range rows {
		item, ok := byID[row.ID]
		if !ok {
			ordered = append(ordered, domain.ProfileRating{
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

func upsertTitle(ctx context.Context, q *gen.Queries, title domain.Title) (int64, error) {
	return q.UpsertTitleSnapshot(ctx, gen.UpsertTitleSnapshotParams{
		TmdbID:        title.TmdbID,
		MediaType:     toGenMediaType(title.MediaType),
		Title:         title.Title,
		OriginalTitle: toNullText(title.OriginalTitle),
		PosterPath:    toNullText(title.PosterPath),
		ReleaseYear:   toNullInt4(title.ReleaseYear),
		Column7:       marshalGenres(title.Genres),
		Overview:      toNullText(title.Overview),
	})
}

type ratingRowData struct {
	RatingID        int64
	AvgScore        pgtype.Numeric
	CreatedAt       pgtype.Timestamptz
	UpdatedAt       pgtype.Timestamptz
	AuthorID        int64
	AuthorTgID      int64
	AuthorUsername  pgtype.Text
	AuthorFirstName string
	AuthorPhotoURL  pgtype.Text
	AuthorCreatedAt pgtype.Timestamptz
	CriterionCode   string
	CriterionScore  int16
}

func ratingRowsToDomain[T any](rows []T, get func(T) ratingRowData) []domain.RatingWithUser {
	ordered := make([]domain.RatingWithUser, 0)
	byID := make(map[int64]*domain.RatingWithUser)
	for _, row := range rows {
		data := get(row)
		rating, ok := byID[data.RatingID]
		if !ok {
			ordered = append(ordered, domain.RatingWithUser{
				User: domain.User{
					ID:        data.AuthorID,
					TgID:      data.AuthorTgID,
					Username:  textToString(data.AuthorUsername),
					FirstName: data.AuthorFirstName,
					PhotoURL:  textToString(data.AuthorPhotoURL),
					CreatedAt: data.AuthorCreatedAt.Time,
				},
				AvgScore:  numericToFloat64(data.AvgScore),
				Scores:    make(map[string]int),
				CreatedAt: data.CreatedAt.Time,
				UpdatedAt: data.UpdatedAt.Time,
			})
			rating = &ordered[len(ordered)-1]
			byID[data.RatingID] = rating
		}
		rating.Scores[data.CriterionCode] = int(data.CriterionScore)
	}
	return ordered
}

func getRatingByUserTitleRowData(row gen.GetRatingByUserTitleRow) ratingRowData {
	return ratingRowData{
		RatingID:        row.ID,
		AvgScore:        row.AvgScore,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		AuthorID:        row.AuthorID,
		AuthorTgID:      row.AuthorTgID,
		AuthorUsername:  row.AuthorUsername,
		AuthorFirstName: row.AuthorFirstName,
		AuthorPhotoURL:  row.AuthorPhotoUrl,
		AuthorCreatedAt: row.AuthorCreatedAt,
		CriterionCode:   row.CriterionCode,
		CriterionScore:  row.CriterionScore,
	}
}

func listFriendRatingsByTitleRowData(row gen.ListFriendRatingsByTitleRow) ratingRowData {
	return ratingRowData{
		RatingID:        row.ID,
		AvgScore:        row.AvgScore,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		AuthorID:        row.AuthorID,
		AuthorTgID:      row.AuthorTgID,
		AuthorUsername:  row.AuthorUsername,
		AuthorFirstName: row.AuthorFirstName,
		AuthorPhotoURL:  row.AuthorPhotoUrl,
		AuthorCreatedAt: row.AuthorCreatedAt,
		CriterionCode:   row.CriterionCode,
		CriterionScore:  row.CriterionScore,
	}
}
