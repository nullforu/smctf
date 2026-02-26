package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"smctf/internal/config"
	apphttp "smctf/internal/http"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/service"
	"smctf/internal/stack"
	"smctf/internal/storage"
	"smctf/internal/utils"
)

func setupStackTest(t *testing.T, cfg config.Config, client stack.API) testEnv {
	t.Helper()
	skipIfIntegrationDisabled(t)
	resetState(t)

	userRepo := repo.NewUserRepo(testDB)
	registrationKeyRepo := repo.NewRegistrationKeyRepo(testDB)
	teamRepo := repo.NewTeamRepo(testDB)
	challengeRepo := repo.NewChallengeRepo(testDB)
	submissionRepo := repo.NewSubmissionRepo(testDB)
	scoreRepo := repo.NewScoreboardRepo(testDB)
	appConfigRepo := repo.NewAppConfigRepo(testDB)
	stackRepo := repo.NewStackRepo(testDB)

	fileStore := storage.NewMemoryChallengeFileStore(10 * time.Minute)

	authSvc := service.NewAuthService(cfg, testDB, userRepo, registrationKeyRepo, teamRepo, testRedis)
	userSvc := service.NewUserService(userRepo, teamRepo)
	scoreSvc := service.NewScoreboardService(scoreRepo)
	teamSvc := service.NewTeamService(teamRepo)
	ctfSvc := service.NewCTFService(cfg, challengeRepo, submissionRepo, testRedis, fileStore)
	appConfigSvc := service.NewAppConfigService(appConfigRepo, testRedis, cfg.Cache.AppConfigTTL)
	stackSvc := service.NewStackService(cfg.Stack, stackRepo, challengeRepo, submissionRepo, client, testRedis)

	router := apphttp.NewRouter(cfg, authSvc, ctfSvc, appConfigSvc, userSvc, scoreSvc, teamSvc, stackSvc, testRedis, testLogger)

	return testEnv{
		cfg:            cfg,
		router:         router,
		userRepo:       userRepo,
		regKeyRepo:     registrationKeyRepo,
		teamRepo:       teamRepo,
		challengeRepo:  challengeRepo,
		submissionRepo: submissionRepo,
		appConfigRepo:  appConfigRepo,
		authSvc:        authSvc,
		ctfSvc:         ctfSvc,
		teamSvc:        teamSvc,
		appConfigSvc:   appConfigSvc,
	}
}

func createStackChallenge(t *testing.T, env testEnv, title string) *models.Challenge {
	t.Helper()
	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: challenge\nspec:\n  containers:\n    - name: app\n      image: nginx:stable\n      ports:\n        - containerPort: 80\n          protocol: TCP\n"

	challenge := &models.Challenge{
		Title:         title,
		Description:   "stack desc",
		Category:      "Web",
		Points:        100,
		MinimumPoints: 100,
		FlagHash:      utils.HMACFlag(env.cfg.Security.FlagHMACSecret, "flag{stack}"),
		StackEnabled:  true,
		StackTargetPorts: stack.TargetPortSpecs{
			{ContainerPort: 80, Protocol: "TCP"},
		},
		StackPodSpec: &podSpec,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
	}

	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func TestStackLifecycle(t *testing.T) {
	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:      true,
		MaxPerUser:   3,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}

	mock := stack.NewProvisionerMock()
	env := setupStackTest(t, cfg, mock.Client())

	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	user, _, _ := registerAndLogin(t, env, "user@example.com", models.UserRole, "strong-pass")
	challenge := createStackChallenge(t, env, "StackChal")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create stack status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(user))
	if rec.Code != http.StatusOK {
		t.Fatalf("get stack status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(user))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete stack status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStackCreateBlockedAfterSolve(t *testing.T) {
	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:      true,
		MaxPerUser:   3,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}

	mock := stack.NewProvisionerMock()
	env := setupStackTest(t, cfg, mock.Client())

	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	access, _, _ := registerAndLogin(t, env, "user2@example.com", "user2", "strong-pass")
	challenge := createStackChallenge(t, env, "SolvedStack")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/submit", map[string]string{"flag": "flag{stack}"}, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(access))
	if rec.Code != http.StatusConflict {
		t.Fatalf("create stack after solve status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStackCreateRateLimit(t *testing.T) {
	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:      true,
		MaxPerUser:   3,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}

	mock := stack.NewProvisionerMock()
	env := setupStackTest(t, cfg, mock.Client())

	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	access, _, _ := registerAndLogin(t, env, "user3@example.com", "user3", "strong-pass")
	challenge1 := createStackChallenge(t, env, "RateLimit1")
	challenge2 := createStackChallenge(t, env, "RateLimit2")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge1.ID)+"/stack", nil, authHeader(access))
	if rec.Code != http.StatusCreated {
		t.Fatalf("first stack status %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge2.ID)+"/stack", nil, authHeader(access))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStackCreateLocked(t *testing.T) {
	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:      true,
		MaxPerUser:   3,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}

	mock := stack.NewProvisionerMock()
	env := setupStackTest(t, cfg, mock.Client())

	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	access, _, userID := registerAndLogin(t, env, "userlocked@example.com", "userlocked", "strong-pass")
	prev := createChallenge(t, env, "Prev", 50, "flag{prev}", true)
	challenge := createStackChallenge(t, env, "LockedStack")
	challenge.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(access))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("create locked stack status %d: %s", rec.Code, rec.Body.String())
	}

	createSubmission(t, env, userID, prev.ID, true, time.Now().UTC())
	rec = doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(access))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create unlocked stack status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStacksBlockedBeforeStart(t *testing.T) {
	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:      true,
		MaxPerUser:   3,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}

	mock := stack.NewProvisionerMock()
	env := setupStackTest(t, cfg, mock.Client())
	start := time.Now().Add(2 * time.Hour)
	end := time.Now().Add(4 * time.Hour)
	setCTFWindow(t, env, &start, &end)

	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	access, _, _ := registerAndLogin(t, env, "user@example.com", models.UserRole, "strong-pass")
	challenge := createStackChallenge(t, env, "StackChal")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("create stack status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/stacks", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("list stack status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}

	rec = doRequest(t, env.router, http.MethodGet, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("get stack status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}

	rec = doRequest(t, env.router, http.MethodDelete, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete stack status %d: %s", rec.Code, rec.Body.String())
	}

	resp = map[string]any{}
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateNotStarted) {
		t.Fatalf("expected ctf_state not_started, got %v", resp["ctf_state"])
	}
}

func TestStacksCreateBlockedAfterEnd(t *testing.T) {
	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:      true,
		MaxPerUser:   3,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}

	mock := stack.NewProvisionerMock()
	env := setupStackTest(t, cfg, mock.Client())
	end := time.Now().Add(-2 * time.Hour)
	setCTFWindow(t, env, nil, &end)

	_ = createUser(t, env, "admin@example.com", models.AdminRole, "adminpass", models.AdminRole)
	access, _, _ := registerAndLogin(t, env, "user2@example.com", "user2", "strong-pass")
	challenge := createStackChallenge(t, env, "EndedStack")

	rec := doRequest(t, env.router, http.MethodPost, "/api/challenges/"+itoa(challenge.ID)+"/stack", nil, authHeader(access))
	if rec.Code != http.StatusOK {
		t.Fatalf("create stack status %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	decodeJSON(t, rec, &resp)
	if resp["ctf_state"] != string(service.CTFStateEnded) {
		t.Fatalf("expected ctf_state ended, got %v", resp["ctf_state"])
	}
}
