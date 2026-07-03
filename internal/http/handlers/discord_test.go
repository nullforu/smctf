package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/discord"
	"smctf/internal/models"
	"smctf/internal/repo"
	"smctf/internal/service"

	"github.com/gin-gonic/gin"
)

type fakeBot struct {
	grantErr  error
	kicked    bool
	announced []string
	nickname  string
}

func (f *fakeBot) JoinGuild(_ context.Context, _, _ string) error { return nil }
func (f *fakeBot) GrantRole(_ context.Context, _ string) error    { return f.grantErr }
func (f *fakeBot) KickMember(_ context.Context, _ string) error {
	f.kicked = true
	return nil
}

func (f *fakeBot) GetMember(_ context.Context, _ string) (*discord.Member, error) {
	return &discord.Member{}, nil
}

func (f *fakeBot) Announce(_ context.Context, content string) error {
	f.announced = append(f.announced, content)
	return nil
}

func (f *fakeBot) SetNickname(_ context.Context, _, nickname string) error {
	f.nickname = nickname
	return nil
}

type fakeOAuth struct {
	user *discord.User
}

func (f fakeOAuth) AuthorizeURL(state string) string {
	return "https://discord.com/oauth2/authorize?state=" + state
}

func (f fakeOAuth) ExchangeCode(_ context.Context, _ string) (*discord.TokenResult, error) {
	return &discord.TokenResult{AccessToken: "at"}, nil
}

func (f fakeOAuth) FetchUser(_ context.Context, _ string) (*discord.User, error) {
	return f.user, nil
}

func discordEnabledHandler(env handlerEnv, bot discord.BotAPI, user *discord.User) *Handler {
	cfg := env.cfg
	cfg.Discord = config.DiscordConfig{
		Enabled:         true,
		StateTTL:        5 * time.Minute,
		AutoJoin:        true,
		SuccessRedirect: "http://localhost:3000/profile",
		InviteURL:       "https://discord.gg/invite",
	}
	discordSvc := service.NewDiscordService(cfg.Discord, repo.NewDiscordRepo(env.db), env.userRepo, bot, fakeOAuth{user: user}, env.redis)
	return New(cfg, env.authSvc, env.ctfSvc, env.appConfigSvc, env.userSvc, env.scoreSvc, env.divisionSvc, env.teamSvc, env.vmSvc, env.redis, discordSvc)
}

func TestDiscordResultMapping(t *testing.T) {
	if got := discordResultForStatus(models.DiscordStatusVerified); got != "verified" {
		t.Errorf("verified -> %q", got)
	}

	if got := discordResultForStatus(models.DiscordStatusNotInGuild); got != "connected_not_joined" {
		t.Errorf("not in guild -> %q", got)
	}

	if got := discordResultForStatus(models.DiscordStatusRoleFailed); got != "role_failed" {
		t.Errorf("role failed -> %q", got)
	}

	if got := discordResultForError(nil); got != "verified" {
		t.Errorf("nil -> %q", got)
	}

	if got := discordResultForError(service.ErrDiscordStateInvalid); got != "state_invalid" {
		t.Errorf("state invalid -> %q", got)
	}

	if got := discordResultForError(service.ErrDiscordAlreadyLinked); got != "already_linked" {
		t.Errorf("already linked -> %q", got)
	}

	if got := discordResultForError(errors.New("boom")); got != "error" {
		t.Errorf("generic -> %q", got)
	}
}

