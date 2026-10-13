package lock

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
)

// Locker menyediakan distributed lock berbasis PostgreSQL advisory lock
type Locker struct {
	db *sql.DB
}

func NewLocker(db *sql.DB) *Locker {
	return &Locker{db: db}
}

// KeyToID mengubah nama string job menjadi int64 hash untuk pg_try_advisory_lock
func KeyToID(jobName string) int64 {
	h := fnv.New64a()
	h.Write([]byte(jobName))
	return int64(h.Sum64())
}

// TryLock mencoba mendapatkan advisory lock berbasis session atau xact
func (l *Locker) TryLock(ctx context.Context, jobID int64) (bool, error) {
	var locked bool
	query := "SELECT pg_try_advisory_lock($1)"
	err := l.db.QueryRowContext(ctx, query, jobID).Scan(&locked)
	if err != nil {
		return false, fmt.Errorf("gagal memeriksa advisory lock: %w", err)
	}
	return locked, nil
}

// Unlock melepaskan advisory lock
func (l *Locker) Unlock(ctx context.Context, jobID int64) error {
	var unlocked bool
	query := "SELECT pg_advisory_unlock($1)"
	err := l.db.QueryRowContext(ctx, query, jobID).Scan(&unlocked)
	if err != nil {
		return fmt.Errorf("gagal melepaskan advisory lock: %w", err)
	}
	return nil
}
