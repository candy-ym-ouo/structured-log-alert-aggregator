package port

import (
	"context"
	"structured-log-alert-aggregator/internal/domain"
	"time"
)

type IngestResult struct {
	Duplicate bool
	AlertID   string
}
type NotificationJob struct {
	ID, AlertID, Channel, Kind string
	Sequence, Attempts         int
	AvailableAt                time.Time
	Reason                     string
}
type Silence struct {
	ID, AlertID      string
	StartsAt, EndsAt time.Time
}
type Repository interface {
	Ingest(context.Context, domain.LogEvent, domain.AlertPolicy) (IngestResult, error)
	ListAlerts(context.Context, string) ([]domain.Alert, error)
	Acknowledge(context.Context, string, string, string) (bool, error)
	DueForRecovery(context.Context, time.Time) ([]domain.Alert, error)
	Transition(context.Context, domain.Alert, domain.AlertState, string) error
	Policies(context.Context) ([]domain.AlertPolicy, error)
	CreateSilence(context.Context, Silence) error
	Silenced(context.Context, string, time.Time) (bool, error)
	ClaimJobs(context.Context, int, time.Time) ([]NotificationJob, error)
	CompleteJob(context.Context, NotificationJob, string, error) error
	Ready(context.Context) error
}
