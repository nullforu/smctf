package handlers

import (
	"testing"
	"time"

	"smctf/internal/models"
)

func TestIndexUsers(t *testing.T) {
	users := []models.ScoreEntry{
		{UserID: 2, Username: "b", Score: 200},
		{UserID: 1, Username: "a", Score: 100},
	}
	ids, names := indexUsers(users)
	if len(ids) != 2 || ids[0] != 2 || ids[1] != 1 {
		t.Fatalf("unexpected ids: %+v", ids)
	}
	if names[1] != "a" || names[2] != "b" {
		t.Fatalf("unexpected names: %+v", names)
	}
}

func TestBuildScoreTimelineBuckets(t *testing.T) {
	base := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	rows := []models.ScoreTimelineRow{
		{Bucket: base, UserID: 1, Username: "a", Score: 100},
		{Bucket: base, UserID: 2, Username: "b", Score: 200},
		{Bucket: base.Add(10 * time.Minute), UserID: 1, Username: "a", Score: 50},
	}
	userIDs := []int64{1, 2}
	usernames := map[int64]string{1: "a", 2: "b"}
	buckets := buildScoreTimelineBuckets(rows, userIDs, usernames)
	if len(buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(buckets))
	}
	if buckets[0].Scores[0].Score != 100 || buckets[0].Scores[1].Score != 200 {
		t.Fatalf("unexpected first bucket scores: %+v", buckets[0].Scores)
	}
	if buckets[1].Scores[0].Score != 150 || buckets[1].Scores[1].Score != 200 {
		t.Fatalf("unexpected second bucket scores: %+v", buckets[1].Scores)
	}
}
