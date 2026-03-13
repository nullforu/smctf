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
		ColumnExpr("stack.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("LEFT JOIN users AS u ON u.id = stack.user_id").
		Join("LEFT JOIN challenges AS c ON c.id = stack.challenge_id").
		Where("stack.user_id = ?", userID).
		Order("stack.created_at DESC").
		Scan(ctx); err != nil {
		return nil, wrapError("stackRepo.ListByUser", err)
	}

	return stacks, nil
}

func (r *StackRepo) ListByTeam(ctx context.Context, teamID int64) ([]models.Stack, error) {
	stacks := make([]models.Stack, 0)
	if err := r.db.NewSelect().
		Model(&stacks).
		ColumnExpr("stack.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("JOIN users AS u ON u.id = stack.user_id").
		Join("JOIN challenges AS c ON c.id = stack.challenge_id").
		Where("u.team_id = ?", teamID).
		Order("stack.created_at DESC").
		Scan(ctx); err != nil {
		return nil, wrapError("stackRepo.ListByTeam", err)
	}

	return stacks, nil
}

func (r *StackRepo) ListAll(ctx context.Context) ([]models.Stack, error) {
	stacks := make([]models.Stack, 0)
	if err := r.db.NewSelect().
		Model(&stacks).
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return nil, wrapError("stackRepo.ListAll", err)
	}

	return stacks, nil
}

func (r *StackRepo) ListAdmin(ctx context.Context) ([]models.AdminStackSummary, error) {
	stacks := make([]models.AdminStackSummary, 0)
	if err := r.db.NewSelect().
		TableExpr("stacks AS s").
		ColumnExpr("s.stack_id AS stack_id").
		ColumnExpr("s.ttl_expires_at AS ttl_expires_at").
		ColumnExpr("s.created_at AS created_at").
		ColumnExpr("s.updated_at AS updated_at").
		ColumnExpr("s.user_id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("u.email AS email").
		ColumnExpr("u.team_id AS team_id").
		ColumnExpr("g.name AS team_name").
		ColumnExpr("s.challenge_id AS challenge_id").
		ColumnExpr("c.title AS challenge_title").
		ColumnExpr("c.category AS challenge_category").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN teams AS g ON g.id = u.team_id").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		OrderExpr("s.created_at DESC").
		Scan(ctx, &stacks); err != nil {
		return nil, wrapError("stackRepo.ListAdmin", err)
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

func (r *StackRepo) CountByUserExcludingStatuses(ctx context.Context, userID int64, statuses []string) (int, error) {
	query := r.db.NewSelect().
		Model((*models.Stack)(nil)).
		Where("user_id = ?", userID)
	if len(statuses) > 0 {
		query = query.Where("status NOT IN (?)", bun.In(statuses))
	}

	count, err := query.Count(ctx)
	if err != nil {
		return 0, wrapError("stackRepo.CountByUserExcludingStatuses", err)
	}

	return count, nil
}

func (r *StackRepo) CountByTeamExcludingStatuses(ctx context.Context, teamID int64, statuses []string) (int, error) {
	query := r.db.NewSelect().
		Model((*models.Stack)(nil)).
		Join("JOIN users AS u ON u.id = stack.user_id").
		Where("u.team_id = ?", teamID)
	if len(statuses) > 0 {
		query = query.Where("stack.status NOT IN (?)", bun.In(statuses))
	}

	count, err := query.Count(ctx)
	if err != nil {
		return 0, wrapError("stackRepo.CountByTeamExcludingStatuses", err)
	}

	return count, nil
}

func (r *StackRepo) TeamIDForUser(ctx context.Context, userID int64) (int64, error) {
	var teamID int64
	if err := r.db.NewSelect().
		TableExpr("users").
		Column("team_id").
		Where("id = ?", userID).
		Scan(ctx, &teamID); err != nil {
		return 0, wrapNotFound("stackRepo.TeamIDForUser", err)
	}

	return teamID, nil
}

func (r *StackRepo) GetByUserAndChallenge(ctx context.Context, userID, challengeID int64) (*models.Stack, error) {
	stack := new(models.Stack)
	if err := r.db.NewSelect().
		Model(stack).
		ColumnExpr("stack.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("LEFT JOIN users AS u ON u.id = stack.user_id").
		Join("LEFT JOIN challenges AS c ON c.id = stack.challenge_id").
		Where("stack.user_id = ?", userID).
		Where("stack.challenge_id = ?", challengeID).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("stackRepo.GetByUserAndChallenge", err)
	}

	return stack, nil
}

func (r *StackRepo) GetByTeamAndChallenge(ctx context.Context, teamID, challengeID int64) (*models.Stack, error) {
	stack := new(models.Stack)
	if err := r.db.NewSelect().
		Model(stack).
		ColumnExpr("stack.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("JOIN users AS u ON u.id = stack.user_id").
		Join("JOIN challenges AS c ON c.id = stack.challenge_id").
		Where("u.team_id = ?", teamID).
		Where("stack.challenge_id = ?", challengeID).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("stackRepo.GetByTeamAndChallenge", err)
	}

	return stack, nil
}

func (r *StackRepo) GetByStackID(ctx context.Context, stackID string) (*models.Stack, error) {
	stack := new(models.Stack)
	if err := r.db.NewSelect().
		Model(stack).
		ColumnExpr("stack.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("LEFT JOIN users AS u ON u.id = stack.user_id").
		Join("LEFT JOIN challenges AS c ON c.id = stack.challenge_id").
		Where("stack.stack_id = ?", stackID).
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
