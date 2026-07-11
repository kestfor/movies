package http

import (
	"errors"
	"net/http"
	"strconv"

	usecasefeed "movies/backend/internal/usecase/feed"

	"github.com/gin-gonic/gin"
)

func listFeed(feed FeedLister) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		limit := 0
		if raw := c.Query("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid limit"))
				return
			}
			limit = parsed
		}

		page, err := feed.List(c.Request.Context(), user.ID, c.Query("cursor"), limit)
		if err != nil {
			writeFeedError(c, err)
			return
		}

		c.JSON(http.StatusOK, page)
	}
}

func writeFeedError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecasefeed.ErrValidation):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
	default:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
	}
}
