package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"smctf/internal/config"
	"smctf/internal/logging"

	"github.com/gin-gonic/gin"
)

func TestRequestLoggerIncludesUserIDAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()

	logger, err := logging.New(config.LoggingConfig{
		Dir:          dir,
		FilePrefix:   "req",
		MaxBodyBytes: 1024,
	}, logging.Options{Service: "smctf", Env: "test"})
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
	httpFields := extractGroup(t, line, "http")
	if httpFields["method"] != "POST" {
		t.Fatalf("expected method in log: %v", httpFields)
	}

	if httpFields["user_id"] != float64(123) {
		t.Fatalf("expected user_id in log: %v", httpFields)
	}

	if body, ok := httpFields["body"].(string); !ok || !strings.Contains(body, "foo") {
		t.Fatalf("expected body in log: %v", httpFields)
	}
}

func TestRequestLoggerSkipsBodyForGET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()

	logger, err := logging.New(config.LoggingConfig{
		Dir:          dir,
		FilePrefix:   "req",
		MaxBodyBytes: 1024,
	}, logging.Options{Service: "smctf", Env: "test"})
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
	httpFields := extractGroup(t, line, "http")
	if _, ok := httpFields["body"]; ok {
		t.Fatalf("expected no body in log: %v", httpFields)
	}
}

func TestRequestLoggerSkipsBodyForSensitivePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()

	logger, err := logging.New(config.LoggingConfig{
		Dir:          dir,
		FilePrefix:   "req",
		MaxBodyBytes: 1024,
	}, logging.Options{Service: "smctf", Env: "test"})
	if err != nil {
		t.Fatalf("logger init: %v", err)
	}

	defer func() {
		_ = logger.Close()
	}()

	r := gin.New()
	r.Use(RequestLogger(config.LoggingConfig{MaxBodyBytes: 1024}, logger))
	r.POST("/api/auth/login", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.POST("/api/challenges/:id/submit", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.POST("/api/admin/challenges", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.PUT("/api/admin/challenges/:id", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}

	line := readLogLine(t, dir, "req")
	httpFields := extractGroup(t, line, "http")
	if _, ok := httpFields["body"]; ok {
		t.Fatalf("expected no body in log: %v", httpFields)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/challenges/123/submit", strings.NewReader(`{"flag":"FLAG{1}"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}

	line = readLogLine(t, dir, "req")
	httpFields = extractGroup(t, line, "http")
	if _, ok := httpFields["body"]; ok {
		t.Fatalf("expected no body in log: %v", httpFields)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/admin/challenges", strings.NewReader(`{"title":"Secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}

	line = readLogLine(t, dir, "req")
	httpFields = extractGroup(t, line, "http")
	if _, ok := httpFields["body"]; ok {
		t.Fatalf("expected no body in log: %v", httpFields)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/admin/challenges/123", strings.NewReader(`{"title":"Secret 2"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}

	line = readLogLine(t, dir, "req")
	httpFields = extractGroup(t, line, "http")
	if _, ok := httpFields["body"]; ok {
		t.Fatalf("expected no body in log: %v", httpFields)
	}
}

func readLogLine(t *testing.T, dir, prefix string) map[string]any {
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

	var payload map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &payload); err != nil {
		t.Fatalf("invalid json log: %v", err)
	}

	return payload
}

func extractGroup(t *testing.T, payload map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := payload[key]
	if !ok {
		t.Fatalf("missing group %s in log: %v", key, payload)
	}

	group, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("invalid group %s in log: %T", key, value)
	}

	return group
}
