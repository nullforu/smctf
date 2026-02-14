package service

import (
	"context"
	"errors"
	"testing"

	"smctf/internal/repo"
)

func TestAppConfigServiceDefaultsPersisted(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo)

	cfg, updatedAt, etag, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if cfg.Title == "" || cfg.Description == "" {
		t.Fatalf("expected defaults, got %+v", cfg)
	}

	if updatedAt.IsZero() || etag == "" {
		t.Fatalf("expected updatedAt and etag")
	}

	rows, err := appRepo.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected defaults stored, got %d rows", len(rows))
	}
}

func TestAppConfigServiceUpdatePartial(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo)

	_, _, _, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	title := "New Title"
	cfg, _, _, err := svc.Update(context.Background(), &title, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if cfg.Title != "New Title" {
		t.Fatalf("expected updated title, got %s", cfg.Title)
	}

	if cfg.Description == "" {
		t.Fatalf("expected description to remain")
	}
}

func TestAppConfigServiceUpdateValidation(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo)

	empty := ""
	_, _, _, err := svc.Update(context.Background(), &empty, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %T", err)
	}
}

func TestAppConfigServiceUpdateCTFTimes(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo)

	start := "2026-02-10T10:00:00Z"
	end := "2026-02-10T12:00:00Z"
	cfg, _, _, err := svc.Update(context.Background(), nil, nil, nil, nil, &start, &end)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if cfg.CTFStartAt != start || cfg.CTFEndAt != end {
		t.Fatalf("expected ctf times, got %+v", cfg)
	}

	invalid := "nope"
	if _, _, _, err := svc.Update(context.Background(), nil, nil, nil, nil, &invalid, nil); err == nil {
		t.Fatalf("expected validation error")
	}

	empty := ""
	if _, _, _, err := svc.Update(context.Background(), nil, nil, nil, nil, &empty, &empty); err != nil {
		t.Fatalf("expected empty times to be allowed, got %v", err)
	}
}
