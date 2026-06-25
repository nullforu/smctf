package models

import (
	"time"

	"github.com/uptrace/bun"
)

const (
	DiscordStatusConnected  = "CONNECTED"
	DiscordStatusVerified   = "VERIFIED"
	DiscordStatusNotInGuild = "NOT_IN_GUILD"
	DiscordStatusRoleFailed = "ROLE_FAILED"
	DiscordStatusRevoked    = "REVOKED"
	DiscordStatusLeftGuild  = "LEFT_GUILD"
)

type DiscordConnection struct {
	bun.BaseModel     `bun:"table:discord_connections"`
	ID                int64      `bun:"id,pk,autoincrement"`
	UserID            int64      `bun:"user_id,unique,notnull"`
	DiscordUserID     string     `bun:"discord_user_id,unique,notnull"`
	DiscordUsername   *string    `bun:"discord_username,nullzero"`
	DiscordGlobalName *string    `bun:"discord_global_name,nullzero"`
	DiscordAvatar     *string    `bun:"discord_avatar,nullzero"`
	RoleStatus        string     `bun:"role_status,notnull,default:'CONNECTED'"`
	ConnectedAt       time.Time  `bun:"connected_at,nullzero,notnull,default:current_timestamp"`
	VerifiedAt        *time.Time `bun:"verified_at,nullzero"`
	RevokedAt         *time.Time `bun:"revoked_at,nullzero"`
	LastSyncedAt      *time.Time `bun:"last_synced_at,nullzero"`
	LastError         *string    `bun:"last_error,nullzero"`
	CreatedAt         time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt         time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
}
