package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"smctf/internal/config"
	"smctf/internal/logging"

	"github.com/gin-gonic/gin"
)

var bodyLogMethods = map[string]struct{}{
	http.MethodPost:  {},
	http.MethodPut:   {},
	http.MethodPatch: {},
}

var bodyLogSkipPaths = map[string]struct{}{
	"/api/auth/login":    {},
	"/api/auth/register": {},
	"/api/auth/refresh":  {},
	"/api/auth/logout":   {},
}

func RequestLogger(cfg config.LoggingConfig, logger *logging.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		log := logging.ComponentLogger(logger, "http")
		start := time.Now().UTC()

		_, bodyStr := readRequestBody(ctx, cfg.MaxBodyBytes)

		ctx.Next()

		status := ctx.Writer.Status()
		latency := time.Since(start)
		clientIP := ctx.ClientIP()
		method := ctx.Request.Method
		path := ctx.Request.URL.Path
		rawQuery := ctx.Request.URL.RawQuery
		userAgent := ctx.Request.UserAgent()
		contentType := ctx.GetHeader("Content-Type")
		contentLength := ctx.Request.ContentLength

		attrs := make([]slog.Attr, 0, 12)
		attrs = append(attrs,
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("ip", clientIP),
		)

		if rawQuery != "" {
			attrs = append(attrs, slog.String("query", rawQuery))
		}

		if userAgent != "" {
			attrs = append(attrs, slog.String("user_agent", userAgent))
		}

		if contentType != "" {
			attrs = append(attrs, slog.String("content_type", contentType))
		}

		if contentLength >= 0 {
			attrs = append(attrs, slog.Int64("content_length", contentLength))
		}

		if userID := UserID(ctx); userID > 0 {
			attrs = append(attrs, slog.Int64("user_id", userID))
		}

		if bodyStr != "" {
			attrs = append(attrs, slog.String("body", bodyStr))
		}

		if log != nil {
			anyAttrs := make([]any, 0, len(attrs))
			for _, attr := range attrs {
				anyAttrs = append(anyAttrs, attr)
			}

			log.Info("http", slog.Group("http", anyAttrs...))
		}
	}
}

func readRequestBody(ctx *gin.Context, maxBodyBytes int) ([]byte, string) {
	if ctx.Request == nil || ctx.Request.Body == nil {
		return nil, ""
	}

	if _, ok := bodyLogMethods[ctx.Request.Method]; !ok {
		return nil, ""
	}

	if shouldSkipBodyLog(ctx.Request.URL.Path) {
		return nil, ""
	}

	if maxBodyBytes <= 0 {
		return nil, ""
	}

	limited := io.LimitReader(ctx.Request.Body, int64(maxBodyBytes))
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return nil, ""
	}

	ctx.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	bodyStr := string(bodyBytes)
	if len(bodyStr) == maxBodyBytes {
		bodyStr = bodyStr + "...(truncated)"
	}

	return bodyBytes, bodyStr
}

func shouldSkipBodyLog(path string) bool {
	if _, ok := bodyLogSkipPaths[path]; ok {
		return true
	}

	return isChallengeSubmitPath(path) || isAdminChallengePath(path)
}

func isChallengeSubmitPath(path string) bool {
	const (
		prefix = "/api/challenges/"
		suffix = "/submit"
	)

	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return false
	}

	rest := strings.TrimPrefix(path, prefix)   // "{id}/submit"
	idPart := strings.TrimSuffix(rest, suffix) // "{id}"
	if idPart == "" || strings.Contains(idPart, "/") {
		return false
	}

	return true
}

func isAdminChallengePath(path string) bool {
	const prefix = "/api/admin/challenges"

	if path == prefix {
		return true
	}

	if !strings.HasPrefix(path, prefix+"/") {
		return false
	}

	idPart := strings.TrimPrefix(path, prefix+"/")
	if idPart == "" || strings.Contains(idPart, "/") {
		return false
	}

	return true
}
