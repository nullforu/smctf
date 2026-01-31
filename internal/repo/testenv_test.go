package repo

import (
	"context"
	"os"
	"testing"
	"time"

	"smctf/internal/auth"
	"smctf/internal/config"
	"smctf/internal/db"
	"smctf/internal/models"
	"smctf/internal/utils"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type repoEnv struct {
	cfg            config.Config
	db             *bun.DB
	userRepo       *UserRepo
	regKeyRepo     *RegistrationKeyRepo
	teamRepo       *TeamRepo
	challengeRepo  *ChallengeRepo
	submissionRepo *SubmissionRepo
}

var (
	repoDB              *bun.DB
	repoCfg             config.Config
	repoPG              testcontainers.Container
	skipRepoIntegration bool
)

func TestMain(m *testing.M) {
	skipRepoIntegration = os.Getenv("SMCTF_SKIP_INTEGRATION") != ""
	if skipRepoIntegration {
		os.Exit(m.Run())
	}

	ctx := context.Background()
	container, dbCfg, err := startPostgres(ctx)
	if err != nil {
		panic(err)
	}

	repoPG = container

	repoDB, err = db.New(dbCfg, "test")
	if err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(ctx, repoDB); err != nil {
		panic(err)
	}

	repoCfg = config.Config{
		AppEnv:             "test",
		HTTPAddr:           ":0",
		ShutdownTimeout:    5 * time.Second,
		AutoMigrate:        false,
		PasswordBcryptCost: bcrypt.MinCost,
		DB:                 dbCfg,
		Security: config.SecurityConfig{
			FlagHMACSecret:   "test-flag-secret",
			SubmissionWindow: 2 * time.Minute,
			SubmissionMax:    5,
		},
	}

	code := m.Run()

	if repoDB != nil {
		_ = repoDB.Close()
	}

	if repoPG != nil {
		_ = repoPG.Terminate(ctx)
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

func setupRepoTest(t *testing.T) repoEnv {
	t.Helper()
	if skipRepoIntegration {
		t.Skip("repo tests disabled via SMCTF_SKIP_INTEGRATION")
	}

	resetRepoState(t)

	return repoEnv{
		cfg:            repoCfg,
		db:             repoDB,
		userRepo:       NewUserRepo(repoDB),
		regKeyRepo:     NewRegistrationKeyRepo(repoDB),
		teamRepo:       NewTeamRepo(repoDB),
		challengeRepo:  NewChallengeRepo(repoDB),
		submissionRepo: NewSubmissionRepo(repoDB),
	}
}

func resetRepoState(t *testing.T) {
	t.Helper()
	if _, err := repoDB.ExecContext(context.Background(), "TRUNCATE TABLE submissions, registration_keys, challenges, users, teams RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func createUser(t *testing.T, env repoEnv, email, username, password, role string) *models.User {
	t.Helper()
	hash, err := auth.HashPassword(password, env.cfg.PasswordBcryptCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &models.User{
		Email:        email,
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := env.userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	return user
}

func createUserWithTeam(t *testing.T, env repoEnv, email, username, password, role string, teamID *int64) *models.User {
	t.Helper()
	hash, err := auth.HashPassword(password, env.cfg.PasswordBcryptCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &models.User{
		Email:        email,
		Username:     username,
		PasswordHash: hash,
		Role:         role,
		TeamID:       teamID,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := env.userRepo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	return user
}

func createTeam(t *testing.T, env repoEnv, name string) *models.Team {
	t.Helper()
	team := &models.Team{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	if err := env.teamRepo.Create(context.Background(), team); err != nil {
		t.Fatalf("create team: %v", err)
	}

	return team
}

func createChallenge(t *testing.T, env repoEnv, title string, points int, flag string, active bool) *models.Challenge {
	t.Helper()
	challenge := &models.Challenge{
		Title:         title,
		Description:   "desc",
		Category:      "Misc",
		Points:        points,
		MinimumPoints: points,
		FlagHash:      utils.HMACFlag(env.cfg.Security.FlagHMACSecret, flag),
		IsActive:      active,
		CreatedAt:     time.Now().UTC(),
	}
	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func createSubmission(t *testing.T, env repoEnv, userID, challengeID int64, correct bool, submittedAt time.Time) *models.Submission {
	t.Helper()
	sub := &models.Submission{
		UserID:      userID,
		ChallengeID: challengeID,
		Provided:    "flag",
		Correct:     correct,
		SubmittedAt: submittedAt,
	}
	if err := env.submissionRepo.Create(context.Background(), sub); err != nil {
		t.Fatalf("create submission: %v", err)
	}

	return sub
}
