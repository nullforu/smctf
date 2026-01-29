package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/logging"

	"github.com/gin-gonic/gin"
)

func TestRequestLoggerIncludesUserIDAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()

	logger, err := logging.New(config.LoggingConfig{
		Dir:              dir,
		FilePrefix:       "req",
		MaxBodyBytes:     1024,
		WebhookQueueSize: 10,
		WebhookTimeout:   time.Second,
		WebhookBatchSize: 1,
		WebhookBatchWait: time.Millisecond,
		WebhookMaxChars:  1000,
	})
	if err != nil {
		t.Fatalf("logger init: %v", err)
	}

	defer func() {
		_ = logger.Close()
	}()

	r := gin.New()
	r.Use(RequestLogger(config.LoggingConfig{MaxBodyBytes: 1024}, logger))
	r.POST("/test", func(ctx *gin.Context) {
		ctx.Set("userID", int64(123))
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"foo":"bar"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}

	line := readLogLine(t, dir, "req")
	if !strings.Contains(line, "method=POST") || !strings.Contains(line, "user_id=123") {
		t.Fatalf("expected method/user_id in log: %s", line)
	}

	if !strings.Contains(line, "body=") || !strings.Contains(line, "foo") {
		t.Fatalf("expected body in log: %s", line)
	}
}

func TestRequestLoggerSkipsBodyForGET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()

	logger, err := logging.New(config.LoggingConfig{
		Dir:              dir,
		FilePrefix:       "req",
		MaxBodyBytes:     1024,
		WebhookQueueSize: 10,
		WebhookTimeout:   time.Second,
		WebhookBatchSize: 1,
		WebhookBatchWait: time.Millisecond,
		WebhookMaxChars:  1000,
	})
	if err != nil {
		t.Fatalf("logger init: %v", err)
	}

	defer func() {
		_ = logger.Close()
	}()

	r := gin.New()
	r.Use(RequestLogger(config.LoggingConfig{MaxBodyBytes: 1024}, logger))
	r.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", strings.NewReader(`{"foo":"bar"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}

	line := readLogLine(t, dir, "req")
	if strings.Contains(line, "body=") {
		t.Fatalf("expected no body in log: %s", line)
	}
}

func readLogLine(t *testing.T, dir, prefix string) string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, prefix+"-*.log"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("log file not found: %v", err)
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatalf("no log lines found")
	}

	return lines[len(lines)-1]
}
