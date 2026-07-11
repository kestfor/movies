package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"movies/backend/internal/clients/tmdb"
	"movies/backend/internal/domain"
	usecasetitles "movies/backend/internal/usecase/titles"

	"github.com/gin-gonic/gin"
)

func searchTitles(titles TitleSearcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		page := 1
		if raw := c.Query("page"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid page"))
				return
			}
			page = parsed
		}

		result, err := titles.Search(c.Request.Context(), query, page)
		if err != nil {
			writeTitleError(c, err)
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func getTitle(titles TitleSearcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		mediaType, ok := parseMediaType(c.Param("type"))
		if !ok {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid media type"))
			return
		}

		tmdbID, err := strconv.ParseInt(c.Param("tmdb_id"), 10, 64)
		if err != nil || tmdbID <= 0 {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid tmdb id"))
			return
		}

		card, err := titles.GetCard(c.Request.Context(), user.ID, mediaType, tmdbID)
		if err != nil {
			writeTitleError(c, err)
			return
		}

		c.JSON(http.StatusOK, card)
	}
}

func parseMediaType(raw string) (domain.MediaType, bool) {
	switch strings.ToLower(raw) {
	case string(domain.MediaTypeMovie):
		return domain.MediaTypeMovie, true
	case string(domain.MediaTypeTV):
		return domain.MediaTypeTV, true
	default:
		return "", false
	}
}

func writeTitleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecasetitles.ErrValidation):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
	case errors.Is(err, usecasetitles.ErrNotFound), errors.Is(err, tmdb.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse("not_found", "title not found"))
	case errors.Is(err, usecasetitles.ErrUpstream), errors.Is(err, tmdb.ErrUpstream), errors.Is(err, tmdb.ErrMissingToken):
		_ = c.Error(err)
		c.JSON(http.StatusBadGateway, errorResponse("upstream_error", "upstream unavailable"))
	default:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
	}
}
