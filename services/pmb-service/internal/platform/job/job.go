package job

import (
	"context"
	"sync"
	"time"

	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/platform/lock"
	"go.uber.org/zap"
)

type JobFunc func(ctx context.Context) error

type jobItem struct {
	name     string
	interval time.Duration
	fn       JobFunc
}

type Runner struct {
	locker *lock.Locker
	logger *zap.Logger
	jobs   []jobItem
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewRunner(locker *lock.Locker, logger *zap.Logger) *Runner {
	return &Runner{
		locker: locker,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

func (r *Runner) Register(name string, interval time.Duration, fn JobFunc) {
	r.jobs = append(r.jobs, jobItem{
		name:     name,
		interval: interval,
		fn:       fn,
	})
}

func (r *Runner) Start(ctx context.Context) {
	for _, j := range r.jobs {
		r.wg.Add(1)
		go r.runJob(ctx, j)
	}
}

func (r *Runner) runJob(ctx context.Context, j jobItem) {
	defer r.wg.Done()
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	jobID := lock.KeyToID(j.name)

	for {
		select {
		case <-r.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			locked, err := r.locker.TryLock(ctx, jobID)
			if err != nil {
				r.logger.Warn("Error memeriksa lock job", zap.String("job", j.name), zap.Error(err))
				continue
			}
			if !locked {
				// Lock sedang dipegang oleh instance lain
				continue
			}

			// Jalankan job
			runCtx, cancel := context.WithTimeout(ctx, j.interval*2)
			if jobErr := j.fn(runCtx); jobErr != nil {
				r.logger.Warn("Job execution error", zap.String("job", j.name), zap.Error(jobErr))
			}
			cancel()

			// Lepaskan lock
			_ = r.locker.Unlock(ctx, jobID)
		}
	}
}

func (r *Runner) Stop() {
	close(r.stopCh)
	r.wg.Wait()
}
