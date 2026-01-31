package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"smctf/internal/auth"
	"smctf/internal/config"
	"smctf/internal/db"
	apphttp "smctf/internal/http"
	"smctf/internal/logging"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/service"
	"smctf/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type testEnv struct {
	cfg            config.Config
	router         *gin.Engine
	userRepo       *repo.UserRepo
	regKeyRepo     *repo.RegistrationKeyRepo
	teamRepo       *repo.TeamRepo
	challengeRepo  *repo.ChallengeRepo
	submissionRepo *repo.SubmissionRepo
	authSvc        *service.AuthService
	ctfSvc         *service.CTFService
	teamSvc        *service.TeamService
}

type errorResp struct {
	Error     string                 `json:"error"`
	Details   []service.FieldError   `json:"details"`
	RateLimit *service.RateLimitInfo `json:"rate_limit"`
}

type registrationKeyResp struct {
	ID                int64      `json:"id"`
	Code              string     `json:"code"`
	CreatedBy         int64      `json:"created_by"`
	CreatedByUsername string     `json:"created_by_username"`
	TeamID            *int64     `json:"team_id"`
	TeamName          *string    `json:"team_name"`
	UsedBy            *int64     `json:"used_by"`
	UsedByUsername    *string    `json:"used_by_username"`
	UsedByIP          *string    `json:"used_by_ip"`
	CreatedAt         time.Time  `json:"created_at"`
	UsedAt            *time.Time `json:"used_at"`
}

var (
	testDB          *bun.DB
	testRedis       *redis.Client
	testCfg         config.Config
	pgContainer     testcontainers.Container
	redisServer     *miniredis.Miniredis
	skipIntegration bool
	regKeyCounter   int64 = 100000
	testLogger      *logging.Logger
	logDir          string
)

