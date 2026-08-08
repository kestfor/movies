package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
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

func (r *RatingRepository) GetUserByUUID(ctx context.Context, rawUUID string) (domain.User, bool, error) {
	uuid, ok := uuidFromString(rawUUID)
	if !ok {
		return domain.User{}, false, nil
	}
	user, err := r.queries.GetUserByUUID(ctx, uuid)
	if err == nil {
		return toDomainUserFields(user.ID, user.Uuid, user.TgID, user.Username, user.FirstName, user.PhotoUrl, user.CreatedAt), true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, false, nil
	}
	return domain.User{}, false, err
}

func (r *RatingRepository) GetUserRelationship(ctx context.Context, viewerID, userID int64) (string, error) {
	return r.queries.GetUserRelationship(ctx, gen.GetUserRelationshipParams{
		Column1: viewerID,
		Column2: userID,
	})
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
	if _, err := q.LockTitleForUpdate(ctx, titleID); err != nil {
		return domain.Rating{}, err
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

	if err := q.DeleteWatchlistItem(ctx, gen.DeleteWatchlistItemParams{UserID: params.UserID, TitleID: titleID}); err != nil {
		return domain.Rating{}, err
	}

	if rating.Inserted {
		eventID, err := q.CreateRatingActivityEvent(ctx, gen.CreateRatingActivityEventParams{
			ActorID:  params.UserID,
			TitleID:  toNullInt8(titleID),
			RatingID: toNullInt8(rating.ID),
		})
		if err != nil {
			return domain.Rating{}, err
		}
		if err := q.DeliverActivityEventToFriends(ctx, gen.DeliverActivityEventToFriendsParams{
			RequesterID: params.UserID,
			EventID:     eventID,
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

func (r *RatingRepository) ListUserRatings(ctx context.Context, userID int64, query usecaseratings.ListQuery) ([]domain.ProfileRating, error) {
	orderExpr, cursorExpr, cursorValue := profileOrder(query)
	args := []any{userID}
	whereCursor := ""
	if query.Cursor.ID > 0 {
		args = append(args, cursorValue, query.Cursor.ID)
		whereCursor = " AND " + cursorExpr
	}
	args = append(args, query.Limit)
	limitArg := len(args)
	rawSQL := fmt.Sprintf(`
WITH page_ratings AS (
    SELECT r.id
    FROM ratings r
    JOIN titles t ON t.id = r.title_id
    WHERE r.user_id = $1%s
    ORDER BY %s
    LIMIT $%d
)
SELECT
    r.id, r.user_id, r.title_id, r.avg_score, r.created_at, r.updated_at,
    t.id, t.tmdb_id, t.media_type, t.title, t.original_title, t.poster_path,
    t.release_year, t.genres, t.overview,
    c.code, rs.score
FROM page_ratings p
JOIN ratings r ON r.id = p.id
JOIN titles t ON t.id = r.title_id
JOIN rating_scores rs ON rs.rating_id = r.id
JOIN criteria c ON c.id = rs.criterion_id
ORDER BY %s, c.sort_order, c.id`, whereCursor, orderExpr, limitArg, orderExpr)

	rows, err := r.pool.Query(ctx, rawSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ordered := make([]domain.ProfileRating, 0)
	byID := make(map[int64]*domain.ProfileRating)
	for rows.Next() {
		var (
			ratingID, ratingUserID, ratingTitleID int64
			avgScore                              pgtype.Numeric
			createdAt, updatedAt                  pgtype.Timestamptz
			titleID, tmdbID                       int64
			mediaType                             gen.MediaType
			title                                 string
			originalTitle, posterPath             pgtype.Text
			releaseYear                           pgtype.Int4
			genres                                []byte
			overview                              pgtype.Text
			criterionCode                         string
			criterionScore                        int16
		)
		if err := rows.Scan(
			&ratingID, &ratingUserID, &ratingTitleID, &avgScore, &createdAt, &updatedAt,
			&titleID, &tmdbID, &mediaType, &title, &originalTitle, &posterPath,
			&releaseYear, &genres, &overview, &criterionCode, &criterionScore,
		); err != nil {
			return nil, err
		}
		item, ok := byID[ratingID]
		if !ok {
			ordered = append(ordered, domain.ProfileRating{
				ID: ratingID,
				Title: domain.Title{
					ID: titleID, TmdbID: tmdbID, MediaType: domain.MediaType(mediaType), Title: title,
					OriginalTitle: textToString(originalTitle), ReleaseYear: int(releaseYear.Int32),
					PosterPath: textToString(posterPath), Genres: unmarshalGenres(genres), Overview: textToString(overview),
				},
				AvgScore:  numericToFloat64(avgScore),
				Scores:    make(map[string]int),
				CreatedAt: createdAt.Time,
				UpdatedAt: updatedAt.Time,
			})
			item = &ordered[len(ordered)-1]
			byID[ratingID] = item
		}
		item.Scores[criterionCode] = int(criterionScore)
	}
	return ordered, rows.Err()
}

func profileOrder(query usecaseratings.ListQuery) (orderExpr, cursorExpr string, cursorValue any) {
	direction := "DESC"
	comparison := "<"
	if query.Order == usecaseratings.OrderAsc {
		direction = "ASC"
		comparison = ">"
	}
	switch query.Sort {
	case usecaseratings.SortScore:
		return "r.avg_score " + direction + ", r.id " + direction,
			fmt.Sprintf("(r.avg_score, r.id) %s ($2::numeric, $3::bigint)", comparison), query.Cursor.Score
	case usecaseratings.SortTitle:
		return "lower(t.title) " + direction + ", r.id " + direction,
			fmt.Sprintf("(lower(t.title), r.id) %s ($2::text, $3::bigint)", comparison), query.Cursor.Title
	default:
		return "r.updated_at " + direction + ", r.id " + direction,
			fmt.Sprintf("(r.updated_at, r.id) %s ($2::timestamptz, $3::bigint)", comparison), query.Cursor.Recent
	}
}

func (r *RatingRepository) GetProfileStats(ctx context.Context, userID int64) (domain.ProfileRatingStats, error) {
	row, err := r.queries.GetUserRatingStats(ctx, userID)
	if err != nil {
		return domain.ProfileRatingStats{}, err
	}
	return domain.ProfileRatingStats{
		Count:    int(row.Count),
		AvgScore: math.Round(numericToFloat64(row.AvgScore)*10) / 10,
	}, nil
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
	AuthorUUID      pgtype.UUID
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
					UUID:      uuidToString(data.AuthorUUID),
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
		AuthorUUID:      row.AuthorUuid,
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
		AuthorUUID:      row.AuthorUuid,
		AuthorTgID:      row.AuthorTgID,
		AuthorUsername:  row.AuthorUsername,
		AuthorFirstName: row.AuthorFirstName,
		AuthorPhotoURL:  row.AuthorPhotoUrl,
		AuthorCreatedAt: row.AuthorCreatedAt,
		CriterionCode:   row.CriterionCode,
		CriterionScore:  row.CriterionScore,
	}
}
