package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/discord"
	"smctf/internal/models"
	"smctf/internal/repo"
)

type fakeBot struct {
	joinErr   error
	grantErr  error
	kickErr   error
	joined    bool
	granted   bool
	kicked    bool
	announced []string
	nickname  string
}

func (f *fakeBot) JoinGuild(_ context.Context, _, _ string) error {
	f.joined = true
	return f.joinErr
}

func (f *fakeBot) GrantRole(_ context.Context, _ string) error {
	f.granted = true
	return f.grantErr
}

func (f *fakeBot) KickMember(_ context.Context, _ string) error {
	f.kicked = true
	return f.kickErr
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
	return &discord.TokenResult{AccessToken: "at-test"}, nil
}

func (f fakeOAuth) FetchUser(_ context.Context, _ string) (*discord.User, error) {
	return f.user, nil
}

func discordTestConfig() config.DiscordConfig {
	return config.DiscordConfig{
		Enabled:     true,
		StateTTL:    5 * time.Minute,
		AutoJoin:    true,
		RedirectURI: "https://example.com/cb",
		Scopes:      "identify guilds.join",
	}
}

func newDiscordServiceForTest(env serviceEnv, bot discord.BotAPI, user *discord.User) (*DiscordService, *repo.DiscordRepo) {
	discordRepo := repo.NewDiscordRepo(env.db)
	svc := NewDiscordService(discordTestConfig(), discordRepo, env.userRepo, bot, fakeOAuth{user: user}, env.redis)
	return svc, discordRepo
}

func TestDiscordBeginConnectStoresState(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "d1@example.com", "duser1", "pass", models.UserRole)

	svc, _ := newDiscordServiceForTest(env, &fakeBot{}, &discord.User{ID: "100", Username: "neo"})

	url, err := svc.BeginConnect(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("BeginConnect: %v", err)
	}

	if url == "" {
		t.Fatal("empty authorize url")
	}

	keys := env.redis.Keys(context.Background(), "discord:oauth:state:*").Val()
	if len(keys) != 1 {
		t.Fatalf("expected 1 state key, got %d", len(keys))
	}
}

func TestDiscordHandleCallbackVerified(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "d2@example.com", "duser2", "pass", models.UserRole)

	bot := &fakeBot{}
	svc, discordRepo := newDiscordServiceForTest(env, bot, &discord.User{ID: "200", Username: "trinity"})

	state := seedState(t, env, user.ID)
	conn, err := svc.HandleCallback(context.Background(), user.ID, "code", state)
	if err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}

	if conn.RoleStatus != models.DiscordStatusVerified {
		t.Errorf("status = %q", conn.RoleStatus)
	}

	if !bot.joined || !bot.granted {
		t.Errorf("expected join+grant, got joined=%v granted=%v", bot.joined, bot.granted)
	}

	stored, err := discordRepo.GetByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}

	if stored.DiscordUserID != "200" {
		t.Errorf("discord id = %q", stored.DiscordUserID)
	}
}

func TestDiscordHandleCallbackNotInGuild(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "d3@example.com", "duser3", "pass", models.UserRole)

	bot := &fakeBot{grantErr: discord.ErrNotInGuild}
	svc, _ := newDiscordServiceForTest(env, bot, &discord.User{ID: "300", Username: "morpheus"})

	state := seedState(t, env, user.ID)
	conn, err := svc.HandleCallback(context.Background(), user.ID, "code", state)
	if err != nil {
		t.Fatalf("HandleCallback: %v", err)
	}

	if conn.RoleStatus != models.DiscordStatusNotInGuild {
		t.Errorf("status = %q", conn.RoleStatus)
	}
}

func TestDiscordHandleCallbackStateMismatch(t *testing.T) {
	env := setupServiceTest(t)
	owner := createUserWithNewTeam(t, env, "d4@example.com", "duser4", "pass", models.UserRole)

	svc, _ := newDiscordServiceForTest(env, &fakeBot{}, &discord.User{ID: "400"})

	if _, err := svc.HandleCallback(context.Background(), owner.ID, "code", "bogus-state"); !errors.Is(err, ErrDiscordStateInvalid) {
		t.Fatalf("bogus state: got %v, want ErrDiscordStateInvalid", err)
	}

	state := seedState(t, env, owner.ID)
	if _, err := svc.HandleCallback(context.Background(), owner.ID+999, "code", state); !errors.Is(err, ErrDiscordStateInvalid) {
		t.Fatalf("mismatched user: got %v, want ErrDiscordStateInvalid", err)
	}
}

func TestDiscordHandleCallbackAlreadyLinked(t *testing.T) {
	env := setupServiceTest(t)
	owner := createUserWithNewTeam(t, env, "d5@example.com", "duser5", "pass", models.UserRole)
	other := createUserWithNewTeam(t, env, "d6@example.com", "duser6", "pass", models.UserRole)

	svc, discordRepo := newDiscordServiceForTest(env, &fakeBot{}, &discord.User{ID: "500", Username: "cypher"})

	state := seedState(t, env, owner.ID)
	if _, err := svc.HandleCallback(context.Background(), owner.ID, "code", state); err != nil {
		t.Fatalf("owner link: %v", err)
	}

	state2 := seedState(t, env, other.ID)
	_, err := svc.HandleCallback(context.Background(), other.ID, "code", state2)
	if !errors.Is(err, ErrDiscordAlreadyLinked) {
		t.Fatalf("got %v, want ErrDiscordAlreadyLinked", err)
	}

	if _, err := discordRepo.GetByUserID(context.Background(), other.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("other user should have no connection, got %v", err)
	}
}

