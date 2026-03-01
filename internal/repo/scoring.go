package repo

import (
	"context"

	"smctf/internal/models"
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

type challengeSolveCountDivisionRow struct {
	DivisionID  int64 `bun:"division_id"`
	ChallengeID int64 `bun:"challenge_id"`
	SolveCount  int   `bun:"solve_count"`
}

type divisionCountRow struct {
	DivisionID int64 `bun:"division_id"`
	Count      int   `bun:"team_count"`
}

func dynamicPointsMap(ctx context.Context, db *bun.DB, divisionID *int64) (map[int64]int, error) {
	challenges, err := listChallengesForScoring(ctx, db)
	if err != nil {
		return nil, err
	}

	solveCounts, err := solveCountsByChallenge(ctx, db, divisionID)
	if err != nil {
		return nil, err
	}

	decay, err := decayFactor(ctx, db, divisionID)
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

func dynamicPointsMapForDivisions(ctx context.Context, db *bun.DB, divisionIDs []int64) (map[int64]map[int64]int, error) {
	if len(divisionIDs) == 0 {
		return map[int64]map[int64]int{}, nil
	}

	challenges, err := listChallengesForScoring(ctx, db)
	if err != nil {
		return nil, err
	}

	solveCounts, err := solveCountsByDivision(ctx, db, divisionIDs)
	if err != nil {
		return nil, err
	}

	decay, err := decayFactorByDivision(ctx, db, divisionIDs)
	if err != nil {
		return nil, err
	}

	pointsByDivision := make(map[int64]map[int64]int, len(divisionIDs))
	for _, divisionID := range divisionIDs {
		decayValue := decay[divisionID]
		counts := solveCounts[divisionID]
		points := make(map[int64]int, len(challenges))

		for _, ch := range challenges {
			solves := counts[ch.ID]
			points[ch.ID] = scoring.DynamicPoints(ch.Points, ch.MinimumPoints, solves, decayValue)
		}

		pointsByDivision[divisionID] = points
	}

	return pointsByDivision, nil
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

func solveCountsByChallenge(ctx context.Context, db *bun.DB, divisionID *int64) (map[int64]int, error) {
	rows := make([]challengeSolveCountRow, 0)
	query := db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.challenge_id").
		ColumnExpr("COUNT(*) AS solve_count").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN teams AS t ON t.id = u.team_id").
		Where("s.correct = true").
		Where("u.role NOT IN (?)", bun.In([]string{models.BlockedRole, models.AdminRole})).
		GroupExpr("challenge_id")

	if divisionID != nil {
		query = query.Where("t.division_id = ?", *divisionID)
	}

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("score.solveCountsByChallenge", err)
	}

	counts := make(map[int64]int, len(rows))
	for _, row := range rows {
		counts[row.ChallengeID] = row.SolveCount
	}

	return counts, nil
}

func solveCountsByDivision(ctx context.Context, db *bun.DB, divisionIDs []int64) (map[int64]map[int64]int, error) {
	rows := make([]challengeSolveCountDivisionRow, 0)
	query := db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("t.division_id AS division_id").
		ColumnExpr("s.challenge_id").
		ColumnExpr("COUNT(*) AS solve_count").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN teams AS t ON t.id = u.team_id").
		Where("s.correct = true").
		Where("u.role NOT IN (?)", bun.In([]string{models.BlockedRole, models.AdminRole})).
		GroupExpr("t.division_id, s.challenge_id")

	if len(divisionIDs) > 0 {
		query = query.Where("t.division_id IN (?)", bun.In(divisionIDs))
	}

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("score.solveCountsByDivision", err)
	}

	counts := make(map[int64]map[int64]int)
	for _, row := range rows {
		if _, exists := counts[row.DivisionID]; !exists {
			counts[row.DivisionID] = make(map[int64]int)
		}
		counts[row.DivisionID][row.ChallengeID] = row.SolveCount
	}

	return counts, nil
}

func challengeSolveCounts(ctx context.Context, db *bun.DB, divisionID *int64) (map[int64]int, error) {
	counts, err := solveCountsByChallenge(ctx, db, divisionID)
	if err != nil {
		return nil, err
	}

	return counts, nil
}

func decayFactor(ctx context.Context, db *bun.DB, divisionID *int64) (int, error) {
	var teamCount int
	query := db.NewSelect().
		TableExpr("teams AS t").
		ColumnExpr("COUNT(DISTINCT t.id)").
		Join("JOIN users AS u ON u.team_id = t.id").
		Where("u.role NOT IN (?)", bun.In([]string{models.BlockedRole, models.AdminRole}))

	if divisionID != nil {
		query = query.Where("t.division_id = ?", *divisionID)
	}

	if err := query.Scan(ctx, &teamCount); err != nil {
		return 0, wrapError("score.teamCount", err)
	}

	return teamCount, nil
}

func decayFactorByDivision(ctx context.Context, db *bun.DB, divisionIDs []int64) (map[int64]int, error) {
	rows := make([]divisionCountRow, 0)
	query := db.NewSelect().
		TableExpr("teams AS t").
		ColumnExpr("t.division_id AS division_id").
		ColumnExpr("COUNT(DISTINCT t.id) AS team_count").
		Join("JOIN users AS u ON u.team_id = t.id").
		Where("u.role NOT IN (?)", bun.In([]string{models.BlockedRole, models.AdminRole})).
		GroupExpr("t.division_id")

	if len(divisionIDs) > 0 {
		query = query.Where("t.division_id IN (?)", bun.In(divisionIDs))
	}

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("score.teamCountByDivision", err)
	}

	decay := make(map[int64]int, len(rows))
	for _, row := range rows {
		decay[row.DivisionID] = row.Count
	}

	return decay, nil
}
