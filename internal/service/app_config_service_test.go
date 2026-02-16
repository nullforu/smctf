package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"smctf/internal/repo"
)

func TestAppConfigServiceDefaultsPersisted(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo, env.redis, env.cfg.Cache.AppConfigTTL)

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

func TestAppConfigServiceUsesRedisCache(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo, env.redis, env.cfg.Cache.AppConfigTTL)

	cfg, updatedAt, etag, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if exists, err := env.redis.Exists(context.Background(), appConfigCacheKey).Result(); err != nil || exists == 0 {
		t.Fatalf("expected app config to be cached in redis")
	}

	cachedCfg, cachedUpdatedAt, cachedETag, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get cached: %v", err)
	}

	if cachedCfg.Title != cfg.Title || cachedETag != etag || !cachedUpdatedAt.Equal(updatedAt) {
		t.Fatalf("cached mismatch: %+v", cachedCfg)
	}
}

func TestAppConfigServiceUpdatePartial(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo, env.redis, env.cfg.Cache.AppConfigTTL)

	_, _, _, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	title := "New Title"
	cfg, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		Title: AppConfigUpdateInput{Set: true, Value: title},
	})
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
	svc := NewAppConfigService(appRepo, env.redis, env.cfg.Cache.AppConfigTTL)

	whitespaceCases := []struct {
		name  string
		input AppConfigUpdate
	}{
		{
			name: "title whitespace",
			input: AppConfigUpdate{
				Title: AppConfigUpdateInput{Set: true, Value: "   "},
			},
		},
		{
			name: "description whitespace",
			input: AppConfigUpdate{
				Description: AppConfigUpdateInput{Set: true, Value: "   "},
			},
		},
		{
			name: "header_title whitespace",
			input: AppConfigUpdate{
				HeaderTitle: AppConfigUpdateInput{Set: true, Value: "   "},
			},
		},
		{
			name: "header_description whitespace",
			input: AppConfigUpdate{
				HeaderDescription: AppConfigUpdateInput{Set: true, Value: "   "},
			},
		},
	}

	for _, tc := range whitespaceCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := svc.Update(context.Background(), tc.input); err != nil {
				t.Fatalf("expected whitespace to be allowed, got %v", err)
			}
		})
	}

	nullCases := []struct {
		name  string
		input AppConfigUpdate
	}{
		{
			name: "title null",
			input: AppConfigUpdate{
				Title: AppConfigUpdateInput{Set: true, Null: true},
			},
		},
		{
			name: "description null",
			input: AppConfigUpdate{
				Description: AppConfigUpdateInput{Set: true, Null: true},
			},
		},
		{
			name: "header_title null",
			input: AppConfigUpdate{
				HeaderTitle: AppConfigUpdateInput{Set: true, Null: true},
			},
		},
		{
			name: "header_description null",
			input: AppConfigUpdate{
				HeaderDescription: AppConfigUpdateInput{Set: true, Null: true},
			},
		},
	}
	for _, tc := range nullCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := svc.Update(context.Background(), tc.input)
			if err == nil {
				t.Fatal("expected validation error for null value")
			}

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected validation error, got %T", err)
			}
		})
	}

	longTitle := strings.Repeat("a", 201)
	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		Title: AppConfigUpdateInput{Set: true, Value: longTitle},
	}); err == nil {
		t.Fatal("expected validation error for title length")
	}

	longDescription := strings.Repeat("b", 2001)
	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		Description: AppConfigUpdateInput{Set: true, Value: longDescription},
	}); err == nil {
		t.Fatal("expected validation error for description length")
	}

	longHeaderTitle := strings.Repeat("c", 81)
	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		HeaderTitle: AppConfigUpdateInput{Set: true, Value: longHeaderTitle},
	}); err == nil {
		t.Fatal("expected validation error for header_title length")
	}

	longHeaderDescription := strings.Repeat("d", 201)
	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		HeaderDescription: AppConfigUpdateInput{Set: true, Value: longHeaderDescription},
	}); err == nil {
		t.Fatal("expected validation error for header_description length")
	}

	cfg := AppConfig{}
	if _, err := applyAppConfigUpdates(&cfg, map[string]AppConfigUpdateInput{
		"bad_key": {Set: true, Value: "x"},
	}); err == nil {
		t.Fatal("expected validation error for invalid key")
	}
}

