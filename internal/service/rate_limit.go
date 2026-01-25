package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func (s *CTFService) rateLimit(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return NewValidationError(FieldError{Field: "user_id", Reason: "invalid"})
	}

	key := rateLimitKey(userID)
	count, ttl, err := s.rateLimitState(ctx, key)
	if err != nil {
		return fmt.Errorf("ctf.rateLimit state: %w", err)
	}

	ttl, err = s.ensureRateLimitTTL(ctx, key, ttl)
	if err != nil {
		return fmt.Errorf("ctf.rateLimit ttl: %w", err)
	}

	return s.evaluateRateLimit(count, ttl)
}

func rateLimitKey(userID int64) string {
	return "submit:" + strconv.FormatInt(userID, 10)
}

func (s *CTFService) rateLimitState(ctx context.Context, key string) (int64, time.Duration, error) {
	pipe := s.redis.TxPipeline()
	cntCmd := pipe.Incr(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return 0, 0, err
	}
	if err := cntCmd.Err(); err != nil {
		return 0, 0, err
	}
	if err := ttlCmd.Err(); err != nil {
		return 0, 0, err
	}
	return cntCmd.Val(), ttlCmd.Val(), nil
}

func (s *CTFService) ensureRateLimitTTL(ctx context.Context, key string, ttl time.Duration) (time.Duration, error) {
	if ttl > 0 {
		return ttl, nil
	}
	if err := s.redis.Expire(ctx, key, s.cfg.Security.SubmissionWindow).Err(); err != nil {
		return 0, err
	}
	return s.cfg.Security.SubmissionWindow, nil
}

func (s *CTFService) evaluateRateLimit(count int64, ttl time.Duration) error {
	remaining := s.cfg.Security.SubmissionMax - int(count)
	if remaining < 0 {
		remaining = 0
	}
	if count > int64(s.cfg.Security.SubmissionMax) {
		resetSeconds := int(math.Ceil(ttl.Seconds()))
		if resetSeconds <= 0 {
			resetSeconds = int(s.cfg.Security.SubmissionWindow.Seconds())
		}
		return &RateLimitError{Info: RateLimitInfo{
			Limit:        s.cfg.Security.SubmissionMax,
			Remaining:    remaining,
			ResetSeconds: resetSeconds,
		}}
	}
	return nil
}
