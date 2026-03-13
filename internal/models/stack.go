package models

import (
	"time"

	"smctf/internal/stack"

	"github.com/uptrace/bun"
)

type Stack struct {
	bun.BaseModel  `bun:"table:stacks"`
	ID             int64              `bun:"id,pk,autoincrement"`
	UserID         int64              `bun:"user_id,notnull"`
	Username       string             `bun:"username,scanonly"`
	ChallengeID    int64              `bun:"challenge_id,notnull"`
	ChallengeTitle string             `bun:"challenge_title,scanonly"`
	StackID        string             `bun:"stack_id,notnull"`
	Status         string             `bun:"status,notnull"`
	NodePublicIP   *string            `bun:"node_public_ip,nullzero"`
	Ports          stack.PortMappings `bun:"ports,type:jsonb,nullzero"`
	TTLExpiresAt   *time.Time         `bun:"ttl_expires_at,nullzero"`
	CreatedAt      time.Time          `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt      time.Time          `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type AdminStackSummary struct {
	StackID           string     `bun:"stack_id" json:"stack_id"`
	TTLExpiresAt      *time.Time `bun:"ttl_expires_at" json:"ttl_expires_at,omitempty"`
	CreatedAt         time.Time  `bun:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `bun:"updated_at" json:"updated_at"`
	UserID            int64      `bun:"user_id" json:"user_id"`
	Username          string     `bun:"username" json:"username"`
	Email             string     `bun:"email" json:"email"`
	TeamID            int64      `bun:"team_id" json:"team_id"`
	TeamName          string     `bun:"team_name" json:"team_name"`
	ChallengeID       int64      `bun:"challenge_id" json:"challenge_id"`
	ChallengeTitle    string     `bun:"challenge_title" json:"challenge_title"`
	ChallengeCategory string     `bun:"challenge_category" json:"challenge_category"`
}
