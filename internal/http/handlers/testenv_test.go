package handlers

import (
	"context"
	"os"
	"testing"
	"time"

	"smctf/internal/auth"
	"smctf/internal/config"
	"smctf/internal/db"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/service"
	"smctf/internal/storage"
	"smctf/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type handlerEnv struct {
	cfg            config.Config
	db             *bun.DB
	redis          *redis.Client
	userRepo       *repo.UserRepo
	regKeyRepo     *repo.RegistrationKeyRepo
	teamRepo       *repo.TeamRepo
	challengeRepo  *repo.ChallengeRepo
	submissionRepo *repo.SubmissionRepo
	appConfigRepo  *repo.AppConfigRepo
	authSvc        *service.AuthService
	ctfSvc         *service.CTFService
	teamSvc        *service.TeamService
	appConfigSvc   *service.AppConfigService
	handler        *Handler
}

var (
	handlerDB          *bun.DB
	handlerRedis       *redis.Client
	handlerCfg         config.Config
	handlerPGContainer testcontainers.Container
	handlerRedisServer *miniredis.Miniredis
	skipHandlerEnv     bool
)

func TestMain(m *testing.M) {
	skipHandlerEnv = os.Getenv("SMCTF_SKIP_INTEGRATION") != ""
	if skipHandlerEnv {
		os.Exit(m.Run())
	}

	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	container, dbCfg, err := startHandlerPostgres(ctx)
	if err != nil {
		panic(err)
	}
	handlerPGContainer = container

	handlerDB, err = db.New(dbCfg, "test")
	if err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(ctx, handlerDB); err != nil {
		panic(err)
	}

	handlerRedisServer, err = miniredis.Run()
	if err != nil {
		panic(err)
	}

	handlerRedis = redis.NewClient(&redis.Options{Addr: handlerRedisServer.Addr()})

	handlerCfg = config.Config{
		AppEnv:             "test",
		HTTPAddr:           ":0",
		ShutdownTimeout:    5 * time.Second,
		AutoMigrate:        false,
		PasswordBcryptCost: bcrypt.MinCost,
		DB:                 dbCfg,
		Redis: config.RedisConfig{
			Addr:     handlerRedisServer.Addr(),
			Password: "",
			DB:       0,
			PoolSize: 5,
		},
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			Issuer:     "smctf-test",
			AccessTTL:  time.Hour,
			RefreshTTL: 24 * time.Hour,
		},
		Security: config.SecurityConfig{
			FlagHMACSecret:   "test-flag-secret",
			SubmissionWindow: 2 * time.Minute,
			SubmissionMax:    5,
		},
		Cache: config.CacheConfig{
			TimelineTTL:    2 * time.Minute,
			LeaderboardTTL: 2 * time.Minute,
		},
	}

	code := m.Run()

	if handlerRedis != nil {
		_ = handlerRedis.Close()
	}

	if handlerRedisServer != nil {
		handlerRedisServer.Close()
	}

	if handlerDB != nil {
		_ = handlerDB.Close()
	}

	if handlerPGContainer != nil {
		_ = handlerPGContainer.Terminate(ctx)
	}

	os.Exit(code)
}

func setHandlerCTFWindow(t *testing.T, env handlerEnv, startAt, endAt *time.Time) {
	t.Helper()

	var startValue *string
	if startAt != nil {
		value := startAt.UTC().Format(time.RFC3339)
		startValue = &value
	} else {
		value := ""
		startValue = &value
	}

	var endValue *string
	if endAt != nil {
		value := endAt.UTC().Format(time.RFC3339)
		endValue = &value
	} else {
		value := ""
		endValue = &value
	}

	if _, _, _, err := env.appConfigSvc.Update(context.Background(), nil, nil, nil, nil, startValue, endValue); err != nil {
		t.Fatalf("set ctf window: %v", err)
	}
}

