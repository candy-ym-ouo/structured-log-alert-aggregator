package app

import (
	"context"
	"errors"
	"testing"

	"structured-log-alert-aggregator/internal/domain"
	"structured-log-alert-aggregator/internal/store"
)

func TestBug04InvalidEventKeepsCause(t *testing.T) {
	_, err := NewWithRepository(store.NewMemory()).Ingest(context.Background(), domain.LogEvent{})
	if err == nil || errors.Unwrap(err) == nil {
		t.Fatalf("invalid event error lost its cause: %v", err)
	}
}
