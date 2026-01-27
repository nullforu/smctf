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
	challengeRepo  *repo.ChallengeRepo
	submissionRepo *repo.SubmissionRepo
	authSvc        *service.AuthService
	ctfSvc         *service.CTFService
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

func setupTest(t *testing.T, cfg config.Config) testEnv {
	t.Helper()
	skipIfIntegrationDisabled(t)
	resetState(t)

	userRepo := repo.NewUserRepo(testDB)
	registrationKeyRepo := repo.NewRegistrationKeyRepo(testDB)
	challengeRepo := repo.NewChallengeRepo(testDB)
	submissionRepo := repo.NewSubmissionRepo(testDB)
	authSvc := service.NewAuthService(cfg, testDB, userRepo, registrationKeyRepo, testRedis)
	ctfSvc := service.NewCTFService(cfg, challengeRepo, submissionRepo, testRedis)
	router := apphttp.NewRouter(cfg, authSvc, ctfSvc, userRepo, testRedis)

	return testEnv{
		cfg:            cfg,
		router:         router,
		userRepo:       userRepo,
		regKeyRepo:     registrationKeyRepo,
		challengeRepo:  challengeRepo,
		submissionRepo: submissionRepo,
		authSvc:        authSvc,
		ctfSvc:         ctfSvc,
	}
}

