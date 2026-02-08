package repo

import (
	"context"
	"sort"
	"time"

	"smctf/internal/models"
	"smctf/internal/scoring"

	"github.com/uptrace/bun"
)

type ScoreboardRepo struct {
	db *bun.DB
}

func NewScoreboardRepo(db *bun.DB) *ScoreboardRepo {
	return &ScoreboardRepo{db: db}
}

type leaderboardChallengeRow struct {
	ID            int64  `bun:"id"`
	Title         string `bun:"title"`
	Category      string `bun:"category"`
	Points        int    `bun:"points"`
	MinimumPoints int    `bun:"minimum_points"`
}

func (r *ScoreboardRepo) leaderboardChallenges(ctx context.Context) ([]models.LeaderboardChallenge, map[int64]int, error) {
	rows := make([]leaderboardChallengeRow, 0)
	if err := r.db.NewSelect().
		TableExpr("challenges AS c").
		ColumnExpr("c.id AS id").
		ColumnExpr("c.title AS title").
		ColumnExpr("c.category AS category").
		ColumnExpr("c.points AS points").
		ColumnExpr("c.minimum_points AS minimum_points").
		OrderExpr("c.id ASC").
		Scan(ctx, &rows); err != nil {
		return nil, nil, wrapError("scoreboardRepo.leaderboardChallenges", err)
	}

	solveCounts, err := solveCountsByChallenge(ctx, r.db)
	if err != nil {
		return nil, nil, wrapError("scoreboardRepo.leaderboardChallenges solve counts", err)
	}

	decay, err := decayFactor(ctx, r.db)
	if err != nil {
		return nil, nil, wrapError("scoreboardRepo.leaderboardChallenges decay", err)
	}

	pointsMap := make(map[int64]int, len(rows))
	challenges := make([]models.LeaderboardChallenge, 0, len(rows))

	for _, row := range rows {
		points := scoring.DynamicPoints(row.Points, row.MinimumPoints, solveCounts[row.ID], decay)
		pointsMap[row.ID] = points
		challenges = append(challenges, models.LeaderboardChallenge{
			ID:       row.ID,
			Title:    row.Title,
			Category: row.Category,
			Points:   points,
		})
	}

	return challenges, pointsMap, nil
}

func (r *ScoreboardRepo) Leaderboard(ctx context.Context) (models.LeaderboardResponse, error) {
	challenges, pointsMap, err := r.leaderboardChallenges(ctx)
	if err != nil {
		return models.LeaderboardResponse{}, wrapError("scoreboardRepo.Leaderboard", err)
	}

	rows := make([]models.LeaderboardEntry, 0)
	if err := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		OrderExpr("u.id ASC").
		Scan(ctx, &rows); err != nil {
		return models.LeaderboardResponse{}, wrapError("scoreboardRepo.Leaderboard", err)
	}

	scores := make(map[int64]int, len(rows))

	type submissionRow struct {
		UserID      int64 `bun:"user_id"`
		ChallengeID int64 `bun:"challenge_id"`
	}

	submissions := make([]submissionRow, 0)
	if err := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.user_id AS user_id").
		ColumnExpr("s.challenge_id AS challenge_id").
		Where("s.correct = true").
		Scan(ctx, &submissions); err != nil {
		return models.LeaderboardResponse{}, wrapError("scoreboardRepo.Leaderboard submissions", err)
	}

	for _, sub := range submissions {
		scores[sub.UserID] += pointsMap[sub.ChallengeID]
	}

	for i := range rows {
		rows[i].Score = scores[rows[i].UserID]
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Score == rows[j].Score {
			return rows[i].UserID < rows[j].UserID
		}
		return rows[i].Score > rows[j].Score
	})

	type solveRow struct {
		UserID       int64     `bun:"user_id"`
		ChallengeID  int64     `bun:"challenge_id"`
		SolvedAt     time.Time `bun:"solved_at"`
		IsFirstBlood bool      `bun:"is_first_blood"`
	}

	solvedRows := make([]solveRow, 0)
	if err := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.user_id AS user_id").
		ColumnExpr("s.challenge_id AS challenge_id").
		ColumnExpr("MIN(s.submitted_at) AS solved_at").
		ColumnExpr("BOOL_OR(s.is_first_blood) AS is_first_blood").
		Where("s.correct = true").
		GroupExpr("s.user_id, s.challenge_id").
		Scan(ctx, &solvedRows); err != nil {
		return models.LeaderboardResponse{}, wrapError("scoreboardRepo.Leaderboard solves", err)
	}

	solvedByUser := make(map[int64][]models.LeaderboardSolve)
	for _, row := range solvedRows {
		solvedByUser[row.UserID] = append(solvedByUser[row.UserID], models.LeaderboardSolve{
			ChallengeID:  row.ChallengeID,
			SolvedAt:     row.SolvedAt,
			IsFirstBlood: row.IsFirstBlood,
		})
	}

	for i := range rows {
		rows[i].Solves = solvedByUser[rows[i].UserID]
		if rows[i].Solves == nil {
			rows[i].Solves = []models.LeaderboardSolve{}
		}

		sort.Slice(rows[i].Solves, func(a, b int) bool {
			return rows[i].Solves[a].ChallengeID < rows[i].Solves[b].ChallengeID
		})
	}

	return models.LeaderboardResponse{
		Challenges: challenges,
		Entries:    rows,
	}, nil
}

