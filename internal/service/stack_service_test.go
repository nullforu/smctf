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

	createCalls := 0
	mock := &stack.MockClient{
		CreateStackFn: func(ctx context.Context, targetPort int, podSpec string) (*stack.StackInfo, error) {
			createCalls++
			return &stack.StackInfo{
				StackID:      "stack-1",
				Status:       "running",
				TargetPort:   targetPort,
				NodePort:     31000,
				NodePublicIP: "127.0.0.1",
				TTLExpiresAt: time.Now().UTC().Add(time.Hour),
			}, nil
		},
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{
				StackID:      stackID,
				Status:       "running",
				TargetPort:   80,
				NodePort:     31000,
				NodePublicIP: "127.0.0.1",
				TTL:          time.Now().UTC().Add(time.Hour),
			}, nil
		},
	}

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPerUser:   2,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}
	stackSvc, _ := newStackService(env, mock, cfg)

	stackModel, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge.ID)
	if err != nil {
		t.Fatalf("GetOrCreateStack: %v", err)
	}

	if stackModel.StackID != "stack-1" || stackModel.TargetPort != 80 {
		t.Fatalf("unexpected stack model: %+v", stackModel)
	}

	again, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge.ID)
	if err != nil {
		t.Fatalf("GetOrCreateStack again: %v", err)
	}

	if again.StackID != stackModel.StackID || createCalls != 1 {
		t.Fatalf("expected cached stack, calls=%d", createCalls)
	}
}

func TestStackServiceRateLimit(t *testing.T) {
	env := setupServiceTest(t)
	challenge1 := createStackChallenge(t, env, "stack-1")
	challenge2 := createStackChallenge(t, env, "stack-2")

	mock := &stack.MockClient{
		CreateStackFn: func(ctx context.Context, targetPort int, podSpec string) (*stack.StackInfo, error) {
			return &stack.StackInfo{StackID: "stack-rl", Status: "running", TargetPort: targetPort}, nil
		},
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{StackID: stackID, Status: "running", TargetPort: 80}, nil
		},
	}

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPerUser:   5,
		CreateWindow: time.Minute,
		CreateMax:    1,
	}
	stackSvc, _ := newStackService(env, mock, cfg)

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

	mock := &stack.MockClient{
		CreateStackFn: func(ctx context.Context, targetPort int, podSpec string) (*stack.StackInfo, error) {
			return &stack.StackInfo{StackID: "stack-limit", Status: "running", TargetPort: targetPort}, nil
		},
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{StackID: stackID, Status: "running", TargetPort: 80}, nil
		},
	}

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPerUser:   1,
		CreateWindow: time.Minute,
		CreateMax:    10,
	}
	stackSvc, _ := newStackService(env, mock, cfg)

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

	mock := &stack.MockClient{
		CreateStackFn: func(ctx context.Context, targetPort int, podSpec string) (*stack.StackInfo, error) {
			return &stack.StackInfo{StackID: "stack-term", Status: "running", TargetPort: targetPort}, nil
		},
		GetStackStatusFn: func(ctx context.Context, stackID string) (*stack.StackStatus, error) {
			return &stack.StackStatus{StackID: stackID, Status: "stopped", TargetPort: 80}, nil
		},
		DeleteStackFn: func(ctx context.Context, stackID string) error {
			return nil
		},
	}

	cfg := config.StackConfig{
		Enabled:      true,
		MaxPerUser:   2,
		CreateWindow: time.Minute,
		CreateMax:    5,
	}
	stackSvc, stackRepo := newStackService(env, mock, cfg)

	stackModel, err := stackSvc.GetOrCreateStack(context.Background(), 1, challenge.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := stackSvc.GetStack(context.Background(), 1, challenge.ID); !errors.Is(err, ErrStackNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	if _, err := stackRepo.GetByStackID(context.Background(), stackModel.StackID); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected repo delete, got %v", err)
	}
}

func TestStackServiceAlreadySolvedDeletesExisting(t *testing.T) {
	env := setupServiceTest(t)
	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
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

	deleted := false
	mock := &stack.MockClient{
		DeleteStackFn: func(ctx context.Context, stackID string) error {
			if stackID == "stack-solved" {
				deleted = true
			}
			return nil
		},
	}

	cfg := config.StackConfig{Enabled: true, MaxPerUser: 2, CreateWindow: time.Minute, CreateMax: 5}
	stackSvc := NewStackService(cfg, stackRepo, env.challengeRepo, env.submissionRepo, mock, env.redis)

	if _, err := stackSvc.GetOrCreateStack(context.Background(), user.ID, challenge.ID); !errors.Is(err, ErrAlreadySolved) {
		t.Fatalf("expected already solved, got %v", err)
	}

	if !deleted {
		t.Fatalf("expected provisioner delete call")
	}

	if _, err := stackRepo.GetByStackID(context.Background(), "stack-solved"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected stack deleted, got %v", err)
	}
}
