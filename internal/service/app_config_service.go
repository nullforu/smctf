package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"smctf/internal/models"
	"smctf/internal/repo"
)

const (
	appConfigKeyTitle       = "title"
	appConfigKeyDescription = "description"
)

type AppConfig struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

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
}

type appConfigCache struct {
	parsed    AppConfig
	updatedAt time.Time
	etag      string
}

type AppConfigService struct {
	repo  *repo.AppConfigRepo
	cache atomic.Value
	mu    sync.Mutex
}

func NewAppConfigService(repo *repo.AppConfigRepo) *AppConfigService {
	return &AppConfigService{repo: repo}
}

func (s *AppConfigService) Get(ctx context.Context) (AppConfig, time.Time, string, error) {
	if cached := s.cached(); cached != nil {
		return cached.parsed, cached.updatedAt, cached.etag, nil
	}

	return s.load(ctx)
}

func (s *AppConfigService) Update(ctx context.Context, title *string, description *string) (AppConfig, time.Time, string, error) {
	cfg, _, _, err := s.Get(ctx)
	if err != nil {
		return AppConfig{}, time.Time{}, "", err
	}

	inputs := map[string]*string{
		appConfigKeyTitle:       title,
		appConfigKeyDescription: description,
	}

	updates, err := applyAppConfigUpdates(&cfg, inputs)
	if err != nil {
		return AppConfig{}, time.Time{}, "", err
	}

	if len(updates) == 0 {
		return cfg, s.cachedUpdatedAt(), buildETag(cfg), nil
	}

	rows, err := s.repo.UpsertMany(ctx, updates)
	if err != nil {
		return AppConfig{}, time.Time{}, "", err
	}

	updatedAt := maxUpdatedAt(rows)
	etag := buildETag(cfg)
	s.cache.Store(&appConfigCache{parsed: cfg, updatedAt: updatedAt, etag: etag})

	return cfg, updatedAt, etag, nil
}

func (s *AppConfigService) cached() *appConfigCache {
	value := s.cache.Load()
	if value == nil {
		return nil
	}

	cached, ok := value.(*appConfigCache)
	if !ok {
		return nil
	}

	return cached
}

func (s *AppConfigService) cachedUpdatedAt() time.Time {
	cached := s.cached()
	if cached == nil {
		return time.Time{}
	}

	return cached.updatedAt
}

func (s *AppConfigService) load(ctx context.Context) (AppConfig, time.Time, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cached := s.cached(); cached != nil {
		return cached.parsed, cached.updatedAt, cached.etag, nil
	}

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
	cached := &appConfigCache{parsed: cfg, updatedAt: updatedAt, etag: etag}
	s.cache.Store(cached)

	return cfg, updatedAt, etag, nil
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
		if value == "" {
			return nil, NewValidationError(FieldError{Field: key, Reason: "required"})
		}

		if field.maxLen > 0 && len(value) > field.maxLen {
			return nil, NewValidationError(FieldError{Field: key, Reason: "too_long"})
		}

		field.set(cfg, value)
		updates[key] = value
	}

	return updates, nil
}
