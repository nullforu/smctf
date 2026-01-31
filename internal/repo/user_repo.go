package repo

import (
	"context"
	"sort"
	"time"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type UserRepo struct {
	db *bun.DB
}

func NewUserRepo(db *bun.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	if _, err := r.db.NewInsert().Model(user).Exec(ctx); err != nil {
		return wrapError("userRepo.Create", err)
	}

	return nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := new(models.User)

	if err := r.db.NewSelect().Model(user).Where("email = ?", email).Scan(ctx); err != nil {
		return nil, wrapNotFound("userRepo.GetByEmail", err)
	}

	return user, nil
}

func (r *UserRepo) GetByEmailOrUsername(ctx context.Context, email, username string) (*models.User, error) {
	user := new(models.User)

	if err := r.db.NewSelect().Model(user).
		Where("email = ? OR username = ?", email, username).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("userRepo.GetByEmailOrUsername", err)
	}

	return user, nil
}

func (r *UserRepo) baseUserWithTeamQuery() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.*").
		ColumnExpr("COALESCE(g.name, 'not affiliated') AS team_name").
		Join("LEFT JOIN teams AS g ON g.id = u.team_id")
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	user := new(models.User)

	if err := r.baseUserWithTeamQuery().
		Model(user).
		Where("u.id = ?", id).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("userRepo.GetByID", err)
	}

	return user, nil
}

func (r *UserRepo) Leaderboard(ctx context.Context) ([]models.LeaderboardEntry, error) {
	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("userRepo.Leaderboard", err)
	}

	rows := make([]models.LeaderboardEntry, 0)
	if err := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		OrderExpr("u.id ASC").
		Scan(ctx, &rows); err != nil {
		return nil, wrapError("userRepo.Leaderboard", err)
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
		return nil, wrapError("userRepo.Leaderboard submissions", err)
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

func (r *UserRepo) TeamLeaderboard(ctx context.Context) ([]models.TeamLeaderboardEntry, error) {
	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("userRepo.TeamLeaderboard", err)
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
		return nil, wrapError("userRepo.TeamLeaderboard users", err)
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
		return nil, wrapError("userRepo.TeamLeaderboard teams", err)
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
		return nil, wrapError("userRepo.TeamLeaderboard submissions", err)
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

func (r *UserRepo) TimelineSubmissions(ctx context.Context, since *time.Time) ([]models.UserTimelineRow, error) {
	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("userRepo.TimelineSubmissions", err)
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
		return nil, wrapError("userRepo.TimelineSubmissions", err)
	}

	for i := range rows {
		rows[i].Points = pointsMap[rows[i].ChallengeID]
	}

	return rows, nil
}

func (r *UserRepo) TimelineTeamSubmissions(ctx context.Context, since *time.Time) ([]models.TeamTimelineRow, error) {
	pointsMap, err := dynamicPointsMap(ctx, r.db)
	if err != nil {
		return nil, wrapError("userRepo.TimelineTeamSubmissions", err)
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
		return nil, wrapError("userRepo.TimelineTeamSubmissions", err)
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

func (r *UserRepo) List(ctx context.Context) ([]models.User, error) {
	users := make([]models.User, 0)

	if err := r.baseUserWithTeamQuery().
		Model(&users).
		Distinct().
		OrderExpr("u.id ASC").
		Scan(ctx); err != nil {
		return nil, wrapError("userRepo.List", err)
	}

	return users, nil
}

func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	if _, err := r.db.NewUpdate().Model(user).WherePK().Exec(ctx); err != nil {
		return wrapError("userRepo.Update", err)
	}

	return nil
}
