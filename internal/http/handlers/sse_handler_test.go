package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"smctf/internal/realtime"

	"github.com/gin-gonic/gin"
)

type sseSafeRecorder struct {
	header http.Header
	buf    strings.Builder
	status int
}

func newSSESafeRecorder() *sseSafeRecorder {
	return &sseSafeRecorder{header: make(http.Header), status: http.StatusOK}
}

func (r *sseSafeRecorder) Header() http.Header {
	return r.header
}

func (r *sseSafeRecorder) WriteHeader(status int) {
	r.status = status
}

func (r *sseSafeRecorder) Write(p []byte) (int, error) {
	return r.buf.Write(p)
}

func (r *sseSafeRecorder) Flush() {}

func (r *sseSafeRecorder) String() string {
	return r.buf.String()
}

func (r *sseSafeRecorder) StatusCode() int {
	return r.status
}

func TestSSEHandlerScoreboardStreamSendsReady(t *testing.T) {
	hub := realtime.NewSSEHub()
	h := NewSSEHandler(hub)

	rec := newSSESafeRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	req := httptest.NewRequest(http.MethodGet, "/api/scoreboard/stream", nil)
	reqCtx, cancel := context.WithCancel(req.Context())
	defer cancel()
	ctx.Request = req.WithContext(reqCtx)

	done := make(chan struct{})
	go func() {
		h.ScoreboardStream(ctx)
		close(done)
	}()

	waitForStringContains(t, rec.String, "event: ready", 2*time.Second)
	payload := "{\"scope\":\"all\"}"
	hub.Broadcast(payload)
	waitForStringContains(t, rec.String, "event: scoreboard", 2*time.Second)
	waitForStringContains(t, rec.String, payload, 2*time.Second)
	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for stream to close")
	}
}

func TestSSEHandlerScoreboardStreamUnavailableWithoutHub(t *testing.T) {
	h := NewSSEHandler(nil)

	ctx, _ := newJSONContext(t, http.MethodGet, "/api/scoreboard/stream", nil)
	h.ScoreboardStream(ctx)

	if ctx.Writer.Status() != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", ctx.Writer.Status())
	}
}

func waitForStringContains(t *testing.T, get func() string, needle string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(get(), needle) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timeout waiting for %q in response body", needle)
}
