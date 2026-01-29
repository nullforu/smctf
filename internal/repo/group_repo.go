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

func (r *GroupRepo) ListWithStats(ctx context.Context) ([]models.GroupSummary, error) {
	rows := make([]models.GroupSummary, 0)
	query := r.db.NewSelect().
		TableExpr("groups AS g").
		ColumnExpr("g.id AS id").
		ColumnExpr("g.name AS name").
		ColumnExpr("g.created_at AS created_at").
		ColumnExpr("COUNT(DISTINCT u.id) AS member_count").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS total_score").
		Join("LEFT JOIN users AS u ON u.group_id = g.id").
		Join("LEFT JOIN submissions AS s ON s.user_id = u.id AND s.correct = true").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		GroupExpr("g.id, g.name, g.created_at").
		OrderExpr("g.name ASC, g.id ASC")

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("groupRepo.ListWithStats", err)
	}

	return rows, nil
}

func (r *GroupRepo) GetStats(ctx context.Context, id int64) (*models.GroupSummary, error) {
	row := new(models.GroupSummary)
	query := r.db.NewSelect().
		TableExpr("groups AS g").
		ColumnExpr("g.id AS id").
		ColumnExpr("g.name AS name").
		ColumnExpr("g.created_at AS created_at").
		ColumnExpr("COUNT(DISTINCT u.id) AS member_count").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS total_score").
		Join("LEFT JOIN users AS u ON u.group_id = g.id").
		Join("LEFT JOIN submissions AS s ON s.user_id = u.id AND s.correct = true").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		Where("g.id = ?", id).
		GroupExpr("g.id, g.name, g.created_at")

	if err := query.Scan(ctx, row); err != nil {
		return nil, wrapNotFound("groupRepo.GetStats", err)
	}

	return row, nil
}

func (r *GroupRepo) ListMembers(ctx context.Context, id int64) ([]models.GroupMember, error) {
	rows := make([]models.GroupMember, 0)
	query := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.id AS id").
		ColumnExpr("u.username AS username").
		ColumnExpr("u.role AS role").
		Where("u.group_id = ?", id).
		OrderExpr("u.id ASC")

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("groupRepo.ListMembers", err)
	}

	return rows, nil
}

func (r *GroupRepo) ListSolvedChallenges(ctx context.Context, id int64) ([]models.GroupSolvedChallenge, error) {
	rows := make([]models.GroupSolvedChallenge, 0)
	query := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("c.id AS challenge_id").
		ColumnExpr("c.title AS title").
		ColumnExpr("c.points AS points").
		ColumnExpr("COUNT(*) AS solve_count").
		ColumnExpr("MAX(s.submitted_at) AS last_solved_at").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Where("s.correct = true").
		Where("u.group_id = ?", id).
		GroupExpr("c.id, c.title, c.points").
		OrderExpr("last_solved_at DESC, c.id ASC")

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("groupRepo.ListSolvedChallenges", err)
	}

	return rows, nil
}
