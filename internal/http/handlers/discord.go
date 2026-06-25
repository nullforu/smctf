package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"smctf/internal/http/middleware"
	"smctf/internal/models"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
)

type discordStatusResponse struct {
	Connected   bool       `json:"connected"`
	DiscordID   *string    `json:"discord_user_id,omitempty"`
	Username    *string    `json:"discord_username,omitempty"`
	GlobalName  *string    `json:"discord_global_name,omitempty"`
	Avatar      *string    `json:"discord_avatar,omitempty"`
	RoleStatus  string     `json:"role_status,omitempty"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	InviteURL   string     `json:"invite_url,omitempty"`
}

func (h *Handler) newDiscordStatusResponse(conn *models.DiscordConnection) discordStatusResponse {
	if conn == nil {
		return discordStatusResponse{Connected: false, InviteURL: h.cfg.Discord.InviteURL}
	}

	connectedAt := conn.ConnectedAt.UTC()
	resp := discordStatusResponse{
		Connected:   true,
		DiscordID:   &conn.DiscordUserID,
		Username:    conn.DiscordUsername,
		GlobalName:  conn.DiscordGlobalName,
		Avatar:      conn.DiscordAvatar,
		RoleStatus:  conn.RoleStatus,
		ConnectedAt: &connectedAt,
		VerifiedAt:  conn.VerifiedAt,
		InviteURL:   h.cfg.Discord.InviteURL,
	}

	return resp
}

func (h *Handler) DiscordConnect(ctx *gin.Context) {
	if h.discord == nil {
		writeError(ctx, service.ErrDiscordDisabled)
		return
	}

	authorizeURL, err := h.discord.BeginConnect(ctx.Request.Context(), middleware.UserID(ctx))
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.Redirect(http.StatusFound, authorizeURL)
}

func (h *Handler) DiscordCallback(ctx *gin.Context) {
	if h.discord == nil {
		writeError(ctx, service.ErrDiscordDisabled)
		return
	}

	code := strings.TrimSpace(ctx.Query("code"))
	state := strings.TrimSpace(ctx.Query("state"))

	conn, err := h.discord.HandleCallback(ctx.Request.Context(), middleware.UserID(ctx), code, state)
	if err != nil {
		h.redirectDiscordResult(ctx, discordResultForError(err))
		return
	}

	h.redirectDiscordResult(ctx, discordResultForStatus(conn.RoleStatus))
}

func (h *Handler) DiscordStatus(ctx *gin.Context) {
	if h.discord == nil {
		writeError(ctx, service.ErrDiscordDisabled)
		return
	}

	conn, err := h.discord.GetConnection(ctx.Request.Context(), middleware.UserID(ctx))
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, h.newDiscordStatusResponse(conn))
}

func (h *Handler) DiscordSyncRole(ctx *gin.Context) {
	if h.discord == nil {
		writeError(ctx, service.ErrDiscordDisabled)
		return
	}

	conn, err := h.discord.SyncRole(ctx.Request.Context(), middleware.UserID(ctx))
	if err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, h.newDiscordStatusResponse(conn))
}

func (h *Handler) DiscordUnlink(ctx *gin.Context) {
	if h.discord == nil {
		writeError(ctx, service.ErrDiscordDisabled)
		return
	}

	if err := h.discord.Unlink(ctx.Request.Context(), middleware.UserID(ctx)); err != nil {
		writeError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func discordResultForStatus(roleStatus string) string {
	switch roleStatus {
	case models.DiscordStatusVerified:
		return "verified"
	case models.DiscordStatusNotInGuild, models.DiscordStatusLeftGuild:
		return "connected_not_joined"
	default:
		return "role_failed"
	}
}

func discordResultForError(err error) string {
	switch {
	case err == nil:
		return "verified"
	case errors.Is(err, service.ErrDiscordStateInvalid):
		return "state_invalid"
	case errors.Is(err, service.ErrDiscordAlreadyLinked):
		return "already_linked"
	default:
		return "error"
	}
}

func (h *Handler) redirectDiscordResult(ctx *gin.Context, result string) {
	target := strings.TrimSpace(h.cfg.Discord.SuccessRedirect)
	if target == "" {
		ctx.JSON(http.StatusOK, gin.H{"discord": result})
		return
	}

	sep := "?"
	if strings.Contains(target, "?") {
		sep = "&"
	}

	ctx.Redirect(http.StatusFound, target+sep+"discord="+url.QueryEscape(result))
}
