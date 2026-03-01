package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for divisions
type Division struct {
	bun.BaseModel `bun:"table:divisions"`
	ID            int64     `bun:"id,pk,autoincrement" json:"id"`
	Name          string    `bun:"name,unique,notnull" json:"name"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
}
