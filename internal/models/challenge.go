package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for challenges
type Challenge struct {
	bun.BaseModel   `bun:"table:challenges"`
	ID              int64      `bun:",pk,autoincrement"`
	Title           string     `bun:",notnull"`
	Description     string     `bun:",notnull"`
	Points          int        `bun:",notnull,default:0"`
	MinimumPoints   int        `bun:"minimum_points,notnull,default:0"`
	Category        string     `bun:",notnull"`
	FlagHash        string     `bun:",notnull"`
	FileKey         *string    `bun:"file_key,nullzero"`
	FileName        *string    `bun:"file_name,nullzero"`
	FileUploadedAt  *time.Time `bun:"file_uploaded_at,nullzero"`
	StackEnabled    bool       `bun:"stack_enabled,notnull,default:false"`
	StackTargetPort int        `bun:"stack_target_port,notnull,default:0"`
	StackPodSpec    *string    `bun:"stack_pod_spec,nullzero"`
	IsActive        bool       `bun:",notnull"`
	CreatedAt       time.Time  `bun:",nullzero,notnull,default:current_timestamp"`
	InitialPoints   int        `bun:"-"`
	SolveCount      int        `bun:"-"`
}
