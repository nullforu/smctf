package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"smctf/internal/models"
	"smctf/internal/repo"

	"github.com/redis/go-redis/v9"
)

const (
	appConfigKeyTitle       = "title"
	appConfigKeyDescription = "description"
	appConfigKeyHeaderTitle = "header_title"
	appConfigKeyHeaderDesc  = "header_description"
	appConfigKeyCTFStartAt  = "ctf_start_at"
	appConfigKeyCTFEndAt    = "ctf_end_at"
)

type AppConfig struct {
	Title             string `json:"title"`
	Description       string `json:"description"`
	HeaderTitle       string `json:"header_title"`
	HeaderDescription string `json:"header_description"`
	CTFStartAt        string `json:"ctf_start_at"`
	CTFEndAt          string `json:"ctf_end_at"`
}

type CTFState string

const (
	CTFStateActive     CTFState = "active"
	CTFStateNotStarted CTFState = "not_started"
	CTFStateEnded      CTFState = "ended"
)

type appConfigField struct {
	key          string
	defaultValue string
	maxLen       int
	get          func(cfg AppConfig) string
	set          func(cfg *AppConfig, value string)
}

var appConfigFields = []appConfigField{
	{
		key:          appConfigKeyTitle,
		defaultValue: "Welcome to SMCTF.",
		maxLen:       200,
		get: func(cfg AppConfig) string {
			return cfg.Title
		},
		set: func(cfg *AppConfig, value string) {
			cfg.Title = value
		},
	},
	{
		key:          appConfigKeyDescription,
		defaultValue: "Check out the repository for setup instructions.",
		maxLen:       2000,
		get: func(cfg AppConfig) string {
			return cfg.Description
		},
		set: func(cfg *AppConfig, value string) {
			cfg.Description = value
		},
	},
	{
		key:          appConfigKeyHeaderTitle,
		defaultValue: "CTF",
		maxLen:       80,
		get: func(cfg AppConfig) string {
			return cfg.HeaderTitle
		},
		set: func(cfg *AppConfig, value string) {
			cfg.HeaderTitle = value
		},
	},
	{
		key:          appConfigKeyHeaderDesc,
		defaultValue: "Capture The Flag",
		maxLen:       200,
		get: func(cfg AppConfig) string {
			return cfg.HeaderDescription
		},
		set: func(cfg *AppConfig, value string) {
			cfg.HeaderDescription = value
		},
	},
	{
		key:          appConfigKeyCTFStartAt,
		defaultValue: "",
		maxLen:       64,
		get: func(cfg AppConfig) string {
			return cfg.CTFStartAt
		},
		set: func(cfg *AppConfig, value string) {
			cfg.CTFStartAt = value
		},
	},
	{
		key:          appConfigKeyCTFEndAt,
		defaultValue: "",
		maxLen:       64,
		get: func(cfg AppConfig) string {
			return cfg.CTFEndAt
		},
		set: func(cfg *AppConfig, value string) {
			cfg.CTFEndAt = value
		},
	},
}

type appConfigCache struct {
	Config    AppConfig `json:"config"`
	UpdatedAt time.Time `json:"updated_at"`
	ETag      string    `json:"etag"`
}

type AppConfigService struct {
	repo     *repo.AppConfigRepo
	redis    *redis.Client
	cacheTTL time.Duration
}

const appConfigCacheKey = "app_config:cached"

func NewAppConfigService(repo *repo.AppConfigRepo, redisClient *redis.Client, cacheTTL time.Duration) *AppConfigService {
	return &AppConfigService{repo: repo, redis: redisClient, cacheTTL: cacheTTL}
}

func (s *AppConfigService) Get(ctx context.Context) (AppConfig, time.Time, string, error) {
	if cached, ok := s.getCache(ctx); ok {
		return cached.Config, cached.UpdatedAt, cached.ETag, nil
	}

	return s.load(ctx)
}

func (s *AppConfigService) Update(ctx context.Context, title *string, description *string, headerTitle *string, headerDescription *string, ctfStartAt *string, ctfEndAt *string) (AppConfig, time.Time, string, error) {
	cfg, cachedUpdatedAt, cachedETag, err := s.Get(ctx)
	if err != nil {
		return AppConfig{}, time.Time{}, "", err
	}

	inputs := map[string]*string{
		appConfigKeyTitle:       title,
		appConfigKeyDescription: description,
		appConfigKeyHeaderTitle: headerTitle,
		appConfigKeyHeaderDesc:  headerDescription,
		appConfigKeyCTFStartAt:  ctfStartAt,
		appConfigKeyCTFEndAt:    ctfEndAt,
	}

	updates, err := applyAppConfigUpdates(&cfg, inputs)
	if err != nil {
		return AppConfig{}, time.Time{}, "", err
	}

	if len(updates) == 0 {
		return cfg, cachedUpdatedAt, cachedETag, nil
	}

	rows, err := s.repo.UpsertMany(ctx, updates)
	if err != nil {
		return AppConfig{}, time.Time{}, "", err
	}

	updatedAt := maxUpdatedAt(rows)
	etag := buildETag(cfg)
	s.invalidateCache(ctx)
	s.storeCache(ctx, appConfigCache{Config: cfg, UpdatedAt: updatedAt, ETag: etag})

	return cfg, updatedAt, etag, nil
}

