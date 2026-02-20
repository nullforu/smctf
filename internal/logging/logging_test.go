package logging

import (
	"bytes"
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

func newTestLogger(dir string) (*Logger, error) {
	return New(config.LoggingConfig{
		Dir:          dir,
		FilePrefix:   "app",
		MaxBodyBytes: 1024,
	}, Options{Service: "smctf", Env: "test"})
}
