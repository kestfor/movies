package postgres

import (
	"context"
	"errors"
	"time"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"
	usecasewatchlist "movies/backend/internal/usecase/watchlist"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WatchlistRepository struct {
	pool    *pgxpool.Pool
	queries *gen.Queries
}

func NewWatchlistRepository(pool *pgxpool.Pool, queries *gen.Queries) *WatchlistRepository {
	return &WatchlistRepository{pool: pool, queries: queries}
}

func (r *WatchlistRepository) GetUserByUUID(ctx context.Context, rawUUID string) (domain.User, bool, error) {
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

func (r *WatchlistRepository) UserCanSeeWatchlist(ctx context.Context, viewerID, userID int64) (bool, error) {
	return r.queries.UserCanSeeRatings(ctx, gen.UserCanSeeRatingsParams{Column1: viewerID, Column2: userID})
}

func (r *WatchlistRepository) GetTitleID(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (int64, bool, error) {
	id, err := r.queries.GetTitleIDByTMDB(ctx, gen.GetTitleIDByTMDBParams{TmdbID: tmdbID, MediaType: toGenMediaType(mediaType)})
	if err == nil {
		return id, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return 0, false, err
}

func (r *WatchlistRepository) Add(ctx context.Context, userID int64, title domain.Title) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	q := r.queries.WithTx(tx)
	titleID := title.ID
	if titleID == 0 {
		titleID, err = upsertTitle(ctx, q, title)
		if err != nil {
			return false, err
		}
	}
	if _, err := q.LockTitleForUpdate(ctx, titleID); err != nil {
		return false, err
	}
	rated, err := q.HasUserRatingForTitle(ctx, gen.HasUserRatingForTitleParams{UserID: userID, TitleID: titleID})
	if err != nil {
		return false, err
	}
	if rated {
		return false, nil
	}
	var createdAt time.Time
	err = tx.QueryRow(ctx, `
INSERT INTO watchlist_items (user_id, title_id)
VALUES ($1, $2)
ON CONFLICT (user_id, title_id) DO NOTHING
RETURNING created_at`, userID, titleID).Scan(&createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if err := insertWatchlistAchievementFact(ctx, tx, userID, titleID, createdAt); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (r *WatchlistRepository) Remove(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) error {
	titleID, ok, err := r.GetTitleID(ctx, mediaType, tmdbID)
	if err != nil || !ok {
		return err
	}
	return r.queries.DeleteWatchlistItem(ctx, gen.DeleteWatchlistItemParams{UserID: userID, TitleID: titleID})
}

func (r *WatchlistRepository) IsInWatchlist(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) (bool, error) {
	titleID, ok, err := r.GetTitleID(ctx, mediaType, tmdbID)
	if err != nil || !ok {
		return false, err
	}
	return r.queries.IsInWatchlist(ctx, gen.IsInWatchlistParams{UserID: userID, TitleID: titleID})
}

func (r *WatchlistRepository) List(ctx context.Context, userID int64, cursor usecasewatchlist.Cursor, limit int) ([]domain.WatchlistItem, error) {
	rows, err := r.queries.ListWatchlistItems(ctx, gen.ListWatchlistItemsParams{
		UserID: userID, Column2: toNullableTimestamp(cursor.AddedAt), Column3: cursor.TitleID, Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	items := make([]domain.WatchlistItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.WatchlistItem{
			AddedAt: row.AddedAt.Time,
			Title: domain.Title{
				ID: row.ID, TmdbID: row.TmdbID, MediaType: domain.MediaType(row.MediaType), Title: row.Title,
				OriginalTitle: textToString(row.OriginalTitle), PosterPath: textToString(row.PosterPath),
				ReleaseYear: int(row.ReleaseYear.Int32), Genres: unmarshalGenres(row.Genres), Overview: textToString(row.Overview),
			},
		})
	}
	return items, nil
}

func (r *WatchlistRepository) Count(ctx context.Context, userID int64) (int64, error) {
	return r.queries.CountWatchlistItems(ctx, userID)
}

func (r *WatchlistRepository) ResolveAcceptedFriendIDs(ctx context.Context, userID int64, friendUUIDs []string) ([]int64, error) {
	if len(friendUUIDs) == 0 {
		return []int64{}, nil
	}
	return r.queries.ListAcceptedFriendIDsByUUIDs(ctx, gen.ListAcceptedFriendIDsByUUIDsParams{
		UserID: userID, FriendUuids: friendUUIDs,
	})
}

func (r *WatchlistRepository) ListMatches(ctx context.Context, userID int64, friendIDs []int64, cursor usecasewatchlist.MatchesCursor, limit int) ([]domain.WatchlistMatchItem, error) {
	rows, err := r.queries.ListWatchlistMatches(ctx, gen.ListWatchlistMatchesParams{
		UserID: userID, FriendIds: friendIDs,
		CursorMatchesCount: toNullInt4(cursor.MatchesCount), CursorLatestAddedAt: toNullableTimestamp(cursor.LatestAddedAt),
		CursorTitleID: toNullInt8(cursor.TitleID), PageLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	items := make([]domain.WatchlistMatchItem, 0)
	for _, row := range rows {
		if len(items) == 0 || items[len(items)-1].Title.ID != row.TitleID {
			items = append(items, domain.WatchlistMatchItem{
				Title: domain.Title{
					ID: row.TitleID, TmdbID: row.TmdbID, MediaType: domain.MediaType(row.MediaType), Title: row.Title,
					OriginalTitle: textToString(row.OriginalTitle), PosterPath: textToString(row.PosterPath),
					ReleaseYear: int(row.ReleaseYear.Int32), Genres: unmarshalGenres(row.Genres), Overview: textToString(row.Overview),
				},
				Users: []domain.User{}, MatchesCount: int(row.MatchesCount), LatestAddedAt: row.LatestAddedAt.Time,
			})
		}
		item := &items[len(items)-1]
		item.Users = append(item.Users, toDomainUserFields(
			row.WatcherID, row.WatcherUuid, row.WatcherTgID, row.WatcherUsername,
			row.WatcherFirstName, row.WatcherPhotoUrl, row.WatcherCreatedAt,
		))
	}
	return items, nil
}

func (r *WatchlistRepository) ListRefs(ctx context.Context, userID int64) ([]domain.TitleRef, error) {
	rows, err := r.queries.ListWatchlistTitleRefs(ctx, userID)
	if err != nil {
		return nil, err
	}
	refs := make([]domain.TitleRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, domain.TitleRef{TmdbID: row.TmdbID, MediaType: domain.MediaType(row.MediaType)})
	}
	return refs, nil
}

func (r *WatchlistRepository) ListRatedRefs(ctx context.Context, userID int64) ([]domain.TitleRef, error) {
	rows, err := r.queries.ListRatedTitleRefs(ctx, userID)
	if err != nil {
		return nil, err
	}
	refs := make([]domain.TitleRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, domain.TitleRef{TmdbID: row.TmdbID, MediaType: domain.MediaType(row.MediaType)})
	}
	return refs, nil
}

func (r *WatchlistRepository) CountRatings(ctx context.Context, userID int64) (int64, error) {
	return r.queries.CountUserRatings(ctx, userID)
}

func (r *WatchlistRepository) ListRecommendationSeeds(ctx context.Context, userID int64) ([]domain.RecommendationSeed, error) {
	rows, err := r.queries.ListRecommendationSeeds(ctx, userID)
	if err != nil {
		return nil, err
	}
	seeds := make([]domain.RecommendationSeed, 0, len(rows))
	for _, row := range rows {
		seeds = append(seeds, domain.RecommendationSeed{
			RatingID: row.RatingID,
			Title:    domain.Title{TmdbID: row.TmdbID, MediaType: domain.MediaType(row.MediaType), Title: row.Title, Genres: unmarshalGenres(row.Genres)},
			AvgScore: numericToFloat64(row.AvgScore), UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return seeds, nil
}

func (r *WatchlistRepository) ListGenreRatings(ctx context.Context, userID int64) ([]domain.GenreRating, error) {
	rows, err := r.queries.ListUserRatingGenres(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]domain.GenreRating, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.GenreRating{AvgScore: numericToFloat64(row.AvgScore), Genres: unmarshalGenres(row.Genres)})
	}
	return items, nil
}
