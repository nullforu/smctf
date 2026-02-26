package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

type Stack struct {
	bun.BaseModel `bun:"table:stacks"`
	ID            int64             `bun:"id,pk,autoincrement"`
	UserID        int64             `bun:"user_id,notnull"`
	ChallengeID   int64             `bun:"challenge_id,notnull"`
	StackID       string            `bun:"stack_id,notnull"`
	Status        string            `bun:"status,notnull"`
	NodePublicIP  *string           `bun:"node_public_ip,nullzero"`
	Ports         StackPortMappings `bun:"ports,type:jsonb,nullzero"`
	TTLExpiresAt  *time.Time        `bun:"ttl_expires_at,nullzero"`
	CreatedAt     time.Time         `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time         `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}

type StackPortSpec struct {
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
}

type StackPortSpecs []StackPortSpec

func (p StackPortSpecs) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}

	payload, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("stack port specs marshal: %w", err)
	}

	return string(payload), nil
}

func (p *StackPortSpecs) Scan(value any) error {
	if value == nil {
		*p = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return fmt.Errorf("stack port specs scan: %T", value)
	}
}

type StackPortMapping struct {
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
	NodePort      int    `json:"node_port"`
}

type StackPortMappings []StackPortMapping

func (p StackPortMappings) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}

	payload, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("stack port mappings marshal: %w", err)
	}

	return string(payload), nil
}

func (p *StackPortMappings) Scan(value any) error {
	if value == nil {
		*p = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return fmt.Errorf("stack port mappings scan: %T", value)
	}
}

type AdminStackSummary struct {
	StackID           string     `bun:"stack_id" json:"stack_id"`
	TTLExpiresAt      *time.Time `bun:"ttl_expires_at" json:"ttl_expires_at,omitempty"`
	CreatedAt         time.Time  `bun:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `bun:"updated_at" json:"updated_at"`
	UserID            int64      `bun:"user_id" json:"user_id"`
	Username          string     `bun:"username" json:"username"`
	Email             string     `bun:"email" json:"email"`
	TeamID            int64      `bun:"team_id" json:"team_id"`
	TeamName          string     `bun:"team_name" json:"team_name"`
	ChallengeID       int64      `bun:"challenge_id" json:"challenge_id"`
	ChallengeTitle    string     `bun:"challenge_title" json:"challenge_title"`
	ChallengeCategory string     `bun:"challenge_category" json:"challenge_category"`
}
