package repo

import (
	"context"
	"sort"
	"time"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type ScoreboardRepo struct {
	db *bun.DB
}

func NewScoreboardRepo(db *bun.DB) *ScoreboardRepo {
	return &ScoreboardRepo{db: db}
}

func (r *ScoreboardRepo) Leaderboard(ctx context.Context) ([]models.LeaderboardEntry, error) {
	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("scoreboardRepo.Leaderboard", err)
	}

	rows := make([]models.LeaderboardEntry, 0)
	if err := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		OrderExpr("u.id ASC").
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("scoreboardRepo.Leaderboard", err)
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
		return nil, wrapError("scoreboardRepo.Leaderboard submissions", err)
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

	return rows, nil
}

func (r *ScoreboardRepo) TeamLeaderboard(ctx context.Context) ([]models.TeamLeaderboardEntry, error) {
	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("scoreboardRepo.TeamLeaderboard", err)
	}

	type userRow struct {
		UserID int64  `bun:"user_id"`
		TeamID *int64 `bun:"team_id"`
	}

	users := make([]userRow, 0)
	if err := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.team_id AS team_id").
		OrderExpr("u.id ASC").
		Scan(ctx, &users); err != nil {
		return nil, wrapError("scoreboardRepo.TeamLeaderboard users", err)
	}

	teamNames := make(map[int64]string)

	var teamRows []struct {
		ID   int64  `bun:"id"`
		Name string `bun:"name"`
	}

	if err := r.db.NewSelect().
		TableExpr("teams AS t").
		ColumnExpr("t.id AS id").
		ColumnExpr("t.name AS name").
		Scan(ctx, &teamRows); err != nil {
		return nil, wrapError("scoreboardRepo.TeamLeaderboard teams", err)
	}

	for _, row := range teamRows {
		teamNames[row.ID] = row.Name
	}

	type teamKey struct {
		hasTeam bool
		teamID  int64
	}

	userTeams := make(map[int64]teamKey, len(users))
	teamEntries := make(map[teamKey]*models.TeamLeaderboardEntry)
	for _, user := range users {
		var key teamKey
		if user.TeamID != nil {
			key = teamKey{hasTeam: true, teamID: *user.TeamID}
		} else {
			key = teamKey{}
		}

		userTeams[user.UserID] = key
		if _, ok := teamEntries[key]; !ok {
			entry := &models.TeamLeaderboardEntry{}
			if key.hasTeam {
				id := key.teamID
				entry.TeamID = &id
				entry.TeamName = teamNames[key.teamID]
				if entry.TeamName == "" {
					entry.TeamName = "unknown team"
				}
			} else {
				entry.TeamName = "not affiliated"
			}
			teamEntries[key] = entry
		}
	}

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
		return nil, wrapError("scoreboardRepo.TeamLeaderboard submissions", err)
	}

	for _, sub := range submissions {
		key, ok := userTeams[sub.UserID]
		if !ok {
			continue
		}

		entry := teamEntries[key]
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

	return rows, nil
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
		ColumnExpr("COALESCE(g.name, 'not affiliated') AS team_name").
		ColumnExpr("s.challenge_id AS challenge_id").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("LEFT JOIN teams AS g ON g.id = u.team_id").
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
