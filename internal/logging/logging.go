package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"smctf/internal/config"
)

type Logger struct {
	writer *rotatingFileWriter
	sender *webhookSender
}

func New(cfg config.LoggingConfig) (*Logger, error) {
	writer, err := newRotatingFileWriter(cfg.Dir, cfg.FilePrefix)
	if err != nil {
		return nil, err
	}

	return &Logger{
		writer: writer,
		sender: newWebhookSender(cfg),
	}, nil
}

func (l *Logger) Write(p []byte) (int, error) {
	if l == nil {
		return len(p), nil
	}

	n, err := l.writer.Write(p)
	if l.sender != nil {
		_ = l.sender.Enqueue(context.Background(), string(p))
	}

	return n, err
}

func (l *Logger) Close() error {
	if l == nil || l.writer == nil {
		return nil
	}

	if l.sender != nil {
		_ = l.sender.Close()
	}

	return l.writer.Close()
}

type rotatingFileWriter struct {
	mu          sync.Mutex
	dir         string
	prefix      string
	currentHour time.Time
	file        *os.File
}

func newRotatingFileWriter(dir, prefix string) (*rotatingFileWriter, error) {
	w := &rotatingFileWriter{
		dir:    dir,
		prefix: prefix,
	}

	if err := w.rotate(time.Now()); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	if !sameHour(now, w.currentHour) {
		if err := w.rotate(now); err != nil {
			return 0, err
		}
	}

	if w.file == nil {
		if err := w.rotate(now); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	if err != nil {
		return n, err
	}

	if err := w.file.Sync(); err != nil {
		return n, err
	}

	return n, nil
}

func (w *rotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}

	if err := w.file.Sync(); err != nil {
		_ = w.file.Close()
		w.file = nil
		return err
	}

	err := w.file.Close()
	w.file = nil
	return err
}

func (w *rotatingFileWriter) rotate(now time.Time) error {
	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return err
	}

	if w.file != nil {
		_ = w.file.Sync()
		_ = w.file.Close()
		w.file = nil
	}

	path := w.pathForTime(now)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) // 0111b = execute for all, 0o644 = rw-r--r--
	if err != nil {
		return err
	}

	w.file = file
	w.currentHour = truncateToHour(now)

	return nil
}

func (w *rotatingFileWriter) pathForTime(t time.Time) string {
	name := fmt.Sprintf("%s-%s.log", w.prefix, t.Format("20060102-15"))
	return filepath.Join(w.dir, name)
}

func truncateToHour(t time.Time) time.Time {
	return t.Truncate(time.Hour)
}

func sameHour(a, b time.Time) bool {
	return !b.IsZero() && truncateToHour(a).Equal(truncateToHour(b))
}

type webhookSender struct {
	discordURL string
	slackURL   string
	client     *http.Client
	queue      chan string
	wg         sync.WaitGroup
	closeOnce  sync.Once
	timeout    time.Duration
	batchSize  int
	batchWait  time.Duration
	maxChars   int
}

func newWebhookSender(cfg config.LoggingConfig) *webhookSender {
	if cfg.DiscordWebhookURL == "" && cfg.SlackWebhookURL == "" {
		return nil
	}

	s := &webhookSender{
		discordURL: cfg.DiscordWebhookURL,
		slackURL:   cfg.SlackWebhookURL,
		client: &http.Client{
			Timeout: cfg.WebhookTimeout,
		},
		queue:     make(chan string, cfg.WebhookQueueSize),
		timeout:   cfg.WebhookTimeout,
		batchSize: cfg.WebhookBatchSize,
		batchWait: cfg.WebhookBatchWait,
		maxChars:  cfg.WebhookMaxChars,
	}
	s.wg.Add(1)
	go s.worker()

	return s
}

func (s *webhookSender) Enqueue(ctx context.Context, msg string) error {
	if s == nil {
		return nil
	}

	select {
	case s.queue <- msg:
		return nil
	default:
		return fmt.Errorf("webhook queue full")
	}
}

func (s *webhookSender) Send(ctx context.Context, msg string) error {
	if s == nil {
		return nil
	}

	for _, chunk := range splitWebhookMessage(strings.TrimRight(msg, "\n"), s.maxChars) {
		payload := "```\n" + chunk + "\n```"

		if s.discordURL != "" {
			if err := s.post(ctx, s.discordURL, map[string]string{"content": payload}); err != nil {
				return err
			}
		}

		if s.slackURL != "" {
			if err := s.post(ctx, s.slackURL, map[string]string{"text": payload}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *webhookSender) worker() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.batchWait)
	defer ticker.Stop()

	batch := make([]string, 0, s.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		payload := strings.Join(batch, "\n")
		_ = s.Send(context.Background(), payload)
		batch = batch[:0]
	}

	for {
		select {
		case msg, ok := <-s.queue:
			if !ok {
				flush()
				return
			}

			batch = append(batch, msg)
			if len(batch) >= s.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *webhookSender) Close() error {
	if s == nil {
		return nil
	}

	s.closeOnce.Do(func() {
		close(s.queue)
	})

	s.wg.Wait()
	return nil
}

func (s *webhookSender) post(ctx context.Context, url string, payload map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}

	return nil
}

func splitWebhookMessage(msg string, maxChars int) []string {
	if maxChars <= 0 {
		return []string{msg}
	}

	const wrapperLen = 8
	if maxChars <= wrapperLen {
		return []string{""}
	}

	limit := maxChars - wrapperLen
	if len(msg) <= limit {
		return []string{msg}
	}

	lines := strings.Split(msg, "\n")
	chunks := make([]string, 0, (len(msg)/limit)+1)
	var b strings.Builder

	flush := func() {
		if b.Len() == 0 {
			return
		}

		chunks = append(chunks, b.String())
		b.Reset()
	}

	for _, line := range lines {
		if len(line) > limit {
			flush()

			for len(line) > 0 {
				n := min(len(line), limit)
				chunks = append(chunks, line[:n])
				line = line[n:]
			}

			continue
		}

		if b.Len() == 0 {
			b.WriteString(line)

			continue
		}

		if b.Len()+1+len(line) > limit {
			flush()
			b.WriteString(line)

			continue
		}

		b.WriteByte('\n')
		b.WriteString(line)
	}

	flush()

	return chunks
}
