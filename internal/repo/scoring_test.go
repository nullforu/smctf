package repo

import (
	"context"
	"testing"
	"time"
)

func TestDynamicPointsMapUsesTeamDecay(t *testing.T) {
	env := setupRepoTest(t)

	team := createTeam(t, env, "Alpha")
	userTeam := createUserWithTeam(t, env, "team@example.com", "team", "pass", "user", team.ID)
	_ = createUser(t, env, "solo@example.com", "solo", "pass", "user")

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
