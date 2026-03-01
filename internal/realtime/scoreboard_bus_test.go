package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"smctf/internal/config"
	"smctf/internal/logging"
	"smctf/internal/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestBus(t *testing.T, score ScoreboardReader, divisions DivisionReader) (*ScoreboardBus, *redis.Client, func()) {
	t.Helper()

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})

	logger, err := logging.New(config.LoggingConfig{}, logging.Options{Service: "smctf", Env: "test"})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}

	cfg := config.Config{Cache: config.CacheConfig{LeaderboardTTL: time.Minute, TimelineTTL: time.Minute}}

	hub := NewSSEHub()
	bus := NewScoreboardBus(client, cfg, score, divisions, logger, hub)

	cleanup := func() {
		_ = client.Close()
		redisServer.Close()
		_ = logger.Close()
	}

	return bus, client, cleanup
}

func TestScoreboardBusPublish(t *testing.T) {
	bus, client, cleanup := newTestBus(t, nil, nil)
	defer cleanup()

	ctx := context.Background()
	sub := client.Subscribe(ctx, scoreboardEventsChannel)
	defer sub.Close()

	event := ScoreboardEvent{Scope: "all", Reason: "test", TS: time.Now().UTC()}
	bus.Publish(ctx, event)

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive message: %v", err)
	}

	var got ScoreboardEvent
	if err := json.Unmarshal([]byte(msg.Payload), &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	if got.Reason != "test" || got.Scope != "all" {
		t.Fatalf("unexpected event: %+v", got)
	}
}

func TestScoreboardBusAcquireReleaseLock(t *testing.T) {
	bus, client, cleanup := newTestBus(t, nil, nil)
	defer cleanup()

	ctx := context.Background()
	ok, token := bus.acquireLock(ctx)
	if !ok {
		t.Fatalf("expected lock to be acquired")
	}

	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	bus.releaseLock(ctx, "wrong-token")
	if got, err := client.Get(ctx, scoreboardLockKey).Result(); err != nil || got != token {
		t.Fatalf("expected lock to remain, got %q err %v", got, err)
	}

	bus.releaseLock(ctx, token)
	if exists, _ := client.Exists(ctx, scoreboardLockKey).Result(); exists != 0 {
		t.Fatalf("expected lock to be released")
	}
}

