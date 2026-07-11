package http

import (
	"errors"
	"net/http"
	"strconv"

	"movies/backend/internal/clients/tmdb"
	usecasecomments "movies/backend/internal/usecase/comments"

	"github.com/gin-gonic/gin"
)

type writeCommentRequest struct {
	Body     string `json:"body"`
	ParentID int64  `json:"parent_id"`
}

func listComments(comments CommentManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		mediaType, tmdbID, ok := titleRefFromPath(c)
		if !ok {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid title"))
			return
		}

		items, err := comments.List(c.Request.Context(), mediaType, tmdbID)
		if err != nil {
			writeCommentError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"comments": items})
	}
}

func postComment(comments CommentManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		mediaType, tmdbID, ok := titleRefFromPath(c)
		if !ok {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid title"))
			return
		}

		var req writeCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
			return
		}

		comment, err := comments.Create(c.Request.Context(), user.ID, mediaType, tmdbID, req.ParentID, req.Body)
		if err != nil {
			writeCommentError(c, err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"comment": comment})
	}
}

func patchComment(comments CommentManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		commentID, ok := commentIDFromPath(c)
		if !ok {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid comment id"))
			return
		}

		var req writeCommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
			return
		}

		comment, err := comments.Update(c.Request.Context(), user.ID, commentID, req.Body)
		if err != nil {
			writeCommentError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"comment": comment})
	}
}

func deleteComment(comments CommentManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		commentID, ok := commentIDFromPath(c)
		if !ok {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid comment id"))
			return
		}

		comment, err := comments.Delete(c.Request.Context(), user.ID, commentID)
		if err != nil {
			writeCommentError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"comment": comment})
	}
}

func commentIDFromPath(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	return id, err == nil && id > 0
}

func writeCommentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecasecomments.ErrValidation):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
	case errors.Is(err, usecasecomments.ErrNotFound), errors.Is(err, tmdb.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse("not_found", "comment not found"))
	case errors.Is(err, usecasecomments.ErrForbidden):
		c.JSON(http.StatusForbidden, errorResponse("forbidden", "forbidden"))
	case errors.Is(err, usecasecomments.ErrConflict):
		c.JSON(http.StatusConflict, errorResponse("conflict", "comment already deleted"))
	case errors.Is(err, tmdb.ErrUpstream), errors.Is(err, tmdb.ErrMissingToken):
		_ = c.Error(err)
		c.JSON(http.StatusBadGateway, errorResponse("upstream_error", "upstream unavailable"))
	default:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
	}
}
