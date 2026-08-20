package store

import (
	"context"
	"testing"

	"structured-log-alert-aggregator/internal/app"
	"structured-log-alert-aggregator/internal/domain"
)

func TestBug01IngestDoesNotRacePolicyRefresh(t *testing.T) {
	m := NewMemory()
	m.policies = []domain.AlertPolicy{{Service: "api", Channels: []string{"email"}}}
	s := app.NewWithRepository(m)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 500; i++ {
			m.mu.Lock()
			m.policies = []domain.AlertPolicy{{Service: "api", Channels: []string{"email"}}}
			m.mu.Unlock()
		}
		close(done)
	}()
	for i := 0; i < 500; i++ {
		_, _ = s.Ingest(context.Background(), domain.LogEvent{ID: "event-" + string(rune(i)), TenantID: "tenant", Service: "api", Environment: "prod", Message: "failure"})
	}
	<-done
}