func TestScoreboardBusRebuiltBroadcast(t *testing.T) {
	bus, client, cleanup := newTestBus(t, nil, nil)
	defer cleanup()

	ctx := t.Context()
	bus.Start(ctx)

	ch, unsubscribe := bus.hub.Subscribe(1)
	defer unsubscribe()

	payload := "{\"scope\":\"all\",\"reason\":\"rebuilt\"}"
	if err := client.Publish(ctx, scoreboardRebuiltChannel, payload).Err(); err != nil {
		t.Fatalf("publish rebuilt: %v", err)
	}

	select {
	case msg := <-ch:
		if msg != payload {
			t.Fatalf("unexpected payload: %q", msg)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for broadcast")
	}
}

func TestScoreboardBusHandleEventSkipsWhenLocked(t *testing.T) {
	bus, client, cleanup := newTestBus(t, nil, nil)
	defer cleanup()

	ctx := context.Background()
	if err := client.Set(ctx, scoreboardLockKey, "held", time.Minute).Err(); err != nil {
		t.Fatalf("seed lock: %v", err)
	}

	sub := client.Subscribe(ctx, scoreboardRebuiltChannel)
	defer sub.Close()

	bus.handleEvent(ctx, ScoreboardEvent{Scope: "all"})

	select {
	case <-sub.Channel():
		t.Fatalf("unexpected rebuilt event")
	case <-time.After(150 * time.Millisecond):
	}
}

type fakeScoreboard struct {
	leaderboard        models.LeaderboardResponse
	teamLeaderboard    models.TeamLeaderboardResponse
	userTimeline       []models.TimelineSubmission
	teamTimeline       []models.TeamTimelineSubmission
	leaderboardErr     error
	teamLeaderboardErr error
	userTimelineErr    error
	teamTimelineErr    error
	leaderboardCalls   []int64
	teamCalls          []int64
	userTimelineCalls  []int64
	teamTimelineCalls  []int64
}

type fakeDivisionReader struct {
	divisions []models.Division
	err       error
}

func (f fakeDivisionReader) ListDivisions(ctx context.Context) ([]models.Division, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.divisions, nil
}

func (f *fakeScoreboard) Leaderboard(ctx context.Context, divisionID *int64) (models.LeaderboardResponse, error) {
	f.leaderboardCalls = append(f.leaderboardCalls, divisionIDValue(divisionID))
	return f.leaderboard, f.leaderboardErr
}

func (f *fakeScoreboard) TeamLeaderboard(ctx context.Context, divisionID *int64) (models.TeamLeaderboardResponse, error) {
	f.teamCalls = append(f.teamCalls, divisionIDValue(divisionID))
	return f.teamLeaderboard, f.teamLeaderboardErr
}

func (f *fakeScoreboard) UserTimeline(ctx context.Context, since *time.Time, divisionID *int64) ([]models.TimelineSubmission, error) {
	f.userTimelineCalls = append(f.userTimelineCalls, divisionIDValue(divisionID))
	return f.userTimeline, f.userTimelineErr
}

func (f *fakeScoreboard) TeamTimeline(ctx context.Context, since *time.Time, divisionID *int64) ([]models.TeamTimelineSubmission, error) {
	f.teamTimelineCalls = append(f.teamTimelineCalls, divisionIDValue(divisionID))
	return f.teamTimeline, f.teamTimelineErr
}

func divisionIDValue(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}

func TestScoreboardBusHandleEventRebuildsAndPublishes(t *testing.T) {
	score := &fakeScoreboard{
		leaderboard:     models.LeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.LeaderboardEntry{}},
		teamLeaderboard: models.TeamLeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.TeamLeaderboardEntry{}},
		userTimeline:    []models.TimelineSubmission{},
		teamTimeline:    []models.TeamTimelineSubmission{},
	}

	divReader := fakeDivisionReader{divisions: []models.Division{{ID: 1, Name: "A"}}}
	bus, client, cleanup := newTestBus(t, score, divReader)
	defer cleanup()

	ctx := context.Background()
	sub := client.Subscribe(ctx, scoreboardRebuiltChannel)
	defer sub.Close()

	bus.handleEvent(ctx, ScoreboardEvent{Scope: "division", Reason: "test", DivisionIDs: []int64{1}})

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive rebuilt: %v", err)
	}

	if msg.Payload == "" {
		t.Fatalf("expected rebuilt payload")
	}

	if _, err := client.Get(ctx, "leaderboard:users:div:1").Result(); err != nil {
		t.Fatalf("expected leaderboard cache, got %v", err)
	}

	if _, err := client.Get(ctx, "leaderboard:teams:div:1").Result(); err != nil {
		t.Fatalf("expected team leaderboard cache, got %v", err)
	}

	if _, err := client.Get(ctx, "timeline:users:div:1").Result(); err != nil {
		t.Fatalf("expected timeline cache, got %v", err)
	}

	if _, err := client.Get(ctx, "timeline:teams:div:1").Result(); err != nil {
		t.Fatalf("expected team timeline cache, got %v", err)
	}
}

func TestScoreboardBusHandleEventRebuildFails(t *testing.T) {
	score := &fakeScoreboard{
		leaderboardErr: errors.New("boom"),
	}

	divReader := fakeDivisionReader{divisions: []models.Division{{ID: 1, Name: "A"}}}
	bus, client, cleanup := newTestBus(t, score, divReader)
	defer cleanup()

	ctx := context.Background()
	sub := client.Subscribe(ctx, scoreboardRebuiltChannel)
	defer sub.Close()

	bus.handleEvent(ctx, ScoreboardEvent{Scope: "all", Reason: "test"})

	select {
	case <-sub.Channel():
		t.Fatalf("unexpected rebuilt message on failure")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestScoreboardBusStoreJSONError(t *testing.T) {
	bus, _, cleanup := newTestBus(t, nil, nil)
	defer cleanup()

	ctx := context.Background()
	value := func() {}
	if err := bus.storeJSON(ctx, "bad", value, time.Minute); err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestScoreboardBusRebuildCachesError(t *testing.T) {
	score := &fakeScoreboard{
		leaderboardErr: errors.New("boom"),
	}

	divReader := fakeDivisionReader{divisions: []models.Division{{ID: 1, Name: "A"}}}
	bus, _, cleanup := newTestBus(t, score, divReader)
	defer cleanup()

	if err := bus.rebuildCaches(context.Background(), []int64{1}); err == nil {
		t.Fatalf("expected rebuild error")
	}
}

func TestScoreboardBusRunDebounce(t *testing.T) {
	score := &fakeScoreboard{
		leaderboard:     models.LeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.LeaderboardEntry{}},
		teamLeaderboard: models.TeamLeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.TeamLeaderboardEntry{}},
		userTimeline:    []models.TimelineSubmission{},
		teamTimeline:    []models.TeamTimelineSubmission{},
	}

	bus, client, cleanup := newTestBus(t, score, nil)
	defer cleanup()

	ctx := t.Context()
	bus.Start(ctx)

	sub := client.Subscribe(ctx, scoreboardRebuiltChannel)
	defer sub.Close()

	if err := client.Publish(ctx, scoreboardEventsChannel, `{"scope":"all","reason":"a"}`).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if err := client.Publish(ctx, scoreboardEventsChannel, `{"scope":"all","reason":"b"}`).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive rebuilt: %v", err)
	}

	if msg.Payload == "" {
		t.Fatalf("expected rebuilt payload")
	}
}

