package realtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"smctf/internal/config"
	"smctf/internal/logging"
	"smctf/internal/models"

	"github.com/redis/go-redis/v9"
)

const (
	scoreboardEventsChannel  = "scoreboard.events"
	scoreboardRebuiltChannel = "scoreboard.rebuilt"
	scoreboardLockKey        = "scoreboard:rebuild:lock"
)

type ScoreboardEvent struct {
	Scope  string    `json:"scope"`
	Reason string    `json:"reason"`
	TS     time.Time `json:"ts"`
}

type ScoreboardBus struct {
	redis     *redis.Client
	cfg       config.Config
	score     ScoreboardReader
	divisions DivisionReader
	logger    *logging.Logger
	hub       *SSEHub
	debounce  time.Duration
	lockTTL   time.Duration
	trigger   chan string
}

type ScoreboardReader interface {
	Leaderboard(ctx context.Context, divisionID *int64) (models.LeaderboardResponse, error)
	TeamLeaderboard(ctx context.Context, divisionID *int64) (models.TeamLeaderboardResponse, error)
	UserTimeline(ctx context.Context, since *time.Time, divisionID *int64) ([]models.TimelineSubmission, error)
	TeamTimeline(ctx context.Context, since *time.Time, divisionID *int64) ([]models.TeamTimelineSubmission, error)
}

type DivisionReader interface {
	ListDivisions(ctx context.Context) ([]models.Division, error)
}

func NewScoreboardBus(redisClient *redis.Client, cfg config.Config, scoreSvc ScoreboardReader, divisionReader DivisionReader, logger *logging.Logger, hub *SSEHub) *ScoreboardBus {
	return &ScoreboardBus{
		redis:     redisClient,
		cfg:       cfg,
		score:     scoreSvc,
		divisions: divisionReader,
		logger:    logger,
		hub:       hub,
		debounce:  300 * time.Millisecond,
		lockTTL:   10 * time.Second,
		trigger:   make(chan string, 16),
	}
}

func (b *ScoreboardBus) Publish(ctx context.Context, event ScoreboardEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	_ = b.redis.Publish(ctx, scoreboardEventsChannel, payload).Err()
}

func (b *ScoreboardBus) Start(ctx context.Context) {
	pubsub := b.redis.Subscribe(ctx, scoreboardEventsChannel)
	rebuilt := b.redis.Subscribe(ctx, scoreboardRebuiltChannel)

	go b.run(ctx, pubsub, rebuilt)
}

func (b *ScoreboardBus) run(ctx context.Context, pubsub *redis.PubSub, rebuilt *redis.PubSub) {
	defer func() {
		if err := pubsub.Close(); err != nil {
			b.logger.Warn("leaderboard pubsub close", slog.Any("error", err))
		}

		if err := rebuilt.Close(); err != nil {
			b.logger.Warn("leaderboard rebuilt close", slog.Any("error", err))
		}
	}()

	go func() {
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}

				select {
				case b.trigger <- msg.Payload:
				default:
				}
			}
		}
	}()

	go func() {
		ch := rebuilt.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}

				b.hub.Broadcast(msg.Payload)
			}
		}
	}()

	var (
		timer       *time.Timer
		lastPayload string
	)

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case payload := <-b.trigger:
			lastPayload = payload
			if timer == nil {
				timer = time.NewTimer(b.debounce)
				continue
			}

			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}

			timer.Reset(b.debounce)
		case <-func() <-chan time.Time {
			if timer == nil {
				return nil
			}
			return timer.C
		}():
			if timer != nil {
				timer.Stop()
				timer = nil
			}

			b.handleEvent(ctx, lastPayload)
		}
	}
}

func (b *ScoreboardBus) handleEvent(ctx context.Context, payload string) {
	locked, token := b.acquireLock(ctx)
	if !locked {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := b.rebuildCaches(ctx); err != nil {
		b.logger.Warn("leaderboard rebuild failed", slog.Any("error", err))
		b.releaseLock(ctx, token)
		return
	}

	b.releaseLock(ctx, token)

	_ = b.redis.Publish(ctx, scoreboardRebuiltChannel, payload).Err()
}

func (b *ScoreboardBus) acquireLock(ctx context.Context) (bool, string) {
	token := randomToken()
	ok, err := b.redis.SetNX(ctx, scoreboardLockKey, token, b.lockTTL).Result()
	if err != nil {
		b.logger.Warn("leaderboard lock error", slog.Any("error", err))
		return false, ""
	}

	return ok, token
}

func (b *ScoreboardBus) releaseLock(ctx context.Context, token string) {
	if token == "" {
		return
	}

	const script = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	_, _ = b.redis.Eval(ctx, script, []string{scoreboardLockKey}, token).Result()
}

func randomToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}

	return hex.EncodeToString(buf)
}

func (b *ScoreboardBus) rebuildCaches(ctx context.Context) error {
	if err := b.rebuildDivisionCaches(ctx, nil); err != nil {
		return err
	}

	if b.divisions == nil {
		return nil
	}

	divisions, err := b.divisions.ListDivisions(ctx)
	if err != nil {
		return err
	}

	for i := range divisions {
		id := divisions[i].ID
		if err := b.rebuildDivisionCaches(ctx, &id); err != nil {
			return err
		}
	}

	return nil
}

func cacheKey(base string, divisionID *int64) string {
	if divisionID == nil {
		return base
	}

	return fmt.Sprintf("%s:div:%d", base, *divisionID)
}

func (b *ScoreboardBus) rebuildDivisionCaches(ctx context.Context, divisionID *int64) error {
	leaderboard, err := b.score.Leaderboard(ctx, divisionID)
	if err != nil {
		return err
	}

	teamLeaderboard, err := b.score.TeamLeaderboard(ctx, divisionID)
	if err != nil {
		return err
	}

	userTimeline, err := b.score.UserTimeline(ctx, nil, divisionID)
	if err != nil {
		return err
	}

	teamTimeline, err := b.score.TeamTimeline(ctx, nil, divisionID)
	if err != nil {
		return err
	}

	if err := b.storeJSON(ctx, cacheKey("leaderboard:users", divisionID), leaderboard, b.cfg.Cache.LeaderboardTTL); err != nil {
		return err
	}

	if err := b.storeJSON(ctx, cacheKey("leaderboard:teams", divisionID), teamLeaderboard, b.cfg.Cache.LeaderboardTTL); err != nil {
		return err
	}

	userTimelineResp := struct {
		Submissions []models.TimelineSubmission `json:"submissions"`
	}{Submissions: userTimeline}
	if err := b.storeJSON(ctx, cacheKey("timeline:users", divisionID), userTimelineResp, b.cfg.Cache.TimelineTTL); err != nil {
		return err
	}

	teamTimelineResp := struct {
		Submissions []models.TeamTimelineSubmission `json:"submissions"`
	}{Submissions: teamTimeline}
	if err := b.storeJSON(ctx, cacheKey("timeline:teams", divisionID), teamTimelineResp, b.cfg.Cache.TimelineTTL); err != nil {
		return err
	}

	return nil
}

func (b *ScoreboardBus) storeJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if err := b.redis.Set(ctx, key, data, ttl).Err(); err != nil {
		return err
	}

	return nil
}
