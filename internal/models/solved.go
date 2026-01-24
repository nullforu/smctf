package models

import "time"

type SolvedChallenge struct {
	ChallengeID int64     `bun:"challenge_id" json:"challenge_id"`
	Title       string    `bun:"title" json:"title"`
	Points      int       `bun:"points" json:"points"`
	SolvedAt    time.Time `bun:"solved_at" json:"solved_at"`
}