func (r *ScoreboardRepo) TeamLeaderboard(ctx context.Context) (models.TeamLeaderboardResponse, error) {
	challenges, pointsMap, err := r.leaderboardChallenges(ctx)
	if err != nil {
		return models.TeamLeaderboardResponse{}, wrapError("scoreboardRepo.TeamLeaderboard", err)
	}

	var teamRows []struct {
		ID   int64  `bun:"id"`
		Name string `bun:"name"`
	}

	if err := r.db.NewSelect().
		TableExpr("teams AS t").
		ColumnExpr("t.id AS id").
		ColumnExpr("t.name AS name").
		Scan(ctx, &teamRows); err != nil {
		return models.TeamLeaderboardResponse{}, wrapError("scoreboardRepo.TeamLeaderboard teams", err)
	}

	teamEntries := make(map[int64]*models.TeamLeaderboardEntry, len(teamRows))
	for _, row := range teamRows {
		teamEntries[row.ID] = &models.TeamLeaderboardEntry{
			TeamID:   row.ID,
			TeamName: row.Name,
		}
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
		Scan(ctx, &submissions); err != nil {
		return models.TeamLeaderboardResponse{}, wrapError("scoreboardRepo.TeamLeaderboard submissions", err)
	}

	for _, sub := range submissions {
		entry, ok := teamEntries[sub.TeamID]
		if !ok {
			continue
		}

		entry.Score += pointsMap[sub.ChallengeID]
	}

	rows := make([]models.TeamLeaderboardEntry, 0, len(teamEntries))
	for _, entry := range teamEntries {
		rows = append(rows, *entry)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Score == rows[j].Score {
			return rows[i].TeamName < rows[j].TeamName
		}

		return rows[i].Score > rows[j].Score
	})

	type solveRow struct {
		TeamID       int64     `bun:"team_id"`
		ChallengeID  int64     `bun:"challenge_id"`
		SolvedAt     time.Time `bun:"solved_at"`
		IsFirstBlood bool      `bun:"is_first_blood"`
	}

	solvedRows := make([]solveRow, 0)
	if err := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("u.team_id AS team_id").
		ColumnExpr("s.challenge_id AS challenge_id").
		ColumnExpr("MIN(s.submitted_at) AS solved_at").
		ColumnExpr("BOOL_OR(s.is_first_blood) AS is_first_blood").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true").
		GroupExpr("u.team_id, s.challenge_id").
		Scan(ctx, &solvedRows); err != nil {
		return models.TeamLeaderboardResponse{}, wrapError("scoreboardRepo.TeamLeaderboard solves", err)
	}

	solvedByTeam := make(map[int64][]models.LeaderboardSolve)
	for _, row := range solvedRows {
		solvedByTeam[row.TeamID] = append(solvedByTeam[row.TeamID], models.LeaderboardSolve{
			ChallengeID:  row.ChallengeID,
			SolvedAt:     row.SolvedAt,
			IsFirstBlood: row.IsFirstBlood,
		})
	}

	for i := range rows {
		rows[i].Solves = solvedByTeam[rows[i].TeamID]
		if rows[i].Solves == nil {
			rows[i].Solves = []models.LeaderboardSolve{}
		}

		sort.Slice(rows[i].Solves, func(a, b int) bool {
			return rows[i].Solves[a].ChallengeID < rows[i].Solves[b].ChallengeID
		})
	}

	return models.TeamLeaderboardResponse{
		Challenges: challenges,
		Entries:    rows,
	}, nil
}

func (r *ScoreboardRepo) TimelineSubmissions(ctx context.Context, since *time.Time) ([]models.UserTimelineRow, error) {
	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("scoreboardRepo.TimelineSubmissions", err)
	}

	rows := make([]models.UserTimelineRow, 0)
	query := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.submitted_at AS submitted_at").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("s.challenge_id AS challenge_id").
		Join("JOIN users AS u ON u.id = s.user_id").
		Where("s.correct = true")

	query = applyTimelineWindow(query, since)

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("scoreboardRepo.TimelineSubmissions", err)
	}

	for i := range rows {
		rows[i].Points = pointsMap[rows[i].ChallengeID]
	}

	return rows, nil
}

func (r *ScoreboardRepo) TimelineTeamSubmissions(ctx context.Context, since *time.Time) ([]models.TeamTimelineRow, error) {
	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("scoreboardRepo.TimelineTeamSubmissions", err)
	}

	rows := make([]models.TeamTimelineRow, 0)
	query := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.submitted_at AS submitted_at").
		ColumnExpr("u.team_id AS team_id").
		ColumnExpr("g.name AS team_name").
		ColumnExpr("s.challenge_id AS challenge_id").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN teams AS g ON g.id = u.team_id").
		Where("s.correct = true")

	query = applyTimelineWindow(query, since)

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("scoreboardRepo.TimelineTeamSubmissions", err)
	}

	for i := range rows {
		rows[i].Points = pointsMap[rows[i].ChallengeID]
	}

	return rows, nil
}

func applyTimelineWindow(query *bun.SelectQuery, since *time.Time) *bun.SelectQuery {
	if since != nil {
		query = query.Where("s.submitted_at >= ?", *since)
	}

	return query.OrderExpr("s.submitted_at ASC, s.id ASC")
}
