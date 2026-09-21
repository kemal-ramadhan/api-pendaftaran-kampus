package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockOutboxRepo struct {
	events       []Event
	publishedIDs []uuid.UUID
	failedIDs    []uuid.UUID
}

func (m *mockOutboxRepo) CreateInTx(tx *gorm.DB, event *Event) error {
	m.events = append(m.events, *event)
	return nil
}

func (m *mockOutboxRepo) FetchPending(ctx context.Context, db *gorm.DB, limit int) ([]Event, error) {
	var result []Event
	for _, e := range m.events {
		if e.Status == StatusPending {
			result = append(result, e)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockOutboxRepo) MarkPublished(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	m.publishedIDs = append(m.publishedIDs, id)
	for i := range m.events {
		if m.events[i].ID == id {
			m.events[i].Status = StatusPublished
		}
	}
	return nil
}

func (m *mockOutboxRepo) MarkFailed(ctx context.Context, db *gorm.DB, id uuid.UUID, err error, maxRetries int) error {
	m.failedIDs = append(m.failedIDs, id)
	for i := range m.events {
		if m.events[i].ID == id {
			m.events[i].RetryCount++
			if m.events[i].RetryCount >= maxRetries {
				m.events[i].Status = StatusFailed
			}
		}
	}
	return nil
}

func TestOutbox_NewEvent(t *testing.T) {
	ev, err := NewEvent("admission", "app-123", "application.verified", "idempotent-key-1", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}
	if ev.AggregateType != "admission" || ev.AggregateID != "app-123" {
		t.Errorf("Aggregate mismatch: %+v", ev)
	}
	if ev.Status != StatusPending {
		t.Errorf("Initial status should be pending, got %s", ev.Status)
	}
	if ev.IdempotencyKey != "idempotent-key-1" {
		t.Errorf("Idempotency key mismatch: %s", ev.IdempotencyKey)
	}
}

func TestOutbox_ProcessPending(t *testing.T) {
	ev1, _ := NewEvent("finance", "inv-1", "invoice.paid", "k1", "payload1")
	ev2, _ := NewEvent("finance", "inv-2", "invoice.paid", "k2", "payload2")

	repo := &mockOutboxRepo{
		events: []Event{*ev1, *ev2},
	}

	dispatcher := func(ctx context.Context, event *Event) error {
		if event.AggregateID == "inv-2" {
			return errors.New("temporary downstream error")
		}
		return nil
	}

	ctx := context.Background()
	processed, err := ProcessPending(ctx, nil, repo, dispatcher, 10, 3)
	if err != nil {
		t.Fatalf("ProcessPending error: %v", err)
	}
	if processed != 1 {
		t.Errorf("Expected 1 processed event, got %d", processed)
	}
	if len(repo.publishedIDs) != 1 || repo.publishedIDs[0] != ev1.ID {
		t.Errorf("Published ID mismatch: %v", repo.publishedIDs)
	}
	if len(repo.failedIDs) != 1 || repo.failedIDs[0] != ev2.ID {
		t.Errorf("Failed ID mismatch: %v", repo.failedIDs)
	}
}
