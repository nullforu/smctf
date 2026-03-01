package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for teams
type Team struct {
	bun.BaseModel `bun:"table:teams"`
	ID            int64     `bun:"id,pk,autoincrement"`
	Name          string    `bun:"name,unique,notnull"`
	DivisionID    int64     `bun:"division_id,notnull"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

type TeamSummary struct {
	ID           int64     `bun:"id" json:"id"`
	Name         string    `bun:"name" json:"name"`
	DivisionID   int64     `bun:"division_id" json:"division_id"`
	DivisionName string    `bun:"division_name" json:"division_name"`
	CreatedAt    time.Time `bun:"created_at" json:"created_at"`
	MemberCount  int       `bun:"member_count" json:"member_count"`
	TotalScore   int       `bun:"total_score" json:"total_score"`
}

type TeamMember struct {
	ID            int64      `bun:"id" json:"id"`
	Username      string     `bun:"username" json:"username"`
	Role          string     `bun:"role" json:"role"`
	BlockedReason *string    `bun:"blocked_reason" json:"blocked_reason"`
	BlockedAt     *time.Time `bun:"blocked_at" json:"blocked_at"`
}

type TeamSolvedChallenge struct {
	ChallengeID  int64     `bun:"challenge_id" json:"challenge_id"`
	Title        string    `bun:"title" json:"title"`
	Points       int       `bun:"points" json:"points"`
	SolveCount   int       `bun:"solve_count" json:"solve_count"`
	LastSolvedAt time.Time `bun:"last_solved_at" json:"last_solved_at"`
}
