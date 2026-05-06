package cronjob

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// helper: create a test logger that writes to the test's output.
func testLogger(t *testing.T) *zap.Logger {
	t.Helper()
	return zaptest.NewLogger(t)
}

// awaitCondition polls fn up to timeout, returning true if fn ever returns
// true. Useful for avoiding arbitrary sleeps in tests.
func awaitCondition(t *testing.T, timeout time.Duration, fn func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// TestScheduler_RunOnStart verifies that every registered job fires exactly
// once immediately when the scheduler starts, before any cron tick can fire.
func TestScheduler_RunOnStart(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	var counter atomic.Int32

	s := NewScheduler(log)
	s.Register(Job{
		Name:     "run-on-start-test",
		Schedule: "@every 1h", // long interval — only the run-on-start should fire
		Fn: func(ctx context.Context) error {
			counter.Add(1)
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	// Poll instead of sleeping: wait until the job has fired.
	if !awaitCondition(t, 2*time.Second, func() bool { return counter.Load() >= 1 }) {
		t.Fatal("job did not execute on start within 2s")
	}

	// Give the scheduler a moment to fire spurious extra runs, then check
	// it didn't over-fire. With a 1h schedule only 1 run is expected.
	time.Sleep(50 * time.Millisecond)
	if got := counter.Load(); got != 1 {
		t.Errorf("expected exactly 1 run-on-start execution, got %d", got)
	}
}

// TestScheduler_PeriodicExecution verifies that jobs recur on schedule after
// the initial run-on-start execution.
func TestScheduler_PeriodicExecution(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	var counter atomic.Int32

	s := NewScheduler(log)
	s.Register(Job{
		Name:     "periodic-test",
		Schedule: "@every 1s",
		Fn: func(ctx context.Context) error {
			counter.Add(1)
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	// 1 run-on-start + at least 2 cron ticks = ≥3 total.
	// Poll rather than sleep for the full window.
	if !awaitCondition(t, 4*time.Second, func() bool { return counter.Load() >= 3 }) {
		t.Errorf("expected at least 3 executions (1 immediate + 2 ticks), got %d", counter.Load())
	}
}

// TestScheduler_OverlapPrevention verifies that a slow job is never run
// concurrently with itself.
func TestScheduler_OverlapPrevention(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	var concurrentCount atomic.Int32
	var maxConcurrent atomic.Int32

	jobStarted := make(chan struct{}, 1)

	s := NewScheduler(log)
	s.Register(Job{
		Name:     "slow-job",
		Schedule: "@every 1s",
		Fn: func(ctx context.Context) error {
			cur := concurrentCount.Add(1)
			defer concurrentCount.Add(-1)

			// Track the high-water mark of concurrency.
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}

			// Signal that at least one run has started.
			select {
			case jobStarted <- struct{}{}:
			default:
			}

			// Hold longer than the 1s tick so subsequent ticks must skip.
			select {
			case <-time.After(1500 * time.Millisecond):
			case <-ctx.Done():
			}
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	// Wait for the first run to start before timing the observation window.
	select {
	case <-jobStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("job never started")
	}

	// Observe for long enough to cover multiple ticks.
	time.Sleep(3 * time.Second)

	if got := maxConcurrent.Load(); got > 1 {
		t.Errorf("overlap prevention failed: max concurrent runs = %d, want ≤ 1", got)
	}
}

// TestScheduler_GracefulShutdown verifies two things:
//  1. An in-flight job is allowed to complete its work after Stop is called.
//  2. Stop does not return before the job finishes.
func TestScheduler_GracefulShutdown(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	const workDuration = 200 * time.Millisecond

	// finishedAt records when the job's work actually completed.
	var finishedAt atomic.Int64 // UnixNano

	s := NewScheduler(log)
	s.Register(Job{
		Name:     "long-job",
		Schedule: "@every 1h",
		Fn: func(ctx context.Context) error {
			// Do a fixed amount of work that does NOT watch ctx, so we can
			// verify the scheduler waits for it rather than cutting it off.
			time.Sleep(workDuration)
			finishedAt.Store(time.Now().UnixNano())
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Wait for run-on-start to begin.
	time.Sleep(50 * time.Millisecond)

	stopStarted := time.Now()
	cancel()
	s.Stop() // must block until the job finishes
	stopReturned := time.Now()

	// The job must have finished before Stop returned.
	jobDone := time.Unix(0, finishedAt.Load())
	if finishedAt.Load() == 0 {
		t.Fatal("job never recorded a finish time — it may have been cancelled before completing")
	}
	if jobDone.After(stopReturned) {
		t.Errorf("Stop returned before job finished: Stop returned at %v, job finished at %v",
			stopReturned, jobDone)
	}

	// Stop should have taken at least as long as the remaining work.
	elapsed := stopReturned.Sub(stopStarted)
	if elapsed < workDuration/2 {
		t.Errorf("Stop returned suspiciously fast (%v) — job may not have been awaited", elapsed)
	}
}

// TestScheduler_PanicRecovery verifies that a panicking job does not crash the
// process or prevent other jobs from continuing to run.
func TestScheduler_PanicRecovery(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	var healthyRuns atomic.Int32

	s := NewScheduler(log)
	s.Register(
		Job{
			Name:     "panicking-job",
			Schedule: "@every 1s",
			Fn: func(ctx context.Context) error {
				panic("test panic — should be recovered")
			},
		},
		Job{
			Name:     "healthy-job",
			Schedule: "@every 1s",
			Fn: func(ctx context.Context) error {
				healthyRuns.Add(1)
				return nil
			},
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	// Healthy job should run on-start and then at least once more via cron tick.
	if !awaitCondition(t, 4*time.Second, func() bool { return healthyRuns.Load() >= 2 }) {
		t.Errorf("expected healthy-job to run at least twice despite panicking-job, got %d runs",
			healthyRuns.Load())
	}
}

// TestScheduler_InvalidSchedule verifies that a job with an unparseable cron
// expression is silently skipped — neither registered nor run-on-started.
func TestScheduler_InvalidSchedule(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	var ran atomic.Bool

	s := NewScheduler(log)
	s.Register(Job{
		Name:     "bad-schedule",
		Schedule: "not a valid cron expression",
		// NOTE: the run-on-start is also skipped for invalid jobs because the
		// wg.Go call is guarded by the continue after the AddFunc error.
		Fn: func(ctx context.Context) error {
			ran.Store(true)
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	time.Sleep(200 * time.Millisecond)

	if ran.Load() {
		t.Error("job with invalid schedule should not have run (not via cron tick, not via run-on-start)")
	}
}

// TestScheduler_ErrorLogging verifies that a job returning an error continues
// to be scheduled on subsequent ticks.
func TestScheduler_ErrorLogging(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	var counter atomic.Int32

	s := NewScheduler(log)
	s.Register(Job{
		Name:     "error-job",
		Schedule: "@every 1s",
		Fn: func(ctx context.Context) error {
			counter.Add(1)
			return errors.New("transient error")
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	// 1 run-on-start + at least 2 ticks = ≥3, same as PeriodicExecution.
	if !awaitCondition(t, 4*time.Second, func() bool { return counter.Load() >= 3 }) {
		t.Errorf("expected at least 3 executions despite errors, got %d", counter.Load())
	}
}

// TestScheduler_MultipleJobs verifies that all registered jobs run
// independently and do not interfere with each other's execution counts.
func TestScheduler_MultipleJobs(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	var countA, countB, countC atomic.Int32

	s := NewScheduler(log)
	s.Register(
		Job{
			Name:     "job-a",
			Schedule: "@every 1s",
			Fn:       func(ctx context.Context) error { countA.Add(1); return nil },
		},
		Job{
			Name:     "job-b",
			Schedule: "@every 1s",
			Fn:       func(ctx context.Context) error { countB.Add(1); return nil },
		},
		Job{
			Name:     "job-c",
			Schedule: "@every 1s",
			Fn:       func(ctx context.Context) error { countC.Add(1); return nil },
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	ok := awaitCondition(t, 4*time.Second, func() bool {
		return countA.Load() >= 2 && countB.Load() >= 2 && countC.Load() >= 2
	})
	if !ok {
		t.Errorf("not all jobs reached 2 executions: A=%d B=%d C=%d",
			countA.Load(), countB.Load(), countC.Load())
	}
}

// TestScheduler_ContextPassedToJob verifies that the context received by the
// job function is cancelled when the scheduler is stopped.
func TestScheduler_ContextPassedToJob(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	ctxCancelledDuringJob := make(chan struct{})

	s := NewScheduler(log)
	s.Register(Job{
		Name:     "ctx-aware-job",
		Schedule: "@every 1h",
		Fn: func(ctx context.Context) error {
			// Block until ctx is cancelled or timeout.
			select {
			case <-ctx.Done():
				close(ctxCancelledDuringJob)
			case <-time.After(5 * time.Second):
			}
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	// Let run-on-start begin.
	time.Sleep(50 * time.Millisecond)

	cancel()
	s.Stop()

	select {
	case <-ctxCancelledDuringJob:
		// ctx.Done() was observed inside the job — correct.
	case <-time.After(2 * time.Second):
		t.Error("job did not observe context cancellation after Stop")
	}
}

// TestScheduler_StopIsIdempotent verifies that calling Stop more than once
// does not panic or deadlock.
func TestScheduler_StopIsIdempotent(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	s := NewScheduler(log)
	s.Register(Job{
		Name:     "noop",
		Schedule: "@every 1h",
		Fn:       func(ctx context.Context) error { return nil },
	})

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	cancel()
	s.Stop()

	// Second Stop must not panic or hang.
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Stop()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("second call to Stop hung")
	}
}

// TestScheduler_NoJobsStartsCleanly verifies that a scheduler with no
// registered jobs starts and stops without error.
func TestScheduler_NoJobsStartsCleanly(t *testing.T) {
	t.Parallel()
	log := testLogger(t)

	s := NewScheduler(log)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	cancel()
	s.Stop() // must not panic or hang
}