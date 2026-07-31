package event

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type poolMetrics struct {
	processed atomic.Int64
	errors    atomic.Int64
	avgTime   atomic.Int64
}

type workerPool struct {
	workers int
	name    string
	jobs    chan func()
	stopped atomic.Bool
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	metrics *poolMetrics
}

func newWorkerPool(name string, workers int) *workerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &workerPool{
		workers: workers,
		name:    name,
		jobs:    make(chan func(), 1024),
		stopped: atomic.Bool{},
		ctx:     ctx,
		cancel:  cancel,
		metrics: &poolMetrics{},
	}
}

func (p *workerPool) executeJob(job func()) {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			p.metrics.errors.Add(1)
		}
	}()

	job()

	p.metrics.processed.Add(1)

	elapsed := time.Since(start).Nanoseconds()
	avg := p.metrics.avgTime.Load()
	if avg == 0 {
		p.metrics.avgTime.Store(elapsed)
	} else {
		p.metrics.avgTime.Store((avg + elapsed) / 2)
	}
}

func (p *workerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			p.executeJob(job)
		case <-p.ctx.Done():
			for {
				select {
				case job, ok := <-p.jobs:
					if !ok {
						return
					}
					p.executeJob(job)
				default:
					return
				}
			}
		}

	}
}

func (p *workerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *workerPool) Submit() bool {
	if p.stopped.Load() {
		return false
	}

	select {
	case p.jobs <- func() {}:
		return true
	default:
		return false
	}
}

func (p *workerPool) SubmitWithContext(ctx context.Context, jobs func()) bool {
	if p.stopped.Load() {
		return false
	}

	select {
	case p.jobs <- jobs:
		return true
	case <-ctx.Done():
		return false
	default:
		return false
	}
}

func (p *workerPool) Stop() {
	p.stopped.Store(true)
	p.cancel()
	p.wg.Wait()
	close(p.jobs)
}

func (p *workerPool) Done() <-chan struct{} {
	return p.ctx.Done()
}

func (p *workerPool) Stats() (processed, errors int64, avgTimeNs int64) {
	return p.metrics.processed.Load(),
		p.metrics.errors.Load(),
		p.metrics.avgTime.Load()
}

func (p *workerPool) PendingJobs() int {
	return len(p.jobs)
}
