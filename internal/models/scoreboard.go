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

type ScoreTimelineBucket struct {
	Bucket time.Time    `json:"bucket"`
	Scores []ScoreEntry `json:"scores"`
}

type ScoreTimelineResponse struct {
	IntervalMinutes int                   `json:"interval_minutes"`
	Users           []ScoreEntry          `json:"users"`
	Buckets         []ScoreTimelineBucket `json:"buckets"`
}
