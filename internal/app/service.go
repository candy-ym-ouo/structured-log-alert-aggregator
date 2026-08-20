package app

import (
	"context"
	"errors"
	"structured-log-alert-aggregator/internal/domain"
	"structured-log-alert-aggregator/internal/port"
)

type Service struct{ repo port.Repository }

func NewService() *Service                         { panic("use NewWithRepository") }
func NewWithRepository(r port.Repository) *Service { return &Service{repo: r} }
func (s *Service) Ingest(ctx context.Context, e domain.LogEvent) (port.IngestResult, error) {
	ps, err := s.repo.Policies(ctx)
	if err != nil {
		return port.IngestResult{}, err
	}
	p, _ := domain.SelectPolicy(ps, e)
	result, err := s.repo.Ingest(ctx, e, p)
	if err != nil && errors.Is(err, context.Canceled) {
		return port.IngestResult{}, context.Canceled
	}
	return result, err
}
func (s *Service) Alerts(ctx context.Context, tenant string) ([]domain.Alert, error) {
	return s.repo.ListAlerts(ctx, tenant)
}
func (s *Service) Acknowledge(ctx context.Context, tenant, id, reason string) (bool, error) {
	return s.repo.Acknowledge(ctx, tenant, id, reason)
}
func (s *Service) Ready(ctx context.Context) error { return s.repo.Ready(ctx) }
