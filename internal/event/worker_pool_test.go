package event

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_StartStop(t *testing.T) {
	pool := newWorkerPool("test-pool", 2)
	pool.Start()

	// Submit some jobs
	var count atomic.Int32
	for i := 0; i < 10; i++ {
		pool.SubmitWithContext(context.Background(), func() {
			count.Add(1)
		})
	}

	time.Sleep(100 * time.Millisecond)
	pool.Stop()

	if count.Load() != 10 {
		t.Errorf("processed %d jobs, want 10", count.Load())
	}
}

func TestWorkerPool_StopWaitsForJobs(t *testing.T) {
	pool := newWorkerPool("test-pool", 1)
	pool.Start()

	var completed atomic.Bool
	pool.SubmitWithContext(context.Background(), func() {
		time.Sleep(50 * time.Millisecond)
		completed.Store(true)
	})

	pool.Stop()
	if !completed.Load() {
		t.Error("Stop() should wait for jobs to complete")
	}
}

func TestWorkerPool_Submit_AfterStop(t *testing.T) {
	pool := newWorkerPool("test-pool", 1)
	pool.Start()
	pool.Stop()

	ok := pool.SubmitWithContext(context.Background(), func() {})
	if ok {
		t.Error("Submit() after Stop() should return false")
	}
}

func TestWorkerPool_SubmitWithContext_Cancelled(t *testing.T) {
	pool := newWorkerPool("test-pool", 1)
	pool.Start()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok := pool.SubmitWithContext(ctx, func() {})
	if ok {
		t.Error("SubmitWithContext() with cancelled context should return false")
	}

	pool.Stop()
}

func TestWorkerPool_Stats(t *testing.T) {
	pool := newWorkerPool("test-pool", 2)
	pool.Start()

	pool.SubmitWithContext(context.Background(), func() {})
	pool.SubmitWithContext(context.Background(), func() {})
	pool.SubmitWithContext(context.Background(), func() {})

	time.Sleep(100 * time.Millisecond)

	processed, errors, avgTime := pool.Stats()
	if processed < 3 {
		t.Errorf("processed = %d, want >= 3", processed)
	}
	if errors != 0 {
		t.Errorf("errors = %d, want 0", errors)
	}
	if avgTime < 0 {
		t.Errorf("avgTime = %d, want >= 0", avgTime)
	}

	pool.Stop()
}

func TestWorkerPool_PanicRecovery(t *testing.T) {
	pool := newWorkerPool("test-pool", 1)
	pool.Start()

	pool.SubmitWithContext(context.Background(), func() {
		panic("test panic")
	})

	time.Sleep(100 * time.Millisecond)

	_, errors, _ := pool.Stats()
	if errors != 1 {
		t.Errorf("errors = %d, want 1 after panic", errors)
	}

	pool.Stop()
}

func TestWorkerPool_PendingJobs(t *testing.T) {
	pool := newWorkerPool("test-pool", 1)
	pool.Start()

	// Fill up with slow jobs
	for i := 0; i < 10; i++ {
		pool.SubmitWithContext(context.Background(), func() {
			time.Sleep(50 * time.Millisecond)
		})
	}

	time.Sleep(10 * time.Millisecond)
	pending := pool.PendingJobs()
	if pending < 0 {
		t.Errorf("PendingJobs() = %d, want >= 0", pending)
	}

	pool.Stop()
}

func TestWorkerPool_Done(t *testing.T) {
	pool := newWorkerPool("test-pool", 1)
	pool.Start()

	done := pool.Done()
	if done == nil {
		t.Fatal("Done() returned nil")
	}

	select {
	case <-done:
		t.Error("Done channel closed before Stop()")
	case <-time.After(10 * time.Millisecond):
		// Expected
	}

	pool.Stop()

	select {
	case <-done:
		// Expected after Stop()
	case <-time.After(time.Second):
		t.Error("Done channel not closed after Stop()")
	}
}

func TestWorkerPool_ExecuteJob_AvgTime(t *testing.T) {
	pool := newWorkerPool("test-pool", 1)
	pool.Start()

	pool.SubmitWithContext(context.Background(), func() {
		time.Sleep(10 * time.Millisecond)
	})
	time.Sleep(50 * time.Millisecond)

	_, _, avgTime := pool.Stats()
	if avgTime <= 0 {
		t.Errorf("avgTime = %d, want > 0", avgTime)
	}

	pool.Stop()
}

func TestWorkerPool_ContextCancel_FlushesRemaining(t *testing.T) {
	pool := newWorkerPool("test-pool", 1)
	pool.Start()

	var count atomic.Int32
	// Submit many slow jobs
	for i := 0; i < 5; i++ {
		pool.SubmitWithContext(context.Background(), func() {
			time.Sleep(20 * time.Millisecond)
			count.Add(1)
		})
	}

	// Cancel context to trigger flush path
	pool.cancel()
	time.Sleep(200 * time.Millisecond)

	pool.Stop()
	// Some jobs should have been processed during flush
	if count.Load() == 0 {
		t.Error("no jobs processed during context cancel flush")
	}
}
