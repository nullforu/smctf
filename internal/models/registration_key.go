package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for registration keys
type RegistrationKey struct {
	bun.BaseModel `bun:"table:registration_keys"`
	ID            int64     `bun:"id,pk,autoincrement"`
	Code          string    `bun:"code,unique,notnull"`
	CreatedBy     int64     `bun:"created_by,notnull"`
	TeamID        int64     `bun:"team_id,notnull"`
	MaxUses       int       `bun:"max_uses,notnull,default:1"`
	UsedCount     int       `bun:"used_count,notnull,default:0"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

type RegistrationKeyUse struct {
	bun.BaseModel     `bun:"table:registration_key_uses"`
	ID                int64     `bun:"id,pk,autoincrement"`
	RegistrationKeyID int64     `bun:"registration_key_id,notnull"`
	UsedBy            int64     `bun:"used_by,notnull"`
	UsedByIP          string    `bun:"used_by_ip,notnull"`
	UsedAt            time.Time `bun:"used_at,nullzero,notnull,default:current_timestamp"`
}

type RegistrationKeySummary struct {
	ID                int64                       `bun:"id" json:"id"`
	Code              string                      `bun:"code" json:"code"`
	CreatedBy         int64                       `bun:"created_by" json:"created_by"`
	CreatedByUsername string                      `bun:"created_by_username" json:"created_by_username"`
	TeamID            int64                       `bun:"team_id" json:"team_id"`
	TeamName          string                      `bun:"team_name" json:"team_name"`
	MaxUses           int                         `bun:"max_uses" json:"max_uses"`
	UsedCount         int                         `bun:"used_count" json:"used_count"`
	CreatedAt         time.Time                   `bun:"created_at" json:"created_at"`
	LastUsedAt        *time.Time                  `bun:"last_used_at" json:"last_used_at,omitempty"`
	Uses              []RegistrationKeyUseSummary `json:"uses,omitempty"`
}

type RegistrationKeyUseSummary struct {
	RegistrationKeyID int64     `bun:"registration_key_id" json:"-"`
	UsedBy            int64     `bun:"used_by" json:"used_by"`
	UsedByUsername    string    `bun:"used_by_username" json:"used_by_username"`
	UsedByIP          string    `bun:"used_by_ip" json:"used_by_ip"`
	UsedAt            time.Time `bun:"used_at" json:"used_at"`
}
