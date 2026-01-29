package handlers

import (
	"testing"
	"time"
)

func TestGroupSubmissions(t *testing.T) {
	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)

	raw := []rawSubmission{
		{SubmittedAt: base.Add(2 * time.Minute), UserID: 1, Username: "user1", Points: 100},
		{SubmittedAt: base.Add(5 * time.Minute), UserID: 1, Username: "user1", Points: 200},
		{SubmittedAt: base.Add(15 * time.Minute), UserID: 1, Username: "user1", Points: 50},
		{SubmittedAt: base.Add(3 * time.Minute), UserID: 2, Username: "user2", Points: 150},
	}

	result := groupSubmissions(raw)

	if len(result) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(result))
	}

	if result[0].UserID != 1 || result[0].Points != 300 || result[0].ChallengeCount != 2 {
		t.Fatalf("unexpected first group: %+v", result[0])
	}

	if result[1].UserID != 2 || result[1].Points != 150 || result[1].ChallengeCount != 1 {
		t.Fatalf("unexpected second group: %+v", result[1])
	}

	if result[2].UserID != 1 || result[2].Points != 50 || result[2].ChallengeCount != 1 {
		t.Fatalf("unexpected third group: %+v", result[2])
	}
}

func TestGroupGroupSubmissions(t *testing.T) {
	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	groupID := int64(10)

	raw := []rawGroupSubmission{
		{SubmittedAt: base.Add(2 * time.Minute), GroupID: &groupID, GroupName: "Alpha", Points: 100},
		{SubmittedAt: base.Add(7 * time.Minute), GroupID: &groupID, GroupName: "Alpha", Points: 50},
		{SubmittedAt: base.Add(12 * time.Minute), GroupID: nil, GroupName: "not affiliated", Points: 30},
	}

	result := groupGroupSubmissions(raw)

	if len(result) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result))
	}

	if result[0].GroupName != "Alpha" || result[0].Points != 150 || result[0].ChallengeCount != 2 {
		t.Fatalf("unexpected first group: %+v", result[0])
	}

	if result[1].GroupName != "not affiliated" || result[1].Points != 30 || result[1].ChallengeCount != 1 {
		t.Fatalf("unexpected second group: %+v", result[1])
	}
}
