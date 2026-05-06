// Package cronjob provides a lightweight, structured cron scheduler for
// background maintenance tasks across Veritas Go microservices.
//
// It wraps github.com/robfig/cron/v3 with opinionated defaults:
//   - Structured zap logging for every job run (name, duration, error).
//   - Overlap prevention: a job that is still running when the next tick
//     fires is skipped with a warning log.
//   - Immediate execution on start: every registered job runs once
//     before the first cron tick.
//   - Graceful shutdown via context cancellation: in-flight jobs are
//     given time to finish.
//
// Usage in a service's main.go:
//
//	scheduler := cronjob.NewScheduler(log)
//	scheduler.Register(
//	    cronjob.Job{
//	        Name:     "purge-expired-tokens",
//	        Schedule: "@every 6h",
//	        Fn:       func(ctx context.Context) error { return repo.DeleteExpired(ctx) },
//	    },
//	)
//	scheduler.Start(ctx)
//	defer scheduler.Stop()
package cronjob

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// JobFunc is the function signature for a cron job.
// It receives a context that is cancelled when the scheduler is stopping.
type JobFunc func(ctx context.Context) error

// Job defines a single scheduled task.
type Job struct {
	// Name is a human-readable identifier used in log messages.
	Name string

	// Schedule is a cron expression parsed by robfig/cron/v3.
	// Standard cron (5 fields) and convenience shorthands are supported:
	//   "*/5 * * * *"   – every 5 minutes
	//   "@every 6h"     – every 6 hours
	//   "@daily"        – once per day at midnight
	//   "@weekly"       – once per week (Sunday midnight)
	// See https://pkg.go.dev/github.com/robfig/cron/v3 for full syntax.
	Schedule string

	// Fn is the work to execute on each tick. Returning a non-nil error
	// causes the error to be logged; the scheduler does NOT stop.
	Fn JobFunc
}

// Scheduler manages and runs registered cron jobs.
type Scheduler struct {
	log  *zap.Logger
	cron *cron.Cron
	jobs []Job

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewScheduler creates a Scheduler that logs to the provided zap logger.
func NewScheduler(log *zap.Logger) *Scheduler {
	return &Scheduler{
		log: log.Named("cronjob"),
	}
}

// Register adds one or more jobs to the scheduler. Must be called before Start.
func (s *Scheduler) Register(jobs ...Job) {
	s.jobs = append(s.jobs, jobs...)
}

// Start begins executing all registered jobs. It:
//  1. Fires every job once immediately (run-on-start).
//  2. Schedules each job according to its cron expression.
//
// Start is non-blocking. Call Stop to shut down.
func (s *Scheduler) Start(ctx context.Context) {
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.cron = cron.New(cron.WithLogger(cron.PrintfLogger(
		newZapCronAdapter(s.log),
	)))

	for _, j := range s.jobs {
		job := j // capture loop variable
		running := &atomic.Bool{}

		wrappedFn := func() {
			s.runJob(job, running)
		}

		// Schedule the recurring cron entry.
		_, err := s.cron.AddFunc(job.Schedule, wrappedFn)
		if err != nil {
			s.log.Error("failed to register job",
				zap.String("job", job.Name),
				zap.String("schedule", job.Schedule),
				zap.Error(err),
			)
			continue
		}

		s.log.Info("registered job",
			zap.String("job", job.Name),
			zap.String("schedule", job.Schedule),
		)

		// Run-on-start: fire each job once immediately in a goroutine.
		s.wg.Go(func() {
			s.runJob(job, running)
		})
	}

	s.cron.Start()
	s.log.Info("scheduler started", zap.Int("job_count", len(s.jobs)))
}

// Stop signals all jobs to finish and blocks until they complete.
// After Stop returns, no more jobs will be executed.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}

	if s.cron != nil {
		// Stop the cron scheduler and wait for running jobs to finish.
		stopCtx := s.cron.Stop()
		<-stopCtx.Done()
	}

	// Wait for any run-on-start goroutines still in flight.
	s.wg.Wait()
	s.log.Info("scheduler stopped")
}

// runJob executes a single job with overlap prevention, structured logging,
// and panic recovery.
func (s *Scheduler) runJob(job Job, running *atomic.Bool) {
	// Overlap guard: skip if the previous run is still in progress.
	if !running.CompareAndSwap(false, true) {
		s.log.Warn("skipping job — previous run still in progress",
			zap.String("job", job.Name),
		)
		return
	}
	defer running.Store(false)

	// Check if the scheduler context has been cancelled.
	if s.ctx.Err() != nil {
		return
	}

	start := time.Now()
	l := s.log.With(zap.String("job", job.Name))
	l.Info("job started")

	// Recover from panics so one bad job can't kill the process.
	defer func() {
		if r := recover(); r != nil {
			l.Error("job panicked",
				zap.Any("panic", r),
				zap.Duration("duration", time.Since(start)),
			)
		}
	}()

	err := job.Fn(s.ctx)
	elapsed := time.Since(start)

	if err != nil {
		l.Error("job failed",
			zap.Error(err),
			zap.Duration("duration", elapsed),
		)
		return
	}

	l.Info("job completed",
		zap.Duration("duration", elapsed),
	)
}

// zapCronAdapter bridges robfig/cron's Printf logger to zap.
type zapCronAdapter struct {
	log *zap.SugaredLogger
}

func newZapCronAdapter(log *zap.Logger) *zapCronAdapter {
	return &zapCronAdapter{log: log.Named("cron-internal").Sugar()}
}

func (a *zapCronAdapter) Printf(format string, args ...any) {
	a.log.Debugf(format, args...)
}
