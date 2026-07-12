package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func listNotifications(manager NotificationManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		limit, _ := strconv.Atoi(c.Query("limit"))
		page, err := manager.List(c.Request.Context(), user.ID, c.Query("cursor"), limit)
		if err != nil {
			notificationError(c, err)
			return
		}

		c.JSON(http.StatusOK, page)
	}
}

func unreadNotificationsCount(manager NotificationManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		count, err := manager.CountUnread(c.Request.Context(), user.ID)
		if err != nil {
			notificationError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"count": count})
	}
}

func markNotificationRead(manager NotificationManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		eventID, err := strconv.ParseInt(c.Param("event_id"), 10, 64)
		if err != nil || eventID <= 0 {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
			return
		}

		if err := manager.MarkRead(c.Request.Context(), user.ID, eventID); err != nil {
			notificationError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func markAllNotificationsRead(manager NotificationManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}

		if err := manager.MarkAllRead(c.Request.Context(), user.ID); err != nil {
			notificationError(c, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}
