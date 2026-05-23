package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowAll bool, allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}

	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if allowAll {
			if origin != "" {
				ctx.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				ctx.Writer.Header().Set("Vary", "Origin")
			}
		} else if origin != "" {
			if _, ok := allowed[origin]; ok {
				ctx.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				ctx.Writer.Header().Set("Vary", "Origin")
			}
		}

		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Cache-Control, Pragma, X-CSRF-Token")
		ctx.Writer.Header().Set("Access-Control-Expose-Headers", "X-CSRF-Token")
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}
