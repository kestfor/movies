package postgres

import (
	"context"
	"fmt"

	"movies/backend/internal/domain"
	gen "movies/backend/internal/repo/postgres/gen"
	usecasenotifications "movies/backend/internal/usecase/notifications"
)

type NotificationRepository struct {
	queries *gen.Queries
}

func NewNotificationRepository(queries *gen.Queries) *NotificationRepository {
	return &NotificationRepository{queries: queries}
}

func (r *NotificationRepository) List(ctx context.Context, userID int64, cursor usecasenotifications.Cursor, limit int) ([]domain.Notification, error) {
	rows, err := r.queries.ListNotifications(ctx, gen.ListNotificationsParams{
		UserID:  userID,
		Column2: toNullableTimestamp(cursor.CreatedAt),
		Column3: cursor.ID,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}

	items := make([]domain.Notification, 0, len(rows))
	for _, row := range rows {
		item := domain.Notification{
			EventID: row.EventID,
			Kind:    domain.ActivityEventKind(row.Kind),
			Actor: domain.User{
				ID:        row.ActorID,
				UUID:      uuidToString(row.ActorUuid),
				TgID:      row.ActorTgID,
				Username:  textToString(row.ActorUsername),
				FirstName: row.ActorFirstName,
				PhotoURL:  textToString(row.ActorPhotoUrl),
				CreatedAt: row.ActorCreatedAt.Time,
			},
			Title: domain.Title{
				ID:            row.TitleID,
				TmdbID:        row.TmdbID,
				MediaType:     domain.MediaType(row.MediaType),
				Title:         row.Title,
				OriginalTitle: textToString(row.OriginalTitle),
				ReleaseYear:   int(row.ReleaseYear.Int32),
				PosterPath:    textToString(row.PosterPath),
				Genres:        unmarshalGenres(row.Genres),
				Overview:      textToString(row.Overview),
			},
			CreatedAt: row.CreatedAt.Time,
		}
		if row.ReadAt.Valid {
			readAt := row.ReadAt.Time
			item.ReadAt = &readAt
		}
		if row.RatingID.Valid {
			item.Rating = &domain.NotificationRating{
				ID:       row.RatingID.Int64,
				AvgScore: numericToFloat64(row.RatingAvgScore),
			}
		}
		if row.CommentID.Valid {
			item.Comment = &domain.NotificationComment{
				ID:   row.CommentID.Int64,
				Body: textToString(row.CommentBody),
			}
		}
		item.DeepLink = notificationDeepLink(item)
		items = append(items, item)
	}
	return items, nil
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID int64) (int64, error) {
	return r.queries.CountUnreadNotifications(ctx, userID)
}

func (r *NotificationRepository) MarkRead(ctx context.Context, userID, eventID int64) (bool, error) {
	rows, err := r.queries.MarkNotificationRead(ctx, gen.MarkNotificationReadParams{
		UserID:  userID,
		EventID: eventID,
	})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID int64) error {
	return r.queries.MarkAllNotificationsRead(ctx, userID)
}

func notificationDeepLink(item domain.Notification) string {
	link := fmt.Sprintf("/title/%s/%d", item.Title.MediaType, item.Title.TmdbID)
	if item.Comment != nil {
		link = fmt.Sprintf("%s?comment_id=%d", link, item.Comment.ID)
	}
	return link
}
