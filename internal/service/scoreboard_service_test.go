package service

import (
	"context"
	"reflect"
	"testing"
	"time"

	"smctf/internal/models"
	"smctf/internal/scoring"
)

func TestAggregateUserTimeline(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	raw := []models.UserTimelineRow{
		{SubmittedAt: base.Add(1 * time.Minute), UserID: 1, Username: "a", Points: 50},
		{SubmittedAt: base.Add(5 * time.Minute), UserID: 1, Username: "a", Points: 100},
		{SubmittedAt: base.Add(11 * time.Minute), UserID: 2, Username: "b", Points: 25},
		{SubmittedAt: base.Add(10 * time.Minute), UserID: 1, Username: "a", Points: 10},
	}

	got := aggregateUserTimeline(raw)
	want := []models.TimelineSubmission{
		{
			Timestamp:      base.Truncate(10 * time.Minute),
			UserID:         1,
			Username:       "a",
			Points:         150,
			ChallengeCount: 2,
		},
		{
			Timestamp:      base.Add(10 * time.Minute),
			UserID:         1,
			Username:       "a",
			Points:         10,
			ChallengeCount: 1,
		},
		{
			Timestamp:      base.Add(10 * time.Minute),
			UserID:         2,
			Username:       "b",
			Points:         25,
			ChallengeCount: 1,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected timeline: %+v", got)
	}
}

func TestAggregateTeamTimeline(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	raw := []models.TeamTimelineRow{
		{SubmittedAt: base.Add(2 * time.Minute), TeamID: 2, TeamName: "bravo", Points: 15},
		{SubmittedAt: base.Add(3 * time.Minute), TeamID: 1, TeamName: "alpha", Points: 20},
		{SubmittedAt: base.Add(12 * time.Minute), TeamID: 2, TeamName: "bravo", Points: 5},
		{SubmittedAt: base.Add(12 * time.Minute), TeamID: 1, TeamName: "alpha", Points: 10},
	}

	got := aggregateTeamTimeline(raw)
	want := []models.TeamTimelineSubmission{
		{
			Timestamp:      base.Truncate(10 * time.Minute),
			TeamID:         1,
			TeamName:       "alpha",
			Points:         20,
			ChallengeCount: 1,
		},
		{
			Timestamp:      base.Truncate(10 * time.Minute),
			TeamID:         2,
			TeamName:       "bravo",
			Points:         15,
			ChallengeCount: 1,
		},
		{
			Timestamp:      base.Add(10 * time.Minute),
			TeamID:         1,
			TeamName:       "alpha",
			Points:         10,
			ChallengeCount: 1,
		},
		{
			Timestamp:      base.Add(10 * time.Minute),
			TeamID:         2,
			TeamName:       "bravo",
			Points:         5,
			ChallengeCount: 1,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected team timeline: %+v", got)
	}
}

func TestScoreboardServiceLeaderboardAndTimeline(t *testing.T) {
	env := setupServiceTest(t)
	team1 := createTeam(t, env, "Alpha")
	team2 := createTeam(t, env, "Beta")
	user1 := createUserWithTeam(t, env, "u1@example.com", "u1", "pass", "user", team1.ID)
	user2 := createUserWithTeam(t, env, "u2@example.com", "u2", "pass", "user", team2.ID)

	ch1 := createChallenge(t, env, "Ch1", 100, "flag{1}", true)
	ch2 := createChallenge(t, env, "Ch2", 50, "flag{2}", true)

	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	createSubmission(t, env, user1.ID, ch1.ID, true, base.Add(2*time.Minute))
	createSubmission(t, env, user2.ID, ch2.ID, true, base.Add(5*time.Minute))

	userBoard, err := env.scoreSvc.Leaderboard(context.Background())
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}

	decay := 2
	p1 := scoring.DynamicPoints(ch1.Points, ch1.MinimumPoints, 1, decay)
	p2 := scoring.DynamicPoints(ch2.Points, ch2.MinimumPoints, 1, decay)

	scores := map[int64]int{}
	for _, entry := range userBoard.Entries {
		scores[entry.UserID] = entry.Score
	}

	if scores[user1.ID] != p1 || scores[user2.ID] != p2 {
		t.Fatalf("unexpected user scores: %+v", scores)
	}

	teamBoard, err := env.scoreSvc.TeamLeaderboard(context.Background())
	if err != nil {
		t.Fatalf("TeamLeaderboard: %v", err)
	}

	teamScores := map[int64]int{}
	for _, entry := range teamBoard.Entries {
		teamScores[entry.TeamID] = entry.Score
	}

	if teamScores[team1.ID] != p1 || teamScores[team2.ID] != p2 {
		t.Fatalf("unexpected team scores: %+v", teamScores)
	}

	userTimeline, err := env.scoreSvc.UserTimeline(context.Background(), nil)
	if err != nil {
		t.Fatalf("UserTimeline: %v", err)
	}

	if len(userTimeline) != 2 {
		t.Fatalf("expected 2 timeline buckets, got %d", len(userTimeline))
	}

	teamTimeline, err := env.scoreSvc.TeamTimeline(context.Background(), nil)
	if err != nil {
		t.Fatalf("TeamTimeline: %v", err)
	}

	if len(teamTimeline) != 2 {
		t.Fatalf("expected 2 team timeline buckets, got %d", len(teamTimeline))
	}
}