func TestAppConfigServiceUpdateCTFTimes(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo, env.redis, env.cfg.Cache.AppConfigTTL)

	now := time.Now().UTC()
	startTime := now.Add(1 * time.Hour)
	endTime := startTime.Add(2 * time.Hour)
	start := startTime.Format(time.RFC3339)
	end := endTime.Format(time.RFC3339)
	cfg, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		CTFStartAt: AppConfigUpdateInput{Set: true, Value: start},
		CTFEndAt:   AppConfigUpdateInput{Set: true, Value: end},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if cfg.CTFStartAt != start || cfg.CTFEndAt != end {
		t.Fatalf("expected ctf times, got %+v", cfg)
	}

	invalid := "nope"
	_, _, _, err = svc.Update(context.Background(), AppConfigUpdate{
		CTFStartAt: AppConfigUpdateInput{Set: true, Value: invalid},
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) || len(ve.Fields) == 0 || ve.Fields[0].Reason != "invalid_format" {
		t.Fatalf("expected invalid_format, got %v", err)
	}

	badEnd := "2026-02-10T09:00:00Z"
	_, _, _, err = svc.Update(context.Background(), AppConfigUpdate{
		CTFStartAt: AppConfigUpdateInput{Set: true, Value: start},
		CTFEndAt:   AppConfigUpdateInput{Set: true, Value: badEnd},
	})
	if err == nil {
		t.Fatalf("expected validation error for end before start")
	}
	if !errors.As(err, &ve) || len(ve.Fields) == 0 || ve.Fields[0].Reason != "end_before_start" {
		t.Fatalf("expected end_before_start, got %v", err)
	}

	null := AppConfigUpdateInput{Set: true, Null: true}
	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		CTFStartAt: null,
		CTFEndAt:   null,
	}); err != nil {
		t.Fatalf("expected null times to be allowed, got %v", err)
	}

	whitespace := "   "
	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		CTFStartAt: AppConfigUpdateInput{Set: true, Value: whitespace},
	}); err == nil {
		t.Fatalf("expected whitespace ctf_start_at to be invalid")
	}

	empty := ""
	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		CTFStartAt: AppConfigUpdateInput{Set: true, Value: empty},
	}); err == nil {
		t.Fatalf("expected empty ctf_start_at to be invalid")
	}

	longCTFValue := strings.Repeat("e", 65)
	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		CTFStartAt: AppConfigUpdateInput{Set: true, Value: longCTFValue},
	}); err == nil {
		t.Fatalf("expected ctf_start_at length to be invalid")
	}

	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		CTFEndAt: AppConfigUpdateInput{Set: true, Value: longCTFValue},
	}); err == nil {
		t.Fatalf("expected ctf_end_at length to be invalid")
	}
}

func TestAppConfigServiceUpdateNoChanges(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo, env.redis, env.cfg.Cache.AppConfigTTL)

	cfg, updatedAt, etag, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	outCfg, outUpdatedAt, outETag, err := svc.Update(context.Background(), AppConfigUpdate{})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if outCfg.Title != cfg.Title || outETag != etag || !outUpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected unchanged update: %+v", outCfg)
	}
}

func TestAppConfigServiceGetCacheInvalidJSON(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo, env.redis, env.cfg.Cache.AppConfigTTL)

	if err := env.redis.Set(context.Background(), appConfigCacheKey, "{not-json}", time.Minute).Err(); err != nil {
		t.Fatalf("set bad cache: %v", err)
	}

	if _, ok := svc.getCache(context.Background()); ok {
		t.Fatalf("expected cache miss for invalid json")
	}

	exists, err := env.redis.Exists(context.Background(), appConfigCacheKey).Result()
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists != 0 {
		t.Fatalf("expected bad cache to be deleted")
	}
}

func TestAppConfigServiceCacheNilRedis(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo, nil, env.cfg.Cache.AppConfigTTL)

	if _, ok := svc.getCache(context.Background()); ok {
		t.Fatalf("expected cache miss with nil redis")
	}

	cfg, _, _, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if cfg.Title == "" {
		t.Fatalf("expected config from DB")
	}

	svc.storeCache(context.Background(), appConfigCache{Config: cfg})
	svc.invalidateCache(context.Background())
}

func TestAppConfigServiceStoreCacheTTLDisabled(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo, env.redis, 0)

	cfg, _, _, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	svc.storeCache(context.Background(), appConfigCache{Config: cfg})

	exists, err := env.redis.Exists(context.Background(), appConfigCacheKey).Result()
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if exists != 0 {
		t.Fatalf("expected cache to be skipped with ttl=0")
	}
}

func TestAppConfigServiceCTFState(t *testing.T) {
	env := setupServiceTest(t)
	appRepo := repo.NewAppConfigRepo(env.db)
	svc := NewAppConfigService(appRepo, env.redis, env.cfg.Cache.AppConfigTTL)

	now := time.Date(2026, 2, 10, 9, 0, 0, 0, time.UTC)
	start := now.Add(2 * time.Hour).Format(time.RFC3339)
	end := now.Add(4 * time.Hour).Format(time.RFC3339)

	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		CTFStartAt: AppConfigUpdateInput{Set: true, Value: start},
		CTFEndAt:   AppConfigUpdateInput{Set: true, Value: end},
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	state, err := svc.CTFState(context.Background(), now)
	if err != nil {
		t.Fatalf("CTFState: %v", err)
	}
	if state != CTFStateNotStarted {
		t.Fatalf("expected not_started, got %s", state)
	}

	start = now.Add(-time.Hour).Format(time.RFC3339)
	end = now.Add(time.Hour).Format(time.RFC3339)
	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		CTFStartAt: AppConfigUpdateInput{Set: true, Value: start},
		CTFEndAt:   AppConfigUpdateInput{Set: true, Value: end},
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	state, err = svc.CTFState(context.Background(), now)
	if err != nil {
		t.Fatalf("CTFState: %v", err)
	}
	if state != CTFStateActive {
		t.Fatalf("expected active, got %s", state)
	}

	end = now.Add(-time.Minute).Format(time.RFC3339)
	if _, _, _, err := svc.Update(context.Background(), AppConfigUpdate{
		CTFStartAt: AppConfigUpdateInput{Set: true, Value: start},
		CTFEndAt:   AppConfigUpdateInput{Set: true, Value: end},
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	state, err = svc.CTFState(context.Background(), now)
	if err != nil {
		t.Fatalf("CTFState: %v", err)
	}
	if state != CTFStateEnded {
		t.Fatalf("expected ended, got %s", state)
	}
}
