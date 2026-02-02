package storage

import (
	"context"
	"errors"
	"time"
)

var ErrNotConfigured = errors.New("storage not configured")

type PresignedPost struct {
	URL       string            `json:"url"`
	Fields    map[string]string `json:"fields"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type PresignedURL struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ChallengeFileStore interface {
	PresignUpload(ctx context.Context, key, contentType string) (PresignedPost, error)
	PresignDownload(ctx context.Context, key, filename string) (PresignedURL, error)
	Delete(ctx context.Context, key string) error
}
