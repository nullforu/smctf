package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
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
	"smctf/internal/storage"
	"smctf/internal/utils"
	"smctf/internal/vm"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type testEnv struct {
	cfg               config.Config
	router            *gin.Engine
	userRepo          *repo.UserRepo
	regKeyRepo        *repo.RegistrationKeyRepo
	divisionRepo      *repo.DivisionRepo
	teamRepo          *repo.TeamRepo
	challengeRepo     *repo.ChallengeRepo
	submissionRepo    *repo.SubmissionRepo
	appConfigRepo     *repo.AppConfigRepo
	vmRepo            *repo.VMRepo
	authSvc           *service.AuthService
	ctfSvc            *service.CTFService
	divisionSvc       *service.DivisionService
	teamSvc           *service.TeamService
	appConfigSvc      *service.AppConfigService
	vmSvc             *service.VMService
	defaultDivisionID int64
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
	TeamID            int64      `json:"team_id"`
	TeamName          string     `json:"team_name"`
	MaxUses           int        `json:"max_uses"`
	UsedCount         int        `json:"used_count"`
	CreatedAt         time.Time  `json:"created_at"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	Uses              []struct {
		UsedBy         int64     `json:"used_by"`
		UsedByUsername string    `json:"used_by_username"`
		UsedByIP       string    `json:"used_by_ip"`
		UsedAt         time.Time `json:"used_at"`
	} `json:"uses"`
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

const (
	testRegistrationCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	testRegistrationCodeLength   = 16
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
		AppEnv:          "test",
		HTTPAddr:        ":0",
		ShutdownTimeout: 5 * time.Second,
		AutoMigrate:     false,
		BcryptCost:      bcrypt.MinCost,
		DB:              dbCfg,
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
			SubmissionWindow: 2 * time.Minute,
			SubmissionMax:    5,
		},
		Cache: config.CacheConfig{
			TimelineTTL:    2 * time.Minute,
			LeaderboardTTL: 2 * time.Minute,
			AppConfigTTL:   2 * time.Minute,
		},
		Logging: config.LoggingConfig{
			Dir:          "",
			FilePrefix:   "test",
			MaxBodyBytes: 1024 * 1024,
		},
		VM: config.VMConfig{
			Enabled:      true,
			MaxPer:       3,
			CreateWindow: time.Minute,
			CreateMax:    1,
		},
	}

	logDir, err = os.MkdirTemp("", "smctf-logs-*")
	if err != nil {
		panic(err)
	}

	testCfg.Logging.Dir = logDir

	testLogger, err = logging.New(testCfg.Logging, logging.Options{Service: "smctf", Env: "test"})
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
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp").SkipExternalCheck(),
			wait.ForLog("database system is ready to accept connections"),
		),
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
	divisionRepo := repo.NewDivisionRepo(testDB)
	teamRepo := repo.NewTeamRepo(testDB)
	challengeRepo := repo.NewChallengeRepo(testDB)
	submissionRepo := repo.NewSubmissionRepo(testDB)
	scoreRepo := repo.NewScoreboardRepo(testDB)
	appConfigRepo := repo.NewAppConfigRepo(testDB)
	vmRepo := repo.NewVMRepo(testDB)

	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)

	authSvc := service.NewAuthService(cfg, testDB, userRepo, registrationKeyRepo, teamRepo, testRedis)
	userSvc := service.NewUserService(userRepo, teamRepo)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	divisionSvc := service.NewDivisionService(divisionRepo)
	teamSvc := service.NewTeamService(teamRepo, divisionRepo)
	ctfSvc := service.NewCTFService(cfg, challengeRepo, submissionRepo, testRedis, fileStore)
	appConfigSvc := service.NewAppConfigService(appConfigRepo, testRedis, cfg.Cache.AppConfigTTL)
	vmSvc := service.NewVMService(cfg.VM, vmRepo, challengeRepo, submissionRepo, &vm.MockClient{}, testRedis)

	router := apphttp.NewRouter(cfg, authSvc, ctfSvc, appConfigSvc, userSvc, scoreSvc, divisionSvc, teamSvc, vmSvc, nil, testRedis, testLogger, nil)

	env := testEnv{
		cfg:            cfg,
		router:         router,
		userRepo:       userRepo,
		regKeyRepo:     registrationKeyRepo,
		divisionRepo:   divisionRepo,
		teamRepo:       teamRepo,
		challengeRepo:  challengeRepo,
		submissionRepo: submissionRepo,
		appConfigRepo:  appConfigRepo,
		vmRepo:         vmRepo,
		authSvc:        authSvc,
		ctfSvc:         ctfSvc,
		divisionSvc:    divisionSvc,
		teamSvc:        teamSvc,
		appConfigSvc:   appConfigSvc,
		vmSvc:          vmSvc,
	}

	division := &models.Division{
		Name:      "Default",
		CreatedAt: time.Now().UTC(),
	}
	if err := divisionRepo.Create(context.Background(), division); err != nil {
		t.Fatalf("create division: %v", err)
	}

	env.defaultDivisionID = division.ID

	return env
}

func setupVMTest(t *testing.T, cfg config.Config, client vm.API) testEnv {
	t.Helper()
	skipIfIntegrationDisabled(t)
	resetState(t)

	userRepo := repo.NewUserRepo(testDB)
	registrationKeyRepo := repo.NewRegistrationKeyRepo(testDB)
	divisionRepo := repo.NewDivisionRepo(testDB)
	teamRepo := repo.NewTeamRepo(testDB)
	challengeRepo := repo.NewChallengeRepo(testDB)
	submissionRepo := repo.NewSubmissionRepo(testDB)
	scoreRepo := repo.NewScoreboardRepo(testDB)
	appConfigRepo := repo.NewAppConfigRepo(testDB)
	vmRepo := repo.NewVMRepo(testDB)

	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)

	authSvc := service.NewAuthService(cfg, testDB, userRepo, registrationKeyRepo, teamRepo, testRedis)
	userSvc := service.NewUserService(userRepo, teamRepo)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	divisionSvc := service.NewDivisionService(divisionRepo)
	teamSvc := service.NewTeamService(teamRepo, divisionRepo)
	ctfSvc := service.NewCTFService(cfg, challengeRepo, submissionRepo, testRedis, fileStore)
	appConfigSvc := service.NewAppConfigService(appConfigRepo, testRedis, cfg.Cache.AppConfigTTL)
	vmSvc := service.NewVMService(cfg.VM, vmRepo, challengeRepo, submissionRepo, client, testRedis)

	router := apphttp.NewRouter(cfg, authSvc, ctfSvc, appConfigSvc, userSvc, scoreSvc, divisionSvc, teamSvc, vmSvc, nil, testRedis, testLogger, nil)

	env := testEnv{
		cfg:            cfg,
		router:         router,
		userRepo:       userRepo,
		regKeyRepo:     registrationKeyRepo,
		divisionRepo:   divisionRepo,
		teamRepo:       teamRepo,
		challengeRepo:  challengeRepo,
		submissionRepo: submissionRepo,
		appConfigRepo:  appConfigRepo,
		vmRepo:         vmRepo,
		authSvc:        authSvc,
		ctfSvc:         ctfSvc,
		divisionSvc:    divisionSvc,
		teamSvc:        teamSvc,
		appConfigSvc:   appConfigSvc,
		vmSvc:          vmSvc,
	}

	division := &models.Division{Name: "Default", CreatedAt: time.Now().UTC()}
	if err := divisionRepo.Create(context.Background(), division); err != nil {
		t.Fatalf("create division: %v", err)
	}
	env.defaultDivisionID = division.ID

	return env
}

func setCTFWindow(t *testing.T, env testEnv, startAt, endAt *time.Time) {
	t.Helper()

	var startValue service.AppConfigUpdateInput
	if startAt != nil {
		value := startAt.UTC().Format(time.RFC3339)
		startValue = service.AppConfigUpdateInput{Set: true, Value: value}
	} else {
		startValue = service.AppConfigUpdateInput{Set: true, Null: true}
	}

	var endValue service.AppConfigUpdateInput
	if endAt != nil {
		value := endAt.UTC().Format(time.RFC3339)
		endValue = service.AppConfigUpdateInput{Set: true, Value: value}
	} else {
		endValue = service.AppConfigUpdateInput{Set: true, Null: true}
	}

	if _, _, _, err := env.appConfigSvc.Update(context.Background(), service.AppConfigUpdate{
		CTFStartAt: startValue,
		CTFEndAt:   endValue,
	}); err != nil {
		t.Fatalf("set ctf window: %v", err)
	}
}

func resetState(t *testing.T) {
	t.Helper()

	if _, err := testDB.ExecContext(context.Background(), "TRUNCATE TABLE app_configs, submissions, registration_key_uses, registration_keys, vms, challenges, discord_connections, users, teams, divisions RESTART IDENTITY CASCADE"); err != nil {
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

func doRequest(t *testing.T, router *gin.Engine, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
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

	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete {
		if req.Header.Get("X-CSRF-Token") == "" {
			if cookieHeader := req.Header.Get("Cookie"); strings.Contains(cookieHeader, "csrf_token=") {
				for part := range strings.SplitSeq(cookieHeader, ";") {
					trimmed := strings.TrimSpace(part)
					if after, ok := strings.CutPrefix(trimmed, "csrf_token="); ok {
						req.Header.Set("X-CSRF-Token", after)
						break
					}
				}
			}
		}
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, dest any) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func authHeader(token string) map[string]string {
	return map[string]string{"Cookie": "access_token=" + token + "; csrf_token=test-csrf", "X-CSRF-Token": "test-csrf"}
}

func refreshHeader(token string) map[string]string {
	return map[string]string{"Cookie": "refresh_token=" + token + "; csrf_token=test-csrf", "X-CSRF-Token": "test-csrf"}
}

func cookieValueFromSetCookie(rec *httptest.ResponseRecorder, name string) string {
	prefix := name + "="
	for _, c := range rec.Header().Values("Set-Cookie") {
		if after, ok := strings.CutPrefix(c, prefix); ok {
			return strings.SplitN(after, ";", 2)[0]
		}
	}

	return ""
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
		User struct {
			ID       int64  `json:"id"`
			TeamID   int64  `json:"team_id"`
			TeamName string `json:"team_name"`
			VMCount  int    `json:"vm_count"`
			VMLimit  int    `json:"vm_limit"`
		} `json:"user"`
	}

	decodeJSON(t, rec, &loginResp)
	if loginResp.User.TeamID == 0 || loginResp.User.TeamName == "" {
		t.Fatalf("missing team fields in login response")
	}

	if loginResp.User.VMCount != 0 {
		t.Fatalf("expected vm_count 0, got %d", loginResp.User.VMCount)
	}

	if loginResp.User.VMLimit != env.cfg.VM.MaxPer {
		t.Fatalf("expected vm_limit %d, got %d", env.cfg.VM.MaxPer, loginResp.User.VMLimit)
	}

	accessToken := cookieValueFromSetCookie(rec, "access_token")
	refreshToken := cookieValueFromSetCookie(rec, "refresh_token")
	if accessToken == "" || refreshToken == "" {
		t.Fatalf("missing auth cookies from login response")
	}

	return accessToken, refreshToken, loginResp.User.ID
}

func createUser(t *testing.T, env testEnv, email, username, password, role string) *models.User {
	t.Helper()
	team := createTeam(t, env, "team-"+username)
	return createUserWithTeam(t, env, email, username, password, role, team.ID)
}

func createUserWithTeam(t *testing.T, env testEnv, email, username, password, role string, teamID int64) *models.User {
	t.Helper()

	hash, err := auth.HashPassword(password, env.cfg.BcryptCost)
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
		Name:       name,
		DivisionID: env.defaultDivisionID,
		CreatedAt:  time.Now().UTC(),
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
		User struct {
			ID       int64  `json:"id"`
			TeamID   int64  `json:"team_id"`
			TeamName string `json:"team_name"`
			VMCount  int    `json:"vm_count"`
			VMLimit  int    `json:"vm_limit"`
		} `json:"user"`
	}

	decodeJSON(t, rec, &resp)
	if resp.User.TeamID == 0 || resp.User.TeamName == "" {
		t.Fatalf("missing team fields in login response")
	}

	accessToken := cookieValueFromSetCookie(rec, "access_token")
	refreshToken := cookieValueFromSetCookie(rec, "refresh_token")
	if accessToken == "" || refreshToken == "" {
		t.Fatalf("missing auth cookies from login response")
	}

	return accessToken, refreshToken, resp.User.ID
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

	return createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
}

func nextRegistrationCode() string {
	value := atomic.AddInt64(&regKeyCounter, 1)
	return formatRegistrationCode(uint64(value))
}

func formatRegistrationCode(value uint64) string {
	alphabet := testRegistrationCodeAlphabet
	base := uint64(len(alphabet))
	out := make([]byte, testRegistrationCodeLength)

	for i := testRegistrationCodeLength - 1; i >= 0; i-- {
		out[i] = alphabet[value%base]
		value /= base
	}

	return string(out)
}

func createRegistrationKey(t *testing.T, env testEnv, createdBy int64) *models.RegistrationKey {
	t.Helper()
	team := createTeam(t, env, "reg-"+nextRegistrationCode())

	key := &models.RegistrationKey{
		Code:      nextRegistrationCode(),
		CreatedBy: createdBy,
		TeamID:    team.ID,
		MaxUses:   1,
		UsedCount: 0,
		CreatedAt: time.Now().UTC(),
	}

	if err := env.regKeyRepo.Create(context.Background(), key); err != nil {
		t.Fatalf("create registration key: %v", err)
	}

	return key
}

func createRegistrationKeyWithTeam(t *testing.T, env testEnv, createdBy int64, teamID int64) *models.RegistrationKey {
	t.Helper()

	key := &models.RegistrationKey{
		Code:      nextRegistrationCode(),
		CreatedBy: createdBy,
		TeamID:    teamID,
		MaxUses:   1,
		UsedCount: 0,
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
		IsActive:      active,
		CreatedAt:     time.Now().UTC(),
	}

	hash, err := utils.HashFlag(flag, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash flag: %v", err)
	}

	challenge.FlagHash = hash

	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func createVMChallenge(t *testing.T, env testEnv, title string) *models.Challenge {
	t.Helper()
	challenge := createChallenge(t, env, title, 100, "flag{vm}", true)
	spec := "apiVersion: v1\nkind: Sandbox\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx\n"
	challenge.VMEnabled = true
	challenge.VMSpec = &spec
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update vm challenge: %v", err)
	}
	return challenge
}

func createSubmission(t *testing.T, env testEnv, userID, challengeID int64, correct bool, submittedAt time.Time) {
	t.Helper()

	sub := &models.Submission{
		UserID:      userID,
		ChallengeID: challengeID,
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
