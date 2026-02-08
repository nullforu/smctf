package models

import "time"

type LeaderboardEntry struct {
	UserID   int64              `bun:"user_id" json:"user_id"`
	Username string             `bun:"username" json:"username"`
	Score    int                `bun:"score" json:"score"`
	Solves   []LeaderboardSolve `json:"solves"`
}

type TeamLeaderboardEntry struct {
	TeamID   int64              `bun:"team_id" json:"team_id"`
	TeamName string             `bun:"team_name" json:"team_name"`
	Score    int                `bun:"score" json:"score"`
	Solves   []LeaderboardSolve `json:"solves"`
}

type LeaderboardChallenge struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Points   int    `json:"points"`
}

type LeaderboardSolve struct {
	ChallengeID  int64     `json:"challenge_id"`
	SolvedAt     time.Time `json:"solved_at"`
	IsFirstBlood bool      `json:"is_first_blood"`
}

type LeaderboardResponse struct {
	Challenges []LeaderboardChallenge `json:"challenges"`
	Entries    []LeaderboardEntry     `json:"entries"`
}

type TeamLeaderboardResponse struct {
	Challenges []LeaderboardChallenge `json:"challenges"`
	Entries    []TeamLeaderboardEntry `json:"entries"`
}

type UserTimelineRow struct {
	SubmittedAt time.Time `bun:"submitted_at"`
	UserID      int64     `bun:"user_id"`
	Username    string    `bun:"username"`
	ChallengeID int64     `bun:"challenge_id"`
	Points      int       `bun:"points"`
}

type TeamTimelineRow struct {
	SubmittedAt time.Time `bun:"submitted_at"`
	TeamID      int64     `bun:"team_id"`
	TeamName    string    `bun:"team_name"`
	ChallengeID int64     `bun:"challenge_id"`
	Points      int       `bun:"points"`
}

type TimelineSubmission struct {
	Timestamp      time.Time `json:"timestamp"`
	UserID         int64     `json:"user_id"`
	Username       string    `json:"username"`
	Points         int       `json:"points"`
	ChallengeCount int       `json:"challenge_count"`
}

type TeamTimelineSubmission struct {
	Timestamp      time.Time `json:"timestamp"`
	TeamID         int64     `json:"team_id"`
	TeamName       string    `json:"team_name"`
	Points         int       `json:"points"`
	ChallengeCount int       `json:"challenge_count"`
}
