package db

import (
	"context"
	"os"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/models"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
)

var (
	testDB            *bun.DB
	testCfg           config.DBConfig
	pgContainer       testcontainers.Container
	skipDBIntegration bool
)

func TestMain(m *testing.M) {
	skipDBIntegration = os.Getenv("SMCTF_SKIP_INTEGRATION") != ""
	if skipDBIntegration {
		os.Exit(m.Run())
	}

	ctx := context.Background()
	container, dbCfg, err := startPostgres(ctx)
	if err != nil {
		panic(err)
	}

	pgContainer = container
	testCfg = dbCfg

	testDB, err = New(dbCfg, "test")
	if err != nil {
		panic(err)
	}

	code := m.Run()

	if testDB != nil {
		_ = testDB.Close()
	}

	if pgContainer != nil {
		_ = pgContainer.Terminate(ctx)
	}

	os.Exit(code)
}

func startPostgres(ctx context.Context) (testcontainers.Container, config.DBConfig, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "smctf",
			"POSTGRES_PASSWORD": "smctf",
			"POSTGRES_DB":       "smctf_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, config.DBConfig{}, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, config.DBConfig{}, err
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, config.DBConfig{}, err
	}

	cfg := config.DBConfig{
		Host:            host,
		Port:            port.Int(),
		User:            "smctf",
		Password:        "smctf",
		Name:            "smctf_test",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    5,
		ConnMaxLifetime: 2 * time.Minute,
	}

	return container, cfg, nil
}

func setupDBTest(t *testing.T) *bun.DB {
	t.Helper()
	if skipDBIntegration {
		t.Skip("db tests disabled via SMCTF_SKIP_INTEGRATION")
	}

	if err := AutoMigrate(context.Background(), testDB); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return testDB
}

func TestNewAndAutoMigrate(t *testing.T) {
	db := setupDBTest(t)

	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	var tableCount int
	if err := db.NewSelect().Table("information_schema.tables").
		ColumnExpr("COUNT(*)").
		Where("table_schema = 'public'").
		Where("table_name IN ('users','challenges','submissions','registration_keys')").
		Scan(context.Background(), &tableCount); err != nil {
		t.Fatalf("query tables: %v", err)
	}

	if tableCount != 4 {
		t.Fatalf("expected 4 tables, got %d", tableCount)
	}
}

func TestEnsureColumnsAndIndexes(t *testing.T) {
	db := setupDBTest(t)

	if _, err := db.ExecContext(context.Background(), "ALTER TABLE challenges DROP COLUMN IF EXISTS category"); err != nil {
		t.Fatalf("drop category: %v", err)
	}

	if err := ensureChallengeCategory(context.Background(), db); err != nil {
		t.Fatalf("ensure category: %v", err)
	}

	if _, err := db.ExecContext(context.Background(), "ALTER TABLE registration_keys DROP COLUMN IF EXISTS used_by_ip"); err != nil {
		t.Fatalf("drop used_by_ip: %v", err)
	}

	if err := ensureRegistrationKeyIP(context.Background(), db); err != nil {
		t.Fatalf("ensure used_by_ip: %v", err)
	}

	if err := createIndexes(context.Background(), db); err != nil {
		t.Fatalf("create indexes: %v", err)
	}

	expected := []string{
		"idx_submissions_user",
		"idx_submissions_challenge",
		"idx_submissions_user_challenge",
		"idx_submissions_correct_time",
	}

	for _, name := range expected {
		var count int
		if err := db.NewSelect().Table("pg_indexes").
			ColumnExpr("COUNT(*)").
			Where("indexname = ?", name).
			Scan(context.Background(), &count); err != nil {
			t.Fatalf("query index %s: %v", name, err)
		}

		if count == 0 {
			t.Fatalf("expected index %s to exist", name)
		}
	}
}

func TestCreateTables(t *testing.T) {
	db := setupDBTest(t)
	tmp := []interface{}{
		(*models.User)(nil),
	}

	if err := createTables(context.Background(), db, tmp); err != nil {
		t.Fatalf("create tables: %v", err)
	}
}
