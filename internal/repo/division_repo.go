package repo

import (
	"context"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type DivisionRepo struct {
	db *bun.DB
}

func NewDivisionRepo(db *bun.DB) *DivisionRepo {
	return &DivisionRepo{db: db}
}

func (r *DivisionRepo) Create(ctx context.Context, division *models.Division) error {
	if _, err := r.db.NewInsert().Model(division).Exec(ctx); err != nil {
		return wrapError("divisionRepo.Create", err)
	}

	return nil
}

func (r *DivisionRepo) Update(ctx context.Context, division *models.Division) error {
	res, err := r.db.NewUpdate().
		Model(division).
		Column("name", "discord_role_id", "discord_announce_channel_id").
		WherePK().
		Exec(ctx)
	if err != nil {
		return wrapError("divisionRepo.Update", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return wrapError("divisionRepo.Update", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *DivisionRepo) List(ctx context.Context) ([]models.Division, error) {
	divisions := make([]models.Division, 0)
	if err := r.db.NewSelect().Model(&divisions).OrderExpr("id ASC").Scan(ctx); err != nil {
		return nil, wrapError("divisionRepo.List", err)
	}

	return divisions, nil
}

func (r *DivisionRepo) ListByIDs(ctx context.Context, ids []int64) ([]models.Division, error) {
	divisions := make([]models.Division, 0)
	if len(ids) == 0 {
		return divisions, nil
	}

	if err := r.db.NewSelect().
		Model(&divisions).
		Where("id IN (?)", bun.In(ids)).
		OrderExpr("id ASC").
		Scan(ctx); err != nil {
		return nil, wrapError("divisionRepo.ListByIDs", err)
	}

	return divisions, nil
}

func (r *DivisionRepo) GetByID(ctx context.Context, id int64) (*models.Division, error) {
	division := new(models.Division)
	if err := r.db.NewSelect().Model(division).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, wrapNotFound("divisionRepo.GetByID", err)
	}

	return division, nil
}

func (r *DivisionRepo) GetByName(ctx context.Context, name string) (*models.Division, error) {
	division := new(models.Division)
	if err := r.db.NewSelect().Model(division).Where("name = ?", name).Scan(ctx); err != nil {
		return nil, wrapNotFound("divisionRepo.GetByName", err)
	}

	return division, nil
}

func (r *DivisionRepo) ImportDivisions(ctx context.Context, divisions []models.Division) ([]models.Division, error) {
	if len(divisions) == 0 {
		return []models.Division{}, nil
	}

	imported := make([]models.Division, 0, len(divisions))
	if err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, division := range divisions {
			division.ID = 0
			if _, err := tx.NewInsert().Model(&division).Exec(ctx); err != nil {
				return wrapError("divisionRepo.ImportDivisions insert", err)
			}
			imported = append(imported, division)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return imported, nil
}
