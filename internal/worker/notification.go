package worker

import (
	"context"
	"log/slog"
	"structured-log-alert-aggregator/internal/port"
	"time"
)

type Sender interface {
	Send(context.Context, port.NotificationJob) (string, error)
}
type LogSender struct{}

func (LogSender) Send(_ context.Context, j port.NotificationJob) (string, error) {
	slog.Info("notification delivered", "job_id", j.ID, "channel", j.Channel, "kind", j.Kind)
	return j.ID, nil
}

type Notification struct {
	Repo      port.Repository
	Sender    Sender
	Interval  time.Duration
	BatchSize int
}

func (w Notification) Run(ctx context.Context) {
	interval := w.Interval
	if interval == 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := w.Once(ctx, time.Now().UTC()); err != nil {
			slog.Error("notification worker failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (w Notification) Once(ctx context.Context, now time.Time) error {
	n := w.BatchSize
	if n <= 0 {
		n = 32
	}
	jobs, err := w.Repo.ClaimJobs(ctx, n, now)
	if err != nil {
		return err
	}
	sender := w.Sender
	if sender == nil {
		sender = LogSender{}
	}
	for _, j := range jobs {
		external, e := sender.Send(ctx, j)
		if x := w.Repo.CompleteJob(ctx, j, external, e); x != nil {
			return x
		}
	}
	return nil
}
