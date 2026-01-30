package repo

import (
	"context"
	"database/sql"

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

	var teamID sql.NullInt64
	if err := tx.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.team_id").
		Where("u.id = ?", sub.UserID).
		For("UPDATE").
		Scan(ctx, &teamID); err != nil {
		_ = tx.Rollback()
		return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByTeam lock user", err)
	}

	if teamID.Valid {
		if _, err := tx.NewSelect().
			TableExpr("teams AS t").
			ColumnExpr("t.id").
			Where("t.id = ?", teamID.Int64).
			For("UPDATE").
			Exec(ctx); err != nil {
			_ = tx.Rollback()
			return false, wrapError("submissionRepo.CreateCorrectIfNotSolvedByTeam lock team", err)
		}
	}

	query := tx.NewSelect().
		TableExpr("submissions AS s").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.challenge_id = ?", sub.ChallengeID).
		Where("s.correct = true")
	if teamID.Valid {
		query = query.Where("u.team_id = ?", teamID.Int64)
	} else {
		query = query.Where("u.id = ?", sub.UserID)
	}

	count, err := query.Count(ctx)
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
	count, err := r.db.NewSelect().
		TableExpr("submissions AS s").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN users AS me ON me.id = ?", userID).
		Where("s.challenge_id = ?", challengeID).
		Where("s.correct = true").
		Where("(me.team_id IS NULL AND u.id = me.id) OR (me.team_id IS NOT NULL AND u.team_id = me.team_id)").
		Count(ctx)

	if err != nil {
		return false, wrapError("submissionRepo.HasCorrect", err)
	}

	return count > 0, nil
}

func (r *SubmissionRepo) SolvedChallenges(ctx context.Context, userID int64) ([]models.SolvedChallenge, error) {
	rows := make([]models.SolvedChallenge, 0)

	err := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.challenge_id AS challenge_id").
		ColumnExpr("c.title AS title").
		ColumnExpr("c.points AS points").
		ColumnExpr("MIN(s.submitted_at) AS solved_at").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true").
		Where("u.id = ?", userID).
		GroupExpr("s.challenge_id, c.title, c.points").
		OrderExpr("solved_at ASC").
		Scan(ctx, &rows)

	if err != nil {
		return nil, wrapError("submissionRepo.SolvedChallenges", err)
	}

	return rows, nil
}

func (r *SubmissionRepo) SolvedChallengesTeam(ctx context.Context, userID int64) ([]models.SolvedChallenge, error) {
	rows := make([]models.SolvedChallenge, 0)

	err := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.challenge_id AS challenge_id").
		ColumnExpr("c.title AS title").
		ColumnExpr("c.points AS points").
		ColumnExpr("MIN(s.submitted_at) AS solved_at").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN users AS me ON me.id = ?", userID).
		Where("s.correct = true").
		Where("(me.team_id IS NULL AND u.id = me.id) OR (me.team_id IS NOT NULL AND u.team_id = me.team_id)").
		GroupExpr("s.challenge_id, c.title, c.points").
		OrderExpr("solved_at ASC").
		Scan(ctx, &rows)

	if err != nil {
		return nil, wrapError("submissionRepo.SolvedChallengesTeam", err)
	}

	return rows, nil
}
