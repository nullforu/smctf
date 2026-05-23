package middleware

import (
	"context"
	"net/http"

	"smctf/internal/auth"
	"smctf/internal/config"
	"smctf/internal/models"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserIDKey = "userID"
	ctxRoleKey   = "role"

	errMissingAuth  = "missing access_token cookie"
	errInvalidToken = "invalid token"
	errForbidden    = "forbidden"
)

func Auth(cfg config.JWTConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token, err := ctx.Cookie("access_token")
		if err != nil || token == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errMissingAuth})
			return
		}

		claims, err := auth.ParseToken(cfg, token)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errInvalidToken})
			return
		}

		if claims.Type != auth.TokenTypeAccess {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errInvalidToken})
			return
		}

		ctx.Set(ctxUserIDKey, claims.UserID)
		ctx.Set(ctxRoleKey, claims.Role)
		ctx.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if Role(ctx) != role {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": errForbidden})
			return
		}

		ctx.Next()
	}
}

type UserLookup interface {
	GetByID(ctx context.Context, id int64) (*models.User, error)
}

func RequireActiveUser(users UserLookup) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID := UserID(ctx)
		if userID == 0 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errInvalidToken})
			return
		}

		user, err := users.GetByID(ctx.Request.Context(), userID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errInvalidToken})
			return
		}

		if user.Role == models.BlockedRole {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": service.ErrUserBlocked.Error()})
			return
		}

		ctx.Next()
	}
}

func UserID(ctx *gin.Context) int64 {
	if v, ok := ctx.Get(ctxUserIDKey); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}

	return 0
}

func Role(ctx *gin.Context) string {
	if v, ok := ctx.Get(ctxRoleKey); ok {
		if role, ok := v.(string); ok {
			return role
		}
	}

	return ""
}
