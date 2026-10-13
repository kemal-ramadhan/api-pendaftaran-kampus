package service

import (
	"context"
	"log"
	"time"
)

type Worker struct {
	service  NotificationService
	interval time.Duration
	limit    int
}

func NewWorker(service NotificationService, interval time.Duration, limit int) *Worker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if limit <= 0 {
		limit = 20
	}
	return &Worker{
		service:  service,
		interval: interval,
		limit:    limit,
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("[NOTIFICATION-WORKER] Background worker dimulai (interval: %v, batch limit: %d)", w.interval, w.limit)

	for {
		select {
		case <-ctx.Done():
			log.Println("[NOTIFICATION-WORKER] Background worker dihentikan.")
			return
		case <-ticker.C:
			count, err := w.service.ProcessPending(ctx, w.limit)
			if err != nil {
				log.Printf("[NOTIFICATION-WORKER] Error saat memproses antrean: %v", err)
			} else if count > 0 {
				log.Printf("[NOTIFICATION-WORKER] Berhasil memproses %d notifikasi tertunda", count)
			}
		}
	}
}

func (w *Worker) ProcessBatch(ctx context.Context) (int, error) {
	return w.service.ProcessPending(ctx, w.limit)
}

