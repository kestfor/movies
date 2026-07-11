package http

import (
	"errors"
	"net/http"
	"strconv"

	"movies/backend/internal/clients/tmdb"
	"movies/backend/internal/domain"
	usecaseratings "movies/backend/internal/usecase/ratings"

	"github.com/gin-gonic/gin"
)

type putRatingRequest struct {
	Scores map[string]int `json:"scores"`
}

func listCriteria(criteria CriteriaLister) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := criteria.ListActive(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			c.JSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
			return
		}
		c.JSON(http.StatusOK, gin.H{"criteria": items})
	}
}

func putRating(ratings RatingManager) gin.HandlerFunc {
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

		var req putRatingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
			return
		}

		rating, err := ratings.Upsert(c.Request.Context(), user.ID, mediaType, tmdbID, req.Scores)
		if err != nil {
			writeRatingError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"rating": rating})
	}
}

func deleteRating(ratings RatingManager) gin.HandlerFunc {
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

		if err := ratings.Delete(c.Request.Context(), user.ID, mediaType, tmdbID); err != nil {
			writeRatingError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func listUserRatings(ratings RatingManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || targetID <= 0 {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid user id"))
			return
		}

		page, err := ratings.ListUserRatings(c.Request.Context(), user.ID, targetID)
		if err != nil {
			writeRatingError(c, err)
			return
		}

		c.JSON(http.StatusOK, page)
	}
}

func titleRefFromPath(c *gin.Context) (mediaType domain.MediaType, tmdbID int64, ok bool) {
	mediaType, ok = parseMediaType(c.Param("type"))
	if !ok {
		return "", 0, false
	}
	tmdbID, err := strconv.ParseInt(c.Param("tmdb_id"), 10, 64)
	if err != nil || tmdbID <= 0 {
		return "", 0, false
	}
	return mediaType, tmdbID, true
}

func writeRatingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecaseratings.ErrValidation):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
	case errors.Is(err, usecaseratings.ErrUpstream), errors.Is(err, tmdb.ErrUpstream), errors.Is(err, tmdb.ErrMissingToken):
		_ = c.Error(err)
		c.JSON(http.StatusBadGateway, errorResponse("upstream_error", "upstream unavailable"))
	default:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
	}
}
