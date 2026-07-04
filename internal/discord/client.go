package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrNotInGuild    = fmt.Errorf("discord user not in guild")
	ErrBotPermission = fmt.Errorf("discord bot permission denied")
	ErrRateLimited   = fmt.Errorf("discord rate limited")
	ErrInvalid       = fmt.Errorf("discord request invalid")
	ErrUnavailable   = fmt.Errorf("discord bot server unavailable")
	ErrUnexpected    = fmt.Errorf("discord bot server error")
)

type StatusError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *StatusError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}

	return fmt.Sprintf("discord bot server returned status %d", e.StatusCode)
}

func (e *StatusError) Unwrap() error {
	switch e.Code {
	case "NOT_IN_GUILD":
		return ErrNotInGuild
	case "BOT_PERMISSION":
		return ErrBotPermission
	case "RATE_LIMITED":
		return ErrRateLimited
	}

	switch e.StatusCode {
	case http.StatusNotFound:
		return ErrNotInGuild
	case http.StatusForbidden:
		return ErrBotPermission
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusBadRequest:
		return ErrInvalid
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return ErrUnavailable
	default:
		return ErrUnexpected
	}
}

type Member struct {
	InGuild bool `json:"in_guild"`
	HasRole bool `json:"has_role"`
}

type BotAPI interface {
	JoinGuild(ctx context.Context, discordUserID, accessToken, roleID string) error
	GrantRole(ctx context.Context, discordUserID, roleID string) error
	KickMember(ctx context.Context, discordUserID string) error
	GetMember(ctx context.Context, discordUserID string) (*Member, error)
	Announce(ctx context.Context, channelID, content string) error
	SetNickname(ctx context.Context, discordUserID, nickname string) error
}

type BotClient struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

func NewBotClient(baseURL, secret string, timeout time.Duration) *BotClient {
	return &BotClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		secret:  secret,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type joinRequest struct {
	AccessToken string `json:"access_token"`
	RoleID      string `json:"role_id,omitempty"`
}

type roleRequest struct {
	RoleID string `json:"role_id"`
}

type announceRequest struct {
	ChannelID string `json:"channel_id"`
	Content   string `json:"content"`
}

type nicknameRequest struct {
	Nickname string `json:"nickname"`
}

func (c *BotClient) JoinGuild(ctx context.Context, discordUserID, accessToken, roleID string) error {
	return c.doJSON(ctx, http.MethodPut, "/internal/guild/members/"+url.PathEscape(discordUserID), joinRequest{AccessToken: accessToken, RoleID: roleID}, nil)
}

func (c *BotClient) GrantRole(ctx context.Context, discordUserID, roleID string) error {
	return c.doJSON(ctx, http.MethodPut, "/internal/guild/members/"+url.PathEscape(discordUserID)+"/role", roleRequest{RoleID: roleID}, nil)
}

func (c *BotClient) KickMember(ctx context.Context, discordUserID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/internal/guild/members/"+url.PathEscape(discordUserID), nil, nil)
}

func (c *BotClient) Announce(ctx context.Context, channelID, content string) error {
	return c.doJSON(ctx, http.MethodPost, "/internal/announce", announceRequest{ChannelID: channelID, Content: content}, nil)
}

func (c *BotClient) SetNickname(ctx context.Context, discordUserID, nickname string) error {
	return c.doJSON(ctx, http.MethodPatch, "/internal/guild/members/"+url.PathEscape(discordUserID)+"/nickname", nicknameRequest{Nickname: nickname}, nil)
}

func (c *BotClient) GetMember(ctx context.Context, discordUserID string) (*Member, error) {
	var member Member
	if err := c.doJSON(ctx, http.MethodGet, "/internal/guild/members/"+url.PathEscape(discordUserID), nil, &member); err != nil {
		return nil, err
	}

	return &member, nil
}

func (c *BotClient) doJSON(ctx context.Context, method, path string, body any, out any) error {
	reader, err := encodeBody(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("discord bot client request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if err := handleStatus(resp); err != nil {
		return err
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("discord bot client decode: %w", err)
	}

	return nil
}

func encodeBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("discord bot client marshal: %w", err)
	}

	return bytes.NewReader(payload), nil
}

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func handleStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	statusErr := &StatusError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(resp.Status)}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err == nil && len(body) > 0 {
		var parsed errorBody
		if jsonErr := json.Unmarshal(body, &parsed); jsonErr == nil {
			if strings.TrimSpace(parsed.Code) != "" {
				statusErr.Code = parsed.Code
			}

			if strings.TrimSpace(parsed.Error) != "" {
				statusErr.Message = strings.TrimSpace(parsed.Error)
			}
		} else if text := strings.TrimSpace(string(body)); text != "" {
			statusErr.Message = text
		}
	}

	return statusErr
}
