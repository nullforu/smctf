package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smctf/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newRateLimitService(t *testing.T, window time.Duration, max int) *CTFService {
	t.Helper()

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start redis: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})

	t.Cleanup(func() {
		_ = client.Close()
		redisServer.Close()
	})

	cfg := config.Config{
		Security: config.SecurityConfig{
			SubmissionWindow: window,
			SubmissionMax:    max,
		},
	}

	return &CTFService{cfg: cfg, redis: client}
}

func TestRateLimitKey(t *testing.T) {
	if got := rateLimitKey(42); got != "submit:42" {
		t.Fatalf("unexpected key: %s", got)
	}
}

func TestRateLimitStateIncrements(t *testing.T) {
	svc := newRateLimitService(t, 10*time.Second, 2)
	ctx := context.Background()
	key := rateLimitKey(11)

	count, ttl, err := svc.rateLimitState(ctx, key)
	if err != nil {
		t.Fatalf("rateLimitState: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	if ttl > 0 {
		t.Fatalf("expected no ttl yet, got %v", ttl)
	}

	count, ttl, err = svc.rateLimitState(ctx, key)
	if err != nil {
		t.Fatalf("rateLimitState second: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	if ttl > 0 {
		t.Fatalf("expected no ttl yet, got %v", ttl)
	}
}

func TestEnsureRateLimitTTLWhenMissing(t *testing.T) {
	svc := newRateLimitService(t, 45*time.Second, 5)
	ctx := context.Background()
	key := rateLimitKey(22)

	if err := svc.redis.Set(ctx, key, "1", 0).Err(); err != nil {
		t.Fatalf("set key: %v", err)
	}

	initialTTL := svc.redis.TTL(ctx, key).Val()
	if initialTTL > 0 {
		t.Fatalf("expected missing ttl, got %v", initialTTL)
	}

	gotTTL, err := svc.ensureRateLimitTTL(ctx, key, initialTTL)
	if err != nil {
		t.Fatalf("ensureRateLimitTTL: %v", err)
	}

	if gotTTL != svc.cfg.Security.SubmissionWindow {
		t.Fatalf("expected window %v, got %v", svc.cfg.Security.SubmissionWindow, gotTTL)
	}

	storedTTL := svc.redis.TTL(ctx, key).Val()
	if storedTTL <= 0 || storedTTL > svc.cfg.Security.SubmissionWindow {
		t.Fatalf("unexpected stored ttl: %v", storedTTL)
	}
}

func TestEvaluateRateLimit(t *testing.T) {
	svc := newRateLimitService(t, 10*time.Second, 2)

	if err := svc.evaluateRateLimit(2, 3*time.Second); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err := svc.evaluateRateLimit(3, 1500*time.Millisecond)
	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected rate limit error, got %v", err)
	}

	if rlErr.Info.Limit != 2 {
		t.Fatalf("expected limit 2, got %d", rlErr.Info.Limit)
	}

	if rlErr.Info.Remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", rlErr.Info.Remaining)
	}

	if rlErr.Info.ResetSeconds != 2 {
		t.Fatalf("expected reset 2, got %d", rlErr.Info.ResetSeconds)
	}

	err = svc.evaluateRateLimit(3, 0)
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected rate limit error, got %v", err)
	}

	if rlErr.Info.ResetSeconds != int(svc.cfg.Security.SubmissionWindow.Seconds()) {
		t.Fatalf("expected reset %d, got %d", int(svc.cfg.Security.SubmissionWindow.Seconds()), rlErr.Info.ResetSeconds)
	}
}

func TestRateLimitInvalidUser(t *testing.T) {
	svc := newRateLimitService(t, 10*time.Second, 2)
	ctx := context.Background()

	err := svc.rateLimit(ctx, 0)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}

	if len(ve.Fields) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(ve.Fields))
	}

	if ve.Fields[0].Field != "user_id" || ve.Fields[0].Reason != "invalid" {
		t.Fatalf("unexpected field error: %+v", ve.Fields[0])
	}
}

func TestRateLimitEnforcesLimit(t *testing.T) {
	svc := newRateLimitService(t, 12*time.Second, 2)
	ctx := context.Background()

	if err := svc.rateLimit(ctx, 101); err != nil {
		t.Fatalf("first rateLimit: %v", err)
	}

	if err := svc.rateLimit(ctx, 101); err != nil {
		t.Fatalf("second rateLimit: %v", err)
	}

	err := svc.rateLimit(ctx, 101)
	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected rate limit error, got %v", err)
	}

	if rlErr.Info.Limit != 2 {
		t.Fatalf("expected limit 2, got %d", rlErr.Info.Limit)
	}

	if rlErr.Info.Remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", rlErr.Info.Remaining)
	}

	if rlErr.Info.ResetSeconds <= 0 || rlErr.Info.ResetSeconds > int(svc.cfg.Security.SubmissionWindow.Seconds()) {
		t.Fatalf("unexpected reset seconds: %d", rlErr.Info.ResetSeconds)
	}
}
