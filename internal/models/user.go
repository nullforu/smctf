package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for users
type User struct {
	bun.BaseModel `bun:"table:users"`
	ID            int64     `bun:",pk,autoincrement"`
	Email         string    `bun:",unique,notnull"`
	Username      string    `bun:",unique,notnull"`
	PasswordHash  string    `bun:",notnull"`
	Role          string    `bun:",notnull"`
	TeamID        int64     `bun:"team_id,notnull"`
	TeamName      string    `bun:"team_name,scanonly"`
	CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}
