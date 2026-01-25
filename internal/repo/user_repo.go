package repo

import (
	"context"
	"errors"
	"fmt"
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

	if err := r.db.NewSelect().Model(user).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, wrapNotFound("userRepo.GetByID", err)
	}

	return user, nil
}

func (r *UserRepo) Scoreboard(ctx context.Context, limit int) ([]models.ScoreEntry, error) {
	var rows []models.ScoreEntry

	q := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS score").
		Join("LEFT JOIN submissions AS s ON s.user_id = u.id AND s.correct = true").
		Join("LEFT JOIN challenges AS c ON c.id = s.challenge_id").
		GroupExpr("u.id, u.username").
		OrderExpr("score DESC, u.id ASC")

	if limit > 0 {
		q = q.Limit(limit)
	}

	if err := q.Scan(ctx, &rows); err != nil {
		return nil, wrapError("userRepo.Scoreboard", err)
	}

	return rows, nil
}

func (r *UserRepo) ScoreboardTimeline(ctx context.Context, userIDs []int64, interval time.Duration, since *time.Time) ([]models.ScoreTimelineRow, error) {
	if len(userIDs) == 0 {
		return []models.ScoreTimelineRow{}, nil
	}

	seconds := int(interval.Seconds())
	if seconds <= 0 {
		return nil, wrapError("userRepo.ScoreboardTimeline", errors.New("interval must be positive"))
	}
	intervalStr := fmt.Sprintf("%d seconds", seconds)

	var rows []models.ScoreTimelineRow
	query := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("date_bin(?::interval, s.submitted_at, '1970-01-01') AS bucket", intervalStr).
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("COALESCE(SUM(c.points), 0) AS score").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Where("s.correct = true").
		Where("s.user_id IN (?)", bun.In(userIDs))

	if since != nil {
		query = query.Where("s.submitted_at >= ?", *since)
	}

	err := query.
		GroupExpr("bucket, u.id, u.username").
		OrderExpr("bucket ASC, u.id ASC").
		Scan(ctx, &rows)

	if err != nil {
		return nil, wrapError("userRepo.ScoreboardTimeline", err)
	}

	return rows, nil
}

func (r *UserRepo) ScoreboardTimelineEvents(ctx context.Context, userIDs []int64, since *time.Time) ([]models.ScoreTimelineEvent, error) {
	if len(userIDs) == 0 {
		return []models.ScoreTimelineEvent{}, nil
	}

	var rows []models.ScoreTimelineEvent

	query := r.db.NewSelect().
		TableExpr("submissions AS s").
		ColumnExpr("s.submitted_at AS submitted_at").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.id AS challenge_id").
		ColumnExpr("c.title AS challenge_title").
		ColumnExpr("c.points AS points").
		Join("JOIN users AS u ON u.id = s.user_id").
		Join("JOIN challenges AS c ON c.id = s.challenge_id").
		Where("s.correct = true").
		Where("s.user_id IN (?)", bun.In(userIDs))

	if since != nil {
		query = query.Where("s.submitted_at >= ?", *since)
	}

	if err := query.OrderExpr("s.submitted_at ASC, s.id ASC").Scan(ctx, &rows); err != nil {
		return nil, wrapError("userRepo.ScoreboardTimelineEvents", err)
	}

	return rows, nil
}
