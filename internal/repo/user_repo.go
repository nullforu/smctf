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

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	user := new(models.User)

	if err := r.db.NewSelect().
		Model(user).
		TableExpr("users AS u").
		ColumnExpr("u.*").
		ColumnExpr("COALESCE(g.name, '무소속') AS group_name").
		Join("LEFT JOIN groups AS g ON g.id = u.group_id").
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

func (r *UserRepo) GroupLeaderboard(ctx context.Context) ([]models.GroupLeaderboardEntry, error) {
	rows := make([]models.GroupLeaderboardEntry, 0)

	q := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.group_id AS group_id").
		ColumnExpr("COALESCE(g.name, '무소속') AS group_name").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS score").
		Join("LEFT JOIN groups AS g ON g.id = u.group_id").
		Join("LEFT JOIN submissions AS s ON s.user_id = u.id AND s.correct = true").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		GroupExpr("u.group_id, g.name").
		OrderExpr("score DESC, group_name ASC")

	if err := q.Scan(ctx, &rows); err != nil {
		return nil, wrapError("userRepo.GroupLeaderboard", err)
	}

	return rows, nil
}

type RawSubmission struct {
	SubmittedAt time.Time `bun:"submitted_at"`
	UserID      int64     `bun:"user_id"`
	Username    string    `bun:"username"`
	Points      int       `bun:"points"`
}

func (r *UserRepo) TimelineSubmissions(ctx context.Context, since *time.Time) ([]RawSubmission, error) {
	rows := make([]RawSubmission, 0)

	query := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.submitted_at AS submitted_at").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.points AS points").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Where("s.correct = true")

	if since != nil {
		query = query.Where("s.submitted_at >= ?", *since)
	}

	if err := query.OrderExpr("s.submitted_at ASC, s.id ASC").Scan(ctx, &rows); err != nil {
		return nil, wrapError("userRepo.TimelineSubmissions", err)
	}

	return rows, nil
}

type RawGroupSubmission struct {
	SubmittedAt time.Time `bun:"submitted_at"`
	GroupID     *int64    `bun:"group_id"`
	GroupName   string    `bun:"group_name"`
	Points      int       `bun:"points"`
}

func (r *UserRepo) TimelineGroupSubmissions(ctx context.Context, since *time.Time) ([]RawGroupSubmission, error) {
	rows := make([]RawGroupSubmission, 0)

	query := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.submitted_at AS submitted_at").
		ColumnExpr("u.group_id AS group_id").
		ColumnExpr("COALESCE(g.name, '무소속') AS group_name").
		ColumnExpr("c.points AS points").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("LEFT JOIN groups AS g ON g.id = u.group_id").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Where("s.correct = true")

	if since != nil {
		query = query.Where("s.submitted_at >= ?", *since)
	}

	if err := query.OrderExpr("s.submitted_at ASC, s.id ASC").Scan(ctx, &rows); err != nil {
		return nil, wrapError("userRepo.TimelineGroupSubmissions", err)
	}

	return rows, nil
}

func (r *UserRepo) List(ctx context.Context) ([]models.User, error) {
	users := make([]models.User, 0)

	if err := r.db.NewSelect().
		Model(&users).
		TableExpr("users AS u").
		ColumnExpr("u.*").
		ColumnExpr("COALESCE(g.name, '무소속') AS group_name").
		Join("LEFT JOIN groups AS g ON g.id = u.group_id").
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
