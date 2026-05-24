package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"smctf/internal/config"
)

type Options struct {
	Service   string
	Env       string
	AddSource bool
}

type Logger struct {
	*slog.Logger
	writer io.Writer
	closer io.Closer
}

func New(cfg config.LoggingConfig, opts Options) (*Logger, error) {
	var fileWriter *rotatingFileWriter
	var err error
	if cfg.Dir != "" {
		fileWriter, err = newRotatingFileWriter(cfg.Dir, cfg.FilePrefix)
		if err != nil {
			return nil, err
		}
	}

	handlerOptions := &slog.HandlerOptions{
		AddSource: opts.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Key = "ts"
			case slog.LevelKey:
				a.Value = slog.StringValue(strings.ToLower(a.Value.String()))
			}
			return a
		},
	}

	handlers := make([]slog.Handler, 0, 3)

	stdoutHandler := slog.NewJSONHandler(os.Stdout, handlerOptions)
	handlers = append(handlers, levelRangeHandler{
		min:     slog.LevelDebug,
		max:     slog.LevelWarn,
		hasMax:  true,
		handler: stdoutHandler,
	})

	stderrHandler := slog.NewJSONHandler(os.Stderr, handlerOptions)
	handlers = append(handlers, levelRangeHandler{
		min:     slog.LevelError,
		handler: stderrHandler,
	})

	if fileWriter != nil {
		var w io.Writer
		w = fileWriter

		fileHandler := slog.NewJSONHandler(w, handlerOptions)
		handlers = append(handlers, levelRangeHandler{
			min:     slog.LevelDebug,
			handler: fileHandler,
		})
	}

	logger := slog.New(teeHandler{handlers: handlers})

	baseAttrs := make([]slog.Attr, 0, 1)
	if app := appName(opts.Service, opts.Env); app != "" {
		baseAttrs = append(baseAttrs, slog.String("app", app))
	}

	if len(baseAttrs) > 0 {
		anyAttrs := make([]any, 0, len(baseAttrs))
		for _, attr := range baseAttrs {
			anyAttrs = append(anyAttrs, attr)
		}

		logger = logger.With(anyAttrs...)
	}

	var closer io.Closer
	if fileWriter != nil {
		closer = fileWriter
	}

	return &Logger{
		Logger: logger,
		writer: &logWriter{logger: logger},
		closer: closer,
	}, nil
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}

	if l.closer != nil {
		return l.closer.Close()
	}

	return nil
}

func (l *Logger) Write(p []byte) (int, error) {
	if l == nil {
		return len(p), nil
	}

	return l.writer.Write(p)
}

func PanicLogger(logger *Logger, ctx context.Context, recovered any) {
	if logger == nil {
		return
	}

	log := logger.Logger

	err := fmt.Errorf("panic: %v", recovered)
	log.Error("panic recovered",
		slog.Any("error", err),
	)
}

type logWriter struct {
	logger *slog.Logger
}

func (w *logWriter) Write(p []byte) (int, error) {
	if w == nil || w.logger == nil {
		return len(p), nil
	}

	msg := strings.TrimRight(string(p), "\n")
	if msg == "" {
		return len(p), nil
	}

	w.logger.Info(msg, slog.Bool("legacy", true))
	return len(p), nil
}

type levelRangeHandler struct {
	min     slog.Level
	max     slog.Level
	hasMax  bool
	handler slog.Handler
}

func (h levelRangeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if level < h.min {
		return false
	}
	if h.hasMax && level > h.max {
		return false
	}
	return h.handler.Enabled(ctx, level)
}

func (h levelRangeHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.handler.Handle(ctx, record)
}

func (h levelRangeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return levelRangeHandler{
		min:     h.min,
		max:     h.max,
		hasMax:  h.hasMax,
		handler: h.handler.WithAttrs(attrs),
	}
}

func (h levelRangeHandler) WithGroup(name string) slog.Handler {
	return levelRangeHandler{
		min:     h.min,
		max:     h.max,
		hasMax:  h.hasMax,
		handler: h.handler.WithGroup(name),
	}
}

type teeHandler struct {
	handlers []slog.Handler
}

func (t teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range t.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (t teeHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range t.handlers {
		if h.Enabled(ctx, record.Level) {
			_ = h.Handle(ctx, record)
		}
	}
	return nil
}

func (t teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(t.handlers))
	for _, h := range t.handlers {
		next = append(next, h.WithAttrs(attrs))
	}
	return teeHandler{handlers: next}
}

func (t teeHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(t.handlers))
	for _, h := range t.handlers {
		next = append(next, h.WithGroup(name))
	}
	return teeHandler{handlers: next}
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

	if err := w.rotate(time.Now().UTC()); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now().UTC()
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
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
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

func appName(service, env string) string {
	parts := make([]string, 0, 2)
	if service != "" {
		parts = append(parts, service)
	}

	if env != "" {
		parts = append(parts, env)
	}

	return strings.Join(parts, ":")
}
