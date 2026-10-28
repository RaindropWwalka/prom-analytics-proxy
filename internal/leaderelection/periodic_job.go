package leaderelection

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"
)

// PeriodicJobOption tunes PeriodicJob.
type PeriodicJobOption func(*periodicJobOptions)

type periodicJobOptions struct {
	firstCycleDue func(context.Context) time.Time
}

// WithFirstCycleDue makes the first cycle after gaining leadership wait until
// due returns - typically the job's last successful cycle, read back from
// storage, plus its interval - so a restart or a deploy moving leadership
// doesn't start a full cycle the previous leader just finished. A time
// already past, or the zero time, runs it at once.
func WithFirstCycleDue(due func(context.Context) time.Time) PeriodicJobOption {
	return func(o *periodicJobOptions) { o.firstCycleDue = due }
}

// PeriodicJob adapts a plain per-cycle function into the shape Elector.Run
// needs: a ticking loop that reports each cycle to whatever's coordinating
// leadership (see CycleReporter) and runs it on a context of its own. A job
// stays an ordinary func(context.Context), never learning that leadership,
// budgets or reporting exist.
//
// Each cycle gets a fresh context, canceled as soon as that cycle returns,
// so canceling one - from the reporter's side or by the outer ctx ending -
// reaches everything that cycle started and nothing else. The loop keeps
// ticking on interval (jittered up to 20%, so replicas started together
// don't tick in lockstep) for as long as the outer ctx is alive, however
// any individual cycle ended.
//
// interval must be positive - a job's own config validation is what
// guarantees that, and a non-positive one panics here rather than ticking
// unboundedly. budget is only ever as meaningful as the reporter receiving
// it: with none, nothing reads it.
//
// By default the first cycle runs as soon as leadership is gained;
// WithFirstCycleDue can hold it back.
func PeriodicJob(interval, budget time.Duration, cycle func(context.Context), opts ...PeriodicJobOption) func(context.Context, CycleReporter) {
	var o periodicJobOptions
	for _, opt := range opts {
		opt(&o)
	}
	return func(ctx context.Context, reporter CycleReporter) {
		jitterBase := interval / 5
		if jitterBase <= 0 {
			jitterBase = 1
		}
		ticker := time.NewTicker(interval + time.Duration(rand.Int64N(int64(jitterBase))))
		defer ticker.Stop()

		runCycle := func() {
			cycleCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			if reporter != nil {
				reporter.CycleStarted(cycleCtx, cancel, budget)
			}
			cycle(cycleCtx)
		}

		if o.firstCycleDue != nil {
			if wait := time.Until(o.firstCycleDue(ctx)); wait > 0 {
				slog.Info("leader election: first cycle not due yet", "in", wait.Round(time.Second))
				timer := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				// Count the interval from this cycle, not from leadership.
				ticker.Reset(interval + time.Duration(rand.Int64N(int64(jitterBase))))
			}
		}
		runCycle()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCycle()
			}
		}
	}
}
