package service

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
	"smctf/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type serviceEnv struct {
	cfg            config.Config
	db             *bun.DB
	redis          *redis.Client
	userRepo       *repo.UserRepo
	regKeyRepo     *repo.RegistrationKeyRepo
	teamRepo       *repo.TeamRepo
	challengeRepo  *repo.ChallengeRepo
	submissionRepo *repo.SubmissionRepo
	authSvc        *AuthService
	ctfSvc         *CTFService
	teamSvc        *TeamService
}

var (
	serviceDB          *bun.DB
	serviceRedis       *redis.Client
	serviceCfg         config.Config
	servicePGContainer testcontainers.Container
	serviceRedisServer *miniredis.Miniredis
	skipServiceEnv     bool
)

func TestMain(m *testing.M) {
	skipServiceEnv = os.Getenv("SMCTF_SKIP_INTEGRATION") != ""
	if skipServiceEnv {
		os.Exit(m.Run())
	}

	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	container, dbCfg, err := startPostgres(ctx)
	if err != nil {
		panic(err)
	}
	servicePGContainer = container

	serviceDB, err = db.New(dbCfg, "test")
	if err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(ctx, serviceDB); err != nil {
		panic(err)
	}

	serviceRedisServer, err = miniredis.Run()
	if err != nil {
		panic(err)
	}

	serviceRedis = redis.NewClient(&redis.Options{Addr: serviceRedisServer.Addr()})

	serviceCfg = config.Config{
		AppEnv:             "test",
		HTTPAddr:           ":0",
		ShutdownTimeout:    5 * time.Second,
		AutoMigrate:        false,
		PasswordBcryptCost: bcrypt.MinCost,
		DB:                 dbCfg,
		Redis: config.RedisConfig{
			Addr:     serviceRedisServer.Addr(),
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
			TimelineTTL: 2 * time.Minute,
		},
	}

	code := m.Run()

	if serviceRedis != nil {
		_ = serviceRedis.Close()
	}

	if serviceRedisServer != nil {
		serviceRedisServer.Close()
	}

	if serviceDB != nil {
		_ = serviceDB.Close()
	}

	if servicePGContainer != nil {
		_ = servicePGContainer.Terminate(ctx)
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

func setupServiceTest(t *testing.T) serviceEnv {
	t.Helper()
	skipIfServiceDisabled(t)
	resetServiceState(t)

	userRepo := repo.NewUserRepo(serviceDB)
	regRepo := repo.NewRegistrationKeyRepo(serviceDB)
	teamRepo := repo.NewTeamRepo(serviceDB)
	challengeRepo := repo.NewChallengeRepo(serviceDB)
	submissionRepo := repo.NewSubmissionRepo(serviceDB)
	authSvc := NewAuthService(serviceCfg, serviceDB, userRepo, regRepo, teamRepo, serviceRedis)
	teamSvc := NewTeamService(teamRepo)
	ctfSvc := NewCTFService(serviceCfg, challengeRepo, submissionRepo, serviceRedis)

	return serviceEnv{
		cfg:            serviceCfg,
		db:             serviceDB,
		redis:          serviceRedis,
		userRepo:       userRepo,
		regKeyRepo:     regRepo,
		teamRepo:       teamRepo,
		challengeRepo:  challengeRepo,
		submissionRepo: submissionRepo,
		authSvc:        authSvc,
		ctfSvc:         ctfSvc,
		teamSvc:        teamSvc,
	}
}

func resetServiceState(t *testing.T) {
	t.Helper()

	if _, err := serviceDB.ExecContext(context.Background(), "TRUNCATE TABLE submissions, registration_keys, challenges, users, teams RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	if err := serviceRedis.FlushAll(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

func skipIfServiceDisabled(t *testing.T) {
	t.Helper()

	if skipServiceEnv {
		t.Skip("service tests disabled via SMCTF_SKIP_INTEGRATION")
	}
}

func createUser(t *testing.T, env serviceEnv, email, username, password, role string) *models.User {
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

func createUserWithTeam(t *testing.T, env serviceEnv, email, username, password, role string, teamID *int64) *models.User {
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

func createTeam(t *testing.T, env serviceEnv, name string) *models.Team {
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

func createRegistrationKeyWithTeam(t *testing.T, env serviceEnv, code string, createdBy int64, teamID *int64) *models.RegistrationKey {
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

func createRegistrationKey(t *testing.T, env serviceEnv, code string, createdBy int64) *models.RegistrationKey {
	t.Helper()

	return createRegistrationKeyWithTeam(t, env, code, createdBy, nil)
}

func createChallenge(t *testing.T, env serviceEnv, title string, points int, flag string, active bool) *models.Challenge {
	t.Helper()
	challenge := &models.Challenge{
		Title:       title,
		Description: "desc",
		Category:    "Misc",
		Points:      points,
		FlagHash:    utils.HMACFlag(env.cfg.Security.FlagHMACSecret, flag),
		IsActive:    active,
		CreatedAt:   time.Now().UTC(),
	}

	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func createSubmission(t *testing.T, env serviceEnv, userID, challengeID int64, correct bool, submittedAt time.Time) *models.Submission {
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
