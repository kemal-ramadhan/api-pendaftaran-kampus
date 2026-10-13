package token

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SessionStore interface {
	SaveSession(ctx context.Context, userID uuid.UUID, sessionID, jti string, ttl time.Duration) error
	ValidateSession(ctx context.Context, userID uuid.UUID, sessionID, jti string) (valid bool, reuseDetected bool, err error)
	RotateSession(ctx context.Context, userID uuid.UUID, sessionID, newJTI string, ttl time.Duration) error
	RevokeSession(ctx context.Context, userID uuid.UUID, sessionID string) error
	RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error
	AddToDenylist(ctx context.Context, jti string, ttl time.Duration) error
	IsDenylisted(ctx context.Context, jti string) (bool, error)
}

type sessionStore struct {
	rdb *redis.Client
}

func NewSessionStore(rdb *redis.Client) SessionStore {
	return &sessionStore{rdb: rdb}
}

func getRefreshKey(userID uuid.UUID, sessionID string) string {
	return fmt.Sprintf("auth:refresh:%s:%s", userID.String(), sessionID)
}

func getDenylistKey(jti string) string {
	return fmt.Sprintf("auth:denylist:%s", jti)
}

func (s *sessionStore) SaveSession(ctx context.Context, userID uuid.UUID, sessionID, jti string, ttl time.Duration) error {
	if s.rdb == nil {
		return nil
	}
	key := getRefreshKey(userID, sessionID)
	return s.rdb.Set(ctx, key, jti, ttl).Err()
}

func (s *sessionStore) ValidateSession(ctx context.Context, userID uuid.UUID, sessionID, jti string) (bool, bool, error) {
	if s.rdb == nil {
		return true, false, nil
	}
	key := getRefreshKey(userID, sessionID)
	val, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		// Sesi tidak ditemukan / sudah kedaluwarsa
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}

	if val != jti {
		// JTI tersimpan tidak sama dengan JTI yang dikirim -> Token Reuse Terdeteksi!
		return false, true, nil
	}

	return true, false, nil
}

func (s *sessionStore) RotateSession(ctx context.Context, userID uuid.UUID, sessionID, newJTI string, ttl time.Duration) error {
	if s.rdb == nil {
		return nil
	}
	key := getRefreshKey(userID, sessionID)
	return s.rdb.Set(ctx, key, newJTI, ttl).Err()
}

func (s *sessionStore) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID string) error {
	if s.rdb == nil {
		return nil
	}
	key := getRefreshKey(userID, sessionID)
	return s.rdb.Del(ctx, key).Err()
}

func (s *sessionStore) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	if s.rdb == nil {
		return nil
	}
	pattern := fmt.Sprintf("auth:refresh:%s:*", userID.String())

	var cursor uint64
	var keys []string
	for {
		var batch []string
		var err error
		batch, cursor, err = s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	if len(keys) > 0 {
		return s.rdb.Del(ctx, keys...).Err()
	}
	return nil
}

func (s *sessionStore) AddToDenylist(ctx context.Context, jti string, ttl time.Duration) error {
	if s.rdb == nil || ttl <= 0 {
		return nil
	}
	key := getDenylistKey(jti)
	return s.rdb.Set(ctx, key, "1", ttl).Err()
}

func (s *sessionStore) IsDenylisted(ctx context.Context, jti string) (bool, error) {
	if s.rdb == nil {
		return false, nil
	}
	key := getDenylistKey(jti)
	val, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}
