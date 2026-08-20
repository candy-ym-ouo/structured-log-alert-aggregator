package worker

import (
	"context"
	"testing"
	"time"

	"structured-log-alert-aggregator/internal/domain"
	"structured-log-alert-aggregator/internal/port"
)

type cancellationRepo struct {
	port.Repository
	seen context.Context
}

func (r *cancellationRepo) DueForRecovery(ctx context.Context, _ time.Time) ([]domain.Alert, error) {
	r.seen = ctx
	return nil, nil
}

func TestBug05RecoveryForwardsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := &cancellationRepo{}
	if err := (Recovery{Repo: repo}).Once(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	if repo.seen.Err() != context.Canceled {
		t.Fatalf("repository context was not canceled: %v", repo.seen.Err())
	}
}
