package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Database model for challenges
type Challenge struct {
	bun.BaseModel   `bun:"table:challenges"`
	ID              int64      `bun:"id,pk,autoincrement"`
	Title           string     `bun:"title,notnull"`
	Description     string     `bun:"description,notnull"`
	Points          int        `bun:"points,notnull,default:0"`
	MinimumPoints   int        `bun:"minimum_points,notnull,default:0"`
	Category        string     `bun:"category,notnull"`
	FlagHash        string     `bun:"flag_hash,notnull"`
	FileKey         *string    `bun:"file_key,nullzero"`
	FileName        *string    `bun:"file_name,nullzero"`
	FileUploadedAt  *time.Time `bun:"file_uploaded_at,nullzero"`
	StackEnabled    bool       `bun:"stack_enabled,notnull,default:false"`
	StackTargetPort int        `bun:"stack_target_port,notnull,default:0"`
	StackPodSpec    *string    `bun:"stack_pod_spec,nullzero"`
	IsActive        bool       `bun:"is_active,notnull"`
	CreatedAt       time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	InitialPoints   int        `bun:"-"`
	SolveCount      int        `bun:"-"`
}
