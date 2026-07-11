package http

import (
	"errors"
	"net/http"

	"movies/backend/internal/domain"
	usecasefriends "movies/backend/internal/usecase/friends"

	"github.com/gin-gonic/gin"
)

type createFriendRequestBody struct {
	UserID   int64  `json:"user_id"`
	UserUUID string `json:"user_uuid"`
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

		var (
			friendship domain.Friendship
			err        error
		)
		if req.UserUUID != "" {
			friendship, err = friends.CreateRequestByUUID(c.Request.Context(), user.ID, req.UserUUID)
		} else {
			friendship, err = friends.CreateRequest(c.Request.Context(), user.ID, req.UserID)
		}
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

		requesterUUID := c.Param("user_uuid")
		if requesterUUID == "" {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid user uuid"))
			return
		}

		friendship, err := friends.AcceptRequestByUUID(c.Request.Context(), user.ID, requesterUUID)
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

		otherUserUUID := c.Param("user_uuid")
		if otherUserUUID == "" {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid user uuid"))
			return
		}

		if err := friends.DeleteRequestByUUID(c.Request.Context(), user.ID, otherUserUUID); err != nil {
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

		friendUUID := c.Param("user_uuid")
		if friendUUID == "" {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid user uuid"))
			return
		}

		if err := friends.DeleteFriendByUUID(c.Request.Context(), user.ID, friendUUID); err != nil {
			writeFriendError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
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
