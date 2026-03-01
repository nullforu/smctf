package bootstrap

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/logging"
	"smctf/internal/models"
	"smctf/internal/repo"

	"golang.org/x/crypto/bcrypt"
)

func newTestLogger(t *testing.T) *logging.Logger {
	t.Helper()
	dir := t.TempDir()
	logger, err := logging.New(config.LoggingConfig{
		Dir:          filepath.Join(dir, "logs"),
		FilePrefix:   "test",
		MaxBodyBytes: 1024,
	}, logging.Options{
		Service: "bootstrap-test",
		Env:     "test",
	})

	if err != nil {
		t.Fatalf("logger init: %v", err)
	}

	t.Cleanup(func() {
		_ = logger.Close()
	})

	return logger
}

func baseBootstrapConfig() config.Config {
	return config.Config{
		AppEnv:     "test",
		BcryptCost: bcrypt.MinCost,
		Bootstrap: config.BootstrapConfig{
			AdminTeamEnabled: true,
			AdminUserEnabled: true,
			AdminEmail:       "admin@example.com",
			AdminPassword:    "adminpass",
			AdminUsername:    "admin",
		},
	}
}

func TestIsDatabaseEmpty(t *testing.T) {
	db := setupBootstrapDB(t)

	empty, err := isDatabaseEmpty(context.Background(), db)
	if err != nil {
		t.Fatalf("isDatabaseEmpty: %v", err)
	}

	if !empty {
		t.Fatalf("expected empty database")
	}

	teamRepo := repo.NewTeamRepo(db)
	team := &models.Team{
		Name:      "Temp",
		CreatedAt: time.Now().UTC(),
	}

	if err := teamRepo.Create(context.Background(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	empty, err = isDatabaseEmpty(context.Background(), db)
	if err != nil {
		t.Fatalf("isDatabaseEmpty after insert: %v", err)
	}

	if empty {
		t.Fatalf("expected database to be non-empty")
	}
}

func TestEnsureAdminTeam(t *testing.T) {
	db := setupBootstrapDB(t)
	teamRepo := repo.NewTeamRepo(db)

	divisionRepo := repo.NewDivisionRepo(db)
	divisionID, err := ensureAdminDivision(context.Background(), divisionRepo)
	if err != nil {
		t.Fatalf("ensureAdminDivision: %v", err)
	}

	team, err := ensureAdminTeam(context.Background(), teamRepo, divisionID)
	if err != nil {
		t.Fatalf("ensureAdminTeam: %v", err)
	}

	if team == nil || team.Name != "Admin" {
		t.Fatalf("expected admin team created")
	}

	team, err = ensureAdminTeam(context.Background(), teamRepo, divisionID)
	if err != nil {
		t.Fatalf("ensureAdminTeam second call: %v", err)
	}

	if team != nil {
		t.Fatalf("expected no team on duplicate create")
	}
}

func TestEnsureAdminUser(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)
	teamRepo := repo.NewTeamRepo(db)
	divisionRepo := repo.NewDivisionRepo(db)

	cfg := baseBootstrapConfig()
	divisionID, err := ensureAdminDivision(context.Background(), divisionRepo)
	if err != nil {
		t.Fatalf("ensureAdminDivision: %v", err)
	}

	team, err := ensureAdminTeam(context.Background(), teamRepo, divisionID)
	if err != nil {
		t.Fatalf("ensureAdminTeam: %v", err)
	}

	if team == nil {
		t.Fatalf("expected team for admin user")
	}

	user, err := ensureAdminUser(context.Background(), cfg, team, userRepo)
	if err != nil {
		t.Fatalf("ensureAdminUser: %v", err)
	}

	if user == nil || user.Role != models.AdminRole || user.TeamID != team.ID {
		t.Fatalf("unexpected admin user")
	}

	user, err = ensureAdminUser(context.Background(), cfg, team, userRepo)
	if err != nil {
		t.Fatalf("ensureAdminUser second call: %v", err)
	}

	if user != nil {
		t.Fatalf("expected no user on duplicate create")
	}
}

func TestBootstrapAdminCreatesTeamAndUser(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)
	teamRepo := repo.NewTeamRepo(db)
	divisionRepo := repo.NewDivisionRepo(db)

	cfg := baseBootstrapConfig()
	logger := newTestLogger(t)

	BootstrapAdmin(context.Background(), cfg, db, userRepo, teamRepo, divisionRepo, logger)

	var teamCount int
	if err := db.NewSelect().TableExpr("teams").ColumnExpr("COUNT(*)").Scan(context.Background(), &teamCount); err != nil {
		t.Fatalf("count teams: %v", err)
	}

	if teamCount != 1 {
		t.Fatalf("expected 1 team, got %d", teamCount)
	}

	var userCount int
	if err := db.NewSelect().TableExpr("users").ColumnExpr("COUNT(*)").Scan(context.Background(), &userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}

	if userCount != 1 {
		t.Fatalf("expected 1 user, got %d", userCount)
	}
}

func TestBootstrapAdminSkipsWhenNotEmpty(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)
	teamRepo := repo.NewTeamRepo(db)
	divisionRepo := repo.NewDivisionRepo(db)

	cfg := baseBootstrapConfig()
	logger := newTestLogger(t)

	team := &models.Team{
		Name:      "Existing",
		CreatedAt: time.Now().UTC(),
	}
	if err := teamRepo.Create(context.Background(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	BootstrapAdmin(context.Background(), cfg, db, userRepo, teamRepo, divisionRepo, logger)

	var userCount int
	if err := db.NewSelect().TableExpr("users").ColumnExpr("COUNT(*)").Scan(context.Background(), &userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}

	if userCount != 0 {
		t.Fatalf("expected no users, got %d", userCount)
	}
}

func TestBootstrapAdminDisabled(t *testing.T) {
	db := setupBootstrapDB(t)
	userRepo := repo.NewUserRepo(db)
	teamRepo := repo.NewTeamRepo(db)
	divisionRepo := repo.NewDivisionRepo(db)

	cfg := baseBootstrapConfig()
	cfg.Bootstrap.AdminTeamEnabled = false
	cfg.Bootstrap.AdminUserEnabled = false
	logger := newTestLogger(t)

	BootstrapAdmin(context.Background(), cfg, db, userRepo, teamRepo, divisionRepo, logger)

	var teamCount int
	if err := db.NewSelect().TableExpr("teams").ColumnExpr("COUNT(*)").Scan(context.Background(), &teamCount); err != nil {
		t.Fatalf("count teams: %v", err)
	}

	if teamCount != 0 {
		t.Fatalf("expected 0 teams, got %d", teamCount)
	}
}