func disabledDiscordHandler(redirect string) *Handler {
	cfg := config.Config{}
	cfg.Discord.SuccessRedirect = redirect
	return New(cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func TestHandlerDiscordDisabled(t *testing.T) {
	h := disabledDiscordHandler("")

	cases := []struct {
		name   string
		method string
		invoke func(*Handler, *gin.Context)
	}{
		{"connect", http.MethodGet, (*Handler).DiscordConnect},
		{"callback", http.MethodGet, (*Handler).DiscordCallback},
		{"status", http.MethodGet, (*Handler).DiscordStatus},
		{"sync", http.MethodPost, (*Handler).DiscordSyncRole},
		{"unlink", http.MethodDelete, (*Handler).DiscordUnlink},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, rec := newJSONContext(t, tc.method, "/api/discord/"+tc.name, nil)
			ctx.Set("userID", int64(1))
			tc.invoke(h, ctx)
			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected 503, got %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRedirectDiscordResult(t *testing.T) {
	jsonCtx, jsonRec := newJSONContext(t, http.MethodGet, "/api/discord/callback", nil)
	disabledDiscordHandler("").redirectDiscordResult(jsonCtx, "verified")
	if jsonRec.Code != http.StatusOK || !strings.Contains(jsonRec.Body.String(), `"discord":"verified"`) {
		t.Fatalf("json fallback: code=%d body=%s", jsonRec.Code, jsonRec.Body.String())
	}

	redirCtx, redirRec := newJSONContext(t, http.MethodGet, "/api/discord/callback", nil)
	disabledDiscordHandler("https://app.example.com/profile?tab=1").redirectDiscordResult(redirCtx, "role_failed")
	if redirRec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", redirRec.Code)
	}

	if loc := redirRec.Header().Get("Location"); !strings.Contains(loc, "?tab=1&discord=role_failed") {
		t.Fatalf("location = %q", loc)
	}
}

func TestHandlerDiscordStatusNotConnected(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "dh1@example.com", "dhuser1", "pass", models.UserRole)
	h := discordEnabledHandler(env, &fakeBot{}, &discord.User{ID: "1"})

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/discord/status", nil)
	ctx.Set("userID", user.ID)
	h.DiscordStatus(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d body=%s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), `"connected":false`) {
		t.Errorf("body = %s", rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), "discord.gg/invite") {
		t.Errorf("invite url missing: %s", rec.Body.String())
	}
}

func TestHandlerDiscordConnectAndCallback(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "dh2@example.com", "dhuser2", "pass", models.UserRole)
	bot := &fakeBot{}
	h := discordEnabledHandler(env, bot, &discord.User{ID: "9001", Username: "neo"})

	connectCtx, connectRec := newJSONContext(t, http.MethodGet, "/api/discord/connect", nil)
	connectCtx.Set("userID", user.ID)
	h.DiscordConnect(connectCtx)
	if connectRec.Code != http.StatusFound {
		t.Fatalf("connect code = %d", connectRec.Code)
	}

	location := connectRec.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}

	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatal("state missing from authorize url")
	}

	cbCtx, cbRec := newJSONContext(t, http.MethodGet, "/api/discord/callback?code=abc&state="+state, nil)
	cbCtx.Set("userID", user.ID)
	h.DiscordCallback(cbCtx)
	if cbRec.Code != http.StatusFound {
		t.Fatalf("callback code = %d body=%s", cbRec.Code, cbRec.Body.String())
	}

	if loc := cbRec.Header().Get("Location"); !strings.Contains(loc, "discord=verified") {
		t.Errorf("callback location = %q", loc)
	}

	stored, err := repo.NewDiscordRepo(env.db).GetByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("connection not persisted: %v", err)
	}

	if stored.DiscordUserID != "9001" || stored.RoleStatus != models.DiscordStatusVerified {
		t.Errorf("stored = %+v", stored)
	}
}

func TestHandlerDiscordSyncRole(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "dh5@example.com", "dhuser5", "pass", models.UserRole)
	bot := &fakeBot{grantErr: discord.ErrNotInGuild}
	h := discordEnabledHandler(env, bot, &discord.User{ID: "5000", Username: "link"})

	connectCtx, connectRec := newJSONContext(t, http.MethodGet, "/api/discord/connect", nil)
	connectCtx.Set("userID", user.ID)
	h.DiscordConnect(connectCtx)
	state, _ := url.Parse(connectRec.Header().Get("Location"))

	cbCtx, cbRec := newJSONContext(t, http.MethodGet, "/api/discord/callback?code=abc&state="+state.Query().Get("state"), nil)
	cbCtx.Set("userID", user.ID)
	h.DiscordCallback(cbCtx)
	if loc := cbRec.Header().Get("Location"); !strings.Contains(loc, "discord=connected_not_joined") {
		t.Fatalf("callback location = %q", loc)
	}

	bot.grantErr = nil
	syncCtx, syncRec := newJSONContext(t, http.MethodPost, "/api/discord/sync-role", nil)
	syncCtx.Set("userID", user.ID)
	h.DiscordSyncRole(syncCtx)

	if syncRec.Code != http.StatusOK {
		t.Fatalf("sync code = %d body=%s", syncRec.Code, syncRec.Body.String())
	}

	body := syncRec.Body.String()
	if !strings.Contains(body, `"connected":true`) || !strings.Contains(body, `"role_status":"VERIFIED"`) {
		t.Errorf("sync body = %s", body)
	}
}

