package repo

import (
	"context"
	"smctf/internal/models"
	"testing"
	"time"
)

func TestDynamicPointsMapUsesTeamDecay(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "Alpha")
	userTeam := createUserWithTeam(t, env, "team@example.com", "team", "pass", models.UserRole, team.ID)
	_ = createUserWithTeam(t, env, "blocked@example.com", "blocked", "pass", models.BlockedRole, team.ID)
	_ = createUserWithNewTeam(t, env, "solo@example.com", "solo", "pass", models.UserRole)

	challenge := createChallenge(t, env, "Dynamic", 500, "FLAG{DYN}", true)
	challenge.MinimumPoints = 100
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge minimum: %v", err)
	}

	createSubmission(t, env, userTeam.ID, challenge.ID, true, time.Now().UTC())

	points, err := dynamicPointsMap(context.Background(), env.db)
	if err != nil {
		t.Fatalf("dynamicPointsMap: %v", err)
	}

	got := points[challenge.ID]
	if got != 400 {
		t.Fatalf("expected 400 with decay=2 and solves=1, got %d", got)
	}
}

func TestDynamicPointsMapIgnoresBlockedAndAdmin(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "Alpha")
	userTeam := createUserWithTeam(t, env, "team@example.com", "team", "pass", models.UserRole, team.ID)
	_ = createUserWithTeam(t, env, "blocked1@example.com", "blocked1", "pass", models.BlockedRole, team.ID)

	admin := createUserWithNewTeam(t, env, "admin@example.com", "admin", "pass", models.AdminRole)
	blocked := createUserWithNewTeam(t, env, "blocked2@example.com", "blocked", "pass", models.BlockedRole)

	challenge := createChallenge(t, env, "Dynamic", 500, "FLAG{DYN}", true)
	challenge.MinimumPoints = 100
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge minimum: %v", err)
	}

	createSubmission(t, env, userTeam.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, admin.ID, challenge.ID, true, time.Now().UTC())
	createSubmission(t, env, blocked.ID, challenge.ID, true, time.Now().UTC())

	countsByChallenge, err := solveCountsByChallenge(context.Background(), env.db)
	if err != nil {
		t.Fatalf("solveCountsByChallenge: %v", err)
	}

	got := countsByChallenge[challenge.ID]
	if got != 1 {
		t.Fatalf("expected 1 solve count (blocked/admin should be ignored), got %d", got)
	}

	points, err := dynamicPointsMap(context.Background(), env.db)
	if err != nil {
		t.Fatalf("dynamicPointsMap: %v", err)
	}

	got = points[challenge.ID]
	if got != 100 {
		t.Fatalf("expected 100 with decay=1 and solves=1 (blocked/admin should be ignored), got %d", got)
	}
}

func TestDynamicPointsMapDecayZeroWhenOnlyBlockedOrAdminTeams(t *testing.T) {
	env := setupRepoTest(t)

	_ = createUserWithNewTeam(t, env, "admin@example.com", "admin", "pass", models.AdminRole)
	blocked := createUserWithNewTeam(t, env, "blocked@example.com", "blocked", "pass", models.BlockedRole)

	challenge := createChallenge(t, env, "Dynamic", 500, "FLAG{DYN}", true)
	challenge.MinimumPoints = 100
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge minimum: %v", err)
	}

	createSubmission(t, env, blocked.ID, challenge.ID, true, time.Now().UTC())

	points, err := dynamicPointsMap(context.Background(), env.db)
	if err != nil {
		t.Fatalf("dynamicPointsMap: %v", err)
	}

	got := points[challenge.ID]
	if got != 500 {
		t.Fatalf("expected initial points with decay=0, got %d", got)
	}
}

func TestDynamicPointsMapDecayCountsDistinctTeams(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "Alpha")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole, team.ID)
	_ = createUserWithTeam(t, env, "u2@example.com", "u2", "pass", models.UserRole, team.ID)
	_ = createUserWithNewTeam(t, env, "blocked@example.com", "blocked", "pass", models.BlockedRole)

	challenge := createChallenge(t, env, "Dynamic", 500, "FLAG{DYN}", true)
	challenge.MinimumPoints = 100
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge minimum: %v", err)
	}

	createSubmission(t, env, user1.ID, challenge.ID, true, time.Now().UTC())

	points, err := dynamicPointsMap(context.Background(), env.db)
	if err != nil {
		t.Fatalf("dynamicPointsMap: %v", err)
	}

	got := points[challenge.ID]
	if got != 100 {
		t.Fatalf("expected 100 with decay=1 and solves=1, got %d", got)
	}
}

func TestDynamicPointsMapZeroSolvesUsesInitialPoints(t *testing.T) {
	env := setupRepoTest(t)

	_ = createUserWithNewTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole)

	challenge := createChallenge(t, env, "Dynamic", 500, "FLAG{DYN}", true)
	challenge.MinimumPoints = 100
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge minimum: %v", err)
	}

	points, err := dynamicPointsMap(context.Background(), env.db)
	if err != nil {
		t.Fatalf("dynamicPointsMap: %v", err)
	}

	got := points[challenge.ID]
	if got != 500 {
		t.Fatalf("expected initial points with zero solves, got %d", got)
	}
}

func TestDynamicPointsMapMinimumGreaterThanInitial(t *testing.T) {
	env := setupRepoTest(t)

	user := createUserWithNewTeam(t, env, "u1@example.com", "u1", "pass", models.UserRole)

	challenge := createChallenge(t, env, "Dynamic", 500, "FLAG{DYN}", true)
	challenge.MinimumPoints = 600
	if err := env.challengeRepo.Update(context.Background(), challenge); err != nil {
		t.Fatalf("update challenge minimum: %v", err)
	}

	createSubmission(t, env, user.ID, challenge.ID, true, time.Now().UTC())

	points, err := dynamicPointsMap(context.Background(), env.db)
	if err != nil {
		t.Fatalf("dynamicPointsMap: %v", err)
	}

	got := points[challenge.ID]
	if got != 500 {
		t.Fatalf("expected initial points when minimum > initial, got %d", got)
	}
}
