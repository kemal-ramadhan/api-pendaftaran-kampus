package limiter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	IsLoginLocked(ctx context.Context, email string) (bool, time.Duration, error)
	RecordLoginFailure(ctx context.Context, email string) (locked bool, remainingAttempts int, err error)
	ResetLoginFailure(ctx context.Context, email string) error

	CheckResendOTPCooldown(ctx context.Context, email string) (inCooldown bool, remaining time.Duration, err error)
	SetResendOTPCooldown(ctx context.Context, email string) error

	CheckForgotPasswordCooldown(ctx context.Context, email string) (inCooldown bool, remaining time.Duration, err error)
	SetForgotPasswordCooldown(ctx context.Context, email string) error
}

type rateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) RateLimiter {
	return &rateLimiter{rdb: rdb}
}

func sanitizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (l *rateLimiter) IsLoginLocked(ctx context.Context, email string) (bool, time.Duration, error) {
	if l.rdb == nil {
		return false, 0, nil
	}
	key := fmt.Sprintf("auth:lockout:login:%s", sanitizeEmail(email))
	ttl, err := l.rdb.TTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	if ttl > 0 {
		return true, ttl, nil
	}
	return false, 0, nil
}

func (l *rateLimiter) RecordLoginFailure(ctx context.Context, email string) (bool, int, error) {
	if l.rdb == nil {
		return false, 5, nil
	}
	clean := sanitizeEmail(email)
	countKey := fmt.Sprintf("auth:fail:login:%s", clean)
	lockKey := fmt.Sprintf("auth:lockout:login:%s", clean)

	pipe := l.rdb.TxPipeline()
	incr := pipe.Incr(ctx, countKey)
	pipe.Expire(ctx, countKey, 15*time.Minute)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	count := incr.Val()
	maxAttempts := int64(5)
	if count >= maxAttempts {
		// Kunci selama 15 menit
		_ = l.rdb.Set(ctx, lockKey, "locked", 15*time.Minute).Err()
		_ = l.rdb.Del(ctx, countKey).Err()
		return true, 0, nil
	}

	remaining := int(maxAttempts - count)
	return false, remaining, nil
}

func (l *rateLimiter) ResetLoginFailure(ctx context.Context, email string) error {
	if l.rdb == nil {
		return nil
	}
	clean := sanitizeEmail(email)
	countKey := fmt.Sprintf("auth:fail:login:%s", clean)
	lockKey := fmt.Sprintf("auth:lockout:login:%s", clean)
	return l.rdb.Del(ctx, countKey, lockKey).Err()
}

func (l *rateLimiter) CheckResendOTPCooldown(ctx context.Context, email string) (bool, time.Duration, error) {
	if l.rdb == nil {
		return false, 0, nil
	}
	key := fmt.Sprintf("auth:cooldown:resend:%s", sanitizeEmail(email))
	val, err := l.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, 0, nil
		}
		return false, 0, err
	}

	var expUnix int64
	if _, scanErr := fmt.Sscanf(val, "%d", &expUnix); scanErr != nil || expUnix <= 0 {
		ttl, ttlErr := l.rdb.TTL(ctx, key).Result()
		if ttlErr == nil && ttl > 0 {
			return true, ttl, nil
		}
		return false, 0, nil
	}

	nowUnix := time.Now().Unix()
	if expUnix > nowUnix {
		rem := time.Duration(expUnix-nowUnix) * time.Second
		return true, rem, nil
	}

	_ = l.rdb.Del(ctx, key)
	return false, 0, nil
}

func (l *rateLimiter) SetResendOTPCooldown(ctx context.Context, email string) error {
	if l.rdb == nil {
		return nil
	}
	key := fmt.Sprintf("auth:cooldown:resend:%s", sanitizeEmail(email))
	expUnix := time.Now().Add(60 * time.Second).Unix()
	return l.rdb.Set(ctx, key, fmt.Sprintf("%d", expUnix), 65*time.Second).Err()
}

func (l *rateLimiter) CheckForgotPasswordCooldown(ctx context.Context, email string) (bool, time.Duration, error) {
	if l.rdb == nil {
		return false, 0, nil
	}
	key := fmt.Sprintf("auth:cooldown:forgot:%s", sanitizeEmail(email))
	val, err := l.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, 0, nil
		}
		return false, 0, err
	}

	var expUnix int64
	if _, scanErr := fmt.Sscanf(val, "%d", &expUnix); scanErr != nil || expUnix <= 0 {
		ttl, ttlErr := l.rdb.TTL(ctx, key).Result()
		if ttlErr == nil && ttl > 0 {
			return true, ttl, nil
		}
		return false, 0, nil
	}

	nowUnix := time.Now().Unix()
	if expUnix > nowUnix {
		rem := time.Duration(expUnix-nowUnix) * time.Second
		return true, rem, nil
	}

	_ = l.rdb.Del(ctx, key)
	return false, 0, nil
}

func (l *rateLimiter) SetForgotPasswordCooldown(ctx context.Context, email string) error {
	if l.rdb == nil {
		return nil
	}
	key := fmt.Sprintf("auth:cooldown:forgot:%s", sanitizeEmail(email))
	expUnix := time.Now().Add(60 * time.Second).Unix()
	return l.rdb.Set(ctx, key, fmt.Sprintf("%d", expUnix), 65*time.Second).Err()
}
