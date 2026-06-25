package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	discordAPIBase   = "https://discord.com/api/v10"
	authorizeBaseURL = "https://discord.com/oauth2/authorize"
	tokenURL         = discordAPIBase + "/oauth2/token"
	currentUserURL   = discordAPIBase + "/users/@me"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       string
}

type TokenResult struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type User struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	GlobalName *string `json:"global_name"`
	Avatar     *string `json:"avatar"`
}

type OAuthClient struct {
	cfg        OAuthConfig
	httpClient *http.Client
}

func NewOAuthClient(cfg OAuthConfig, timeout time.Duration) *OAuthClient {
	return &OAuthClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *OAuthClient) AuthorizeURL(state string) string {
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {c.cfg.ClientID},
		"scope":         {c.cfg.Scopes},
		"state":         {state},
		"redirect_uri":  {c.cfg.RedirectURI},
		"prompt":        {"consent"},
	}

	return authorizeBaseURL + "?" + params.Encode()
}

func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (*TokenResult, error) {
	form := url.Values{
		"client_id":     {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {c.cfg.RedirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("discord oauth token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, oauthStatusError(resp)
	}

	var token TokenResult
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("discord oauth token decode: %w", err)
	}

	if token.AccessToken == "" {
		return nil, ErrUnexpected
	}

	return &token, nil
}

func (c *OAuthClient) FetchUser(ctx context.Context, accessToken string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, currentUserURL, nil)
	if err != nil {
		return nil, fmt.Errorf("discord oauth user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, oauthStatusError(resp)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("discord oauth user decode: %w", err)
	}

	if user.ID == "" {
		return nil, ErrUnexpected
	}

	return &user, nil
}

func oauthStatusError(resp *http.Response) error {
	message := strings.TrimSpace(resp.Status)
	if body, err := io.ReadAll(io.LimitReader(resp.Body, 4096)); err == nil {
		if text := strings.TrimSpace(string(body)); text != "" {
			message = text
		}
	}

	return &StatusError{StatusCode: resp.StatusCode, Message: message}
}
