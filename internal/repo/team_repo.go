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

func (r *TeamRepo) baseTeamStatsQuery() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("teams AS t").
		ColumnExpr("t.id AS id").
		ColumnExpr("t.name AS name").
		ColumnExpr("t.created_at AS created_at").
		ColumnExpr("COUNT(DISTINCT u.id) AS member_count").
		Join("LEFT JOIN users AS u ON u.team_id = t.id").
		GroupExpr("t.id, t.name, t.created_at")
}

func (r *TeamRepo) ListWithStats(ctx context.Context) ([]models.TeamSummary, error) {
	rows := make([]models.TeamSummary, 0)
	query := r.baseTeamStatsQuery().OrderExpr("t.name ASC, t.id ASC")
	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("teamRepo.ListWithStats", err)
	}

	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("teamRepo.ListWithStats", err)
	}

	type submissionRow struct {
		TeamID      int64 `bun:"team_id"`
		ChallengeID int64 `bun:"challenge_id"`
	}

	submissions := make([]submissionRow, 0)
	if err := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("u.team_id AS team_id").
		ColumnExpr("s.challenge_id AS challenge_id").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true").
		Where("u.team_id IS NOT NULL").
		Scan(ctx, &submissions); err != nil {
		return nil, wrapError("teamRepo.ListWithStats submissions", err)
	}

	scores := make(map[int64]int, len(rows))
	for _, sub := range submissions {
		scores[sub.TeamID] += pointsMap[sub.ChallengeID]
	}

	for i := range rows {
		rows[i].TotalScore = scores[rows[i].ID]
	}

	return rows, nil
}

func (r *TeamRepo) GetStats(ctx context.Context, id int64) (*models.TeamSummary, error) {
	row := new(models.TeamSummary)
	query := r.baseTeamStatsQuery().Where("t.id = ?", id)
	if err := query.Scan(ctx, row); err != nil {
		return nil, wrapNotFound("teamRepo.GetStats", err)
	}

	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("teamRepo.GetStats", err)
	}

	var submissions []struct {
		ChallengeID int64 `bun:"challenge_id"`
	}
	if err := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.challenge_id AS challenge_id").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true").
		Where("u.team_id = ?", id).
		Scan(ctx, &submissions); err != nil {
		return nil, wrapError("teamRepo.GetStats submissions", err)
	}

	score := 0
	for _, sub := range submissions {
		score += pointsMap[sub.ChallengeID]
	}

	row.TotalScore = score

	return row, nil
}

func (r *TeamRepo) baseTeamSolvedQuery() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("c.id AS challenge_id").
		ColumnExpr("c.title AS title").
		ColumnExpr("c.points AS points").
		ColumnExpr("COUNT(*) AS solve_count").
		ColumnExpr("MAX(s.submitted_at) AS last_solved_at").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Where("s.correct = true")
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
	query := r.baseTeamSolvedQuery().
		Where("u.team_id = ?", id).
		GroupExpr("c.id, c.title, c.points").
		OrderExpr("last_solved_at DESC, c.id ASC")

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("teamRepo.ListSolvedChallenges", err)
	}

	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("teamRepo.ListSolvedChallenges", err)
	}

	for i := range rows {
		rows[i].Points = pointsMap[rows[i].ChallengeID]
	}

	return rows, nil
}
