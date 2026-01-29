package repo

import (
	"context"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type GroupRepo struct {
	db *bun.DB
}

func NewGroupRepo(db *bun.DB) *GroupRepo {
	return &GroupRepo{db: db}
}

func (r *GroupRepo) Create(ctx context.Context, group *models.Group) error {
	if _, err := r.db.NewInsert().Model(group).Exec(ctx); err != nil {
		return wrapError("groupRepo.Create", err)
	}

	return nil
}

func (r *GroupRepo) List(ctx context.Context) ([]models.Group, error) {
	groups := make([]models.Group, 0)
	if err := r.db.NewSelect().Model(&groups).OrderExpr("id ASC").Scan(ctx); err != nil {
		return nil, wrapError("groupRepo.List", err)
	}

	return groups, nil
}

func (r *GroupRepo) GetByID(ctx context.Context, id int64) (*models.Group, error) {
	group := new(models.Group)
	if err := r.db.NewSelect().Model(group).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, wrapNotFound("groupRepo.GetByID", err)
	}

	return group, nil
}
