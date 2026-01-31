package models

import (
	"time"

	"github.com/uptrace/bun"
)

type AppConfig struct {
	bun.BaseModel `bun:"table:app_configs"`
	Key           string    `bun:"key,pk,notnull"`
	Value         string    `bun:"value,notnull"`
	UpdatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}
