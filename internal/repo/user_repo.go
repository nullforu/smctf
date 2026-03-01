package repo

import (
	"context"

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

	if err := r.baseUserWithTeamQuery().
		Model(user).
		Where("u.email = ?", email).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("userRepo.GetByEmail", err)
	}

	return user, nil
}

func (r *UserRepo) GetByEmailOrUsername(ctx context.Context, email, username string) (*models.User, error) {
	user := new(models.User)

	if err := r.baseUserWithTeamQuery().
		Model(user).
		Where("u.email = ? OR u.username = ?", email, username).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("userRepo.GetByEmailOrUsername", err)
	}

	return user, nil
}

func (r *UserRepo) baseUserWithTeamQuery() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("u.*").
		ColumnExpr("g.name AS team_name").
		ColumnExpr("g.division_id AS division_id").
		ColumnExpr("d.name AS division_name").
		Join("JOIN teams AS g ON g.id = u.team_id").
		Join("JOIN divisions AS d ON d.id = g.division_id")
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

func (r *UserRepo) List(ctx context.Context, divisionID *int64) ([]models.User, error) {
	users := make([]models.User, 0)

	query := r.baseUserWithTeamQuery()
	if divisionID != nil {
		query = query.Where("g.division_id = ?", *divisionID)
	}

	if err := query.
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

func (r *UserRepo) GetDivisionID(ctx context.Context, userID int64) (int64, error) {
	var divisionID int64
	if err := r.db.NewSelect().
		TableExpr("users AS u").
		ColumnExpr("t.division_id").
		Join("JOIN teams AS t ON t.id = u.team_id").
		Where("u.id = ?", userID).
		Scan(ctx, &divisionID); err != nil {
		return 0, wrapNotFound("userRepo.GetDivisionID", err)
	}

	return divisionID, nil
}