func TestMain(m *testing.M) {
	skipIntegration = os.Getenv("SMCTF_SKIP_INTEGRATION") != ""
	if skipIntegration {
		os.Exit(m.Run())
	}

	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

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

	redisServer, err = miniredis.Run()
	if err != nil {
		panic(err)
	}

	testRedis = redis.NewClient(&redis.Options{Addr: redisServer.Addr()})

	testCfg = config.Config{
		AppEnv:             "test",
		HTTPAddr:           ":0",
		ShutdownTimeout:    5 * time.Second,
		AutoMigrate:        false,
		PasswordBcryptCost: bcrypt.MinCost,
		DB:                 dbCfg,
		Redis: config.RedisConfig{
			Addr:     redisServer.Addr(),
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
		Logging: config.LoggingConfig{
			Dir:              "",
			FilePrefix:       "test",
			MaxBodyBytes:     1024 * 1024,
			WebhookQueueSize: 100,
			WebhookTimeout:   time.Second,
			WebhookBatchSize: 10,
			WebhookBatchWait: time.Second,
			WebhookMaxChars:  1000,
		},
	}

	logDir, err = os.MkdirTemp("", "smctf-logs-*")
	if err != nil {
		panic(err)
	}

	testCfg.Logging.Dir = logDir

	testLogger, err = logging.New(testCfg.Logging)
	if err != nil {
		panic(err)
	}

	code := m.Run()

	if testRedis != nil {
		_ = testRedis.Close()
	}

	if redisServer != nil {
		redisServer.Close()
	}

	if testDB != nil {
		_ = testDB.Close()
	}

	if testLogger != nil {
		_ = testLogger.Close()
	}

	if pgContainer != nil {
		_ = pgContainer.Terminate(ctx)
	}

	if logDir != "" {
		_ = os.RemoveAll(logDir)
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

func setupTest(t *testing.T, cfg config.Config) testEnv {
	t.Helper()
	skipIfIntegrationDisabled(t)
	resetState(t)

	userRepo := repo.NewUserRepo(testDB)
	registrationKeyRepo := repo.NewRegistrationKeyRepo(testDB)
	teamRepo := repo.NewTeamRepo(testDB)
	challengeRepo := repo.NewChallengeRepo(testDB)
	submissionRepo := repo.NewSubmissionRepo(testDB)
	authSvc := service.NewAuthService(cfg, testDB, userRepo, registrationKeyRepo, teamRepo, testRedis)
	teamSvc := service.NewTeamService(teamRepo)
	ctfSvc := service.NewCTFService(cfg, challengeRepo, submissionRepo, testRedis)
	router := apphttp.NewRouter(cfg, authSvc, ctfSvc, userRepo, teamSvc, testRedis, testLogger)

	return testEnv{
		cfg:            cfg,
		router:         router,
		userRepo:       userRepo,
		regKeyRepo:     registrationKeyRepo,
		teamRepo:       teamRepo,
		challengeRepo:  challengeRepo,
		submissionRepo: submissionRepo,
		authSvc:        authSvc,
		ctfSvc:         ctfSvc,
		teamSvc:        teamSvc,
	}
}

func resetState(t *testing.T) {
	t.Helper()

	if _, err := testDB.ExecContext(context.Background(), "TRUNCATE TABLE submissions, registration_keys, challenges, users, teams RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	if err := testRedis.FlushAll(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

func skipIfIntegrationDisabled(t *testing.T) {
	t.Helper()

	if skipIntegration {
		t.Skip("integration tests disabled via SMCTF_SKIP_INTEGRATION")
	}
}

func doRequest(t *testing.T, router *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader

	if body != nil {
		switch v := body.(type) {
		case string:
			reader = bytes.NewBufferString(v)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			reader = bytes.NewBuffer(data)
		}
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dest interface{}) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func authHeader(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func registerAndLogin(t *testing.T, env testEnv, email, username, password string) (string, string, int64) {
	t.Helper()

	admin := ensureAdminUser(t, env)
	key := createRegistrationKey(t, env, admin.ID)
	regBody := map[string]string{
		"email":            email,
		"username":         username,
		"password":         password,
		"registration_key": key.Code,
	}

	rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", regBody, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d: %s", rec.Code, rec.Body.String())
	}

	var regResp struct {
		ID int64 `json:"id"`
	}

	decodeJSON(t, rec, &regResp)

	loginBody := map[string]string{
		"email":    email,
		"password": password,
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/login", loginBody, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
	}

	var loginResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}

	decodeJSON(t, rec, &loginResp)

	return loginResp.AccessToken, loginResp.RefreshToken, loginResp.User.ID
}

func createUser(t *testing.T, env testEnv, email, username, password, role string) *models.User {
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

func createUserWithTeam(t *testing.T, env testEnv, email, username, password, role string, teamID *int64) *models.User {
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

func createTeam(t *testing.T, env testEnv, name string) *models.Team {
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

func loginUser(t *testing.T, router *gin.Engine, email, password string) (string, string, int64) {
	t.Helper()
	body := map[string]string{"email": email, "password": password}
	rec := doRequest(t, router, http.MethodPost, "/api/auth/login", body, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}

	decodeJSON(t, rec, &resp)

	return resp.AccessToken, resp.RefreshToken, resp.User.ID
}

func ensureAdminUser(t *testing.T, env testEnv) *models.User {
	t.Helper()

	user, err := env.userRepo.GetByEmail(context.Background(), "admin@example.com")
	if err == nil {
		return user
	}
	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("get admin: %v", err)
	}

	return createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")
}

func nextRegistrationCode() string {
	value := atomic.AddInt64(&regKeyCounter, 1) % 1000000
	return fmt.Sprintf("%06d", value)
}

func createRegistrationKey(t *testing.T, env testEnv, createdBy int64) *models.RegistrationKey {
	t.Helper()

	key := &models.RegistrationKey{
		Code:      nextRegistrationCode(),
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
	}

	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("create registration key: %v", err)
	}

	return key
}

func createRegistrationKeyWithTeam(t *testing.T, env testEnv, createdBy int64, teamID *int64) *models.RegistrationKey {
	t.Helper()

	key := &models.RegistrationKey{
		Code:      nextRegistrationCode(),
		CreatedBy: createdBy,
		TeamID:    teamID,
		CreatedAt: time.Now().UTC(),
	}

	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("create registration key: %v", err)
	}

	return key
}

func createChallenge(t *testing.T, env testEnv, title string, points int, flag string, active bool) *models.Challenge {
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

func createSubmission(t *testing.T, env testEnv, userID, challengeID int64, correct bool, submittedAt time.Time) {
	t.Helper()

	sub := &models.Submission{
		UserID:      userID,
		ChallengeID: challengeID,
		Provided:    "flag{test}",
		Correct:     correct,
		SubmittedAt: submittedAt,
	}

	if err := env.submissionRepo.Create(context.Background(), sub); err != nil {
		t.Fatalf("create submission: %v", err)
	}
}

func assertFieldErrors(t *testing.T, got []service.FieldError, expected map[string]string) {
	t.Helper()

	found := make(map[string]string, len(got))

	for _, fe := range got {
		found[fe.Field] = fe.Reason
	}

	for field, reason := range expected {
		if found[field] != reason {
			t.Fatalf("expected field %s reason %s, got %q", field, reason, found[field])
		}
	}
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}
