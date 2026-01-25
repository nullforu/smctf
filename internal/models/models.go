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
	CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

type Challenge struct {
	bun.BaseModel `bun:"table:challenges"`
	ID            int64     `bun:",pk,autoincrement"`
	Title         string    `bun:",notnull"`
	Description   string    `bun:",notnull"`
	Points        int       `bun:",notnull,default:0"`
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
