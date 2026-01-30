package repo

import (
	"context"
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
	rows := make([]models.LeaderboardEntry, 0)

	q := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS score").
		Join("LEFT JOIN submissions AS s ON s.user_id = u.id AND s.correct = true").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		GroupExpr("u.id, u.username").
		OrderExpr("score DESC, u.id ASC")

	if err := q.Scan(ctx, &rows); err != nil {
		return nil, wrapError("userRepo.Leaderboard", err)
	}

	return rows, nil
}

func (r *UserRepo) TeamLeaderboard(ctx context.Context) ([]models.TeamLeaderboardEntry, error) {
	rows := make([]models.TeamLeaderboardEntry, 0)

	q := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.team_id AS team_id").
		ColumnExpr("COALESCE(g.name, 'not affiliated') AS team_name").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS score").
		Join("LEFT JOIN teams AS g ON g.id = u.team_id").
		Join("LEFT JOIN submissions AS s ON s.user_id = u.id AND s.correct = true").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		GroupExpr("u.team_id, g.name").
		OrderExpr("score DESC, team_name ASC")

	if err := q.Scan(ctx, &rows); err != nil {
		return nil, wrapError("userRepo.TeamLeaderboard", err)
	}

	return rows, nil
}

func (r *UserRepo) TimelineSubmissions(ctx context.Context, since *time.Time) ([]models.UserTimelineRow, error) {
	rows := make([]models.UserTimelineRow, 0)

	query := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.submitted_at AS submitted_at").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.points AS points").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Where("s.correct = true")

	query = applyTimelineWindow(query, since)

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("userRepo.TimelineSubmissions", err)
	}

	return rows, nil
}

func (r *UserRepo) TimelineTeamSubmissions(ctx context.Context, since *time.Time) ([]models.TeamTimelineRow, error) {
	rows := make([]models.TeamTimelineRow, 0)

	query := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.submitted_at AS submitted_at").
		ColumnExpr("u.team_id AS team_id").
		ColumnExpr("COALESCE(g.name, 'not affiliated') AS team_name").
		ColumnExpr("c.points AS points").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("LEFT JOIN teams AS g ON g.id = u.team_id").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Where("s.correct = true")

	query = applyTimelineWindow(query, since)

	if err := query.Scan(ctx, &rows); err != nil {
		return nil, wrapError("userRepo.TimelineTeamSubmissions", err)
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
