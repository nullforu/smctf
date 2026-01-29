package models

import "time"

type GroupSummary struct {
	ID          int64     `bun:"id" json:"id"`
	Name        string    `bun:"name" json:"name"`
	CreatedAt   time.Time `bun:"created_at" json:"created_at"`
	MemberCount int       `bun:"member_count" json:"member_count"`
	TotalScore  int       `bun:"total_score" json:"total_score"`
}

type GroupMember struct {
	ID       int64  `bun:"id" json:"id"`
	Username string `bun:"username" json:"username"`
	Role     string `bun:"role" json:"role"`
}

type GroupSolvedChallenge struct {
	ChallengeID  int64     `bun:"challenge_id" json:"challenge_id"`
	Title        string    `bun:"title" json:"title"`
	Points       int       `bun:"points" json:"points"`
	SolveCount   int       `bun:"solve_count" json:"solve_count"`
	LastSolvedAt time.Time `bun:"last_solved_at" json:"last_solved_at"`
}
