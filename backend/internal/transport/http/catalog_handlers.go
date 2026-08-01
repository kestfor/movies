package http

import (
	"errors"
	"net/http"
	"strconv"

	"movies/backend/internal/clients/tmdb"
	usecasecatalog "movies/backend/internal/usecase/catalog"

	"github.com/gin-gonic/gin"
)

func listDiscover(catalog CatalogManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}
		limit, ok := catalogLimit(c)
		if !ok {
			return
		}
		page, err := catalog.Discover(c.Request.Context(), user.ID, c.Query("type"), c.Query("cursor"), limit)
		if err != nil {
			writeCatalogError(c, err)
			return
		}
		c.JSON(http.StatusOK, page)
	}
}

func listRecommendations(catalog CatalogManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}
		limit, ok := catalogLimit(c)
		if !ok {
			return
		}
		page, err := catalog.Recommendations(c.Request.Context(), user.ID, c.Query("cursor"), limit)
		if err != nil {
			writeCatalogError(c, err)
			return
		}
		c.JSON(http.StatusOK, page)
	}
}

func catalogLimit(c *gin.Context) (int, bool) {
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid limit"))
			return 0, false
		}
		return limit, true
	}
	return 0, true
}

func writeCatalogError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecasecatalog.ErrValidation):
		c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
	case errors.Is(err, usecasecatalog.ErrUpstream), errors.Is(err, tmdb.ErrUpstream), errors.Is(err, tmdb.ErrMissingToken):
		_ = c.Error(err)
		c.JSON(http.StatusBadGateway, errorResponse("upstream_error", "upstream unavailable"))
	default:
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, errorResponse("internal", "internal error"))
	}
}
