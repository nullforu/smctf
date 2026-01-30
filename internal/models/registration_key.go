package models

import "time"

type RegistrationKeyView struct {
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
