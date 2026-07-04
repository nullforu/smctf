package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"smctf/internal/config"
	"smctf/internal/discord"
	"smctf/internal/models"
	"smctf/internal/repo"

	"github.com/redis/go-redis/v9"
)

type DiscordOAuth interface {
	AuthorizeURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*discord.TokenResult, error)
	FetchUser(ctx context.Context, accessToken string) (*discord.User, error)
}

type DiscordService struct {
	cfg   config.DiscordConfig
	repo  *repo.DiscordRepo
	users *repo.UserRepo
	bot   discord.BotAPI
	oauth DiscordOAuth
	redis *redis.Client
	wg    sync.WaitGroup
}

func (s *DiscordService) Wait() {
	s.wg.Wait()
}

func NewDiscordService(cfg config.DiscordConfig, discordRepo *repo.DiscordRepo, userRepo *repo.UserRepo, bot discord.BotAPI, oauth DiscordOAuth, redisClient *redis.Client) *DiscordService {
	return &DiscordService{
		cfg:   cfg,
		repo:  discordRepo,
		users: userRepo,
		bot:   bot,
		oauth: oauth,
		redis: redisClient,
	}
}

func (s *DiscordService) ensureEnabled() error {
	if s == nil || !s.cfg.Enabled {
		return ErrDiscordDisabled
	}

	return nil
}

func stateKey(state string) string {
	return "discord:oauth:state:" + state
}

func (s *DiscordService) BeginConnect(ctx context.Context, userID int64) (string, error) {
	if err := s.ensureEnabled(); err != nil {
		return "", err
	}

	state, err := randomState()
	if err != nil {
		return "", fmt.Errorf("discord.BeginConnect state: %w", err)
	}

	if err := s.redis.Set(ctx, stateKey(state), userID, s.cfg.StateTTL).Err(); err != nil {
		return "", fmt.Errorf("discord.BeginConnect store state: %w", err)
	}

	return s.oauth.AuthorizeURL(state), nil
}

func (s *DiscordService) HandleCallback(ctx context.Context, userID int64, code, state string) (*models.DiscordConnection, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	if code == "" || state == "" {
		return nil, ErrDiscordStateInvalid
	}

	if err := s.consumeState(ctx, state, userID); err != nil {
		return nil, err
	}

	token, err := s.oauth.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiscordExchangeFailed, err)
	}

	discordUser, err := s.oauth.FetchUser(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiscordExchangeFailed, err)
	}

	conn, err := s.upsertConnection(ctx, userID, discordUser)
	if err != nil {
		return nil, err
	}

	s.provision(ctx, conn, token.AccessToken)

	if err := s.repo.Update(ctx, conn); err != nil {
		return nil, err
	}

	s.SyncNickname(ctx, userID)

	return conn, nil
}

func (s *DiscordService) SyncRole(ctx context.Context, userID int64) (*models.DiscordConnection, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	conn, err := s.getConnection(ctx, userID)
	if err != nil {
		return nil, err
	}

	s.provision(ctx, conn, "")

	if err := s.repo.Update(ctx, conn); err != nil {
		return nil, err
	}

	return conn, nil
}

func (s *DiscordService) Unlink(ctx context.Context, userID int64) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}

	conn, err := s.getConnection(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.bot.KickMember(ctx, conn.DiscordUserID); err != nil {
		slog.Warn("discord guild kick failed during unlink",
			slog.Int64("user_id", userID),
			slog.Any("error", err),
		)
	}

	return s.repo.Delete(ctx, conn)
}

func (s *DiscordService) GetConnection(ctx context.Context, userID int64) (*models.DiscordConnection, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}

	conn, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return conn, nil
}

func (s *DiscordService) getConnection(ctx context.Context, userID int64) (*models.DiscordConnection, error) {
	conn, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrDiscordNotConnected
		}

		return nil, err
	}

	return conn, nil
}

func (s *DiscordService) consumeState(ctx context.Context, state string, userID int64) error {
	stored, err := s.redis.GetDel(ctx, stateKey(state)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrDiscordStateInvalid
		}

		return fmt.Errorf("discord.consumeState: %w", err)
	}

	if stored != strconv.FormatInt(userID, 10) {
		return ErrDiscordStateInvalid
	}

	return nil
}

