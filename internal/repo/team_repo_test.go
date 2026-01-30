package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/models"
)

func TestTeamRepoCRUD(t *testing.T) {
	env := setupRepoTest(t)

	team := &models.Team{
		Name:      "Alpha School",
		CreatedAt: time.Now().UTC(),
	}

	if err := env.teamRepo.Create(context.Background(), team); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := env.teamRepo.GetByID(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.ID != team.ID || got.Name != team.Name {
		t.Fatalf("unexpected team: %+v", got)
	}

	list, err := env.teamRepo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("expected 1 team, got %d", len(list))
	}
}

func TestTeamRepoNotFound(t *testing.T) {
	env := setupRepoTest(t)

	_, err := env.teamRepo.GetByID(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTeamRepoListWithStats(t *testing.T) {
	env := setupRepoTest(t)

	teamA := createTeam(t, env, "Alpha School")
	teamB := createTeam(t, env, "Beta School")

	userA1 := createUserWithTeam(t, env, "a1@example.com", "alpha1", "pass", "user", &teamA.ID)
	userA2 := createUserWithTeam(t, env, "a2@example.com", "alpha2", "pass", "user", &teamA.ID)
	_ = createUserWithTeam(t, env, "b1@example.com", "beta1", "pass", "user", &teamB.ID)
	_ = createUser(t, env, "noteam@example.com", "noteam", "pass", "user")

	chal1 := createChallenge(t, env, "Basic", 100, "flag{basic}", true)
	chal2 := createChallenge(t, env, "Hard", 200, "flag{hard}", true)

	now := time.Now().UTC()
	createSubmission(t, env, userA1.ID, chal1.ID, true, now.Add(-2*time.Minute))
	createSubmission(t, env, userA2.ID, chal2.ID, true, now.Add(-1*time.Minute))
	createSubmission(t, env, userA1.ID, chal2.ID, false, now.Add(-30*time.Second))

	rows, err := env.teamRepo.ListWithStats(context.Background())
	if err != nil {
		t.Fatalf("ListWithStats: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(rows))
	}

	var gotA, gotB *models.TeamSummary
	for i := range rows {
		switch rows[i].ID {
		case teamA.ID:
			gotA = &rows[i]
		case teamB.ID:
			gotB = &rows[i]
		}
	}

	if gotA == nil || gotB == nil {
		t.Fatalf("missing team rows: %+v", rows)
	}

	if gotA.MemberCount != 2 || gotA.TotalScore != 300 {
		t.Fatalf("unexpected alpha stats: %+v", *gotA)
	}

	if gotB.MemberCount != 1 || gotB.TotalScore != 0 {
		t.Fatalf("unexpected beta stats: %+v", *gotB)
	}
}

func TestTeamRepoGetStats(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "Gamma School")
	user := createUserWithTeam(t, env, "g1@example.com", "gamma1", "pass", "user", &team.ID)
	chal := createChallenge(t, env, "Gamma", 150, "flag{gamma}", true)
	createSubmission(t, env, user.ID, chal.ID, true, time.Now().UTC())

	row, err := env.teamRepo.GetStats(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	if row.ID != team.ID || row.MemberCount != 1 || row.TotalScore != 150 {
		t.Fatalf("unexpected stats: %+v", row)
	}
}

func TestTeamRepoGetStatsNotFound(t *testing.T) {
	env := setupRepoTest(t)

	_, err := env.teamRepo.GetStats(context.Background(), 404)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTeamRepoListMembers(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "Members School")
	user1 := createUserWithTeam(t, env, "m1@example.com", "member1", "pass", "user", &team.ID)
	user2 := createUserWithTeam(t, env, "m2@example.com", "member2", "pass", "admin", &team.ID)
	_ = createUser(t, env, "other@example.com", "other", "pass", "user")

	rows, err := env.teamRepo.ListMembers(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 members, got %d", len(rows))
	}

	if rows[0].ID != user1.ID || rows[1].ID != user2.ID {
		t.Fatalf("unexpected member order: %+v", rows)
	}

	if rows[0].Username != user1.Username || rows[1].Role != user2.Role {
		t.Fatalf("unexpected member data: %+v", rows)
	}
}

func TestTeamRepoListSolvedChallenges(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "Solves School")
	user1 := createUserWithTeam(t, env, "s1@example.com", "solver1", "pass", "user", &team.ID)
	user2 := createUserWithTeam(t, env, "s2@example.com", "solver2", "pass", "user", &team.ID)

	chal1 := createChallenge(t, env, "Intro", 50, "flag{intro}", true)
	chal2 := createChallenge(t, env, "Advanced", 250, "flag{adv}", true)

	now := time.Now().UTC()
	createSubmission(t, env, user1.ID, chal1.ID, true, now.Add(-3*time.Minute))
	createSubmission(t, env, user2.ID, chal1.ID, false, now.Add(-2*time.Minute))
	createSubmission(t, env, user1.ID, chal2.ID, true, now.Add(-1*time.Minute))
	createSubmission(t, env, user2.ID, chal2.ID, false, now.Add(-30*time.Second))

	rows, err := env.teamRepo.ListSolvedChallenges(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("ListSolvedChallenges: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 solved challenges, got %d", len(rows))
	}

	if rows[0].ChallengeID != chal2.ID || rows[0].SolveCount != 1 || rows[0].Points != 250 {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}

	if rows[1].ChallengeID != chal1.ID || rows[1].SolveCount != 1 || rows[1].Points != 50 {
		t.Fatalf("unexpected second row: %+v", rows[1])
	}

	if !rows[0].LastSolvedAt.After(rows[1].LastSolvedAt) {
		t.Fatalf("expected rows ordered by last_solved_at desc: %+v", rows)
	}
}
