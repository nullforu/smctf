package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Stack struct {
	bun.BaseModel `bun:"table:stacks"`
	ID            int64      `bun:",pk,autoincrement"`
	UserID        int64      `bun:"user_id,notnull"`
	ChallengeID   int64      `bun:"challenge_id,notnull"`
	StackID       string     `bun:"stack_id,notnull"`
	Status        string     `bun:"status,notnull"`
	NodePublicIP  *string    `bun:"node_public_ip,nullzero"`
	NodePort      *int       `bun:"node_port,nullzero"`
	TargetPort    int        `bun:"target_port,notnull"`
	TTLExpiresAt  *time.Time `bun:"ttl_expires_at,nullzero"`
	CreatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp"`
}
