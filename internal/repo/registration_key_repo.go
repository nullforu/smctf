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

func (r *RegistrationKeyRepo) GetByCode(ctx context.Context, code string) (*models.RegistrationKey, error) {
	key := new(models.RegistrationKey)
	if err := r.db.NewSelect().Model(key).Where("code = ?", code).Scan(ctx); err != nil {
		return nil, wrapNotFound("registrationKeyRepo.GetByCode", err)
	}

	return key, nil
}

func (r *RegistrationKeyRepo) baseRegistrationKeySummaryQuery() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("registration_keys AS rk").
		ColumnExpr("rk.id AS id").
		ColumnExpr("rk.code AS code").
		ColumnExpr("rk.created_by AS created_by").
		ColumnExpr("creator.username AS created_by_username").
		ColumnExpr("rk.team_id AS team_id").
		ColumnExpr("g.name AS team_name").
		ColumnExpr("rk.max_uses AS max_uses").
		ColumnExpr("rk.used_count AS used_count").
		ColumnExpr("rk.created_at AS created_at").
		ColumnExpr("MAX(rku.used_at) AS last_used_at").
		Join("JOIN users AS creator ON creator.id = rk.created_by").
		Join("JOIN teams AS g ON g.id = rk.team_id").
		Join("LEFT JOIN registration_key_uses AS rku ON rku.registration_key_id = rk.id").
		GroupExpr("rk.id, creator.username, g.name")
}

func (r *RegistrationKeyRepo) List(ctx context.Context) ([]models.RegistrationKeySummary, error) {
	keys := make([]models.RegistrationKeySummary, 0)

	query := r.baseRegistrationKeySummaryQuery().OrderExpr("rk.id DESC")
	if err := query.Scan(ctx, &keys); err != nil {
		return nil, wrapError("registrationKeyRepo.List", err)
	}

	if len(keys) == 0 {
		return keys, nil
	}

	keyIDs := make([]int64, 0, len(keys))
	for _, key := range keys {
		keyIDs = append(keyIDs, key.ID)
	}

	uses := make([]models.RegistrationKeyUseSummary, 0)
	if err := r.db.NewSelect().
		TableExpr("registration_key_uses AS rku").
		ColumnExpr("rku.registration_key_id AS registration_key_id").
		ColumnExpr("rku.used_by AS used_by").
		ColumnExpr("used.username AS used_by_username").
		ColumnExpr("rku.used_by_ip AS used_by_ip").
		ColumnExpr("rku.used_at AS used_at").
		Join("JOIN users AS used ON used.id = rku.used_by").
		Where("rku.registration_key_id IN (?)", bun.In(keyIDs)).
		OrderExpr("rku.used_at DESC").
		Scan(ctx, &uses); err != nil {
		return nil, wrapError("registrationKeyRepo.List uses", err)
	}

	usesByKey := make(map[int64][]models.RegistrationKeyUseSummary, len(keyIDs))
	for _, use := range uses {
		usesByKey[use.RegistrationKeyID] = append(usesByKey[use.RegistrationKeyID], use)
	}

	for i := range keys {
		keys[i].Uses = usesByKey[keys[i].ID]
	}

	return keys, nil
}

func (r *RegistrationKeyRepo) ImportRegistrationKeys(ctx context.Context, keys []models.RegistrationKey) ([]models.RegistrationKey, error) {
	if len(keys) == 0 {
		return []models.RegistrationKey{}, nil
	}

	imported := make([]models.RegistrationKey, 0, len(keys))
	if err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, key := range keys {
			key.ID = 0
			key.UsedCount = 0
			if _, err := tx.NewInsert().Model(&key).Exec(ctx); err != nil {
				return wrapError("registrationKeyRepo.ImportRegistrationKeys insert", err)
			}
			imported = append(imported, key)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return imported, nil
}
