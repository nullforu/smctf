package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for challenges
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
