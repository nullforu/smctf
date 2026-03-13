package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/models"
	"smctf/internal/stack"
)

func createStack(t *testing.T, env repoEnv, userID, challengeID int64, stackID string, createdAt time.Time) *models.Stack {
	t.Helper()
	stack := &models.Stack{
		UserID:      userID,
		ChallengeID: challengeID,
		StackID:     stackID,
		Status:      "running",
		Ports:       stack.PortMappings{{ContainerPort: 80, Protocol: "TCP", NodePort: 31001}},
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

	if got.Username != user.Username {
		t.Fatalf("expected username %s, got %s", user.Username, got.Username)
	}

	if got.ChallengeTitle != challenge.Title {
		t.Fatalf("expected challenge title %s, got %s", challenge.Title, got.ChallengeTitle)
	}

	got, err = env.stackRepo.GetByStackID(context.Background(), stack.StackID)
	if err != nil {
		t.Fatalf("GetByStackID: %v", err)
	}

	if got.ID != stack.ID {
		t.Fatalf("expected stack id %d, got %d", stack.ID, got.ID)
	}

	if got.Username != user.Username {
		t.Fatalf("expected username %s, got %s", user.Username, got.Username)
	}

	if got.ChallengeTitle != challenge.Title {
		t.Fatalf("expected challenge title %s, got %s", challenge.Title, got.ChallengeTitle)
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

	if stacks[0].Username != user.Username {
		t.Fatalf("expected username %s, got %s", user.Username, stacks[0].Username)
	}

	if stacks[0].ChallengeTitle != challenge.Title {
		t.Fatalf("expected challenge title %s, got %s", challenge.Title, stacks[0].ChallengeTitle)
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

func TestStackRepoCountByUserExcludingStatuses(t *testing.T) {
	env := setupRepoTest(t)

	user := createUserWithNewTeam(t, env, "count@example.com", "count", "pass", models.UserRole)
	challenge := createChallenge(t, env, "CountCh", 100, "flag{count}", true)
	terminalChallenge := createChallenge(t, env, "CountChTerm", 100, "flag{count2}", true)
	if terminalChallenge.ID == challenge.ID {
		terminalChallenge = createChallenge(t, env, "CountChTerm2", 100, "flag{count3}", true)
	}

	now := time.Now().UTC()
	createStack(t, env, user.ID, challenge.ID, "stack-running", now)

	terminal := &models.Stack{
		UserID:      user.ID,
		ChallengeID: terminalChallenge.ID,
		StackID:     "stack-stopped",
		Status:      "stopped",
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now.Add(-time.Minute),
	}
	if err := env.stackRepo.Create(context.Background(), terminal); err != nil {
		t.Fatalf("create terminal stack: %v", err)
	}

	count, err := env.stackRepo.CountByUserExcludingStatuses(context.Background(), user.ID, []string{"stopped"})
	if err != nil {
		t.Fatalf("CountByUserExcludingStatuses: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	count, err = env.stackRepo.CountByUserExcludingStatuses(context.Background(), user.ID, nil)
	if err != nil {
		t.Fatalf("CountByUserExcludingStatuses nil: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}

func TestStackRepoListByTeam(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "ListTeam")
	userA := createUserWithTeam(t, env, "teamA@example.com", "teamA", "pass", models.UserRole, team.ID)
	userB := createUserWithTeam(t, env, "teamB@example.com", "teamB", "pass", models.UserRole, team.ID)
	otherTeam := createTeam(t, env, "OtherTeam")
	otherUser := createUserWithTeam(t, env, "other@example.com", "other", "pass", models.UserRole, otherTeam.ID)

	challengeA := createChallenge(t, env, "TeamCh1", 100, "flag{ta}", true)
	challengeB := createChallenge(t, env, "TeamCh2", 100, "flag{tb}", true)
	otherChallenge := createChallenge(t, env, "OtherCh", 100, "flag{tc}", true)

	createStack(t, env, userA.ID, challengeA.ID, "stack-team-1", time.Now().UTC().Add(-time.Minute))
	createStack(t, env, userB.ID, challengeB.ID, "stack-team-2", time.Now().UTC())
	createStack(t, env, otherUser.ID, otherChallenge.ID, "stack-other", time.Now().UTC())

	stacks, err := env.stackRepo.ListByTeam(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("ListByTeam: %v", err)
	}

	if len(stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(stacks))
	}

	if stacks[0].StackID != "stack-team-2" {
		t.Fatalf("expected newest team stack first, got %s", stacks[0].StackID)
	}

	if stacks[0].Username == "" || stacks[0].ChallengeTitle == "" {
		t.Fatalf("expected username and challenge title set, got %+v", stacks[0])
	}
}

func TestStackRepoGetByTeamAndChallenge(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "TeamGet")
	user := createUserWithTeam(t, env, "teamget@example.com", "teamget", "pass", models.UserRole, team.ID)
	challenge := createChallenge(t, env, "TeamGetCh", 100, "flag{tg}", true)

	createStack(t, env, user.ID, challenge.ID, "stack-team-get", time.Now().UTC())

	got, err := env.stackRepo.GetByTeamAndChallenge(context.Background(), team.ID, challenge.ID)
	if err != nil {
		t.Fatalf("GetByTeamAndChallenge: %v", err)
	}

	if got.StackID != "stack-team-get" {
		t.Fatalf("expected stack-team-get, got %s", got.StackID)
	}

	if got.Username != user.Username || got.ChallengeTitle != challenge.Title {
		t.Fatalf("expected username %s and title %s, got %+v", user.Username, challenge.Title, got)
	}
}

func TestStackRepoCountByTeamExcludingStatuses(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "CountTeam")
	user := createUserWithTeam(t, env, "countteam@example.com", "countteam", "pass", models.UserRole, team.ID)
	challenge := createChallenge(t, env, "CountTeamCh", 100, "flag{ct}", true)
	terminalChallenge := createChallenge(t, env, "CountTeamChTerm", 100, "flag{ct2}", true)

	now := time.Now().UTC()
	createStack(t, env, user.ID, challenge.ID, "stack-team-running", now)

	terminal := &models.Stack{
		UserID:      user.ID,
		ChallengeID: terminalChallenge.ID,
		StackID:     "stack-team-stopped",
		Status:      "stopped",
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now.Add(-time.Minute),
	}
	if err := env.stackRepo.Create(context.Background(), terminal); err != nil {
		t.Fatalf("create terminal stack: %v", err)
	}

	count, err := env.stackRepo.CountByTeamExcludingStatuses(context.Background(), team.ID, []string{"stopped"})
	if err != nil {
		t.Fatalf("CountByTeamExcludingStatuses: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
}

func TestStackRepoTeamIDForUser(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "TeamLookup")
	user := createUserWithTeam(t, env, "lookup@example.com", "lookup", "pass", models.UserRole, team.ID)

	teamID, err := env.stackRepo.TeamIDForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("TeamIDForUser: %v", err)
	}

	if teamID != team.ID {
		t.Fatalf("expected team id %d, got %d", team.ID, teamID)
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