func resetState(t *testing.T) {
	t.Helper()

	if _, err := testDB.ExecContext(context.Background(), "TRUNCATE TABLE submissions, registration_keys, challenges, users RESTART IDENTITY CASCADE"); err != nil {
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

func createChallenge(t *testing.T, env testEnv, title string, points int, flag string, active bool) *models.Challenge {
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

func TestRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupTest(t, testCfg)
		admin := ensureAdminUser(t, env)
		key := createRegistrationKey(t, env, admin.ID)
		body := map[string]string{
			"email":            "user@example.com",
			"username":         "user1",
			"password":         "strong-password",
			"registration_key": key.Code,
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			ID       int64  `json:"id"`
			Email    string `json:"email"`
			Username string `json:"username"`
		}

		decodeJSON(t, rec, &resp)

		if resp.ID == 0 || resp.Email != body["email"] || resp.Username != body["username"] {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		env := setupTest(t, testCfg)
		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", map[string]string{}, nil)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrInvalidInput.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}

		assertFieldErrors(t, resp.Details, map[string]string{
			"email":            "required",
			"username":         "required",
			"password":         "required",
			"registration_key": "required",
		})
	})

	t.Run("invalid key format", func(t *testing.T) {
		env := setupTest(t, testCfg)
		body := map[string]string{
			"email":            "user@example.com",
			"username":         "user1",
			"password":         "strong-password",
			"registration_key": "abc123",
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{
			"registration_key": "invalid",
		})
	})

	t.Run("invalid key", func(t *testing.T) {
		env := setupTest(t, testCfg)
		body := map[string]string{
			"email":            "user@example.com",
			"username":         "user1",
			"password":         "strong-password",
			"registration_key": "123456",
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{
			"registration_key": "invalid",
		})
	})

	t.Run("used key", func(t *testing.T) {
		env := setupTest(t, testCfg)
		admin := ensureAdminUser(t, env)
		key := createRegistrationKey(t, env, admin.ID)
		body := map[string]string{
			"email":            "user@example.com",
			"username":         "user1",
			"password":         "strong-password",
			"registration_key": key.Code,
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(t, env.router, http.MethodPost, "/api/auth/register", map[string]string{
			"email":            "user2@example.com",
			"username":         "user2",
			"password":         "strong-password",
			"registration_key": key.Code,
		}, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{
			"registration_key": "used",
		})
	})

	t.Run("duplicate", func(t *testing.T) {
		env := setupTest(t, testCfg)
		admin := ensureAdminUser(t, env)
		key := createRegistrationKey(t, env, admin.ID)
		body := map[string]string{
			"email":            "user@example.com",
			"username":         "user1",
			"password":         "strong-password",
			"registration_key": key.Code,
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		secondKey := createRegistrationKey(t, env, admin.ID)
		body["registration_key"] = secondKey.Code
		rec = doRequest(t, env.router, http.MethodPost, "/api/auth/register", body, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrUserExists.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})
}

func TestLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, refresh, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

		if access == "" || refresh == "" {
			t.Fatalf("tokens should not be empty")
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		env := setupTest(t, testCfg)
		_, _, _ = registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		body := map[string]string{"email": "user@example.com", "password": "wrong"}

		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/login", body, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrInvalidCreds.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		env := setupTest(t, testCfg)
		rec := doRequest(t, env.router, http.MethodPost, "/api/auth/login", map[string]string{"email": "user@example.com"}, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{
			"password": "required",
		})
	})
}

func TestRefreshAndLogout(t *testing.T) {
	env := setupTest(t, testCfg)
	_, refresh, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

	rec := doRequest(t, env.router, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": refresh}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var refreshResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	decodeJSON(t, rec, &refreshResp)
	if refreshResp.AccessToken == "" || refreshResp.RefreshToken == "" {
		t.Fatalf("tokens should not be empty")
	}

	if refreshResp.RefreshToken == refresh {
		t.Fatalf("refresh token should rotate")
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": refresh}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/logout", map[string]string{"refresh_token": refreshResp.RefreshToken}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/refresh", map[string]string{"refresh_token": refreshResp.RefreshToken}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMe(t *testing.T) {
	env := setupTest(t, testCfg)
	access, refresh, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

	rec := doRequest(t, env.router, http.MethodGet, "/api/me", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/me", nil, map[string]string{"Authorization": "Token " + access})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/me", nil, authHeader(refresh))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/me", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID       int64  `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}

	decodeJSON(t, rec, &resp)

	if resp.Email != "user@example.com" || resp.Username != "user1" || resp.Role != "user" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestUpdateMe(t *testing.T) {
	env := setupTest(t, testCfg)
	access, _, userID := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")

	rec := doRequest(t, env.router, http.MethodPut, "/api/me", map[string]string{"username": "newuser"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPut, "/api/me", map[string]string{"username": "newuser"}, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID       int64  `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}

	decodeJSON(t, rec, &resp)

	if resp.ID != userID || resp.Email != "user@example.com" || resp.Username != "newuser" || resp.Role != "user" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestMeSolved(t *testing.T) {
	env := setupTest(t, testCfg)
	access, _, userID := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
	challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/me/solved", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var solved []models.SolvedChallenge
	decodeJSON(t, rec, &solved)

	if len(solved) != 1 {
		t.Fatalf("expected 1 solved, got %d", len(solved))
	}

	if solved[0].ChallengeID != challenge.ID || solved[0].Points != 100 || solved[0].Title != "Warmup" {
		t.Fatalf("unexpected solved entry: %+v", solved[0])
	}

	if solved[0].ChallengeID == 0 || solved[0].SolvedAt.IsZero() {
		t.Fatalf("expected solved timestamp and id, got %+v for user %d", solved[0], userID)
	}
}

func TestListChallenges(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createChallenge(t, env, "Active 1", 100, "flag{1}", true)
	_ = createChallenge(t, env, "Inactive", 50, "flag{2}", false)
	_ = createChallenge(t, env, "Active 2", 200, "flag{3}", true)

	rec := doRequest(t, env.router, http.MethodGet, "/api/challenges", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp []map[string]interface{}
	decodeJSON(t, rec, &resp)

	if len(resp) != 3 {
		t.Fatalf("expected 3 challenges, got %d", len(resp))
	}

	expectedTitles := []string{"Active 1", "Inactive", "Active 2"}
	expectedActive := []bool{true, false, true}
	expectedCategories := []string{"Misc", "Misc", "Misc"}

	for i, row := range resp {
		if row["title"] != expectedTitles[i] {
			t.Fatalf("expected title %q, got %q", expectedTitles[i], row["title"])
		}
		if row["category"] != expectedCategories[i] {
			t.Fatalf("expected category %q, got %q", expectedCategories[i], row["category"])
		}
		if isActive, ok := row["is_active"].(bool); !ok || isActive != expectedActive[i] {
			t.Fatalf("expected is_active to be %v for %q, got %v", expectedActive[i], row["title"], isActive)
		}
	}
}

func TestSubmitFlag(t *testing.T) {
	t.Run("missing auth", func(t *testing.T) {
		env := setupTest(t, testCfg)
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)
		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, nil)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/abc/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrInvalidInput.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)
		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{}, authHeader(access))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		assertFieldErrors(t, resp.Details, map[string]string{"flag": "required"})
	})

	t.Run("challenge not found", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/999/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrChallengeNotFound.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("inactive challenge", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", false)
		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("correct and wrong", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{nope}"}, authHeader(access))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var wrongResp struct {
			Correct bool `json:"correct"`
		}

		decodeJSON(t, rec, &wrongResp)

		if wrongResp.Correct {
			t.Fatalf("expected incorrect flag")
		}

		rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var correctResp struct {
			Correct bool `json:"correct"`
		}

		decodeJSON(t, rec, &correctResp)

		if !correctResp.Correct {
			t.Fatalf("expected correct flag")
		}
	})

	t.Run("already solved", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{ok}"}, authHeader(access))
		if rec.Code != http.StatusConflict {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrAlreadySolved.Error() {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
	})

	t.Run("rate limited", func(t *testing.T) {
		env := setupTest(t, testCfg)
		access, _, _ := registerAndLogin(t, env, "user@example.com", "user1", "strong-password")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)

		for i := 0; i < env.cfg.Security.SubmissionMax; i++ {
			rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{nope}"}, authHeader(access))
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d at attempt %d: %s", rec.Code, i+1, rec.Body.String())
			}
		}

		rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{nope}"}, authHeader(access))
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp errorResp
		decodeJSON(t, rec, &resp)

		if resp.Error != service.ErrRateLimited.Error() || resp.RateLimit == nil {
			t.Fatalf("unexpected rate limit response: %+v", resp)
		}

		if resp.RateLimit.Limit != env.cfg.Security.SubmissionMax || resp.RateLimit.Remaining != 0 {
			t.Fatalf("unexpected rate limit info: %+v", resp.RateLimit)
		}

		if rec.Header().Get("X-RateLimit-Limit") == "" || rec.Header().Get("X-RateLimit-Remaining") == "" || rec.Header().Get("X-RateLimit-Reset") == "" {
			t.Fatalf("missing rate limit headers")
		}
	})
}

func TestScoreboard(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	createSubmission(t, env, user1.ID, challenge1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge1.ID, true, time.Now().UTC())
	createSubmission(t, env, user2.ID, challenge2.ID, true, time.Now().UTC())

	rec := doRequest(t, env.router, http.MethodGet, "/api/leaderboard", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var rows []models.LeaderboardEntry
	decodeJSON(t, rec, &rows)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].UserID != user2.ID || rows[0].Score != 300 {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}
}

func TestScoreboardTimeline(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	createSubmission(t, env, user1.ID, challenge1.ID, true, base.Add(3*time.Minute))
	createSubmission(t, env, user2.ID, challenge2.ID, true, base.Add(7*time.Minute))
	createSubmission(t, env, user1.ID, challenge2.ID, true, base.Add(16*time.Minute))

	rec := doRequest(t, env.router, http.MethodGet, "/api/timeline", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Submissions []struct {
			Timestamp      time.Time `json:"timestamp"`
			UserID         int64     `json:"user_id"`
			Username       string    `json:"username"`
			Points         int       `json:"points"`
			ChallengeCount int       `json:"challenge_count"`
		} `json:"submissions"`
	}

	decodeJSON(t, rec, &resp)

	if len(resp.Submissions) != 3 {
		t.Fatalf("expected 3 submissions, got %d", len(resp.Submissions))
	}

	if resp.Submissions[0].UserID != 1 || resp.Submissions[0].Points != 100 || resp.Submissions[0].ChallengeCount != 1 {
		t.Fatalf("unexpected first submission: %+v", resp.Submissions[0])
	}

	if resp.Submissions[1].UserID != 2 || resp.Submissions[1].Points != 200 || resp.Submissions[1].ChallengeCount != 1 {
		t.Fatalf("unexpected second submission: %+v", resp.Submissions[1])
	}

	if resp.Submissions[2].UserID != 1 || resp.Submissions[2].Points != 200 || resp.Submissions[2].ChallengeCount != 1 {
		t.Fatalf("unexpected third submission: %+v", resp.Submissions[2])
	}
}

func TestScoreboardTimelineWindow(t *testing.T) {
	env := setupTest(t, testCfg)
	user1 := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 200, "flag{2}", true)

	now := time.Now().UTC()
	createSubmission(t, env, user1.ID, challenge1.ID, true, now.Add(-2*time.Hour))

	recent := now.Add(-20 * time.Minute)
	createSubmission(t, env, user2.ID, challenge2.ID, true, recent)

	rec := doRequest(t, env.router, http.MethodGet, "/api/timeline?window=60", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Submissions []struct {
			Timestamp      time.Time `json:"timestamp"`
			UserID         int64     `json:"user_id"`
			Username       string    `json:"username"`
			Points         int       `json:"points"`
			ChallengeCount int       `json:"challenge_count"`
		} `json:"submissions"`
	}

	decodeJSON(t, rec, &resp)

	if len(resp.Submissions) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(resp.Submissions))
	}

	if resp.Submissions[0].UserID != user2.ID {
		t.Fatalf("unexpected user: %d", resp.Submissions[0].UserID)
	}

	windowStart := now.Add(-60 * time.Minute)
	if resp.Submissions[0].Timestamp.Before(windowStart) {
		t.Fatalf("submission outside window: %s", resp.Submissions[0].Timestamp)
	}
}

func TestScoreboardTimelineInvalidWindow(t *testing.T) {
	env := setupTest(t, testCfg)
	rec := doRequest(t, env.router, http.MethodGet, "/api/timeline?window=0", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp errorResp
	decodeJSON(t, rec, &resp)

	if resp.Error != service.ErrInvalidInput.Error() {
		t.Fatalf("unexpected error: %s", resp.Error)
	}
}

func TestAdminCreateChallenge(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")

	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]string{"title": "Ch1"}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	accessUser, _, _ := registerAndLogin(t, env, "user2@example.com", "user2", "strong-password")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]interface{}{
		"title":       "Ch1",
		"description": "desc",
		"category":    "Web",
		"points":      100,
		"flag":        "flag{1}",
		"is_active":   true,
	}, authHeader(accessUser))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]interface{}{
		"title":       "Ch1",
		"description": "desc",
		"category":    "Web",
		"points":      100,
		"flag":        "flag{1}",
		"is_active":   true,
	}, authHeader(adminAccess))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]interface{}{
		"title": "Ch2",
	}, authHeader(adminAccess))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/challenges", map[string]interface{}{
		"title":       "Ch3",
		"description": "desc",
		"category":    "Unknown",
		"points":      100,
		"flag":        "flag{1}",
		"is_active":   true,
	}, authHeader(adminAccess))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp errorResp
	decodeJSON(t, rec, &resp)
	assertFieldErrors(t, resp.Details, map[string]string{"category": "invalid"})
}

func TestAdminRegistrationKeys(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")

	rec := doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 1}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	accessUser, _, _ := registerAndLogin(t, env, "user2@example.com", "user2", "strong-password")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 1}, authHeader(accessUser))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	adminAccess, _, _ := loginUser(t, env.router, "admin@example.com", "adminpass")
	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 0}, authHeader(adminAccess))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var errResp errorResp
	decodeJSON(t, rec, &errResp)
	assertFieldErrors(t, errResp.Details, map[string]string{"count": "must be >= 1"})

	rec = doRequest(t, env.router, http.MethodPost, "/api/admin/registration-keys", map[string]int{"count": 2}, authHeader(adminAccess))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var created []registrationKeyResp
	decodeJSON(t, rec, &created)

	if len(created) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(created))
	}

	if len(created[0].Code) != 6 || len(created[1].Code) != 6 {
		t.Fatalf("expected 6-digit codes, got %q and %q", created[0].Code, created[1].Code)
	}

	if created[0].CreatedByUsername != "admin" {
		t.Fatalf("expected created_by_username admin, got %q", created[0].CreatedByUsername)
	}

	regBody := map[string]string{
		"email":            "user1@example.com",
		"username":         "user1",
		"password":         "strong-password",
		"registration_key": created[0].Code,
	}

	regHeaders := map[string]string{"X-Forwarded-For": "203.0.113.7"}

	rec = doRequest(t, env.router, http.MethodPost, "/api/auth/register", regBody, regHeaders)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/admin/registration-keys", nil, authHeader(adminAccess))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var listed []registrationKeyResp
	decodeJSON(t, rec, &listed)

	var found *registrationKeyResp
	for i := range listed {
		if listed[i].Code == created[0].Code {
			found = &listed[i]
			break
		}
	}

	if found == nil {
		t.Fatalf("expected key %s in list", created[0].Code)
	}

	if found.CreatedByUsername != "admin" {
		t.Fatalf("expected created_by_username admin, got %q", found.CreatedByUsername)
	}

	if found.UsedByUsername == nil || *found.UsedByUsername != "user1" {
		t.Fatalf("expected used_by_username user1, got %v", found.UsedByUsername)
	}

	if found.UsedByIP == nil || *found.UsedByIP != "203.0.113.7" {
		t.Fatalf("expected used_by_ip 203.0.113.7, got %v", found.UsedByIP)
	}
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

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

func TestListUsers(t *testing.T) {
	env := setupTest(t, testCfg)
	_ = createUser(t, env, "user1@example.com", "user1", "pass1", "user")
	_ = createUser(t, env, "user2@example.com", "user2", "pass2", "user")
	_ = createUser(t, env, "admin@example.com", "admin", "pass3", "admin")

	rec := doRequest(t, env.router, http.MethodGet, "/api/users", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp []struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}

	decodeJSON(t, rec, &resp)

	if len(resp) != 3 {
		t.Fatalf("expected 3 users, got %d", len(resp))
	}

	if resp[0].Username != "user1" || resp[1].Username != "user2" || resp[2].Username != "admin" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupTest(t, testCfg)
		user := createUser(t, env, "user1@example.com", "user1", "pass1", "user")

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(user.ID), nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		}

		decodeJSON(t, rec, &resp)

		if resp.ID != user.ID || resp.Username != "user1" || resp.Role != "user" {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("not found", func(t *testing.T) {
		env := setupTest(t, testCfg)

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/999", nil, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		env := setupTest(t, testCfg)

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/invalid", nil, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestGetUserSolved(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := setupTest(t, testCfg)
		user := createUser(t, env, "user1@example.com", "user1", "pass1", "user")
		challenge := createChallenge(t, env, "Warmup", 100, "flag{ok}", true)
		createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(user.ID)+"/solved", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp []struct {
			ChallengeID int64  `json:"challenge_id"`
			Title       string `json:"title"`
			Points      int    `json:"points"`
			SolvedAt    string `json:"solved_at"`
		}

		decodeJSON(t, rec, &resp)

		if len(resp) != 1 {
			t.Fatalf("expected 1 solved challenge, got %d", len(resp))
		}

		if resp[0].ChallengeID != challenge.ID || resp[0].Title != "Warmup" || resp[0].Points != 100 {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		env := setupTest(t, testCfg)
		user := createUser(t, env, "user1@example.com", "user1", "pass1", "user")

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/"+itoa(user.ID)+"/solved", nil, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}

		var resp []interface{}
		decodeJSON(t, rec, &resp)

		if len(resp) != 0 {
			t.Fatalf("expected empty list, got %d", len(resp))
		}
	})

	t.Run("not found", func(t *testing.T) {
		env := setupTest(t, testCfg)

		rec := doRequest(t, env.router, http.MethodGet, "/api/users/999/solved", nil, nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})
}
