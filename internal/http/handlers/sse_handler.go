package handlers

import (
	"net/http"
	"time"

	"smctf/internal/realtime"

	"github.com/gin-gonic/gin"
)

type SSEHandler struct {
	hub *realtime.SSEHub
}

func NewSSEHandler(hub *realtime.SSEHub) *SSEHandler {
	return &SSEHandler{hub: hub}
}

func (h *SSEHandler) ScoreboardStream(ctx *gin.Context) {
	if h == nil || h.hub == nil {
		ctx.Status(http.StatusServiceUnavailable)
		return
	}

	writer := ctx.Writer
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := writer.(http.Flusher)
	if !ok {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	events, unsubscribe := h.hub.Subscribe(16)
	defer unsubscribe()

	_, _ = writer.WriteString("event: ready\ndata: {}\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Request.Context().Done():
			return
		case payload, ok := <-events:
			if !ok {
				return
			}
			_, _ = writer.WriteString("event: scoreboard\n")
			_, _ = writer.WriteString("data: ")
			_, _ = writer.WriteString(payload)
			_, _ = writer.WriteString("\n\n")
			flusher.Flush()
		case <-ticker.C:
			_, _ = writer.WriteString(": ping\n\n")
			flusher.Flush()
		}
	}
}
