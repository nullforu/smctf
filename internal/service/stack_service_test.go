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

	"golang.org/x/crypto/bcrypt"
)

func createStackChallenge(t *testing.T, env serviceEnv, title string) *models.Challenge {
	t.Helper()
	podSpec := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: test\nspec:\n  containers:\n    - name: app\n      image: nginx\n      ports:\n        - containerPort: 80\n"
	challenge := &models.Challenge{
		Title:         title,
		Description:   "desc",
		Category:      "Web",
		Points:        100,
		MinimumPoints: 100,
		StackEnabled:  true,
		StackTargetPorts: stack.TargetPortSpecs{
			{ContainerPort: 80, Protocol: "TCP"},
		},
		StackPodSpec: &podSpec,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
	}
	hash, err := utils.HashFlag("flag", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash flag: %v", err)
	}
	challenge.FlagHash = hash

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
		MaxPer:       2,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}
	stackSvc, _ := newStackService(env, mock.Client(), cfg)

	stackModel, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge.ID)
	if err != nil {
		t.Fatalf("GetOrCreateStack: %v", err)
	}

	if stackModel.StackID == "" || len(stackModel.Ports) != 1 || stackModel.Ports[0].ContainerPort != 80 {
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

func TestStackServiceUserStackSummary(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "stack-summary@example.com", "stack-summary", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "stack-summary")
	terminalChallenge := createStackChallenge(t, env, "stack-summary-term")

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPer:       3,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}
	stackSvc, stackRepo := newStackService(env, stack.NewProvisionerMock().Client(), cfg)

	disabledSvc, _ := newStackService(env, stack.NewProvisionerMock().Client(), config.StackConfig{Enabled: false})
	count, limit, err := disabledSvc.UserStackSummary(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("UserStackSummary disabled: %v", err)
	}

	if count != 0 || limit != 0 {
		t.Fatalf("expected disabled summary 0/0, got %d/%d", count, limit)
	}

	count, limit, err = stackSvc.UserStackSummary(context.Background(), 0)
	if err != nil {
		t.Fatalf("UserStackSummary empty: %v", err)
	}

	if count != 0 || limit != cfg.MaxPer {
		t.Fatalf("expected empty summary 0/%d, got %d/%d", cfg.MaxPer, count, limit)
	}

	now := time.Now().UTC()
	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-summary-1",
		Status:      "running",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	terminal := &models.Stack{
		UserID:      user.ID,
		ChallengeID: terminalChallenge.ID,
		StackID:     "stack-summary-stopped",
		Status:      "stopped",
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now.Add(-time.Minute),
	}
	if err := stackRepo.Create(context.Background(), terminal); err != nil {
		t.Fatalf("create terminal stack: %v", err)
	}

	count, limit, err = stackSvc.UserStackSummary(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("UserStackSummary: %v", err)
	}
	if count != 1 || limit != cfg.MaxPer {
		t.Fatalf("expected summary 1/%d, got %d/%d", cfg.MaxPer, count, limit)
	}
}

func TestStackServiceUserStackSummaryTeamScope(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "team-summary-1@example.com", "team-summary-1", "pass", models.UserRole)
	user2 := createUserWithTeam(t, env, "team-summary-2@example.com", "team-summary-2", "pass", models.UserRole, user.TeamID)
	challenge1 := createStackChallenge(t, env, "team-summary-1")
	challenge2 := createStackChallenge(t, env, "team-summary-2")

	cfg := config.StackConfig{
		Enabled:      true,
		MaxScope:     "team",
		MaxPer:       5,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}
	stackSvc, stackRepo := newStackService(env, stack.NewProvisionerMock().Client(), cfg)

	now := time.Now().UTC()
	stackOne := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge1.ID,
		StackID:     "team-summary-1",
		Status:      "running",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := stackRepo.Create(context.Background(), stackOne); err != nil {
		t.Fatalf("create stack one: %v", err)
	}

	stack2 := &models.Stack{
		UserID:      user2.ID,
		ChallengeID: challenge2.ID,
		StackID:     "team-summary-2",
		Status:      "running",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := stackRepo.Create(context.Background(), stack2); err != nil {
		t.Fatalf("create stack 2: %v", err)
	}

	count, limit, err := stackSvc.UserStackSummary(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("UserStackSummary team: %v", err)
	}

	if count != 2 || limit != cfg.MaxPer {
		t.Fatalf("expected team summary 2/%d, got %d/%d", cfg.MaxPer, count, limit)
	}
}

