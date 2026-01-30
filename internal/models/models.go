package models

import (
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`
	ID            int64     `bun:",pk,autoincrement"`
	Email         string    `bun:",unique,notnull"`
	Username      string    `bun:",unique,notnull"`
	PasswordHash  string    `bun:",notnull"`
	Role          string    `bun:",notnull"`
	TeamID        *int64    `bun:"team_id,nullzero"`
	TeamName      *string   `bun:"team_name,scanonly"`
	CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

type Team struct {
	bun.BaseModel `bun:"table:teams"`
	ID            int64     `bun:",pk,autoincrement"`
	Name          string    `bun:",unique,notnull"`
	CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

type Challenge struct {
	bun.BaseModel `bun:"table:challenges"`
	ID            int64     `bun:",pk,autoincrement"`
	Title         string    `bun:",notnull"`
	Description   string    `bun:",notnull"`
	Points        int       `bun:",notnull,default:0"`
	Category      string    `bun:",notnull"`
	FlagHash      string    `bun:",notnull"`
	IsActive      bool      `bun:",notnull"`
	CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

type Submission struct {
	bun.BaseModel `bun:"table:submissions"`
	ID            int64     `bun:",pk,autoincrement"`
	UserID        int64     `bun:",notnull"`
	ChallengeID   int64     `bun:",notnull"`
	Provided      string    `bun:",notnull"`
	Correct       bool      `bun:",notnull,default:false"`
	SubmittedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

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
