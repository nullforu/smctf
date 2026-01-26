package repo

import (
	"context"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type RegistrationKeyRepo struct {
	db *bun.DB
}

func NewRegistrationKeyRepo(db *bun.DB) *RegistrationKeyRepo {
	return &RegistrationKeyRepo{db: db}
}

func (r *RegistrationKeyRepo) Create(ctx context.Context, key *models.RegistrationKey) error {
	if _, err := r.db.NewInsert().Model(key).Exec(ctx); err != nil {
		return wrapError("registrationKeyRepo.Create", err)
	}

	return nil
}

func (r *RegistrationKeyRepo) GetByCodeForUpdate(ctx context.Context, db bun.IDB, code string) (*models.RegistrationKey, error) {
	key := new(models.RegistrationKey)

	if err := db.NewSelect().
		Model(key).
		Where("code = ?", code).
		For("UPDATE").
		Scan(ctx); err != nil {
		return nil, wrapNotFound("registrationKeyRepo.GetByCodeForUpdate", err)
	}

	return key, nil
}

func (r *RegistrationKeyRepo) List(ctx context.Context) ([]models.RegistrationKeyView, error) {
	keys := make([]models.RegistrationKeyView, 0)

	query := r.db.NewSelect().
		TableExpr("registration_keys AS rk").
		ColumnExpr("rk.id AS id").
		ColumnExpr("rk.code AS code").
		ColumnExpr("rk.created_by AS created_by").
		ColumnExpr("creator.username AS created_by_username").
		ColumnExpr("rk.used_by AS used_by").
		ColumnExpr("used.username AS used_by_username").
		ColumnExpr("rk.used_by_ip AS used_by_ip").
		ColumnExpr("rk.created_at AS created_at").
		ColumnExpr("rk.used_at AS used_at").
		Join("JOIN users AS creator ON creator.id = rk.created_by").
		Join("LEFT JOIN users AS used ON used.id = rk.used_by").
		OrderExpr("rk.id DESC")

	if err := query.Scan(ctx, &keys); err != nil {
		return nil, wrapError("registrationKeyRepo.List", err)
	}

	return keys, nil
}
