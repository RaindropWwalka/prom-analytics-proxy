package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/oklog/run"

	"github.com/nicolastakashi/prom-analytics-proxy/internal/leaderelection"
)

// periodicJob is the shape cmd/api needs from every job it schedules: one
// cycle per call, bounded by the context it's given, neither looping nor
// aware of who's calling it - see docs/jobs.md.
type periodicJob interface {
	RunOnce(ctx context.Context)
}

// recordedJob is a periodicJob that records its successful cycles in the
// database: a replica gaining leadership waits until the last one is an
// interval old rather than starting a cycle at once, so restarts and deploys
// don't each cost a full cycle.
type recordedJob interface {
	LastSuccess(ctx context.Context) (time.Time, bool, error)
}

// addPeriodicJob wires job into g under leaseName, scheduling its cycles for
// as long as elector says this replica holds that lease. Every provider has
// an elector - a sole instance holds leadership unconditionally where
// there's nothing to coordinate - so leadership governs who runs cycles and
// never how often they run, and there is one path here rather than one per
// provider.
//
// interval and budget are the caller's own config values (see docs/jobs.md
// for what makes a job's budget a true bound on one of its cycles), not
// something this package derives.
func addPeriodicJob(g *run.Group, elector leaderelection.Elector, leaseName string, interval, budget time.Duration, job periodicJob) {
	ctx, cancel := context.WithCancel(context.Background())
	var opts []leaderelection.PeriodicJobOption
	if r, ok := job.(recordedJob); ok {
		opts = append(opts, leaderelection.WithFirstCycleDue(func(ctx context.Context) time.Time {
			last, ok, err := r.LastSuccess(ctx)
			if err != nil {
				slog.Warn("last successful cycle unreadable, running now", "lease", leaseName, "err", err)
				return time.Time{}
			}
			if !ok {
				return time.Time{}
			}
			return last.Add(interval)
		}))
	}
	loop := leaderelection.PeriodicJob(interval, budget, job.RunOnce, opts...)

	g.Add(func() error {
		return elector.Run(ctx, leaseName, loop)
	}, func(err error) { cancel() })
}
