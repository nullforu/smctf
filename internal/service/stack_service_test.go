package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/stack"
	"smctf/internal/utils"
)

func createStackChallenge(t *testing.T, env serviceEnv, title string) *models.Challenge {
	t.Helper()
	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: test\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	challenge := &models.Challenge{
		Title:           title,
		Description:     "desc",
		Category:        "Web",
		Points:          100,
		MinimumPoints:   100,
		FlagHash:        utils.HMACFlag(env.cfg.Security.FlagHMACSecret, "flag"),
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

func newStackService(env serviceEnv, client stack.API, cfg config.StackConfig) (*StackService, *repo.StackRepo) {
	stackRepo := repo.NewStackRepo(env.db)
	return NewStackService(cfg, stackRepo, env.challengeRepo, env.submissionRepo, client, env.redis), stackRepo
}

func TestStackServiceGetOrCreateStack(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createStackChallenge(t, env, "stack")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPerUser:   2,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}
	stackSvc, _ := newStackService(env, mock.Client(), cfg)

	stackModel, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge.ID)
	if err != nil {
		t.Fatalf("GetOrCreateStack: %v", err)
	}

	if stackModel.StackID == "" || stackModel.TargetPort != 80 {
		t.Fatalf("unexpected stack model: %+v", stackModel)
	}

	again, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge.ID)
	if err != nil {
		t.Fatalf("GetOrCreateStack again: %v", err)
	}

	if again.StackID != stackModel.StackID || mock.CreateCount() != 1 {
		t.Fatalf("expected cached stack, calls=%d", mock.CreateCount())
	}
}

func TestStackServiceRateLimit(t *testing.T) {
	env := setupServiceTest(t)
	challenge1 := createStackChallenge(t, env, "stack-1")
	challenge2 := createStackChallenge(t, env, "stack-2")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPerUser:   5,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}
	stackSvc, _ := newStackService(env, mock.Client(), cfg)

	if _, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge1.ID); err != nil {
		t.Fatalf("first create: %v", err)
	}

	if _, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge2.ID); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}

func TestStackServiceUserLimit(t *testing.T) {
	env := setupServiceTest(t)
	challenge1 := createStackChallenge(t, env, "stack-1")
	challenge2 := createStackChallenge(t, env, "stack-2")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPerUser:   1,
		CreateWindow: time.Minute,
		CreateMax:    10,
	}
	stackSvc, _ := newStackService(env, mock.Client(), cfg)

	if _, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge1.ID); err != nil {
		t.Fatalf("first create: %v", err)
	}

	if _, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge2.ID); !errors.Is(err, ErrStackLimitReached) {
		t.Fatalf("expected stack limit error, got %v", err)
	}
}

func TestStackServiceTerminalStatusDeletes(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createStackChallenge(t, env, "stack")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPerUser:   2,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}
	stackSvc, stackRepo := newStackService(env, mock.Client(), cfg)

	stackModel, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	mock.SetStatus(stackModel.StackID, "stopped")

	if _, err := stackSvc.GetStack(context.Background(), 1, challenge.ID); !errors.Is(err, ErrStackNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	if _, err := stackRepo.GetByStackID(context.Background(), stackModel.StackID); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected repo delete, got %v", err)
	}
}

func TestStackServiceAlreadySolvedDeletesExisting(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "stack")

	stackRepo := repo.NewStackRepo(env.db)
	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-solved",
		Status:      "running",
		TargetPort:  80,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{Enabled: true, MaxPerUser: 2, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc := NewStackService(cfg, stackRepo, env.challengeRepo, env.submissionRepo, mock.Client(), env.redis)

	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrAlreadySolved) {
		t.Fatalf("expected already solved, got %v", err)
	}

	if mock.DeleteCount("stack-solved") == 0 {
		t.Fatalf("expected provisioner delete call")
	}

	if _, err := stackRepo.GetByStackID(context.Background(), "stack-solved"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected stack deleted, got %v", err)
	}
}

func TestStackServiceDeleteStackByStackID(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "admin-del@example.com", "admin-del", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "admin-del-stack")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{Enabled: true, MaxPerUser: 2, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc, stackRepo := newStackService(env, mock.Client(), cfg)

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-del",
		Status:      "running",
		TargetPort:  80,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	mock.AddStack(stack.StackInfo{StackID: "stack-del", Status: "running", TargetPort: 80})

	if err := stackSvc.DeleteStackByStackID(context.Background(), "stack-del"); err != nil {
		t.Fatalf("DeleteStackByStackID: %v", err)
	}

	if mock.DeleteCount("stack-del") == 0 {
		t.Fatalf("expected provisioner delete call")
	}

	if _, err := stackRepo.GetByStackID(context.Background(), "stack-del"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected stack deleted, got %v", err)
	}
}

