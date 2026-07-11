package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"movies/backend/internal/domain"
	"movies/backend/internal/usecase/auth"

	"github.com/gin-gonic/gin"
)

type Authenticator interface {
	Authenticate(ctx context.Context, initData string) (domain.User, error)
}

type UserGetter interface {
	GetByID(ctx context.Context, id int64) (domain.User, error)
	GetByUUID(ctx context.Context, uuid string) (domain.User, bool, error)
	SearchByUsernamePrefix(ctx context.Context, currentUserID int64, query string, limit int32) ([]domain.UserSearchResult, error)
}

type TitleSearcher interface {
	Search(ctx context.Context, query string, page int) (domain.SearchPage, error)
	Get(ctx context.Context, mediaType domain.MediaType, tmdbID int64) (domain.Title, error)
	GetCard(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) (domain.TitleCard, error)
}

type CriteriaLister interface {
	ListActive(ctx context.Context) ([]domain.Criterion, error)
}

type RatingManager interface {
	Upsert(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64, scores map[string]int) (domain.Rating, error)
	Delete(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64) error
	ListUserRatingsByUUID(ctx context.Context, viewerID int64, userUUID string) (domain.ProfileRatingsPage, error)
}

type CommentManager interface {
	List(ctx context.Context, mediaType domain.MediaType, tmdbID int64) ([]domain.Comment, error)
	Create(ctx context.Context, userID int64, mediaType domain.MediaType, tmdbID int64, parentID int64, body string) (domain.Comment, error)
	Update(ctx context.Context, userID, commentID int64, body string) (domain.Comment, error)
	Delete(ctx context.Context, userID, commentID int64) (domain.Comment, error)
}

type FriendManager interface {
	ListFriends(ctx context.Context, userID int64) ([]domain.User, error)
	ListIncomingRequests(ctx context.Context, userID int64) ([]domain.FriendRequest, error)
	CreateRequest(ctx context.Context, requesterID, addresseeID int64) (domain.Friendship, error)
	CreateRequestByUUID(ctx context.Context, requesterID int64, addresseeUUID string) (domain.Friendship, error)
	AcceptRequest(ctx context.Context, currentUserID, requesterID int64) (domain.Friendship, error)
	AcceptRequestByUUID(ctx context.Context, currentUserID int64, requesterUUID string) (domain.Friendship, error)
	DeleteRequest(ctx context.Context, currentUserID, otherUserID int64) error
	DeleteRequestByUUID(ctx context.Context, currentUserID int64, otherUserUUID string) error
	DeleteFriend(ctx context.Context, currentUserID, friendID int64) error
	DeleteFriendByUUID(ctx context.Context, currentUserID int64, friendUUID string) error
}

type FeedLister interface {
	List(ctx context.Context, userID int64, cursor string, limit int) (domain.FeedPage, error)
}

func NewRouter(authSvc Authenticator, users UserGetter, titles TitleSearcher, criteria CriteriaLister, ratings RatingManager, comments CommentManager, friends FriendManager, feed FeedLister) *gin.Engine {
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.Use(gin.Logger(), requestErrorLogger(slog.Default()), gin.Recovery())

	router.GET("/health", health)
	router.GET("/me", authMiddleware(authSvc), me(users))
	router.GET("/users/search", authMiddleware(authSvc), searchUsers(users))
	router.GET("/criteria", authMiddleware(authSvc), listCriteria(criteria))
	router.GET("/search", authMiddleware(authSvc), searchTitles(titles))
	router.GET("/titles/:type/:tmdb_id", authMiddleware(authSvc), getTitle(titles))
	router.PUT("/titles/:type/:tmdb_id/rating", authMiddleware(authSvc), putRating(ratings))
	router.DELETE("/titles/:type/:tmdb_id/rating", authMiddleware(authSvc), deleteRating(ratings))
	router.GET("/titles/:type/:tmdb_id/comments", authMiddleware(authSvc), listComments(comments))
	router.POST("/titles/:type/:tmdb_id/comments", authMiddleware(authSvc), postComment(comments))
	router.PATCH("/comments/:id", authMiddleware(authSvc), patchComment(comments))
	router.DELETE("/comments/:id", authMiddleware(authSvc), deleteComment(comments))
	router.GET("/friends", authMiddleware(authSvc), listFriends(friends))
	router.GET("/friends/requests", authMiddleware(authSvc), listFriendRequests(friends))
	router.POST("/friends/requests", authMiddleware(authSvc), postFriendRequest(friends))
	router.POST("/friends/requests/:user_uuid/accept", authMiddleware(authSvc), acceptFriendRequest(friends))
	router.DELETE("/friends/requests/:user_uuid", authMiddleware(authSvc), deleteFriendRequest(friends))
	router.DELETE("/friends/:user_uuid", authMiddleware(authSvc), deleteFriend(friends))
	router.GET("/users/:id/ratings", authMiddleware(authSvc), listUserRatings(ratings))
	router.GET("/feed", authMiddleware(authSvc), listFeed(feed))

	return router
}

func searchUsers(users UserGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		query := strings.TrimSpace(c.Query("q"))
		if strings.TrimPrefix(query, "@") == "" {
			c.JSON(http.StatusOK, gin.H{"users": []domain.UserSearchResult{}})
			return
		}

		items, err := users.SearchByUsernamePrefix(c.Request.Context(), user.ID, query, 20)
		if err != nil {
			_ = c.Error(err)
			c.JSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
			return
		}
		c.JSON(http.StatusOK, gin.H{"users": items})
	}
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func me(users UserGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		fresh, err := users.GetByID(c.Request.Context(), user.ID)
		if err != nil {
			_ = c.Error(err)
			c.JSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"uuid":       fresh.UUID,
			"username":   fresh.Username,
			"first_name": fresh.FirstName,
			"photo_url":  fresh.PhotoURL,
			"created_at": fresh.CreatedAt,
		})
	}
}

func authMiddleware(authSvc Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		initData, ok := strings.CutPrefix(header, "tma ")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("unauthorized", "missing auth header"))
			return
		}

		user, err := authSvc.Authenticate(c.Request.Context(), initData)
		if err != nil {
			if errors.Is(err, auth.ErrUnauthorized) || errors.Is(err, auth.ErrValidation) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse("unauthorized", "invalid auth data"))
				return
			}
			_ = c.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
			return
		}

		withUser(c, user)
		c.Next()
	}
}

func errorResponse(code, message string) gin.H {
	return gin.H{"error": gin.H{"code": code, "message": message}}
}

func requestErrorLogger(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(c *gin.Context) {
		c.Next()

		status := c.Writer.Status()
		if len(c.Errors) == 0 && status < http.StatusInternalServerError {
			return
		}

		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"client_ip", c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		logger.Error("http request failed", attrs...)
	}
}