func TestDiscordUnlinkRemovesConnection(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "d7@example.com", "duser7", "pass", models.UserRole)

	bot := &fakeBot{}
	svc, discordRepo := newDiscordServiceForTest(env, bot, &discord.User{ID: "700"})

	state := seedState(t, env, user.ID)
	if _, err := svc.HandleCallback(context.Background(), user.ID, "code", state); err != nil {
		t.Fatalf("link: %v", err)
	}

	if err := svc.Unlink(context.Background(), user.ID); err != nil {
		t.Fatalf("Unlink: %v", err)
	}

	if !bot.kicked {
		t.Error("expected kick call")
	}

	if _, err := discordRepo.GetByUserID(context.Background(), user.ID); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("connection should be gone, got %v", err)
	}
}

func TestDiscordGetConnectionNilWhenAbsent(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "d8@example.com", "duser8", "pass", models.UserRole)

	svc, _ := newDiscordServiceForTest(env, &fakeBot{}, &discord.User{ID: "800"})

	conn, err := svc.GetConnection(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}

	if conn != nil {
		t.Fatalf("expected nil connection, got %+v", conn)
	}
}

func TestDiscordDisabledReturnsError(t *testing.T) {
	env := setupServiceTest(t)
	discordRepo := repo.NewDiscordRepo(env.db)
	svc := NewDiscordService(config.DiscordConfig{Enabled: false}, discordRepo, env.userRepo, &fakeBot{}, fakeOAuth{user: &discord.User{ID: "1"}}, env.redis)

	if _, err := svc.BeginConnect(context.Background(), 1); !errors.Is(err, ErrDiscordDisabled) {
		t.Fatalf("got %v, want ErrDiscordDisabled", err)
	}
}

func seedState(t *testing.T, env serviceEnv, userID int64) string {
	t.Helper()
	state := "teststate-" + time.Now().Format("150405.000000000")
	if err := env.redis.Set(context.Background(), stateKey(state), userID, time.Minute).Err(); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	return state
}

func TestDiscordAnnounceSolve(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "ann@example.com", "annuser", "pass", models.UserRole)
	bot := &fakeBot{}
	svc, _ := newDiscordServiceForTest(env, bot, &discord.User{ID: "1"})

	svc.AnnounceSolve(context.Background(), user.ID, "Web-101", true)
	svc.AnnounceSolve(context.Background(), user.ID, "Web-102", false)

	if len(bot.announced) != 2 {
		t.Fatalf("expected 2 announcements, got %d", len(bot.announced))
	}

	if !strings.Contains(bot.announced[0], "First Blood") || !strings.Contains(bot.announced[0], "Web-101") || !strings.Contains(bot.announced[0], "annuser") {
		t.Errorf("first blood msg = %q", bot.announced[0])
	}

	if strings.Contains(bot.announced[1], "First Blood") || !strings.Contains(bot.announced[1], "solved") {
		t.Errorf("normal msg = %q", bot.announced[1])
	}
}

func TestDiscordSyncNickname(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "nick@example.com", "nickuser", "pass", models.UserRole)
	bot := &fakeBot{}
	svc, discordRepo := newDiscordServiceForTest(env, bot, &discord.User{ID: "555"})

	svc.SyncNickname(context.Background(), user.ID)
	if bot.nickname != "" {
		t.Fatalf("expected skip without connection, got %q", bot.nickname)
	}

	conn := &models.DiscordConnection{UserID: user.ID, DiscordUserID: "555", RoleStatus: models.DiscordStatusVerified, ConnectedAt: time.Now().UTC()}
	if err := discordRepo.Create(context.Background(), conn); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	svc.SyncNickname(context.Background(), user.ID)
	if !strings.Contains(bot.nickname, "nickuser") {
		t.Fatalf("nickname = %q (expected division_team_nickuser)", bot.nickname)
	}
}

func TestCTFServiceWasFirstBlood(t *testing.T) {
	env := setupServiceTest(t)
	user := createUserWithNewTeam(t, env, "cfb@example.com", "cfb", "pass", models.UserRole)
	ch := createChallenge(t, env, "cfbch", 100, "FLAG{CFB}", true)

	correct, err := env.ctfSvc.SubmitFlag(context.Background(), user.ID, ch.ID, "FLAG{CFB}")
	if err != nil || !correct {
		t.Fatalf("submit: correct=%v err=%v", correct, err)
	}

	fb, err := env.ctfSvc.WasFirstBlood(context.Background(), user.ID, ch.ID)
	if err != nil {
		t.Fatalf("WasFirstBlood: %v", err)
	}

	if !fb {
		t.Fatal("expected first blood for first solver")
	}
}