func TestStackServiceListUserStacksTeamScope(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "team-list-1@example.com", "team-list-1", "pass", models.UserRole)
	user2 := createUserWithTeam(t, env, "team-list-2@example.com", "team-list-2", "pass", models.UserRole, user.TeamID)
	challenge1 := createStackChallenge(t, env, "team-list-1")
	challenge2 := createStackChallenge(t, env, "team-list-2")

	client := &stack.MockClient{
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{StackID: stackID, Status: "running"}, nil
		},
	}

	cfg := config.StackConfig{
		Enabled:      true,
		MaxScope:     "team",
		MaxPer:       5,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}
	stackSvc, stackRepo := newStackService(env, client, cfg)

	now := time.Now().UTC()
	stackOne := &models.Stack{UserID: user.ID, ChallengeID: challenge1.ID, StackID: "team-list-1", Status: "running", CreatedAt: now, UpdatedAt: now}
	stack2 := &models.Stack{UserID: user2.ID, ChallengeID: challenge2.ID, StackID: "team-list-2", Status: "running", CreatedAt: now, UpdatedAt: now}
	if err := stackRepo.Create(context.Background(), stackOne); err != nil {
		t.Fatalf("create stack one: %v", err)
	}

	if err := stackRepo.Create(context.Background(), stack2); err != nil {
		t.Fatalf("create stack 2: %v", err)
	}

	stacks, err := stackSvc.ListUserStacks(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("ListUserStacks team: %v", err)
	}

	if len(stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(stacks))
	}
}

func TestStackServiceGetStackTeamScope(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "team-get-1@example.com", "team-get-1", "pass", models.UserRole)
	user2 := createUserWithTeam(t, env, "team-get-2@example.com", "team-get-2", "pass", models.UserRole, user.TeamID)
	challenge := createStackChallenge(t, env, "team-get")

	client := &stack.MockClient{
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{StackID: stackID, Status: "running"}, nil
		},
	}

	cfg := config.StackConfig{
		Enabled:      true,
		MaxScope:     "team",
		MaxPer:       5,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}
	stackSvc, stackRepo := newStackService(env, client, cfg)

	now := time.Now().UTC()
	stackModel := &models.Stack{UserID: user2.ID, ChallengeID: challenge.ID, StackID: "team-get", Status: "running", CreatedAt: now, UpdatedAt: now}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	got, err := stackSvc.GetStack(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetStack team: %v", err)
	}

	if got.StackID != "team-get" {
		t.Fatalf("expected team stack, got %+v", got)
	}
}

func TestStackServiceCreateStackInvalidPorts(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createStackChallenge(t, env, "stack-invalid")
	challenge.StackTargetPorts = stack.TargetPortSpecs{{ContainerPort: 0, Protocol: "TCP"}}
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	mock := stack.NewProvisionerMock()
	cfg := config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc, _ := newStackService(env, mock.Client(), cfg)

	if _, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge.ID); !errors.Is(err, ErrStackInvalidSpec) {
		t.Fatalf("expected ErrStackInvalidSpec, got %v", err)
	}
}

func TestStackServiceCreateStackProvisionerUnavailable(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createStackChallenge(t, env, "stack-provisioner-down")

	client := &stack.MockClient{
		CreateStackFn: func(ctx context.Context, targetPorts []stack.TargetPortSpec, podSpec string) (*stack.StackInfo, error) {
			return nil, stack.ErrUnavailable
		},
	}
	cfg := config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc, _ := newStackService(env, client, cfg)

	if _, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge.ID); !errors.Is(err, ErrStackProvisionerDown) {
		t.Fatalf("expected ErrStackProvisionerDown, got %v", err)
	}
}

func TestStackServiceLockedChallenge(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "locked-stack@example.com", "locked-stack", "pass", models.UserRole)
	prev := createChallenge(t, env, "Prev", 50, "flag{prev}", true)
	challenge := createStackChallenge(t, env, "stack-locked")
	challenge.PreviousChallengeID = &prev.ID
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge: %v", err)
	}

	mock := stack.NewProvisionerMock()
	cfg := config.StackConfig{
		Enabled:      true,
		MaxPer:       2,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}
	stackSvc, _ := newStackService(env, mock.Client(), cfg)

	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrChallengeLocked) {
		t.Fatalf("expected locked error, got %v", err)
	}

	createSubmission(t, env, user.ID, prev.ID, true, time.Now().UTC())
	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, challenge.ID); err != nil {
		t.Fatalf("expected stack after unlock, got %v", err)
	}
}

func TestStackServiceRateLimit(t *testing.T) {
	env := setupServiceTest(t)
	challenge1 := createStackChallenge(t, env, "stack-1")
	challenge2 := createStackChallenge(t, env, "stack-2")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPer:       5,
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

func TestStackServiceRateLimitTeamScope(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "rate-team-1@example.com", "rate-team-1", "pass", models.UserRole)
	user2 := createUserWithTeam(t, env, "rate-team-2@example.com", "rate-team-2", "pass", models.UserRole, user.TeamID)
	challenge1 := createStackChallenge(t, env, "rate-team-1")
	challenge2 := createStackChallenge(t, env, "rate-team-2")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{
		Enabled:      true,
		MaxScope:     "team",
		MaxPer:       5,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}
	stackSvc, _ := newStackService(env, mock.Client(), cfg)

	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, challenge1.ID); err != nil {
		t.Fatalf("first create: %v", err)
	}

	if _, err := stackSvc.GetOrCreateStack(context.Background(), user2.ID, challenge2.ID); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected team rate limit error, got %v", err)
	}
}

