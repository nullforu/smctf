package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for registration keys
type RegistrationKey struct {
	bun.BaseModel `bun:"table:registration_keys"`
	ID            int64      `bun:",pk,autoincrement"`
	Code          string     `bun:",unique,notnull"`
	CreatedBy     int64      `bun:",notnull"`
	TeamID        *int64     `bun:"team_id,nullzero"`
	UsedBy        *int64     `bun:",nullzero"`
	UsedByIP      *string    `bun:",nullzero"`
	CreatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp"`
	UsedAt        *time.Time `bun:",nullzero"`
}

type RegistrationKeySummary struct {
	ID                int64      `bun:"id" json:"id"`
	Code              string     `bun:"code" json:"code"`
	CreatedBy         int64      `bun:"created_by" json:"created_by"`
	CreatedByUsername string     `bun:"created_by_username" json:"created_by_username"`
	TeamID            *int64     `bun:"team_id" json:"team_id,omitempty"`
	TeamName          *string    `bun:"team_name" json:"team_name,omitempty"`
	UsedBy            *int64     `bun:"used_by" json:"used_by,omitempty"`
	UsedByUsername    *string    `bun:"used_by_username" json:"used_by_username,omitempty"`
	UsedByIP          *string    `bun:"used_by_ip" json:"used_by_ip,omitempty"`
	CreatedAt         time.Time  `bun:"created_at" json:"created_at"`
	UsedAt            *time.Time `bun:"used_at" json:"used_at,omitempty"`
}
