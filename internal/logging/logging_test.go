package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"smctf/internal/config"
)

func TestLoggerWriteAndRotate(t *testing.T) {
	dir := t.TempDir()
	writer, err := newRotatingFileWriter(dir, "app")
	if err != nil {
		t.Fatalf("newRotatingFileWriter: %v", err)
	}

	t1 := time.Date(2026, 1, 1, 10, 15, 0, 0, time.UTC)
	if err := writer.rotate(t1); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	firstPath := filepath.Join(dir, "app-20260101-10.log")
	if _, err := os.Stat(firstPath); err != nil {
		t.Fatalf("expected first file: %v", err)
	}

	t2 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)
	if err := writer.rotate(t2); err != nil {
		t.Fatalf("rotate second: %v", err)
	}

	secondPath := filepath.Join(dir, "app-20260101-11.log")
	if _, err := os.Stat(secondPath); err != nil {
		t.Fatalf("expected second file: %v", err)
	}

	if _, err := writer.Write([]byte("hello\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "app-*.log"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("expected log files, err %v", err)
	}

	found := false
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}

		if strings.Contains(string(data), "hello") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected log content in files")
	}
}

func TestLoggerConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	logger, err := newTestLogger(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() {
		_ = logger.Close()
	}()

	const goroutines = 20
	const perG = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range make([]struct{}, goroutines) {
		go func(i int) {
			defer wg.Done()
			for range make([]struct{}, perG) {
				msg := []byte("log line " + strconv.Itoa(i) + "\n")
				if _, err := logger.Write(msg); err != nil {
					t.Errorf("write error: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	matches, err := filepath.Glob(filepath.Join(dir, "app-*.log"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("expected log files, err %v", err)
	}

	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if len(bytes.TrimSpace(content)) == 0 {
		t.Fatalf("expected content")
	}
}

func TestWebhookSender(t *testing.T) {
	var discordPayload map[string]string
	discordSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &discordPayload)
		w.WriteHeader(http.StatusOK)
	}))

	defer discordSrv.Close()

	var slackPayload map[string]string
	slackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &slackPayload)
		w.WriteHeader(http.StatusOK)
	}))

	defer slackSrv.Close()

	sender := newWebhookSender(config.LoggingConfig{
		DiscordWebhookURL: discordSrv.URL,
		SlackWebhookURL:   slackSrv.URL,
		WebhookQueueSize:  10,
		WebhookTimeout:    time.Second,
		WebhookBatchSize:  1,
		WebhookBatchWait:  time.Millisecond,
		WebhookMaxChars:   1000,
	})
	if sender == nil {
		t.Fatalf("expected sender")
	}

	if err := sender.Send(context.Background(), "hello"); err != nil {
		t.Fatalf("send: %v", err)
	}

	if discordPayload["content"] != "```\nhello\n```" {
		t.Fatalf("unexpected discord payload: %+v", discordPayload)
	}

	if slackPayload["text"] != "```\nhello\n```" {
		t.Fatalf("unexpected slack payload: %+v", slackPayload)
	}
}

func TestWebhookSenderNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	defer srv.Close()

	sender := newWebhookSender(config.LoggingConfig{
		DiscordWebhookURL: srv.URL,
		WebhookQueueSize:  10,
		WebhookTimeout:    time.Second,
		WebhookBatchSize:  1,
		WebhookBatchWait:  time.Millisecond,
		WebhookMaxChars:   1000,
	})
	if sender == nil {
		t.Fatalf("expected sender")
	}

	if err := sender.Send(context.Background(), "hello"); err == nil {
		t.Fatalf("expected error on non-2xx")
	}
}

func TestWebhookSenderQueueFull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sender := newWebhookSender(config.LoggingConfig{
		DiscordWebhookURL: srv.URL,
		WebhookQueueSize:  1,
		WebhookTimeout:    time.Second,
		WebhookBatchSize:  1,
		WebhookBatchWait:  time.Second,
		WebhookMaxChars:   1000,
	})
	if sender == nil {
		t.Fatalf("expected sender")
	}

	defer func() {
		_ = sender.Close()
	}()

	if err := sender.Enqueue(context.Background(), "one"); err != nil {
		t.Fatalf("enqueue first: %v", err)
	}

	if err := sender.Enqueue(context.Background(), "two"); err == nil {
		t.Fatalf("expected queue full error")
	}
}

func newTestLogger(dir string) (*Logger, error) {
	return New(config.LoggingConfig{
		Dir:              dir,
		FilePrefix:       "app",
		MaxBodyBytes:     1024,
		WebhookQueueSize: 100,
		WebhookTimeout:   time.Second,
		WebhookBatchSize: 5,
		WebhookBatchWait: time.Second,
		WebhookMaxChars:  1000,
	})
}

func TestWebhookBatchingBySize(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)

		got = append(got, payload["content"])
		w.WriteHeader(http.StatusOK)
	}))

	defer srv.Close()

	sender := newWebhookSender(config.LoggingConfig{
		DiscordWebhookURL: srv.URL,
		WebhookQueueSize:  10,
		WebhookTimeout:    time.Second,
		WebhookBatchSize:  2,
		WebhookBatchWait:  time.Second,
		WebhookMaxChars:   1000,
	})
	if sender == nil {
		t.Fatalf("expected sender")
	}

	_ = sender.Enqueue(context.Background(), "one")
	_ = sender.Enqueue(context.Background(), "two")
	_ = sender.Close()

	if len(got) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(got))
	}

	if got[0] != "```\none\ntwo\n```" {
		t.Fatalf("unexpected batch content: %s", got[0])
	}
}

func TestWebhookBatchingByTime(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)

		got = append(got, payload["content"])
		w.WriteHeader(http.StatusOK)
	}))

	defer srv.Close()

	sender := newWebhookSender(config.LoggingConfig{
		DiscordWebhookURL: srv.URL,
		WebhookQueueSize:  10,
		WebhookTimeout:    time.Second,
		WebhookBatchSize:  10,
		WebhookBatchWait:  50 * time.Millisecond,
		WebhookMaxChars:   1000,
	})
	if sender == nil {
		t.Fatalf("expected sender")
	}

	_ = sender.Enqueue(context.Background(), "one")
	time.Sleep(80 * time.Millisecond)
	_ = sender.Close()

	if len(got) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(got))
	}

	if got[0] != "```\none\n```" {
		t.Fatalf("unexpected batch content: %s", got[0])
	}
}

func TestWebhookMessageSplit(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)

		got = append(got, payload["content"])
		w.WriteHeader(http.StatusOK)
	}))

	defer srv.Close()

	sender := newWebhookSender(config.LoggingConfig{
		DiscordWebhookURL: srv.URL,
		WebhookQueueSize:  10,
		WebhookTimeout:    time.Second,
		WebhookBatchSize:  1,
		WebhookBatchWait:  time.Millisecond,
		WebhookMaxChars:   20,
	})
	if sender == nil {
		t.Fatalf("expected sender")
	}

	long := strings.Repeat("a", 30)
	if err := sender.Send(context.Background(), long); err != nil {
		t.Fatalf("send: %v", err)
	}

	if len(got) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(got))
	}
}
