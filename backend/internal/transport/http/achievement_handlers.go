package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func listUserAchievements(manager AchievementManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}
		page, err := manager.GetByUUID(c.Request.Context(), user.ID, c.Param("id"))
		if err != nil {
			achievementError(c, err)
			return
		}
		c.JSON(http.StatusOK, page)
	}
}

func achievementsLeaderboard(manager AchievementManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}
		leaderboard, err := manager.Leaderboard(c.Request.Context(), user.ID)
		if err != nil {
			achievementError(c, err)
			return
		}
		c.JSON(http.StatusOK, leaderboard)
	}
}

func unseenAchievements(manager AchievementManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}
		result, err := manager.Unseen(c.Request.Context(), user.ID)
		if err != nil {
			achievementError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

type markAchievementsSeenRequest struct {
	AwardIDs []string `json:"award_ids" binding:"required"`
}

func markAchievementsSeen(manager AchievementManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := currentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, errorResponse("unauthorized", "authentication required"))
			return
		}
		var request markAchievementsSeenRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusUnprocessableEntity, errorResponse("validation_failed", "invalid request"))
			return
		}
		if err := manager.MarkSeen(c.Request.Context(), user.ID, request.AwardIDs); err != nil {
			achievementError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
