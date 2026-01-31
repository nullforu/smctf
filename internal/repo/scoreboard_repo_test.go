package repo

import (
	"context"
	"testing"
	"time"
)

func TestScoreboardRepoLeaderboardAndTimeline(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	team := createTeam(t, env, "Alpha")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", &team.ID)
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")

	ch1 := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	ch2 := createChallenge(t, env, "ch2", 50, "FLAG{2}", true)

	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-3*time.Minute))
	createSubmission(t, env, user1.ID, ch2.ID, true, time.Now().Add(-2*time.Minute))
	createSubmission(t, env, user2.ID, ch2.ID, false, time.Now().Add(-1*time.Minute))

	leaderboard, err := scoreRepo.Leaderboard(context.Background())
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}

	if len(leaderboard) != 2 {
		t.Fatalf("expected 2 leaderboard rows, got %d", len(leaderboard))
	}

	if leaderboard[0].UserID != user1.ID || leaderboard[0].Score != 150 {
		t.Fatalf("unexpected leaderboard first row: %+v", leaderboard[0])
	}

	if leaderboard[1].UserID != user2.ID || leaderboard[1].Score != 0 {
		t.Fatalf("unexpected leaderboard second row: %+v", leaderboard[1])
	}

	since := time.Now().Add(-2*time.Minute - time.Second)
	rows, err := scoreRepo.TimelineSubmissions(context.Background(), &since)
	if err != nil {
		t.Fatalf("TimelineSubmissions: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 timeline row, got %d", len(rows))
	}

	if rows[0].UserID != user1.ID {
		t.Fatalf("unexpected timeline row: %+v", rows[0])
	}
}

func TestScoreboardRepoTeamLeaderboardAndTimeline(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	teamA := createTeam(t, env, "Alpha")
	teamB := createTeam(t, env, "Beta")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", &teamA.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", &teamB.ID)
	user3 := createUser(t, env, "u3@example.com", "u3", "pass", "user")

	ch1 := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	ch2 := createChallenge(t, env, "ch2", 50, "FLAG{2}", true)

	createSubmission(t, env, user1.ID, ch1.ID, true, time.Now().Add(-3*time.Minute))
	createSubmission(t, env, user2.ID, ch2.ID, true, time.Now().Add(-2*time.Minute))
	createSubmission(t, env, user3.ID, ch2.ID, true, time.Now().Add(-1*time.Minute))

	leaderboard, err := scoreRepo.TeamLeaderboard(context.Background())
	if err != nil {
		t.Fatalf("TeamLeaderboard: %v", err)
	}

	if len(leaderboard) != 3 {
		t.Fatalf("expected 3 team rows, got %d", len(leaderboard))
	}

	if leaderboard[0].TeamName != "Alpha" || leaderboard[0].Score != 100 {
		t.Fatalf("unexpected team leaderboard first row: %+v", leaderboard[0])
	}

	if leaderboard[2].TeamName != "not affiliated" || leaderboard[2].Score != 50 {
		t.Fatalf("unexpected team leaderboard last row: %+v", leaderboard[2])
	}

	rows, err := scoreRepo.TimelineTeamSubmissions(context.Background(), nil)
	if err != nil {
		t.Fatalf("TimelineTeamSubmissions: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("expected 3 team timeline rows, got %d", len(rows))
	}

	if rows[0].TeamName == "" {
		t.Fatalf("expected team name in row")
	}
}

func TestScoreboardRepoTimelineNoSince(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	createSubmission(t, env, user.ID, ch.ID, true, time.Now().Add(-time.Minute))

	rows, err := scoreRepo.TimelineSubmissions(context.Background(), nil)
	if err != nil {
		t.Fatalf("TimelineSubmissions: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
}

func TestScoreboardRepoTimelineOrdering(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	now := time.Now().UTC()
	createSubmission(t, env, user.ID, ch.ID, true, now.Add(-2*time.Minute))
	createSubmission(t, env, user.ID, ch.ID, true, now.Add(-time.Minute))

	rows, err := scoreRepo.TimelineSubmissions(context.Background(), nil)
	if err != nil {
		t.Fatalf("TimelineSubmissions: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].SubmittedAt.After(rows[1].SubmittedAt) {
		t.Fatalf("expected ascending order")
	}
}

func TestScoreboardRepoLeaderboardTieBreak(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	user1 := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	user2 := createUser(t, env, "u2@example.com", "u2", "pass", "user")

	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	createSubmission(t, env, user1.ID, ch.ID, true, time.Now().Add(-time.Minute))
	createSubmission(t, env, user2.ID, ch.ID, true, time.Now().Add(-time.Minute))

	rows, err := scoreRepo.Leaderboard(context.Background())
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	if rows[0].UserID != user1.ID {
		t.Fatalf("expected lower id first in tie, got %+v", rows[0])
	}
}

func TestScoreboardRepoTimelineIncludesUsername(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	user := createUser(t, env, "u1@example.com", "u1", "pass", "user")
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)

	createSubmission(t, env, user.ID, ch.ID, true, time.Now().Add(-time.Minute))

	rows, err := scoreRepo.TimelineSubmissions(context.Background(), nil)
	if err != nil {
		t.Fatalf("TimelineSubmissions: %v", err)
	}

	if rows[0].Username == "" {
		t.Fatalf("expected username in row")
	}
}
