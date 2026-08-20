package store

import (
	"context"
	"testing"

	"structured-log-alert-aggregator/internal/app"
	"structured-log-alert-aggregator/internal/domain"
)

func TestBug01IngestDoesNotChangeStoredPolicyChannel(t *testing.T) {
	m := NewMemory()
	m.policies = []domain.AlertPolicy{{Service: "api", Channels: []string{"email"}}}
	s := app.NewWithRepository(m)
	_, err := s.Ingest(context.Background(), domain.LogEvent{ID: "event-1", TenantID: "tenant", Service: "api", Environment: "prod", Message: "failure"})
	if err != nil {
		t.Fatal(err)
	}
	if m.policies[0].Channels[0] != "email" {
		t.Fatalf("stored policy channel changed to %q", m.policies[0].Channels[0])
	}
}