func TestStackServiceGetStackByStackID(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "admin-get@example.com", "admin-get", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "admin-get-stack")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{Enabled: true, MaxPerUser: 2, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc, stackRepo := newStackService(env, mock.Client(), cfg)

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-get",
		Status:      "running",
		TargetPort:  80,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	mock.AddStack(stack.StackInfo{StackID: "stack-get", Status: "running", TargetPort: 80})

	got, err := stackSvc.GetStackByStackID(context.Background(), "stack-get")
	if err != nil {
		t.Fatalf("GetStackByStackID: %v", err)
	}

	if got.StackID != "stack-get" || got.ChallengeID != challenge.ID {
		t.Fatalf("unexpected stack: %+v", got)
	}
}

func TestStackServiceListAdminStacks(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "admin-list@example.com", "admin-list", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "admin-stack")

	mock := stack.NewProvisionerMock()
	stackSvc, stackRepo := newStackService(env, mock.Client(), config.StackConfig{Enabled: true, MaxPerUser: 2, CreateWindow: time.Minute, CreateMax: 5})

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-admin",
		Status:      "running",
		TargetPort:  80,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	stacks, err := stackSvc.ListAdminStacks(context.Background())
	if err != nil {
		t.Fatalf("ListAdminStacks: %v", err)
	}

	if len(stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(stacks))
	}

	if stacks[0].StackID != "stack-admin" {
		t.Fatalf("unexpected stack: %+v", stacks[0])
	}
}

func TestStackServiceListAdminStacksDisabled(t *testing.T) {
	env := setupServiceTest(t)
	mock := stack.NewProvisionerMock()
	stackSvc, _ := newStackService(env, mock.Client(), config.StackConfig{Enabled: false})

	if _, err := stackSvc.ListAdminStacks(context.Background()); !errors.Is(err, ErrStackDisabled) {
		t.Fatalf("expected ErrStackDisabled, got %v", err)
	}
}

func TestStackServiceDeleteStackByStackIDNotFound(t *testing.T) {
	env := setupServiceTest(t)
	mock := stack.NewProvisionerMock()
	stackSvc, _ := newStackService(env, mock.Client(), config.StackConfig{Enabled: true})

	if err := stackSvc.DeleteStackByStackID(context.Background(), "missing"); !errors.Is(err, ErrStackNotFound) {
		t.Fatalf("expected ErrStackNotFound, got %v", err)
	}
}

func TestStackServiceDeleteStackByStackIDProvisionerDown(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "admin-del-down@example.com", "admin-del-down", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "admin-del-down")

	mock := stack.NewProvisionerMock()

	stackSvc, stackRepo := newStackService(env, mock.Client(), config.StackConfig{Enabled: true})

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-down",
		Status:      "running",
		TargetPort:  80,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	mock.SetDeleteError("stack-down", stack.ErrUnavailable)

	if err := stackSvc.DeleteStackByStackID(context.Background(), "stack-down"); !errors.Is(err, ErrStackProvisionerDown) {
		t.Fatalf("expected ErrStackProvisionerDown, got %v", err)
	}
}

func TestStackServiceGetStackByStackIDNotFound(t *testing.T) {
	env := setupServiceTest(t)
	mock := stack.NewProvisionerMock()
	stackSvc, _ := newStackService(env, mock.Client(), config.StackConfig{Enabled: true})

	if _, err := stackSvc.GetStackByStackID(context.Background(), "missing"); !errors.Is(err, ErrStackNotFound) {
		t.Fatalf("expected ErrStackNotFound, got %v", err)
	}
}

func TestStackServiceGetStackByStackIDDisabled(t *testing.T) {
	env := setupServiceTest(t)
	mock := stack.NewProvisionerMock()
	stackSvc, _ := newStackService(env, mock.Client(), config.StackConfig{Enabled: false})

	if _, err := stackSvc.GetStackByStackID(context.Background(), "stack"); !errors.Is(err, ErrStackDisabled) {
		t.Fatalf("expected ErrStackDisabled, got %v", err)
	}
}

func TestStackServiceListAllStacks(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "stack-all@example.com", "stackall", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "stack-all")

	mock := stack.NewProvisionerMock()
	stackSvc, stackRepo := newStackService(env, mock.Client(), config.StackConfig{Enabled: false})

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-all",
		Status:      "running",
		TargetPort:  80,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	stacks, err := stackSvc.ListAllStacks(context.Background())
	if err != nil {
		t.Fatalf("ListAllStacks: %v", err)
	}
	if len(stacks) != 1 || stacks[0].StackID != "stack-all" {
		t.Fatalf("unexpected stacks: %+v", stacks)
	}
}
