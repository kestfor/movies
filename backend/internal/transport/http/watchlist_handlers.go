package http

import (
	"errors"
	"net/http"
	"strconv"

	"movies/backend/internal/clients/tmdb"
	usecasewatchlist "movies/backend/internal/usecase/watchlist"

	"github.com/gin-gonic/gin"
)

func putWatchlistItem(watchlist WatchlistManager) gin.HandlerFunc {
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
		if err := watchlist.Add(c.Request.Context(), user.ID, mediaType, tmdbID); err != nil {
			writeWatchlistError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"in_watchlist": true})
	}
}

func deleteWatchlistItem(watchlist WatchlistManager) gin.HandlerFunc {
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
		if err := watchlist.Remove(c.Request.Context(), user.ID, mediaType, tmdbID); err != nil {
			writeWatchlistError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func listUserWatchlist(watchlist WatchlistManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}
		limit := 0
		if raw := c.Query("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid limit"))
				return
			}
			limit = parsed
		}
		page, err := watchlist.ListByUUID(c.Request.Context(), user.ID, c.Param("id"), c.Query("cursor"), limit)
		if err != nil {
			writeWatchlistError(c, err)
			return
		}
		c.JSON(http.StatusOK, page)
	}
}

func listWatchlistMatches(watchlist WatchlistManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}
		limit := 0
		if raw := c.Query("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid limit"))
				return
			}
			limit = parsed
		}
		page, err := watchlist.ListMatches(
			c.Request.Context(), user.ID, c.QueryArray("friend_id"), c.Query("cursor"), limit,
		)
		if err != nil {
			writeWatchlistError(c, err)
			return
		}
		c.JSON(http.StatusOK, page)
	}
}

func writeWatchlistError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecasewatchlist.ErrValidation):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
	case errors.Is(err, usecasewatchlist.ErrConflict):
		c.JSON(http.StatusConflict, errorResponse("already_rated", "rated title cannot be added to watchlist"))
	case errors.Is(err, usecasewatchlist.ErrUpstream), errors.Is(err, tmdb.ErrUpstream), errors.Is(err, tmdb.ErrMissingToken):
		_ = c.Error(err)
		c.JSON(http.StatusBadGateway, errorResponse("upstream_error", "upstream unavailable"))
	default:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
	}
}
