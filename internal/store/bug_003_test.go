package store

import (
	"context"
	"testing"

	"structured-log-alert-aggregator/internal/app"
	"structured-log-alert-aggregator/internal/domain"
)

func TestBug03PolicyChannelsAreNotMutatedDuringIngest(t *testing.T) {
	m := NewMemory()
	m.policies = []domain.AlertPolicy{{Service: "api", Channels: []string{"email"}}}
	m.policies[0].Channels[0] = "email"
	_, err := app.NewWithRepository(m).Ingest(context.Background(), domain.LogEvent{ID: "event-1", TenantID: "tenant", Service: "api", Environment: "prod", Message: "failure"})
	if err != nil {
		t.Fatal(err)
	}
	if m.policies[0].Channels[0] != "email" {
		t.Fatalf("policy channels changed to %#v", m.policies[0].Channels)
	}
}
