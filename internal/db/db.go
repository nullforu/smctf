package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"smctf/internal/config"
	"smctf/internal/models"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

func New(cfg config.DBConfig, appEnv string) (*bun.DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))

	sqldb := sql.OpenDB(connector)
	sqldb.SetMaxOpenConns(cfg.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.MaxIdleConns)
	sqldb.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	db := bun.NewDB(sqldb, pgdialect.New())
	if appEnv != "production" {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(false)))
	}

	return db, nil
}

func AutoMigrate(ctx context.Context, db *bun.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	modelsToCreate := []any{
		(*models.AppConfig)(nil),
		(*models.Division)(nil),
		(*models.Team)(nil),
		(*models.User)(nil),
		(*models.Challenge)(nil),
		(*models.VM)(nil),
		(*models.Submission)(nil),
		(*models.RegistrationKey)(nil),
		(*models.RegistrationKeyUse)(nil),
		(*models.DiscordConnection)(nil),
	}

	if err := createTables(ctx, db, modelsToCreate); err != nil {
		return err
	}

	return createIndexes(ctx, db)
}

func createTables(ctx context.Context, db *bun.DB, modelsToCreate []any) error {
	for _, m := range modelsToCreate {
		if _, err := db.NewCreateTable().Model(m).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("auto migrate create table %T: %w", m, err)
		}
	}

	return nil
}

func createIndexes(ctx context.Context, db *bun.DB) error {
	indexes := []struct {
		name  string
		query string
	}{
		{
			name:  "idx_discord_connections_user",
			query: "CREATE UNIQUE INDEX IF NOT EXISTS idx_discord_connections_user ON discord_connections (user_id)",
		},
		{
			name:  "idx_discord_connections_discord_user",
			query: "CREATE UNIQUE INDEX IF NOT EXISTS idx_discord_connections_discord_user ON discord_connections (discord_user_id)",
		},
		{
			name:  "idx_submissions_user",
			query: "CREATE INDEX IF NOT EXISTS idx_submissions_user ON submissions (user_id)",
		},
		{
			name:  "idx_submissions_challenge",
			query: "CREATE INDEX IF NOT EXISTS idx_submissions_challenge ON submissions (challenge_id)",
		},
		{
			name:  "idx_submissions_user_challenge",
			query: "CREATE INDEX IF NOT EXISTS idx_submissions_user_challenge ON submissions (user_id, challenge_id)",
		},
		{
			name:  "idx_submissions_correct_time",
			query: "CREATE INDEX IF NOT EXISTS idx_submissions_correct_time ON submissions (correct, submitted_at) WHERE correct = true",
		},
		{
			name:  "idx_users_team_id",
			query: "CREATE INDEX IF NOT EXISTS idx_users_team_id ON users (team_id)",
		},
		{
			name:  "idx_teams_division_id",
			query: "CREATE INDEX IF NOT EXISTS idx_teams_division_id ON teams (division_id)",
		},
		{
			name:  "idx_registration_keys_team_id",
			query: "CREATE INDEX IF NOT EXISTS idx_registration_keys_team_id ON registration_keys (team_id)",
		},
		{
			name:  "idx_registration_key_uses_key_id",
			query: "CREATE INDEX IF NOT EXISTS idx_registration_key_uses_key_id ON registration_key_uses (registration_key_id)",
		},
		{
			name:  "idx_vms_user_id",
			query: "CREATE INDEX IF NOT EXISTS idx_vms_user_id ON vms (user_id)",
		},
		{
			name:  "idx_vms_user_challenge",
			query: "CREATE UNIQUE INDEX IF NOT EXISTS idx_vms_user_challenge ON vms (user_id, challenge_id)",
		},
		{
			name:  "idx_vms_vm_id",
			query: "CREATE UNIQUE INDEX IF NOT EXISTS idx_vms_vm_id ON vms (vm_id)",
		},
	}

	for _, idx := range indexes {
		if _, err := db.ExecContext(ctx, idx.query); err != nil {
			return fmt.Errorf("auto migrate create index %s: %w", idx.name, err)
		}
	}

	return nil
}