func (s *AppConfigService) CTFState(ctx context.Context, now time.Time) (CTFState, error) {
	cfg, _, _, err := s.Get(ctx)
	if err != nil {
		return CTFStateActive, err
	}

	startAt, startSet, err := parseRFC3339Optional(cfg.CTFStartAt)
	if err != nil {
		return CTFStateActive, err
	}

	endAt, endSet, err := parseRFC3339Optional(cfg.CTFEndAt)
	if err != nil {
		return CTFStateActive, err
	}

	if startSet && now.Before(startAt) {
		return CTFStateNotStarted, nil
	}

	if endSet && now.After(endAt) {
		return CTFStateEnded, nil
	}

	return CTFStateActive, nil
}

func (s *AppConfigService) getCache(ctx context.Context) (appConfigCache, bool) {
	if s.redis == nil {
		return appConfigCache{}, false
	}

	cached, err := s.redis.Get(ctx, appConfigCacheKey).Result()
	if err != nil {
		return appConfigCache{}, false
	}

	var data appConfigCache
	if err := json.Unmarshal([]byte(cached), &data); err != nil {
		_ = s.redis.Del(ctx, appConfigCacheKey).Err()
		return appConfigCache{}, false
	}

	return data, true
}

func (s *AppConfigService) load(ctx context.Context) (AppConfig, time.Time, string, error) {
	rows, err := s.repo.GetAll(ctx)
	if err != nil {
		return AppConfig{}, time.Time{}, "", err
	}

	cfg, updatedAt, missing := buildConfigFromRows(rows)
	if len(missing) > 0 {
		if _, err := s.repo.UpsertMany(ctx, missing); err != nil {
			return AppConfig{}, time.Time{}, "", err
		}

		updatedAt = time.Now().UTC()
	}

	etag := buildETag(cfg)
	s.storeCache(ctx, appConfigCache{Config: cfg, UpdatedAt: updatedAt, ETag: etag})

	return cfg, updatedAt, etag, nil
}

func (s *AppConfigService) storeCache(ctx context.Context, data appConfigCache) {
	if s.redis == nil || s.cacheTTL <= 0 {
		return
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	_ = s.redis.Set(ctx, appConfigCacheKey, payload, s.cacheTTL).Err()
}

func (s *AppConfigService) invalidateCache(ctx context.Context) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Del(ctx, appConfigCacheKey).Err()
}

func buildConfigFromRows(rows []models.AppConfig) (AppConfig, time.Time, map[string]string) {
	cfg := defaultAppConfig()
	updatedAt := time.Time{}
	missing := map[string]string{}
	seen := map[string]bool{}

	fieldMap := appConfigFieldMap()
	for _, row := range rows {
		field, ok := fieldMap[row.Key]
		if !ok {
			continue
		}

		field.set(&cfg, row.Value)
		seen[row.Key] = true
		updatedAt = maxTime(updatedAt, row.UpdatedAt)
	}

	for _, field := range appConfigFields {
		if !seen[field.key] {
			missing[field.key] = field.get(cfg)
		}
	}

	return cfg, updatedAt, missing
}

func maxUpdatedAt(rows []models.AppConfig) time.Time {
	updated := time.Time{}
	for _, row := range rows {
		updated = maxTime(updated, row.UpdatedAt)
	}

	return updated
}

func maxTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return b
	}

	return a
}

func buildETag(cfg AppConfig) string {
	var b strings.Builder
	for i, field := range appConfigFields {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(field.get(cfg))
	}
	hash := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("\"%x\"", hash[:])
}

func defaultAppConfig() AppConfig {
	cfg := AppConfig{}
	for _, field := range appConfigFields {
		field.set(&cfg, field.defaultValue)
	}
	return cfg
}

func appConfigFieldMap() map[string]appConfigField {
	fields := make(map[string]appConfigField, len(appConfigFields))
	for _, field := range appConfigFields {
		fields[field.key] = field
	}
	return fields
}

func applyAppConfigUpdates(cfg *AppConfig, inputs map[string]*string) (map[string]string, error) {
	fields := appConfigFieldMap()
	updates := make(map[string]string)

	for key, valuePtr := range inputs {
		if valuePtr == nil {
			continue
		}

		field, ok := fields[key]
		if !ok {
			return nil, NewValidationError(FieldError{Field: key, Reason: "invalid"})
		}

		value := strings.TrimSpace(*valuePtr)
		if value == "" && !isOptionalConfigField(key) {
			return nil, NewValidationError(FieldError{Field: key, Reason: "required"})
		}

		if field.maxLen > 0 && len(value) > field.maxLen {
			return nil, NewValidationError(FieldError{Field: key, Reason: "too_long"})
		}

		if key == appConfigKeyCTFStartAt || key == appConfigKeyCTFEndAt {
			if _, _, err := parseRFC3339Optional(value); err != nil {
				return nil, NewValidationError(FieldError{Field: key, Reason: "invalid_format"})
			}
		}

		field.set(cfg, value)
		updates[key] = value
	}

	startAt, startSet, err := parseRFC3339Optional(cfg.CTFStartAt)
	if err != nil {
		return nil, NewValidationError(FieldError{Field: appConfigKeyCTFStartAt, Reason: "invalid_format"})
	}

	endAt, endSet, err := parseRFC3339Optional(cfg.CTFEndAt)
	if err != nil {
		return nil, NewValidationError(FieldError{Field: appConfigKeyCTFEndAt, Reason: "invalid_format"})
	}

	if startSet && endSet && !endAt.After(startAt) {
		return nil, NewValidationError(FieldError{Field: appConfigKeyCTFEndAt, Reason: "end_before_start"})
	}

	return updates, nil
}

func isOptionalConfigField(key string) bool {
	return key == appConfigKeyCTFStartAt || key == appConfigKeyCTFEndAt
}

func parseRFC3339Optional(value string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false, err
	}

	return parsed, true, nil
}
