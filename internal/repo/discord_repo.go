package repo

import (
	"context"

	"smctf/internal/models"

	"github.com/uptrace/bun"
)

type DiscordRepo struct {
	db *bun.DB
}

func NewDiscordRepo(db *bun.DB) *DiscordRepo {
	return &DiscordRepo{db: db}
}

func (r *DiscordRepo) GetByUserID(ctx context.Context, userID int64) (*models.DiscordConnection, error) {
	conn := new(models.DiscordConnection)
	if err := r.db.NewSelect().Model(conn).Where("user_id = ?", userID).Scan(ctx); err != nil {
		return nil, wrapNotFound("discordRepo.GetByUserID", err)
	}

	return conn, nil
}

func (r *DiscordRepo) GetByDiscordUserID(ctx context.Context, discordUserID string) (*models.DiscordConnection, error) {
	conn := new(models.DiscordConnection)
	if err := r.db.NewSelect().Model(conn).Where("discord_user_id = ?", discordUserID).Scan(ctx); err != nil {
		return nil, wrapNotFound("discordRepo.GetByDiscordUserID", err)
	}

	return conn, nil
}

func (r *DiscordRepo) Create(ctx context.Context, conn *models.DiscordConnection) error {
	if _, err := r.db.NewInsert().Model(conn).Exec(ctx); err != nil {
		return wrapError("discordRepo.Create", err)
	}

	return nil
}

func (r *DiscordRepo) Update(ctx context.Context, conn *models.DiscordConnection) error {
	if _, err := r.db.NewUpdate().Model(conn).WherePK().Exec(ctx); err != nil {
		return wrapError("discordRepo.Update", err)
	}

	return nil
}

func (r *DiscordRepo) Delete(ctx context.Context, conn *models.DiscordConnection) error {
	if _, err := r.db.NewDelete().Model(conn).WherePK().Exec(ctx); err != nil {
		return wrapError("discordRepo.Delete", err)
	}

	return nil
}
