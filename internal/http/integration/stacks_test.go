package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

type provisionerStub struct {
	mu     sync.Mutex
	nextID int
	stacks map[string]stack.StackInfo
}

func newProvisionerStub() *provisionerStub {
	return &provisionerStub{
		nextID: 1,
		stacks: make(map[string]stack.StackInfo),
	}
}

func (p *provisionerStub) handler(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/stacks":
		var req stack.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		p.mu.Lock()
		id := p.nextID
		p.nextID++
		stackID := "stack-test-" + itoa(int64(id))
		now := time.Now().UTC()
		info := stack.StackInfo{
			StackID:      stackID,
			PodID:        stackID,
			Namespace:    "stacks",
			NodeID:       "dev-worker",
			NodePublicIP: "127.0.0.1",
			PodSpec:      req.PodSpec,
			TargetPort:   req.TargetPort,
			NodePort:     31001 + id,
			ServiceName:  "svc-" + stackID,
			Status:       "running",
			TTLExpiresAt: now.Add(2 * time.Hour),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		p.stacks[stackID] = info
		p.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(info)
		return
	case r.Method == http.MethodGet && r.URL.Path == "/stacks":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"stacks": []stack.StackInfo{}})
		return
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/stacks/") && strings.HasSuffix(r.URL.Path, "/status"):
		stackID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/stacks/"), "/status")
		p.mu.Lock()
		info, ok := p.stacks[stackID]
		p.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := stack.StackStatus{
			StackID:      info.StackID,
			Status:       info.Status,
			TTL:          info.TTLExpiresAt,
			NodePort:     info.NodePort,
			TargetPort:   info.TargetPort,
			NodePublicIP: info.NodePublicIP,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
		return
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/stacks/"):
		stackID := strings.TrimPrefix(r.URL.Path, "/stacks/")
		p.mu.Lock()
		_, ok := p.stacks[stackID]
		delete(p.stacks, stackID)
		p.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true, "stack_id": stackID})
		return
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func setupStackTest(t *testing.T, cfg config.Config, client *stack.Client) testEnv {
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
	teamSvc := service.NewTeamService(teamRepo)
	ctfSvc := service.NewCTFService(cfg, challengeRepo, submissionRepo, testRedis, fileStore)
	appConfigSvc := service.NewAppConfigService(appConfigRepo)
	stackSvc := service.NewStackService(cfg.Stack, stackRepo, challengeRepo, submissionRepo, client, testRedis)

	router := apphttp.NewRouter(cfg, authSvc, ctfSvc, appConfigSvc, userRepo, scoreRepo, teamSvc, stackSvc, testRedis, testLogger)

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
		Title:           title,
		Description:     "stack desc",
		Category:        "Web",
		Points:          100,
		MinimumPoints:   100,
		FlagHash:        utils.HMACFlag(env.cfg.Security.FlagHMACSecret, "flag{stack}"),
		StackEnabled:    true,
		StackTargetPort: 80,
		StackPodSpec:    &podSpec,
		IsActive:        true,
		CreatedAt:       time.Now().UTC(),
	}

	if err := env.challengeRepo.Create(context.Background(), challenge); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	return challenge
}

func TestStackLifecycle(t *testing.T) {
	stub := newProvisionerStub()
	server := httptest.NewServer(http.HandlerFunc(stub.handler))
	defer server.Close()

	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:            true,
		MaxPerUser:         3,
		ProvisionerBaseURL: server.URL,
		ProvisionerAPIKey:  "test-key",
		ProvisionerTimeout: 2 * time.Second,
		CreateWindow:       time.Minute,
		CreateMax:          1,
	}

	client := stack.NewClient(cfg.Stack.ProvisionerBaseURL, cfg.Stack.ProvisionerAPIKey, cfg.Stack.ProvisionerTimeout)
	env := setupStackTest(t, cfg, client)

	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")
	user, _, _ := registerAndLogin(t, env, "user@example.com", "user", "strong-pass")
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
	stub := newProvisionerStub()
	server := httptest.NewServer(http.HandlerFunc(stub.handler))
	defer server.Close()

	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:            true,
		MaxPerUser:         3,
		ProvisionerBaseURL: server.URL,
		ProvisionerAPIKey:  "test-key",
		ProvisionerTimeout: 2 * time.Second,
		CreateWindow:       time.Minute,
		CreateMax:          1,
	}

	client := stack.NewClient(cfg.Stack.ProvisionerBaseURL, cfg.Stack.ProvisionerAPIKey, cfg.Stack.ProvisionerTimeout)
	env := setupStackTest(t, cfg, client)

	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")
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
	stub := newProvisionerStub()
	server := httptest.NewServer(http.HandlerFunc(stub.handler))
	defer server.Close()

	cfg := testCfg
	cfg.Stack = config.StackConfig{
		Enabled:            true,
		MaxPerUser:         3,
		ProvisionerBaseURL: server.URL,
		ProvisionerAPIKey:  "test-key",
		ProvisionerTimeout: 2 * time.Second,
		CreateWindow:       time.Minute,
		CreateMax:          1,
	}

	client := stack.NewClient(cfg.Stack.ProvisionerBaseURL, cfg.Stack.ProvisionerAPIKey, cfg.Stack.ProvisionerTimeout)
	env := setupStackTest(t, cfg, client)

	_ = createUser(t, env, "admin@example.com", "admin", "adminpass", "admin")
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
