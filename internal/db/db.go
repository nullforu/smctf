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

	modelsToCreate := []interface{}{
		(*models.User)(nil),
		(*models.Challenge)(nil),
		(*models.Submission)(nil),
	}

	if err := createTables(ctx, db, modelsToCreate); err != nil {
		return err
	}

	return createIndexes(ctx, db)
}

func createTables(ctx context.Context, db *bun.DB, modelsToCreate []interface{}) error {
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
	}

	for _, idx := range indexes {
		if _, err := db.ExecContext(ctx, idx.query); err != nil {
			return fmt.Errorf("auto migrate create index %s: %w", idx.name, err)
		}
	}
	return nil
}
