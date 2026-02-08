package repo

import (
	"context"
	"testing"
	"time"

	"smctf/internal/models"
)

func TestScoreboardRepoLeaderboardAndTimeline(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	team := createTeam(t, env, "Alpha")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", team.ID)
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

	if len(leaderboard.Entries) != 2 {
		t.Fatalf("expected 2 leaderboard rows, got %d", len(leaderboard.Entries))
	}

	if leaderboard.Entries[0].UserID != user1.ID || leaderboard.Entries[0].Score != 150 {
		t.Fatalf("unexpected leaderboard first row: %+v", leaderboard.Entries[0])
	}

	if leaderboard.Entries[1].UserID != user2.ID || leaderboard.Entries[1].Score != 0 {
		t.Fatalf("unexpected leaderboard second row: %+v", leaderboard.Entries[1])
	}

	if len(leaderboard.Challenges) != 2 {
		t.Fatalf("expected 2 challenges, got %d", len(leaderboard.Challenges))
	}

	if len(leaderboard.Entries[0].Solves) != 2 {
		t.Fatalf("expected 2 solves for first entry, got %d", len(leaderboard.Entries[0].Solves))
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
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", teamA.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", teamB.ID)
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

	if len(leaderboard.Entries) != 3 {
		t.Fatalf("expected 3 team rows, got %d", len(leaderboard.Entries))
	}

	if leaderboard.Entries[0].TeamName != "Alpha" || leaderboard.Entries[0].Score != 100 {
		t.Fatalf("unexpected team leaderboard first row: %+v", leaderboard.Entries[0])
	}

	if leaderboard.Entries[2].TeamName != "team-u3" || leaderboard.Entries[2].Score != 50 {
		t.Fatalf("unexpected team leaderboard last row: %+v", leaderboard.Entries[2])
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

	if len(rows.Entries) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows.Entries))
	}

	if rows.Entries[0].UserID != user1.ID {
		t.Fatalf("expected lower id first in tie, got %+v", rows.Entries[0])
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

func TestScoreboardRepoTeamLeaderboardIncludesEmptyTeam(t *testing.T) {
	env := setupRepoTest(t)
	scoreRepo := NewScoreboardRepo(env.db)

	teamA := createTeam(t, env, "Alpha")
	teamB := createTeam(t, env, "Beta")
	user := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", teamA.ID)
	ch := createChallenge(t, env, "ch1", 100, "FLAG{1}", true)
	createSubmission(t, env, user.ID, ch.ID, true, time.Now().UTC())

	rows, err := scoreRepo.TeamLeaderboard(context.Background())
	if err != nil {
		t.Fatalf("TeamLeaderboard: %v", err)
	}

	var alpha, beta *models.TeamLeaderboardEntry
	for i := range rows.Entries {
		switch rows.Entries[i].TeamName {
		case teamA.Name:
			alpha = &rows.Entries[i]
		case teamB.Name:
			beta = &rows.Entries[i]
		}
	}

	if alpha == nil || beta == nil {
		t.Fatalf("expected both teams in leaderboard, got %+v", rows)
	}

	if alpha.Score != 100 {
		t.Fatalf("expected alpha score 100, got %d", alpha.Score)
	}

	if beta.Score != 0 {
		t.Fatalf("expected beta score 0, got %d", beta.Score)
	}
}