func startHandlerPostgres(ctx context.Context) (testcontainers.Container, config.DBConfig, error) {
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

func setupHandlerTest(t *testing.T) handlerEnv {
	t.Helper()
	skipIfHandlerDisabled(t)
	resetHandlerState(t)

	userRepo := repo.NewUserRepo(handlerDB)
	regRepo := repo.NewRegistrationKeyRepo(handlerDB)
	teamRepo := repo.NewTeamRepo(handlerDB)
	challengeRepo := repo.NewChallengeRepo(handlerDB)
	submissionRepo := repo.NewSubmissionRepo(handlerDB)
	scoreRepo := repo.NewScoreboardRepo(handlerDB)
	appConfigRepo := repo.NewAppConfigRepo(handlerDB)

	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)

	appConfigSvc := service.NewAppConfigService(appConfigRepo)
	authSvc := service.NewAuthService(handlerCfg, handlerDB, userRepo, regRepo, teamRepo, handlerRedis)
	teamSvc := service.NewTeamService(teamRepo)
	ctfSvc := service.NewCTFService(handlerCfg, challengeRepo, submissionRepo, handlerRedis, fileStore)

	handler := New(handlerCfg, authSvc, ctfSvc, appConfigSvc, userRepo, scoreRepo, teamSvc, nil, handlerRedis)

	return handlerEnv{
		cfg:            handlerCfg,
		db:             handlerDB,
		redis:          handlerRedis,
		userRepo:       userRepo,
		regKeyRepo:     regRepo,
		teamRepo:       teamRepo,
		challengeRepo:  challengeRepo,
		submissionRepo: submissionRepo,
		appConfigRepo:  appConfigRepo,
		authSvc:        authSvc,
		ctfSvc:         ctfSvc,
		teamSvc:        teamSvc,
		appConfigSvc:   appConfigSvc,
		handler:        handler,
	}
}

func resetHandlerState(t *testing.T) {
	t.Helper()

	if _, err := handlerDB.ExecContext(context.Background(), "TRUNCATE TABLE app_configs, submissions, registration_keys, stacks, challenges, users, teams RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	if err := handlerRedis.FlushAll(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

func skipIfHandlerDisabled(t *testing.T) {
	t.Helper()

	if skipHandlerEnv {
		t.Skip("handler tests disabled via SMCTF_SKIP_INTEGRATION")
	}
}

func createHandlerUser(t *testing.T, env handlerEnv, email, username, password, role string) *models.User {
	t.Helper()
	team := createHandlerTeam(t, env, "team-"+username)

	return createHandlerUserWithTeam(t, env, email, username, password, role, team.ID)
}

func createHandlerUserWithTeam(t *testing.T, env handlerEnv, email, username, password, role string, teamID int64) *models.User {
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

func createHandlerRegistrationKey(t *testing.T, env handlerEnv, code string, createdBy int64) *models.RegistrationKey {
	t.Helper()

	team := createHandlerTeam(t, env, "reg-"+code)
	key := &models.RegistrationKey{
		Code:      code,
		CreatedBy: createdBy,
		TeamID:    team.ID,
		CreatedAt: time.Now().UTC(),
	}

	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("create registration key: %v", err)
	}

	return key
}

func createHandlerRegistrationKeyWithTeam(t *testing.T, env handlerEnv, code string, createdBy int64, teamID int64) *models.RegistrationKey {
	t.Helper()

	key := &models.RegistrationKey{
		Code:      code,
		CreatedBy: createdBy,
		TeamID:    teamID,
		CreatedAt: time.Now().UTC(),
	}

	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("create registration key: %v", err)
	}

	return key
}

func createHandlerTeam(t *testing.T, env handlerEnv, name string) *models.Team {
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

func createHandlerChallenge(t *testing.T, env handlerEnv, title string, points int, flag string, active bool) *models.Challenge {
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

func createHandlerSubmission(t *testing.T, env handlerEnv, userID, challengeID int64, correct bool, submittedAt time.Time) *models.Submission {
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
