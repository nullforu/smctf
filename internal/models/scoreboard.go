package models

import "time"

type LeaderboardEntry struct {
	UserID   int64  `bun:"user_id" json:"user_id"`
	Username string `bun:"username" json:"username"`
	Score    int    `bun:"score" json:"score"`
}

type TimelineSubmission struct {
	Timestamp      time.Time `json:"timestamp"`
	UserID         int64     `json:"user_id"`
	Username       string    `json:"username"`
	Points         int       `json:"points"`
	ChallengeCount int       `json:"challenge_count"`
}
