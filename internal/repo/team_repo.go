package repo

import (
	"context"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type TeamRepo struct {
	db *bun.DB
}

func NewTeamRepo(db *bun.DB) *TeamRepo {
	return &TeamRepo{db: db}
}

func (r *TeamRepo) Create(ctx context.Context, team *models.Team) error {
	if _, err := r.db.NewInsert().Model(team).Exec(ctx); err != nil {
		return wrapError("teamRepo.Create", err)
	}

	return nil
}

func (r *TeamRepo) List(ctx context.Context) ([]models.Team, error) {
	teams := make([]models.Team, 0)
	if err := r.db.NewSelect().Model(&teams).OrderExpr("id ASC").Scan(ctx); err != nil {
		return nil, wrapError("teamRepo.List", err)
	}

	return teams, nil
}

func (r *TeamRepo) GetByID(ctx context.Context, id int64) (*models.Team, error) {
	team := new(models.Team)
	if err := r.db.NewSelect().Model(team).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, wrapNotFound("teamRepo.GetByID", err)
	}

	return team, nil
}

func (r *TeamRepo) ListWithStats(ctx context.Context) ([]models.TeamSummary, error) {
	rows := make([]models.TeamSummary, 0)
	query := r.db.NewSelect().
		TableExpr("teams AS t").
		ColumnExpr("t.id AS id").
		ColumnExpr("t.name AS name").
		ColumnExpr("t.created_at AS created_at").
		ColumnExpr("COUNT(DISTINCT u.id) AS member_count").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS total_score").
		Join("LEFT JOIN users AS u ON u.team_id = t.id").
		Join("LEFT JOIN submissions AS s ON s.user_id = u.id AND s.correct = true").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		GroupExpr("t.id, t.name, t.created_at").
		OrderExpr("t.name ASC, t.id ASC")
	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("teamRepo.ListWithStats", err)
	}

	return rows, nil
}

func (r *TeamRepo) GetStats(ctx context.Context, id int64) (*models.TeamSummary, error) {
	row := new(models.TeamSummary)
	query := r.db.NewSelect().
		TableExpr("teams AS t").
		ColumnExpr("t.id AS id").
		ColumnExpr("t.name AS name").
		ColumnExpr("t.created_at AS created_at").
		ColumnExpr("COUNT(DISTINCT u.id) AS member_count").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS total_score").
		Join("LEFT JOIN users AS u ON u.team_id = t.id").
		Join("LEFT JOIN submissions AS s ON s.user_id = u.id AND s.correct = true").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		Where("t.id = ?", id).
		GroupExpr("t.id, t.name, t.created_at")
	if err := query.Scan(ctx, row); err != nil {
		return nil, wrapNotFound("teamRepo.GetStats", err)
	}

	return row, nil
}

func (r *TeamRepo) ListMembers(ctx context.Context, id int64) ([]models.TeamMember, error) {
	rows := make([]models.TeamMember, 0)
	query := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.id AS id").
		ColumnExpr("u.username AS username").
		ColumnExpr("u.role AS role").
		Where("u.team_id = ?", id).
		OrderExpr("u.id ASC")

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("teamRepo.ListMembers", err)
	}

	return rows, nil
}

func (r *TeamRepo) ListSolvedChallenges(ctx context.Context, id int64) ([]models.TeamSolvedChallenge, error) {
	rows := make([]models.TeamSolvedChallenge, 0)
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
		Where("u.team_id = ?", id).
		GroupExpr("c.id, c.title, c.points").
		OrderExpr("last_solved_at DESC, c.id ASC")

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("teamRepo.ListSolvedChallenges", err)
	}

	return rows, nil
}
