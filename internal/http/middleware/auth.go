package middleware

import (
	"net/http"
	"strings"

	"smctf/internal/auth"
	"smctf/internal/config"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserIDKey = "userID"
	ctxRoleKey   = "role"
)

func Auth(cfg config.JWTConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization"})
			return
		}
		claims, err := auth.ParseToken(cfg, parts[1])
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		if claims.Type != auth.TokenTypeAccess {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
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
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
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
