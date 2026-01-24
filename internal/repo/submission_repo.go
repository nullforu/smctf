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
	_, err := r.db.NewInsert().Model(sub).Exec(ctx)
	return err
}

func (r *SubmissionRepo) HasCorrect(ctx context.Context, userID, challengeID int64) (bool, error) {
	count, err := r.db.NewSelect().Model((*models.Submission)(nil)).
		Where("user_id = ?", userID).
		Where("challenge_id = ?", challengeID).
		Where("correct = true").
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *SubmissionRepo) SolvedChallenges(ctx context.Context, userID int64) ([]models.SolvedChallenge, error) {
	var rows []models.SolvedChallenge
	err := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.challenge_id AS challenge_id").
		ColumnExpr("c.title AS title").
		ColumnExpr("c.points AS points").
		ColumnExpr("MIN(s.submitted_at) AS solved_at").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Where("s.user_id = ?", userID).
		Where("s.correct = true").
		GroupExpr("s.challenge_id, c.title, c.points").
		OrderExpr("solved_at ASC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	return rows, nil
}
