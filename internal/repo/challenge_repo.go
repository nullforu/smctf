package repo

import (
	"context"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type ChallengeRepo struct {
	db *bun.DB
}

func NewChallengeRepo(db *bun.DB) *ChallengeRepo {
	return &ChallengeRepo{db: db}
}

func (r *ChallengeRepo) ListActive(ctx context.Context) ([]models.Challenge, error) {
	var challenges []models.Challenge
	err := r.db.NewSelect().Model(&challenges).Where("is_active = true").Order("id ASC").Scan(ctx)
	return challenges, err
}

func (r *ChallengeRepo) GetByID(ctx context.Context, id int64) (*models.Challenge, error) {
	ch := new(models.Challenge)
	err := r.db.NewSelect().Model(ch).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func (r *ChallengeRepo) Create(ctx context.Context, ch *models.Challenge) error {
	_, err := r.db.NewInsert().Model(ch).Exec(ctx)
	return err
}