func TestScoreboardBusRunDebounceMergesDivisions(t *testing.T) {
	score := &fakeScoreboard{
		leaderboard:     models.LeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.LeaderboardEntry{}},
		teamLeaderboard: models.TeamLeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.TeamLeaderboardEntry{}},
		userTimeline:    []models.TimelineSubmission{},
		teamTimeline:    []models.TeamTimelineSubmission{},
	}

	divReader := fakeDivisionReader{divisions: []models.Division{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}}
	bus, client, cleanup := newTestBus(t, score, divReader)
	defer cleanup()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	bus.Start(ctx)

	sub := client.Subscribe(ctx, scoreboardRebuiltChannel)
	defer sub.Close()

	payloadA, _ := json.Marshal(ScoreboardEvent{Scope: "division", Reason: "a", DivisionIDs: []int64{1}})
	payloadB, _ := json.Marshal(ScoreboardEvent{Scope: "division", Reason: "b", DivisionIDs: []int64{2}})

	if err := client.Publish(ctx, scoreboardEventsChannel, string(payloadA)).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err := client.Publish(ctx, scoreboardEventsChannel, string(payloadB)).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive rebuilt: %v", err)
	}

	var got ScoreboardEvent
	if err := json.Unmarshal([]byte(msg.Payload), &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if got.Scope != "division" {
		t.Fatalf("expected division scope, got %q", got.Scope)
	}
	if got.Reason != "batch" {
		t.Fatalf("expected batch reason, got %q", got.Reason)
	}
	if len(got.DivisionIDs) != 2 {
		t.Fatalf("expected 2 divisions, got %v", got.DivisionIDs)
	}

	if _, err := client.Get(ctx, "leaderboard:users:div:1").Result(); err != nil {
		t.Fatalf("expected leaderboard cache for div 1, got %v", err)
	}
	if _, err := client.Get(ctx, "leaderboard:users:div:2").Result(); err != nil {
		t.Fatalf("expected leaderboard cache for div 2, got %v", err)
	}
}

func TestScoreboardBusRunDebounceAllOverridesDivision(t *testing.T) {
	score := &fakeScoreboard{
		leaderboard:     models.LeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.LeaderboardEntry{}},
		teamLeaderboard: models.TeamLeaderboardResponse{Challenges: []models.LeaderboardChallenge{}, Entries: []models.TeamLeaderboardEntry{}},
		userTimeline:    []models.TimelineSubmission{},
		teamTimeline:    []models.TeamTimelineSubmission{},
	}

	divReader := fakeDivisionReader{divisions: []models.Division{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}}
	bus, client, cleanup := newTestBus(t, score, divReader)
	defer cleanup()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	bus.Start(ctx)

	sub := client.Subscribe(ctx, scoreboardRebuiltChannel)
	defer sub.Close()

	payloadDivision, _ := json.Marshal(ScoreboardEvent{Scope: "division", Reason: "a", DivisionIDs: []int64{1}})
	payloadAll, _ := json.Marshal(ScoreboardEvent{Scope: "all", Reason: "b"})

	if err := client.Publish(ctx, scoreboardEventsChannel, string(payloadDivision)).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err := client.Publish(ctx, scoreboardEventsChannel, string(payloadAll)).Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("receive rebuilt: %v", err)
	}

	var got ScoreboardEvent
	if err := json.Unmarshal([]byte(msg.Payload), &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if got.Scope != "all" {
		t.Fatalf("expected all scope, got %q", got.Scope)
	}
	if len(got.DivisionIDs) != 0 {
		t.Fatalf("expected no division_ids, got %v", got.DivisionIDs)
	}

	if _, err := client.Get(ctx, "leaderboard:users:div:1").Result(); err != nil {
		t.Fatalf("expected leaderboard cache for div 1, got %v", err)
	}
	if _, err := client.Get(ctx, "leaderboard:users:div:2").Result(); err != nil {
		t.Fatalf("expected leaderboard cache for div 2, got %v", err)
	}
}
