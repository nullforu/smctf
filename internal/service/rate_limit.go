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
	if err := rateLimit(ctx, s.redis, key, s.cfg.Security.SubmissionWindow, s.cfg.Security.SubmissionMax); err != nil {
		return fmt.Errorf("ctf.rateLimit: %w", err)
	}

	return nil
}

func rateLimitKey(userID int64) string {
	return redisSubmitPrefix + strconv.FormatInt(userID, 10)
}

func rateLimit(ctx context.Context, redisClient *redis.Client, key string, window time.Duration, max int) error {
	count, ttl, err := rateLimitState(ctx, redisClient, key)
	if err != nil {
		return err
	}

	ttl, err = ensureRateLimitTTL(ctx, redisClient, key, ttl, window)
	if err != nil {
		return err
	}

	return evaluateRateLimit(count, ttl, max, window)
}

func rateLimitState(ctx context.Context, redisClient *redis.Client, key string) (int64, time.Duration, error) {
	pipe := redisClient.TxPipeline()
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

func ensureRateLimitTTL(ctx context.Context, redisClient *redis.Client, key string, ttl time.Duration, window time.Duration) (time.Duration, error) {
	if ttl > 0 {
		return ttl, nil
	}

	if err := redisClient.Expire(ctx, key, window).Err(); err != nil {
		return 0, err
	}

	return window, nil
}

func evaluateRateLimit(count int64, ttl time.Duration, maxAllowed int, window time.Duration) error {
	remaining := max(maxAllowed-int(count), 0)

	if count > int64(maxAllowed) {
		resetSeconds := int(math.Ceil(ttl.Seconds()))
		if resetSeconds <= 0 {
			resetSeconds = int(window.Seconds())
		}

		return &RateLimitError{Info: RateLimitInfo{
			Limit:        maxAllowed,
			Remaining:    remaining,
			ResetSeconds: resetSeconds,
		}}
	}

	return nil
}
