package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/models"
)

func createStack(t *testing.T, env repoEnv, userID, challengeID int64, stackID string, createdAt time.Time) *models.Stack {
	t.Helper()
	stack := &models.Stack{
		UserID:      userID,
		ChallengeID: challengeID,
		StackID:     stackID,
		Status:      "running",
		Ports:       models.StackPortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
	if err := env.stackRepo.Create(context.Background(), stack); err != nil {
		t.Fatalf("create stack: %v", err)
	}

	return stack
}

func TestStackRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)

	user := createUserWithNewTeam(t, env, "stacker@example.com", "stacker", "pass", models.UserRole)
	challenge := createChallenge(t, env, "Stacked", 100, "flag{1}", true)

	now := time.Now().UTC()
	stack := createStack(t, env, user.ID, challenge.ID, "stack-1", now)

	got, err := env.stackRepo.GetByUserAndChallenge(context.Background(), user.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetByUserAndChallenge: %v", err)
	}

	if got.StackID != stack.StackID {
		t.Fatalf("expected stack %s, got %s", stack.StackID, got.StackID)
	}

	got, err = env.stackRepo.GetByStackID(context.Background(), stack.StackID)
	if err != nil {
		t.Fatalf("GetByStackID: %v", err)
	}

	if got.ID != stack.ID {
		t.Fatalf("expected stack id %d, got %d", stack.ID, got.ID)
	}

	count, err := env.stackRepo.CountByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("CountByUser: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	stacks, err := env.stackRepo.ListByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}

	if len(stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(stacks))
	}

	if stacks[0].StackID != stack.StackID {
		t.Fatalf("expected stack %s, got %s", stack.StackID, stacks[0].StackID)
	}

	if err := env.stackRepo.DeleteByUserAndChallenge(context.Background(), user.ID, challenge.ID); err != nil {
		t.Fatalf("DeleteByUserAndChallenge: %v", err)
	}

	if _, err := env.stackRepo.GetByStackID(context.Background(), stack.StackID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStackRepoListByUserOrdering(t *testing.T) {
	env := setupRepoTest(t)

	user := createUserWithNewTeam(t, env, "order@example.com", "order", "pass", models.UserRole)
	challenge1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	challenge2 := createChallenge(t, env, "Ch2", 100, "flag{2}", true)

	createStack(t, env, user.ID, challenge1.ID, "stack-old", time.Now().UTC().Add(-time.Hour))
	createStack(t, env, user.ID, challenge2.ID, "stack-new", time.Now().UTC())

	stacks, err := env.stackRepo.ListByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}

	if len(stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(stacks))
	}

	if stacks[0].StackID != "stack-new" {
		t.Fatalf("expected newest stack first, got %s", stacks[0].StackID)
	}
}

func TestStackRepoListAll(t *testing.T) {
	env := setupRepoTest(t)

	user := createUserWithNewTeam(t, env, "all@example.com", "all", "pass", models.UserRole)
	challenge1 := createChallenge(t, env, "ChAll1", 100, "flag{all1}", true)
	challenge2 := createChallenge(t, env, "ChAll2", 100, "flag{all2}", true)

	createStack(t, env, user.ID, challenge1.ID, "stack-all-1", time.Now().UTC().Add(-time.Minute))
	createStack(t, env, user.ID, challenge2.ID, "stack-all-2", time.Now().UTC())

	stacks, err := env.stackRepo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}

	if len(stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(stacks))
	}

	if stacks[0].StackID != "stack-all-2" {
		t.Fatalf("expected newest stack first, got %s", stacks[0].StackID)
	}
}

func TestStackRepoListAdmin(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "Alpha")
	user := createUserWithTeam(t, env, "admin-stack@example.com", "adminstack", "pass", models.UserRole, team.ID)
	challenge := createChallenge(t, env, "AdminStack", 200, "flag{admin}", true)

	createStack(t, env, user.ID, challenge.ID, "stack-admin", time.Now().UTC())

	stacks, err := env.stackRepo.ListAdmin(context.Background())
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}

	if len(stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(stacks))
	}

	item := stacks[0]
	if item.StackID != "stack-admin" {
		t.Fatalf("expected stack-admin, got %s", item.StackID)
	}

	if item.Username != user.Username || item.Email != user.Email {
		t.Fatalf("expected user info, got %+v", item)
	}

	if item.TeamName != team.Name {
		t.Fatalf("expected team name %s, got %s", team.Name, item.TeamName)
	}

	if item.ChallengeTitle != challenge.Title || item.ChallengeCategory != challenge.Category {
		t.Fatalf("expected challenge info, got %+v", item)
	}
}

func TestStackRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)
	_, err := env.stackRepo.GetByStackID(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
