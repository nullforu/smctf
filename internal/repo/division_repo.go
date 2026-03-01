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

func (r *DivisionRepo) List(ctx context.Context) ([]models.Division, error) {
	divisions := make([]models.Division, 0)
	if err := r.db.NewSelect().Model(&divisions).OrderExpr("id ASC").Scan(ctx); err != nil {
		return nil, wrapError("divisionRepo.List", err)
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
