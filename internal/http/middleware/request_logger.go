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

func RequestLogger(cfg config.LoggingConfig, logger *logging.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

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

	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, ""
	}

	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	bodyStr := string(bodyBytes)
	if maxBodyBytes > 0 && len(bodyStr) > maxBodyBytes {
		bodyStr = bodyStr[:maxBodyBytes] + "...(truncated)"
	}

	return bodyBytes, bodyStr
}
