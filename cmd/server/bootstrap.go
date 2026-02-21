package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"smctf/internal/auth"
	"smctf/internal/config"
	"smctf/internal/db"
	"smctf/internal/logging"
	"smctf/internal/models"
	"smctf/internal/repo"

	"github.com/uptrace/bun"
)

const (
	bootstrapAdminTeamName = "Admin"
)

func bootstrapAdmin(ctx context.Context, cfg config.Config, database *bun.DB, userRepo *repo.UserRepo, teamRepo *repo.TeamRepo, logger *logging.Logger) {
	if !cfg.Bootstrap.AdminTeamEnabled && !cfg.Bootstrap.AdminUserEnabled {
		return
	}

	empty, err := isDatabaseEmpty(ctx, database)
	if err != nil {
		logger.Error("bootstrap database check error", slog.Any("error", err))
		return
	}

	if !empty {
		logger.Info("bootstrap skipped: database is not empty")
		return
	}

	var team *models.Team
	var err error

	if cfg.Bootstrap.AdminTeamEnabled {
		team, err = ensureAdminTeam(ctx, cfg, database, teamRepo)
		if err != nil {
			logger.Error("bootstrap admin team error", slog.Any("error", err))
			return
		}

		if team != nil {
			logger.Info("admin team created", slog.Any("team_id", team.ID), slog.Any("team_name", team.Name))
		}
	}

	if team != nil && cfg.Bootstrap.AdminUserEnabled {
		user, err := ensureAdminUser(ctx, cfg, team, userRepo)
		if err != nil {
			logger.Error("bootstrap admin user error", slog.Any("error", err))
			return
		}

		if user != nil {
			logger.Info("admin user created", slog.Any("user_id", user.ID))
		}
	}
}

func ensureAdminTeam(ctx context.Context, cfg config.Config, database *bun.DB, teamRepo *repo.TeamRepo) (*models.Team, error) {
	team := &models.Team{
		Name:      bootstrapAdminTeamName,
		CreatedAt: time.Now().UTC(),
	}

	if err := teamRepo.Create(ctx, team); err != nil {
		if db.IsUniqueViolation(err) {
			return ensureAdminTeam(ctx, cfg, database, teamRepo)
		}

		return nil, fmt.Errorf("create team: %w", err)
	}

	return team, nil
}

func ensureAdminUser(ctx context.Context, cfg config.Config, team *models.Team, userRepo *repo.UserRepo) (*models.User, error) {
	email := strings.TrimSpace(cfg.Bootstrap.AdminEmail)
	password := strings.TrimSpace(cfg.Bootstrap.AdminPassword)
	if email == "" || password == "" {
		return nil, nil
	}

	username := strings.TrimSpace(cfg.Bootstrap.AdminUsername)
	if username == "" {
		username = "admin"
	}

	hash, err := auth.HashPassword(password, cfg.PasswordBcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash admin password: %w", err)
	}

	now := time.Now().UTC()
	user := &models.User{
		Email:        email,
		Username:     username,
		PasswordHash: hash,
		Role:         models.AdminRole,
		TeamID:       team.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := userRepo.Create(ctx, user); err != nil {
		if db.IsUniqueViolation(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("create admin user: %w", err)
	}

	return user, nil
}

func isDatabaseEmpty(ctx context.Context, database *bun.DB) (bool, error) {
	tables := []string{"users", "teams", "registration_keys"}
	for _, table := range tables {
		count, err := database.NewSelect().TableExpr(table).Count(ctx)
		if err != nil {
			return false, fmt.Errorf("count %s: %w", table, err)
		}

		if count > 0 {
			return false, nil
		}
	}

	return true, nil
}
