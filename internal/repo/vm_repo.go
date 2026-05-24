package repo

import (
	"context"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type VMRepo struct {
	db *bun.DB
}

func NewVMRepo(db *bun.DB) *VMRepo {
	return &VMRepo{db: db}
}

func (r *VMRepo) ListByUser(ctx context.Context, userID int64) ([]models.VM, error) {
	vms := make([]models.VM, 0)
	if err := r.db.NewSelect().
		Model(&vms).
		ColumnExpr("vm.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("LEFT JOIN users AS u ON u.id = vm.user_id").
		Join("LEFT JOIN challenges AS c ON c.id = vm.challenge_id").
		Where("vm.user_id = ?", userID).
		Order("vm.created_at DESC").
		Scan(ctx); err != nil {
		return nil, wrapError("vmRepo.ListByUser", err)
	}

	return vms, nil
}

func (r *VMRepo) ListByTeamUser(ctx context.Context, userID int64) ([]models.VM, error) {
	vms := make([]models.VM, 0)
	if err := r.db.NewSelect().
		Model(&vms).
		ColumnExpr("vm.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("LEFT JOIN users AS u ON u.id = vm.user_id").
		Join("LEFT JOIN challenges AS c ON c.id = vm.challenge_id").
		Where("vm.user_id IN (SELECT u2.id FROM users AS u2 JOIN users AS me ON me.id = ? WHERE u2.team_id = me.team_id)", userID).
		Order("vm.created_at DESC").
		Scan(ctx); err != nil {
		return nil, wrapError("vmRepo.ListByTeamUser", err)
	}

	return vms, nil
}

func (r *VMRepo) ListAdmin(ctx context.Context) ([]models.AdminVMSummary, error) {
	vms := make([]models.AdminVMSummary, 0)
	if err := r.db.NewSelect().
		TableExpr("vms AS v").
		ColumnExpr("v.vm_id AS vm_id").
		ColumnExpr("v.ttl_expires_at AS ttl_expires_at").
		ColumnExpr("v.created_at AS created_at").
		ColumnExpr("v.updated_at AS updated_at").
		ColumnExpr("v.user_id AS user_id").
		ColumnExpr("u.username AS username").
		ColumnExpr("u.email AS email").
		ColumnExpr("u.team_id AS team_id").
		ColumnExpr("t.name AS team_name").
		ColumnExpr("v.challenge_id AS challenge_id").
		ColumnExpr("c.title AS challenge_title").
		ColumnExpr("c.category AS challenge_category").
		Join("JOIN users AS u ON u.id = v.user_id").
		Join("JOIN teams AS t ON t.id = u.team_id").
		Join("JOIN challenges AS c ON c.id = v.challenge_id").
		OrderExpr("v.created_at DESC").
		Scan(ctx, &vms); err != nil {
		return nil, wrapError("vmRepo.ListAdmin", err)
	}

	return vms, nil
}

func (r *VMRepo) ListAll(ctx context.Context) ([]models.VM, error) {
	vms := make([]models.VM, 0)
	if err := r.db.NewSelect().
		Model(&vms).
		ColumnExpr("vm.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("LEFT JOIN users AS u ON u.id = vm.user_id").
		Join("LEFT JOIN challenges AS c ON c.id = vm.challenge_id").
		Order("vm.created_at DESC").
		Scan(ctx); err != nil {
		return nil, wrapError("vmRepo.ListAll", err)
	}

	return vms, nil
}

func (r *VMRepo) CountByUser(ctx context.Context, userID int64) (int, error) {
	count, err := r.db.NewSelect().Model((*models.VM)(nil)).Where("user_id = ?", userID).Count(ctx)
	if err != nil {
		return 0, wrapError("vmRepo.CountByUser", err)
	}

	return count, nil
}

func (r *VMRepo) CountByTeamUser(ctx context.Context, userID int64) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.VM)(nil)).
		Where("user_id IN (SELECT u2.id FROM users AS u2 JOIN users AS me ON me.id = ? WHERE u2.team_id = me.team_id)", userID).
		Count(ctx)
	if err != nil {
		return 0, wrapError("vmRepo.CountByTeamUser", err)
	}

	return count, nil
}

func (r *VMRepo) GetByUserAndChallenge(ctx context.Context, userID, challengeID int64) (*models.VM, error) {
	vm := new(models.VM)
	if err := r.db.NewSelect().
		Model(vm).
		ColumnExpr("vm.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("LEFT JOIN users AS u ON u.id = vm.user_id").
		Join("LEFT JOIN challenges AS c ON c.id = vm.challenge_id").
		Where("vm.user_id = ?", userID).
		Where("vm.challenge_id = ?", challengeID).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("vmRepo.GetByUserAndChallenge", err)
	}

	return vm, nil
}

func (r *VMRepo) GetByTeamUserAndChallenge(ctx context.Context, userID, challengeID int64) (*models.VM, error) {
	vm := new(models.VM)
	if err := r.db.NewSelect().
		Model(vm).
		ColumnExpr("vm.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("LEFT JOIN users AS u ON u.id = vm.user_id").
		Join("LEFT JOIN challenges AS c ON c.id = vm.challenge_id").
		Where("vm.challenge_id = ?", challengeID).
		Where("vm.user_id IN (SELECT u2.id FROM users AS u2 JOIN users AS me ON me.id = ? WHERE u2.team_id = me.team_id)", userID).
		OrderExpr("vm.created_at DESC").
		Limit(1).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("vmRepo.GetByTeamUserAndChallenge", err)
	}

	return vm, nil
}

func (r *VMRepo) GetByVMID(ctx context.Context, vmID string) (*models.VM, error) {
	vm := new(models.VM)
	if err := r.db.NewSelect().
		Model(vm).
		ColumnExpr("vm.*").
		ColumnExpr("u.username AS username").
		ColumnExpr("c.title AS challenge_title").
		Join("LEFT JOIN users AS u ON u.id = vm.user_id").
		Join("LEFT JOIN challenges AS c ON c.id = vm.challenge_id").
		Where("vm.vm_id = ?", vmID).
		Scan(ctx); err != nil {
		return nil, wrapNotFound("vmRepo.GetByVMID", err)
	}

	return vm, nil
}

func (r *VMRepo) Create(ctx context.Context, vm *models.VM) error {
	if _, err := r.db.NewInsert().Model(vm).Exec(ctx); err != nil {
		return wrapError("vmRepo.Create", err)
	}

	return nil
}

func (r *VMRepo) Update(ctx context.Context, vm *models.VM) error {
	if _, err := r.db.NewUpdate().Model(vm).WherePK().Exec(ctx); err != nil {
		return wrapError("vmRepo.Update", err)
	}

	return nil
}

func (r *VMRepo) Delete(ctx context.Context, vm *models.VM) error {
	if _, err := r.db.NewDelete().Model(vm).WherePK().Exec(ctx); err != nil {
		return wrapError("vmRepo.Delete", err)
	}

	return nil
}