func TestStackServiceUserLimit(t *testing.T) {
	env := setupServiceTest(t)
	challenge1 := createStackChallenge(t, env, "stack-1")
	challenge2 := createStackChallenge(t, env, "stack-2")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPer:       1,
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

func TestStackServiceUserLimitTeamScope(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "limit-team-1@example.com", "limit-team-1", "pass", models.UserRole)
	user2 := createUserWithTeam(t, env, "limit-team-2@example.com", "limit-team-2", "pass", models.UserRole, user.TeamID)
	challenge1 := createStackChallenge(t, env, "limit-team-1")
	challenge2 := createStackChallenge(t, env, "limit-team-2")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{
		Enabled:      true,
		MaxScope:     "team",
		MaxPer:       1,
		CreateWindow: time.Minute,
		CreateMax:    10,
	}
	stackSvc, _ := newStackService(env, mock.Client(), cfg)

	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, challenge1.ID); err != nil {
		t.Fatalf("first create: %v", err)
	}

	if _, err := stackSvc.GetOrCreateStack(context.Background(), user2.ID, challenge2.ID); !errors.Is(err, ErrStackLimitReached) {
		t.Fatalf("expected team stack limit error, got %v", err)
	}
}

func TestStackServiceTerminalStatusDeletes(t *testing.T) {
	env := setupServiceTest(t)
	challenge := createStackChallenge(t, env, "stack")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPer:       2,
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
	user := createUserWithNewTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "stack")

	stackRepo := repo.NewStackRepo(env.db)
	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-solved",
		Status:      "running",
		Ports:       stack.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5}
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
	user := createUserWithNewTeam(t, env, "admin-del@example.com", "admin-del", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "admin-del-stack")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc, stackRepo := newStackService(env, mock.Client(), cfg)

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-del",
		Status:      "running",
		Ports:       stack.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	mock.AddStack(stack.StackInfo{
		StackID: "stack-del",
		Status:  "running",
		Ports:   []stack.PortMapping{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
	})

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
	user := createUserWithNewTeam(t, env, "admin-get@example.com", "admin-get", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "admin-get-stack")

	mock := stack.NewProvisionerMock()

	cfg := config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc, stackRepo := newStackService(env, mock.Client(), cfg)

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-get",
		Status:      "running",
		Ports:       stack.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := stackRepo.Create(context.Background(), stackModel); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	mock.AddStack(stack.StackInfo{
		StackID: "stack-get",
		Status:  "running",
		Ports:   []stack.PortMapping{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
	})

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
	user := createUserWithNewTeam(t, env, "admin-list@example.com", "admin-list", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "admin-stack")

	mock := stack.NewProvisionerMock()
	stackSvc, stackRepo := newStackService(env, mock.Client(), config.StackConfig{Enabled: true, MaxPer: 2, CreateWindow: time.Minute, CreateMax: 5})

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-admin",
		Status:      "running",
		Ports:       stack.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
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
	user := createUserWithNewTeam(t, env, "admin-del-down@example.com", "admin-del-down", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "admin-del-down")

	mock := stack.NewProvisionerMock()

	stackSvc, stackRepo := newStackService(env, mock.Client(), config.StackConfig{Enabled: true})

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-down",
		Status:      "running",
		Ports:       stack.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
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
	user := createUserWithNewTeam(t, env, "stack-all@example.com", "stackall", "pass", models.UserRole)
	challenge := createStackChallenge(t, env, "stack-all")

	mock := stack.NewProvisionerMock()
	stackSvc, stackRepo := newStackService(env, mock.Client(), config.StackConfig{Enabled: false})

	stackModel := &models.Stack{
		UserID:      user.ID,
		ChallengeID: challenge.ID,
		StackID:     "stack-all",
		Status:      "running",
		Ports:       stack.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
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

func TestToTargetPortSpecsValidation(t *testing.T) {
	if _, err := toTargetPortSpecs(nil); !errors.Is(err, ErrStackInvalidSpec) {
		t.Fatalf("expected ErrStackInvalidSpec for empty ports, got %v", err)
	}

	if _, err := toTargetPortSpecs(stack.TargetPortSpecs{{ContainerPort: 70000, Protocol: "TCP"}}); !errors.Is(err, ErrStackInvalidSpec) {
		t.Fatalf("expected ErrStackInvalidSpec for invalid port, got %v", err)
	}

	if _, err := toTargetPortSpecs(stack.TargetPortSpecs{{ContainerPort: 80, Protocol: "icmp"}}); !errors.Is(err, ErrStackInvalidSpec) {
		t.Fatalf("expected ErrStackInvalidSpec for invalid protocol, got %v", err)
	}

	ports, err := toTargetPortSpecs(stack.TargetPortSpecs{{ContainerPort: 80, Protocol: "tcp"}})
	if err != nil {
		t.Fatalf("expected normalized ports, got %v", err)
	}

	if len(ports) != 1 || ports[0].Protocol != "TCP" {
		t.Fatalf("expected TCP normalized, got %+v", ports)
	}
}
