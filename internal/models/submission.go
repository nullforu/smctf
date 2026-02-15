package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for submissions
type Submission struct {
	bun.BaseModel `bun:"table:submissions"`
	ID            int64     `bun:"id,pk,autoincrement"`
	UserID        int64     `bun:"user_id,notnull"`
	ChallengeID   int64     `bun:"challenge_id,notnull"`
	Provided      string    `bun:"provided,notnull"`
	Correct       bool      `bun:"correct,notnull,default:false"`
	IsFirstBlood  bool      `bun:"is_first_blood,notnull,default:false"`
	SubmittedAt   time.Time `bun:"submitted_at,nullzero,notnull,default:current_timestamp"`
}

type SolvedChallenge struct {
	ChallengeID int64     `bun:"challenge_id" json:"challenge_id"`
	Title       string    `bun:"title" json:"title"`
	Points      int       `bun:"points" json:"points"`
	SolvedAt    time.Time `bun:"solved_at" json:"solved_at"`
}
