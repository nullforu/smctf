package handlers

import (
	"testing"
	"time"
)

func TestTeamSubmissions(t *testing.T) {
	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)

	raw := []rawSubmission{
		{SubmittedAt: base.Add(2 * time.Minute), UserID: 1, Username: "user1", Points: 100},
		{SubmittedAt: base.Add(5 * time.Minute), UserID: 1, Username: "user1", Points: 200},
		{SubmittedAt: base.Add(15 * time.Minute), UserID: 1, Username: "user1", Points: 50},
		{SubmittedAt: base.Add(3 * time.Minute), UserID: 2, Username: "user2", Points: 150},
	}

	result := teamSubmissions(raw)

	if len(result) != 3 {
		t.Fatalf("expected 3 teams, got %d", len(result))
	}

	if result[0].UserID != 1 || result[0].Points != 300 || result[0].ChallengeCount != 2 {
		t.Fatalf("unexpected first team: %+v", result[0])
	}

	if result[1].UserID != 2 || result[1].Points != 150 || result[1].ChallengeCount != 1 {
		t.Fatalf("unexpected second team: %+v", result[1])
	}

	if result[2].UserID != 1 || result[2].Points != 50 || result[2].ChallengeCount != 1 {
		t.Fatalf("unexpected third team: %+v", result[2])
	}
}

func TestTeamTeamSubmissions(t *testing.T) {
	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	teamID := int64(10)

	raw := []rawTeamSubmission{
		{SubmittedAt: base.Add(2 * time.Minute), TeamID: &teamID, TeamName: "Alpha", Points: 100},
		{SubmittedAt: base.Add(7 * time.Minute), TeamID: &teamID, TeamName: "Alpha", Points: 50},
		{SubmittedAt: base.Add(12 * time.Minute), TeamID: nil, TeamName: "not affiliated", Points: 30},
	}

	result := teamTeamSubmissions(raw)

	if len(result) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(result))
	}

	if result[0].TeamName != "Alpha" || result[0].Points != 150 || result[0].ChallengeCount != 2 {
		t.Fatalf("unexpected first team: %+v", result[0])
	}

	if result[1].TeamName != "not affiliated" || result[1].Points != 30 || result[1].ChallengeCount != 1 {
		t.Fatalf("unexpected second team: %+v", result[1])
	}
}
