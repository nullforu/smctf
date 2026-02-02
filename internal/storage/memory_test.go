package storage

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryPresignUploadAndDownload(t *testing.T) {
	store := NewMemoryChallengeFileStore(5 * time.Minute)

	upload, err := store.PresignUpload(context.Background(), "key.zip", "application/zip")
	if err != nil {
		t.Fatalf("presign upload: %v", err)
	}

	if upload.URL == "" {
		t.Fatalf("expected upload url")
	}

	if upload.Fields["key"] != "key.zip" {
		t.Fatalf("expected key field")
	}

	if upload.Fields["Content-Type"] != "application/zip" {
		t.Fatalf("expected Content-Type")
	}

	download, err := store.PresignDownload(context.Background(), "key.zip", "bundle.zip")
	if err != nil {
		t.Fatalf("presign download: %v", err)
	}

	if download.URL == "" {
		t.Fatalf("expected download url")
	}
}

func TestMemoryStoreConcurrentAccess(t *testing.T) {
	store := NewMemoryChallengeFileStore(5 * time.Minute)
	var wg sync.WaitGroup

	for i := range 20 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key-" + string(rune('a'+i)) + ".zip"
			_, _ = store.PresignUpload(context.Background(), key, "application/zip")
			_ = store.Delete(context.Background(), key)
		}(i)
	}

	wg.Wait()
}