func TestHandlerDiscordCallbackInvalidState(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "dh3@example.com", "dhuser3", "pass", models.UserRole)
	h := discordEnabledHandler(env, &fakeBot{}, &discord.User{ID: "3"})

	ctx, rec := newJSONContext(t, http.MethodGet, "/api/discord/callback?code=abc&state=bogus", nil)
	ctx.Set("userID", user.ID)
	h.DiscordCallback(ctx)

	if rec.Code != http.StatusFound {
		t.Fatalf("code = %d", rec.Code)
	}

	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "discord=state_invalid") {
		t.Errorf("location = %q", loc)
	}
}

func TestHandlerDiscordUnlink(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "dh4@example.com", "dhuser4", "pass", models.UserRole)
	bot := &fakeBot{}
	h := discordEnabledHandler(env, bot, &discord.User{ID: "4000", Username: "trinity"})

	connectCtx, connectRec := newJSONContext(t, http.MethodGet, "/api/discord/connect", nil)
	connectCtx.Set("userID", user.ID)
	h.DiscordConnect(connectCtx)
	state, _ := url.Parse(connectRec.Header().Get("Location"))

	cbCtx, _ := newJSONContext(t, http.MethodGet, "/api/discord/callback?code=abc&state="+state.Query().Get("state"), nil)
	cbCtx.Set("userID", user.ID)
	h.DiscordCallback(cbCtx)

	unlinkCtx, unlinkRec := newJSONContext(t, http.MethodDelete, "/api/discord/unlink", nil)
	unlinkCtx.Set("userID", user.ID)
	h.DiscordUnlink(unlinkCtx)

	if unlinkRec.Code != http.StatusOK {
		t.Fatalf("unlink code = %d", unlinkRec.Code)
	}

	if !bot.kicked {
		t.Error("expected kick call")
	}

	if _, err := repo.NewDiscordRepo(env.db).GetByUserID(context.Background(), user.ID); err == nil {
		t.Error("connection should be deleted")
	}
}

func TestHandlerSubmitFlagAnnounces(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "solveann@example.com", "solveann", "pass", models.UserRole)
	ch := createHandlerChallenge(t, env, "AnnounceChal", 100, "FLAG{ANN}", true)
	bot := &fakeBot{}
	h := discordEnabledHandler(env, bot, &discord.User{ID: "1"})

	ctx, rec := newJSONContext(t, http.MethodPost, "/api/challenges/x/submit", map[string]string{"flag": "FLAG{ANN}"})
	ctx.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", ch.ID)}}
	ctx.Set("userID", user.ID)

	h.SubmitFlag(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit status %d: %s", rec.Code, rec.Body.String())
	}

	if len(bot.announced) != 1 {
		t.Fatalf("expected 1 announcement, got %d", len(bot.announced))
	}

	if !strings.Contains(bot.announced[0], "AnnounceChal") || !strings.Contains(bot.announced[0], "First Blood") {
		t.Errorf("announce = %q", bot.announced[0])
	}
}

func TestHandlerUpdateMeSyncsNickname(t *testing.T) {
	env := setupHandlerTest(t)
	user := createHandlerUser(t, env, "nickupd@example.com", "nickupd", "pass", models.UserRole)
	bot := &fakeBot{}
	h := discordEnabledHandler(env, bot, &discord.User{ID: "1"})

	conn := &models.DiscordConnection{UserID: user.ID, DiscordUserID: "1", RoleStatus: models.DiscordStatusVerified, ConnectedAt: time.Now().UTC()}
	if err := repo.NewDiscordRepo(env.db).Create(context.Background(), conn); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	ctx, rec := newJSONContext(t, http.MethodPut, "/api/me", map[string]string{"username": "renamed"})
	ctx.Set("userID", user.ID)

	h.UpdateMe(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(bot.nickname, "renamed") {
		t.Fatalf("expected nickname synced to renamed, got %q", bot.nickname)
	}
}
