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
	if err := r.db.NewSelect().Model(&challenges).Order("id ASC").Scan(ctx); err != nil {
		return nil, wrapError("challengeRepo.ListActive", err)
	}
	return challenges, nil
}

func (r *ChallengeRepo) GetByID(ctx context.Context, id int64) (*models.Challenge, error) {
	challenge := new(models.Challenge)
	if err := r.db.NewSelect().Model(challenge).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, wrapNotFound("challengeRepo.GetByID", err)
	}
	return challenge, nil
}

func (r *ChallengeRepo) Create(ctx context.Context, challenge *models.Challenge) error {
	if _, err := r.db.NewInsert().Model(challenge).Exec(ctx); err != nil {
		return wrapError("challengeRepo.Create", err)
	}
	return nil
}
