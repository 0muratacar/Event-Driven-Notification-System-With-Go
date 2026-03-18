package worker

import (
	"context"
	"log/slog"
	"sync"
)

type Pool struct {
	size   int
	wg     sync.WaitGroup
	logger *slog.Logger
}

func NewPool(size int, logger *slog.Logger) *Pool {
	return &Pool{size: size, logger: logger}
}

// Start launches worker goroutines that each call the provided work function.
// The work function should block until ctx is cancelled.
func (p *Pool) Start(ctx context.Context, work func(ctx context.Context, workerID int)) {
	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			p.logger.Info("worker started", "worker_id", id)
			work(ctx, id)
			p.logger.Info("worker stopped", "worker_id", id)
		}(i)
	}
}

func (p *Pool) Wait() {
	p.wg.Wait()
}
