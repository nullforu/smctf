package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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

		var b strings.Builder
		b.Grow(256 + len(bodyStr))
		fmt.Fprintf(&b, "ts=%s level=INFO msg=\"http request\" method=%s path=%s status=%d latency=%s ip=%s",
			start.UTC().Format(time.RFC3339Nano),
			method,
			path,
			status,
			latency,
			clientIP,
		)

		if rawQuery != "" {
			fmt.Fprintf(&b, " query=%s", strconv.Quote(rawQuery))
		}

		if userAgent != "" {
			fmt.Fprintf(&b, " ua=%s", strconv.Quote(userAgent))
		}

		if contentType != "" {
			fmt.Fprintf(&b, " content_type=%s", strconv.Quote(contentType))
		}

		if contentLength >= 0 {
			fmt.Fprintf(&b, " content_length=%d", contentLength)
		}

		if userID := UserID(ctx); userID > 0 {
			fmt.Fprintf(&b, " user_id=%d", userID)
		}

		if bodyStr != "" {
			fmt.Fprintf(&b, " body=%s", strconv.Quote(bodyStr))
		}

		if logger != nil {
			_, _ = logger.Write([]byte(b.String() + "\n"))
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

	return isChallengeSubmitPath(path)
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
