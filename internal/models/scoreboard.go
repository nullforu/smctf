package models

import "time"

type ScoreEntry struct {
	UserID   int64  `bun:"user_id" json:"user_id"`
	Username string `bun:"username" json:"username"`
	Score    int    `bun:"score" json:"score"`
}

type ScoreTimelineRow struct {
	Bucket   time.Time `bun:"bucket"`
	UserID   int64     `bun:"user_id"`
	Username string    `bun:"username"`
	Score    int       `bun:"score"`
}

type ScoreTimelineEvent struct {
	SubmittedAt    time.Time `bun:"submitted_at" json:"submitted_at"`
	UserID         int64     `bun:"user_id" json:"user_id"`
	Username       string    `bun:"username" json:"username"`
	ChallengeID    int64     `bun:"challenge_id" json:"challenge_id"`
	ChallengeTitle string    `bun:"challenge_title" json:"challenge_title"`
	Points         int       `bun:"points" json:"points"`
}

type ScoreTimelineBucket struct {
	Bucket time.Time    `json:"bucket"`
	Scores []ScoreEntry `json:"scores"`
}

type ScoreTimelineResponse struct {
	IntervalMinutes int                   `json:"interval_minutes"`
	Users           []ScoreEntry          `json:"users"`
	Buckets         []ScoreTimelineBucket `json:"buckets"`
	Events          []ScoreTimelineEvent  `json:"events"`
}
