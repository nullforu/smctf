package repo

import (
	"context"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type StackRepo struct {
	db *bun.DB
}

func NewStackRepo(db *bun.DB) *StackRepo {
	return &StackRepo{db: db}
}

func (r *StackRepo) ListByUser(ctx context.Context, userID int64) ([]models.Stack, error) {
	stacks := make([]models.Stack, 0)
	if err := r.db.NewSelect().
		Model(&stacks).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return nil, wrapError("stackRepo.ListByUser", err)
	}

	return stacks, nil
}

func (r *StackRepo) CountByUser(ctx context.Context, userID int64) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.Stack)(nil)).
		Where("user_id = ?", userID).
		Count(ctx)
	if err != nil {
		return 0, wrapError("stackRepo.CountByUser", err)
	}

	return count, nil
}

func (r *StackRepo) GetByUserAndChallenge(ctx context.Context, userID, challengeID int64) (*models.Stack, error) {
	stack := new(models.Stack)
	if err := r.db.NewSelect().
		Model(stack).
		Where("user_id = ?", userID).
		Where("challenge_id = ?", challengeID).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("stackRepo.GetByUserAndChallenge", err)
	}

	return stack, nil
}

func (r *StackRepo) GetByStackID(ctx context.Context, stackID string) (*models.Stack, error) {
	stack := new(models.Stack)
	if err := r.db.NewSelect().
		Model(stack).
		Where("stack_id = ?", stackID).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("stackRepo.GetByStackID", err)
	}

	return stack, nil
}

func (r *StackRepo) Create(ctx context.Context, stack *models.Stack) error {
	if _, err := r.db.NewInsert().Model(stack).Exec(ctx); err != nil {
		return wrapError("stackRepo.Create", err)
	}

	return nil
}

func (r *StackRepo) Update(ctx context.Context, stack *models.Stack) error {
	if _, err := r.db.NewUpdate().Model(stack).WherePK().Exec(ctx); err != nil {
		return wrapError("stackRepo.Update", err)
	}

	return nil
}

func (r *StackRepo) Delete(ctx context.Context, stack *models.Stack) error {
	if _, err := r.db.NewDelete().Model(stack).WherePK().Exec(ctx); err != nil {
		return wrapError("stackRepo.Delete", err)
	}

	return nil
}

func (r *StackRepo) DeleteByUserAndChallenge(ctx context.Context, userID, challengeID int64) error {
	if _, err := r.db.NewDelete().
		Model((*models.Stack)(nil)).
		Where("user_id = ?", userID).
		Where("challenge_id = ?", challengeID).
		Exec(ctx); err != nil {
		return wrapError("stackRepo.DeleteByUserAndChallenge", err)
	}

	return nil
}
