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
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization"})
			return
		}
		claims, err := auth.ParseToken(cfg, parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		if claims.Type != auth.TokenTypeAccess {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ctxUserIDKey, claims.UserID)
		c.Set(ctxRoleKey, claims.Role)
		c.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if Role(c) != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

func UserID(c *gin.Context) int64 {
	if v, ok := c.Get(ctxUserIDKey); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}

func Role(c *gin.Context) string {
	if v, ok := c.Get(ctxRoleKey); ok {
		if role, ok := v.(string); ok {
			return role
		}
	}
	return ""
}
