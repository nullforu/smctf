package repo

import (
	"context"
	"time"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type AppConfigRepo struct {
	db *bun.DB
}

func NewAppConfigRepo(db *bun.DB) *AppConfigRepo {
	return &AppConfigRepo{db: db}
}

func (r *AppConfigRepo) GetAll(ctx context.Context) ([]models.AppConfig, error) {
	rows := make([]models.AppConfig, 0)
	if err := r.db.NewSelect().Model(&rows).OrderExpr("key ASC").Scan(ctx); err != nil {
		return nil, wrapError("appConfigRepo.GetAll", err)
	}

	return rows, nil
}

func (r *AppConfigRepo) Upsert(ctx context.Context, key, value string) (*models.AppConfig, error) {
	now := time.Now().UTC()
	cfg := &models.AppConfig{
		Key:       key,
		Value:     value,
		UpdatedAt: now,
	}

	if _, err := r.db.NewInsert().Model(cfg).
		On("CONFLICT (key) DO UPDATE").
		Set("value = EXCLUDED.value").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("*").
		Exec(ctx); err != nil {
		return nil, wrapError("appConfigRepo.Upsert", err)
	}

	return cfg, nil
}

func (r *AppConfigRepo) UpsertMany(ctx context.Context, values map[string]string) ([]models.AppConfig, error) {
	if len(values) == 0 {
		return nil, nil
	}

	now := time.Now().UTC()
	rows := make([]models.AppConfig, 0, len(values))
	for key, value := range values {
		rows = append(rows, models.AppConfig{Key: key, Value: value, UpdatedAt: now})
	}

	if _, err := r.db.NewInsert().Model(&rows).
		On("CONFLICT (key) DO UPDATE").
		Set("value = EXCLUDED.value").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("*").
		Exec(ctx); err != nil {
		return nil, wrapError("appConfigRepo.UpsertMany", err)
	}

	return rows, nil
}
