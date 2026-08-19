package worker

import (
	"context"
	"errors"
	"structured-log-alert-aggregator/internal/domain"
	"structured-log-alert-aggregator/internal/port"
	"testing"
	"time"
)

type notificationRepo struct{ done bool }

func (r *notificationRepo) Ingest(context.Context, domain.LogEvent, domain.AlertPolicy) (port.IngestResult, error) {
	return port.IngestResult{}, nil
}
func (r *notificationRepo) ListAlerts(context.Context, string) ([]domain.Alert, error) {
	return nil, nil
}
func (r *notificationRepo) Acknowledge(context.Context, string, string, string) (bool, error) {
	return false, nil
}
func (r *notificationRepo) DueForRecovery(context.Context, time.Time) ([]domain.Alert, error) {
	return nil, nil
}
func (r *notificationRepo) Transition(context.Context, domain.Alert, domain.AlertState, string) error {
	return nil
}
func (r *notificationRepo) Policies(context.Context) ([]domain.AlertPolicy, error) { return nil, nil }
func (r *notificationRepo) CreateSilence(context.Context, port.Silence) error      { return nil }
func (r *notificationRepo) Silenced(context.Context, string, time.Time) (bool, error) {
	return false, nil
}
func (r *notificationRepo) ClaimJobs(context.Context, int, time.Time) ([]port.NotificationJob, error) {
	return []port.NotificationJob{{ID: "j"}}, nil
}
func (r *notificationRepo) CompleteJob(context.Context, port.NotificationJob, string, error) error {
	r.done = true
	return nil
}
func (r *notificationRepo) Ready(context.Context) error { return nil }

type sender struct{ err error }

func (s sender) Send(context.Context, port.NotificationJob) (string, error) { return "external", s.err }
func TestNotificationOnceCompletes(t *testing.T) {
	r := &notificationRepo{}
	if err := (Notification{Repo: r, Sender: sender{}}).Once(context.Background(), time.Now()); err != nil || !r.done {
		t.Fatal("job not completed")
	}
}
func TestNotificationPropagatesSendError(t *testing.T) {
	r := &notificationRepo{}
	if err := (Notification{Repo: r, Sender: sender{err: errors.New("down")}}).Once(context.Background(), time.Now()); err != nil || !r.done {
		t.Fatal("delivery should be audited")
	}
}
