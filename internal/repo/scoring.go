package repo

import (
	"context"

	"smctf/internal/scoring"

	"github.com/uptrace/bun"
)

type challengeScoreRow struct {
	ID            int64 `bun:"id"`
	Points        int   `bun:"points"`
	MinimumPoints int   `bun:"minimum_points"`
}

type challengeSolveCountRow struct {
	ChallengeID int64 `bun:"challenge_id"`
	SolveCount  int   `bun:"solve_count"`
}

func dynamicPointsMap(ctx context.Context, db *bun.DB) (map[int64]int, error) {
	challenges, err := listChallengesForScoring(ctx, db)
	if err != nil {
		return nil, err
	}

	solveCounts, err := solveCountsByChallenge(ctx, db)
	if err != nil {
		return nil, err
	}

	decay, err := decayFactor(ctx, db)
	if err != nil {
		return nil, err
	}

	points := make(map[int64]int, len(challenges))
	for _, ch := range challenges {
		solves := solveCounts[ch.ID]
		points[ch.ID] = scoring.DynamicPoints(ch.Points, ch.MinimumPoints, solves, decay)
	}

	return points, nil
}

func listChallengesForScoring(ctx context.Context, db *bun.DB) ([]challengeScoreRow, error) {
	rows := make([]challengeScoreRow, 0)
	if err := db.NewSelect().
		TableExpr("challenges").
		ColumnExpr("id").
		ColumnExpr("points").
		ColumnExpr("minimum_points").
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("score.listChallenges", err)
	}

	return rows, nil
}

func solveCountsByChallenge(ctx context.Context, db *bun.DB) (map[int64]int, error) {
	rows := make([]challengeSolveCountRow, 0)
	if err := db.NewSelect().
		TableExpr("submissions").
		ColumnExpr("challenge_id").
		ColumnExpr("COUNT(*) AS solve_count").
		Where("correct = true").
		GroupExpr("challenge_id").
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("score.solveCountsByChallenge", err)
	}

	counts := make(map[int64]int, len(rows))
	for _, row := range rows {
		counts[row.ChallengeID] = row.SolveCount
	}

	return counts, nil
}

func challengeSolveCounts(ctx context.Context, db *bun.DB) (map[int64]int, error) {
	counts, err := solveCountsByChallenge(ctx, db)
	if err != nil {
		return nil, err
	}

	return counts, nil
}

func decayFactor(ctx context.Context, db *bun.DB) (int, error) {
	var teamCount int
	if err := db.NewSelect().
		TableExpr("teams").
		ColumnExpr("COUNT(*)").
		Scan(ctx, &teamCount); err != nil {
		return 0, wrapError("score.teamCount", err)
	}

	return teamCount, nil
}
