package http

import (
	"movies/backend/internal/domain"

	"github.com/gin-gonic/gin"
)

type contextKey string

const userContextKey contextKey = "auth_user"

func withUser(ctx *gin.Context, user domain.User) {
	ctx.Set(string(userContextKey), user)
}

func currentUser(ctx *gin.Context) (domain.User, bool) {
	value, ok := ctx.Get(string(userContextKey))
	if !ok {
		return domain.User{}, false
	}
	user, ok := value.(domain.User)
	return user, ok
}
