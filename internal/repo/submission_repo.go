package repo

import (
	"context"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type SubmissionRepo struct {
	db *bun.DB
}

func NewSubmissionRepo(db *bun.DB) *SubmissionRepo {
	return &SubmissionRepo{db: db}
}

func (r *SubmissionRepo) Create(ctx context.Context, sub *models.Submission) error {
	if _, err := r.db.NewInsert().Model(sub).Exec(ctx); err != nil {
		return wrapError("submissionRepo.Create", err)
	}

	return nil
}

func (r *SubmissionRepo) lockTeamScope(ctx context.Context, db bun.IDB, userID int64) (int64, error) {
	var teamID int64
	if err := db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.team_id").
		Where("u.id = ?", userID).
		For("UPDATE").
		Scan(ctx, &teamID); err != nil {
		return teamID, err
	}

	if _, err := db.NewSelect().
		TableExpr("teams AS t").
		ColumnExpr("t.id").
		Where("t.id = ?", teamID).
		For("UPDATE").
		Exec(ctx); err != nil {
		return teamID, err
	}

	return teamID, nil
}

func (r *SubmissionRepo) correctSubmissionCount(
	ctx context.Context,
	db bun.IDB,
	challengeID int64,
	teamID int64,
) (int, error) {
	query := r.baseCorrectSubmissionsQuery(db).
		Where("s.challenge_id = ?", challengeID)

	query = query.Where("u.team_id = ?", teamID)

	return query.Count(ctx)
}

func (r *SubmissionRepo) baseCorrectSubmissionsQuery(db bun.IDB) *bun.SelectQuery {
	return db.NewSelect().
		TableExpr("submissions AS s").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true")
}

func (r *SubmissionRepo) solvedChallengesQuery(db bun.IDB) *bun.SelectQuery {
	return db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.challenge_id AS challenge_id").
		ColumnExpr("c.title AS title").
		ColumnExpr("c.points AS points").
		ColumnExpr("MIN(s.submitted_at) AS solved_at").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true")
}

func (r *SubmissionRepo) CreateCorrectIfNotSolvedByTeam(ctx context.Context, sub *models.Submission) (bool, error) {
	if !sub.Correct {
		if err := r.Create(ctx, sub); err != nil {
			return false, err
		}
		return true, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByTeam begin", err)
	}

	teamID, err := r.lockTeamScope(ctx, tx, sub.UserID)
	if err != nil {
		_ = tx.Rollback()
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByTeam lock user", err)
	}

	count, err := r.correctSubmissionCount(ctx, tx, sub.ChallengeID, teamID)
	if err != nil {
		_ = tx.Rollback()
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByTeam check", err)
	}

	if count > 0 {
		_ = tx.Rollback()
		return false, nil
	}

	if _, err := tx.NewInsert().Model(sub).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByTeam insert", err)
	}

	if err := tx.Commit(); err != nil {
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByTeam commit", err)
	}

	return true, nil
}

func (r *SubmissionRepo) HasCorrect(ctx context.Context, userID, challengeID int64) (bool, error) {
	count, err := r.baseCorrectSubmissionsQuery(r.db).
		Join("JOIN users AS me ON me.id = ?", userID).
		Where("s.challenge_id = ?", challengeID).
		Where("u.team_id = me.team_id").
		Count(ctx)

	if err != nil {
		return false, wrapError("submissionRepo.HasCorrect", err)
	}

	return count > 0, nil
}

func (r *SubmissionRepo) SolvedChallenges(ctx context.Context, userID int64) ([]models.SolvedChallenge, error) {
	rows := make([]models.SolvedChallenge, 0)

	err := r.solvedChallengesQuery(r.db).
		Where("u.id = ?", userID).
		GroupExpr("s.challenge_id, c.title, c.points").
		OrderExpr("solved_at ASC").
		Scan(ctx, &rows)

	if err != nil {
		return nil, wrapError("submissionRepo.SolvedChallenges", err)
	}

	return rows, nil
}