func (s *DiscordService) upsertConnection(ctx context.Context, userID int64, discordUser *discord.User) (*models.DiscordConnection, error) {
	existing, err := s.repo.GetByDiscordUserID(ctx, discordUser.ID)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}

	if existing != nil && existing.UserID != userID {
		return nil, ErrDiscordAlreadyLinked
	}

	conn, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			return nil, err
		}

		conn = &models.DiscordConnection{
			UserID:      userID,
			ConnectedAt: time.Now().UTC(),
			RoleStatus:  models.DiscordStatusConnected,
		}
	}

	conn.DiscordUserID = discordUser.ID
	conn.DiscordUsername = &discordUser.Username
	conn.DiscordGlobalName = discordUser.GlobalName
	conn.DiscordAvatar = discordUser.Avatar
	conn.RevokedAt = nil
	conn.UpdatedAt = time.Now().UTC()

	if conn.ID == 0 {
		if err := s.repo.Create(ctx, conn); err != nil {
			return nil, err
		}
	}

	return conn, nil
}

func (s *DiscordService) provision(ctx context.Context, conn *models.DiscordConnection, accessToken string) {
	now := time.Now().UTC()
	conn.LastSyncedAt = &now
	conn.UpdatedAt = now
	conn.LastError = nil

	if s.cfg.AutoJoin && accessToken != "" {
		if err := s.bot.JoinGuild(ctx, conn.DiscordUserID, accessToken); err != nil {
			slog.Warn("discord guild join failed",
				slog.Int64("user_id", conn.UserID),
				slog.Any("error", err),
			)
		}
	}

	if err := s.bot.GrantRole(ctx, conn.DiscordUserID); err != nil {
		s.applyGrantError(conn, err)
		return
	}

	conn.RoleStatus = models.DiscordStatusVerified
	conn.VerifiedAt = &now
}

func (s *DiscordService) applyGrantError(conn *models.DiscordConnection, err error) {
	msg := err.Error()
	conn.LastError = &msg
	conn.VerifiedAt = nil

	switch {
	case errors.Is(err, discord.ErrNotInGuild):
		conn.RoleStatus = models.DiscordStatusNotInGuild
	default:
		conn.RoleStatus = models.DiscordStatusRoleFailed
	}
}

func (s *DiscordService) AnnounceSolve(_ context.Context, userID int64, challengeTitle string, firstBlood bool) {
	if err := s.ensureEnabled(); err != nil {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.announceSolve(context.Background(), userID, challengeTitle, firstBlood)
	}()
}

func (s *DiscordService) announceSolve(ctx context.Context, userID int64, challengeTitle string, firstBlood bool) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		slog.Warn("discord announce: user lookup failed", slog.Int64("user_id", userID), slog.Any("error", err))
		return
	}

	team := fmt.Sprintf("%s_%s", user.DivisionName, user.TeamName)
	var content string
	if firstBlood {
		content = fmt.Sprintf("\U0001FA78 **First Blood!** **%s** — %s solved **%s**!", team, user.Username, challengeTitle)
	} else {
		content = fmt.Sprintf("**%s** — %s solved **%s**", team, user.Username, challengeTitle)
	}

	if err := s.bot.Announce(ctx, content); err != nil {
		slog.Warn("discord announce failed", slog.Int64("user_id", userID), slog.Any("error", err))
	}
}

func (s *DiscordService) SyncNickname(_ context.Context, userID int64) {
	if err := s.ensureEnabled(); err != nil {
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.syncNickname(context.Background(), userID)
	}()
}

func (s *DiscordService) syncNickname(ctx context.Context, userID int64) {
	conn, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			slog.Warn("discord nickname: connection lookup failed", slog.Int64("user_id", userID), slog.Any("error", err))
		}
		return
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		slog.Warn("discord nickname: user lookup failed", slog.Int64("user_id", userID), slog.Any("error", err))
		return
	}

	nickname := fmt.Sprintf("%s_%s_%s", user.DivisionName, user.TeamName, user.Username)
	if err := s.bot.SetNickname(ctx, conn.DiscordUserID, nickname); err != nil {
		if !errors.Is(err, discord.ErrNotInGuild) {
			slog.Warn("discord nickname sync failed", slog.Int64("user_id", userID), slog.Any("error", err))
		}
	}
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
