package bootstrap

import (
	"context"
	"os"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/db"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
)

var (
	testDB      *bun.DB
	pgContainer testcontainers.Container
	skipDBTests bool
)

func TestMain(m *testing.M) {
	skipDBTests = os.Getenv("SMCTF_SKIP_INTEGRATION") != ""
	if skipDBTests {
		os.Exit(m.Run())
	}

	ctx := context.Background()
	container, dbCfg, err := startPostgres(ctx)
	if err != nil {
		panic(err)
	}

	pgContainer = container

	testDB, err = db.New(dbCfg, "test")
	if err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(ctx, testDB); err != nil {
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

func setupBootstrapDB(t *testing.T) *bun.DB {
	t.Helper()
	if skipDBTests {
		t.Skip("db tests disabled via SMCTF_SKIP_INTEGRATION")
	}

	if err := db.AutoMigrate(context.Background(), testDB); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	if _, err := testDB.ExecContext(context.Background(), "TRUNCATE TABLE submissions, registration_key_uses, registration_keys, challenges, users, teams, divisions RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	return testDB
}
