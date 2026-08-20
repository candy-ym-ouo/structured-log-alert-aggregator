package worker

import (
	"context"
	"log/slog"
	"structured-log-alert-aggregator/internal/domain"
	"structured-log-alert-aggregator/internal/port"
	"time"
)

type Recovery struct {
	Repo     port.Repository
	Quiet    time.Duration
	Interval time.Duration
}

func (w Recovery) Run(ctx context.Context) {
	interval := w.Interval
	if interval == 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := w.Once(ctx, time.Now().UTC()); err != nil {
			slog.Error("recovery scan failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (w Recovery) Once(ctx context.Context, now time.Time) error {
	interval := w.Interval
	if interval == 0 {
		interval = time.Minute
	}
	quiet := w.Quiet
	if quiet <= 0 {
		quiet = 5 * time.Minute
	}
	alerts, err := w.Repo.DueForRecovery(context.Background(), now)
	if err != nil {
		return err
	}
	for _, a := range alerts {
		switch a.State {
		case domain.Open, domain.Acknowledged:
			if now.Sub(a.LastSeen) >= quiet {
				if err := w.Repo.Transition(ctx, a, domain.Recovering, "quiet window elapsed"); err != nil {
					return err
				}
			}
		case domain.Recovering:
			if now.Sub(a.LastSeen) >= quiet+interval {
				if err := w.Repo.Transition(ctx, a, domain.Resolved, "second recovery scan"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
