package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusPublished Status = "published"
	StatusFailed    Status = "failed"
)

// Event entitas penyimpanan event outbox untuk menjamin keandalan pengiriman lintas service
type Event struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	AggregateType  string     `gorm:"type:varchar(50);not null;index" json:"aggregate_type"`
	AggregateID    string     `gorm:"type:varchar(100);not null;index" json:"aggregate_id"`
	EventType      string     `gorm:"type:varchar(100);not null" json:"event_type"`
	Payload        string     `gorm:"type:text;not null" json:"payload"`
	IdempotencyKey string     `gorm:"type:varchar(150);uniqueIndex;not null" json:"idempotency_key"`
	Status         Status     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	RetryCount     int        `gorm:"not null;default:0" json:"retry_count"`
	LastError      string     `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt      time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	ProcessedAt    *time.Time `gorm:"type:timestamptz" json:"processed_at,omitempty"`
}

// TableName menentukan nama tabel database untuk outbox events
func (Event) TableName() string {
	return "outbox_events"
}

// NewEvent membuat objek Event outbox baru dengan UUID dan status pending
func NewEvent(aggregateType, aggregateID, eventType, idempotencyKey string, payloadData interface{}) (*Event, error) {
	if aggregateType == "" || aggregateID == "" || eventType == "" {
		return nil, errors.New("aggregateType, aggregateID, dan eventType wajib diisi")
	}

	var payloadStr string
	if s, ok := payloadData.(string); ok {
		payloadStr = s
	} else {
		b, err := json.Marshal(payloadData)
		if err != nil {
			return nil, fmt.Errorf("gagal marshal payload event outbox: %w", err)
		}
		payloadStr = string(b)
	}

	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("%s:%s:%s", aggregateType, aggregateID, uuid.NewString())
	}

	return &Event{
		ID:             uuid.New(),
		AggregateType:  aggregateType,
		AggregateID:    aggregateID,
		EventType:      eventType,
		Payload:        payloadStr,
		IdempotencyKey: idempotencyKey,
		Status:         StatusPending,
		RetryCount:     0,
		CreatedAt:      time.Now(),
	}, nil
}

// Repository interface untuk manajemen basis data tabel outbox
type Repository interface {
	CreateInTx(tx *gorm.DB, event *Event) error
	FetchPending(ctx context.Context, db *gorm.DB, limit int) ([]Event, error)
	MarkPublished(ctx context.Context, db *gorm.DB, id uuid.UUID) error
	MarkFailed(ctx context.Context, db *gorm.DB, id uuid.UUID, err error, maxRetries int) error
}

type repository struct{}

// NewRepository membuat repository outbox baru
func NewRepository() Repository {
	return &repository{}
}

// CreateInTx menyimpan record outbox event dalam transaksi DB yang sama dengan operasi bisnis
func (r *repository) CreateInTx(tx *gorm.DB, event *Event) error {
	if event == nil {
		return errors.New("event outbox tidak boleh nil")
	}
	return tx.Create(event).Error
}

// FetchPending mengambil daftar event pending dengan locking untuk pemrosesan paralel
func (r *repository) FetchPending(ctx context.Context, db *gorm.DB, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 50
	}

	var events []Event
	err := db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ?", StatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error

	return events, err
}

// MarkPublished menandai event telah berhasil dikirim
func (r *repository) MarkPublished(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	now := time.Now()
	return db.WithContext(ctx).
		Model(&Event{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       StatusPublished,
			"processed_at": &now,
			"last_error":   "",
		}).Error
}

// MarkFailed mencatat kegagalan dan mengubah status jika melebihi batas maksimal retry
func (r *repository) MarkFailed(ctx context.Context, db *gorm.DB, id uuid.UUID, err error, maxRetries int) error {
	now := time.Now()
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	if maxRetries <= 0 {
		maxRetries = 3
	}

	var current Event
	if err := db.WithContext(ctx).Select("retry_count").First(&current, "id = ?", id).Error; err != nil {
		return err
	}

	newRetryCount := current.RetryCount + 1
	status := StatusPending
	if newRetryCount >= maxRetries {
		status = StatusFailed
	}

	return db.WithContext(ctx).
		Model(&Event{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       status,
			"retry_count":  newRetryCount,
			"last_error":   errMsg,
			"processed_at": &now,
		}).Error
}

// DispatcherFunc fungsi handler untuk memproses/mengirimkan satu event outbox
type DispatcherFunc func(ctx context.Context, event *Event) error

// ProcessPending mengambil dan memproses satu batch event pending
func ProcessPending(ctx context.Context, db *gorm.DB, repo Repository, dispatcher DispatcherFunc, batchSize, maxRetries int) (int, error) {
	events, err := repo.FetchPending(ctx, db, batchSize)
	if err != nil {
		return 0, err
	}

	processed := 0
	for i := range events {
		ev := &events[i]
		if err := dispatcher(ctx, ev); err != nil {
			_ = repo.MarkFailed(ctx, db, ev.ID, err, maxRetries)
		} else {
			_ = repo.MarkPublished(ctx, db, ev.ID)
			processed++
		}
	}

	return processed, nil
}

// StartWorker menjalankan background worker untuk memproses event outbox secara berkala
func StartWorker(ctx context.Context, db *gorm.DB, repo Repository, dispatcher DispatcherFunc, interval time.Duration, batchSize, maxRetries int) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 20
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = ProcessPending(ctx, db, repo, dispatcher, batchSize, maxRetries)
			}
		}
	}()
}
