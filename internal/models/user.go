package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for users
type User struct {
	bun.BaseModel `bun:"table:users"`
	ID            int64      `bun:"id,pk,autoincrement"`
	Email         string     `bun:"email,unique,notnull"`
	Username      string     `bun:"username,unique,notnull"`
	PasswordHash  string     `bun:"password_hash,notnull"`
	Role          string     `bun:"role,notnull"`
	TeamID        int64      `bun:"team_id,notnull"`
	TeamName      string     `bun:"team_name,scanonly"`
	DivisionID    int64      `bun:"division_id,scanonly"`
	DivisionName  string     `bun:"division_name,scanonly"`
	BlockedReason *string    `bun:"blocked_reason,nullzero"`
	BlockedAt     *time.Time `bun:"blocked_at,nullzero"`
	CreatedAt     time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
