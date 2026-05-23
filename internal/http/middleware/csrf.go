package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
)

func CSRF() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !requiresCSRF(ctx.Request.Method) {
			ctx.Next()
			return
		}

		if isCSRFIgnoredPath(ctx.Request.URL.Path) {
			ctx.Next()
			return
		}

		_, accessErr := ctx.Cookie("access_token")
		_, refreshErr := ctx.Cookie("refresh_token")
		if accessErr != nil && refreshErr != nil {
			ctx.Next()
			return
		}

		cookieToken, err := ctx.Cookie(csrfCookieName)
		if err != nil || cookieToken == "" {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		headerToken := strings.TrimSpace(ctx.GetHeader(csrfHeaderName))
		if headerToken == "" || headerToken != cookieToken {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		ctx.Next()
	}
}

func requiresCSRF(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func isCSRFIgnoredPath(path string) bool {
	return path == "/api/auth/login" ||
		path == "/api/auth/register" ||
		path == "/api/auth/refresh" ||
		path == "/api/auth/logout"
}
