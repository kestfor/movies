package http

import (
	"errors"
	"net/http"
	"strconv"

	usecasefriends "movies/backend/internal/usecase/friends"

	"github.com/gin-gonic/gin"
)

type createFriendRequestBody struct {
	UserID int64 `json:"user_id"`
}

func listFriends(friends FriendManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		items, err := friends.ListFriends(c.Request.Context(), user.ID)
		if err != nil {
			writeFriendError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"friends": items})
	}
}

func listFriendRequests(friends FriendManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		items, err := friends.ListIncomingRequests(c.Request.Context(), user.ID)
		if err != nil {
			writeFriendError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"requests": items})
	}
}

func postFriendRequest(friends FriendManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		var req createFriendRequestBody
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
			return
		}

		friendship, err := friends.CreateRequest(c.Request.Context(), user.ID, req.UserID)
		if err != nil {
			writeFriendError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"friendship": friendship})
	}
}

func acceptFriendRequest(friends FriendManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		requesterID, ok := userIDFromPath(c)
		if !ok {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid user id"))
			return
		}

		friendship, err := friends.AcceptRequest(c.Request.Context(), user.ID, requesterID)
		if err != nil {
			writeFriendError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"friendship": friendship})
	}
}

func deleteFriendRequest(friends FriendManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		otherUserID, ok := userIDFromPath(c)
		if !ok {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid user id"))
			return
		}

		if err := friends.DeleteRequest(c.Request.Context(), user.ID, otherUserID); err != nil {
			writeFriendError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func deleteFriend(friends FriendManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		friendID, ok := userIDFromPath(c)
		if !ok {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid user id"))
			return
		}

		if err := friends.DeleteFriend(c.Request.Context(), user.ID, friendID); err != nil {
			writeFriendError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func userIDFromPath(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	return id, err == nil && id > 0
}

func writeFriendError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecasefriends.ErrValidation):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
	case errors.Is(err, usecasefriends.ErrConflict):
		c.JSON(http.StatusConflict, errorResponse("conflict", "friendship conflict"))
	case errors.Is(err, usecasefriends.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse("not_found", "friendship not found"))
	case errors.Is(err, usecasefriends.ErrForbidden):
		c.JSON(http.StatusForbidden, errorResponse("forbidden", "forbidden"))
	default:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
	}
}
