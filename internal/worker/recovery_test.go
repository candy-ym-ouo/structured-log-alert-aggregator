package worker

import (
	"context"
	"structured-log-alert-aggregator/internal/domain"
	"structured-log-alert-aggregator/internal/port"
	"testing"
	"time"
)

type recoveryRepo struct {
	a  domain.Alert
	to domain.AlertState
}

func (r *recoveryRepo) Ingest(context.Context, domain.LogEvent, domain.AlertPolicy) (port.IngestResult, error) {
	return port.IngestResult{}, nil
}
func (r *recoveryRepo) ListAlerts(context.Context, string) ([]domain.Alert, error) { return nil, nil }
func (r *recoveryRepo) Acknowledge(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (r *recoveryRepo) DueForRecovery(context.Context, time.Time) ([]domain.Alert, error) {
	return []domain.Alert{r.a}, nil
}
func (r *recoveryRepo) Transition(_ context.Context, a domain.Alert, to domain.AlertState, _ string) error {
	r.to = to
	return nil
}
func (r *recoveryRepo) Policies(context.Context) ([]domain.AlertPolicy, error)    { return nil, nil }
func (r *recoveryRepo) CreateSilence(context.Context, port.Silence) error         { return nil }
func (r *recoveryRepo) Silenced(context.Context, string, time.Time) (bool, error) { return false, nil }
func (r *recoveryRepo) ClaimJobs(context.Context, int, time.Time) ([]port.NotificationJob, error) {
	return nil, nil
}
func (r *recoveryRepo) CompleteJob(context.Context, port.NotificationJob, string, error) error {
	return nil
}
func (r *recoveryRepo) Ready(context.Context) error { return nil }
func TestRecoveryUsesDefaultInterval(t *testing.T) {
	now := time.Now()
	r := &recoveryRepo{a: domain.Alert{State: domain.Recovering, LastSeen: now.Add(-6 * time.Minute)}}
	if err := (Recovery{Repo: r, Quiet: 5 * time.Minute}).Once(context.Background(), now); err != nil || r.to != domain.Resolved {
		t.Fatalf("expected resolve, got %q %v", r.to, err)
	}
}
func TestRecoveryUsesDefaultQuietWindow(t *testing.T) {
	now := time.Now()
	r := &recoveryRepo{a: domain.Alert{State: domain.Open, LastSeen: now.Add(-6 * time.Minute)}}
	if err := (Recovery{Repo: r}).Once(context.Background(), now); err != nil || r.to != domain.Recovering {
		t.Fatalf("expected recovering, got %q %v", r.to, err)
	}
}
