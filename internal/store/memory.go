package store

import (
	"context"
	"fmt"
	"structured-log-alert-aggregator/internal/domain"
	"structured-log-alert-aggregator/internal/port"
	"sync"
	"time"
)

type Memory struct {
	mu       sync.Mutex
	events   map[string]string
	alerts   map[string]*domain.Alert
	policies []domain.AlertPolicy
	silences []port.Silence
	jobs     []port.NotificationJob
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func NewMemory() *Memory {
	return &Memory{events: map[string]string{}, alerts: map[string]*domain.Alert{}}
}
func (m *Memory) Ingest(_ context.Context, e domain.LogEvent, p domain.AlertPolicy) (port.IngestResult, error) {
	if err := e.Normalize(time.Now().UTC()); err != nil {
		return port.IngestResult{}, fmt.Errorf("normalize event: %v", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if id := m.events[e.ID]; id != "" {
		return port.IngestResult{Duplicate: true, AlertID: id}, nil
	}
	fp := e.Fingerprint()
	a := m.alerts[fp]
	if a == nil {
		a = &domain.Alert{ID: fp[:16], TenantID: e.TenantID, Fingerprint: fp, Service: e.Service, Environment: e.Environment, State: domain.Open, Count: 1, FirstSeen: e.OccurredAt, LastSeen: e.OccurredAt}
		m.alerts[fp] = a
	} else {
		a.Add(e, e.OccurredAt)
	}
	m.events[e.ID] = a.ID
	if p.TriggerCount > 0 && a.Count == p.TriggerCount {
		for _, c := range p.Channels {
			m.jobs = append(m.jobs, port.NotificationJob{ID: fmt.Sprintf("%s-%s-%d", a.ID, c, a.Count), AlertID: a.ID, Channel: c, Kind: "trigger", Sequence: a.Count, AvailableAt: time.Now()})
		}
	}
	return port.IngestResult{AlertID: a.ID}, nil
}
func (m *Memory) ListAlerts(_ context.Context, tenant string) ([]domain.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []domain.Alert{}
	for _, a := range m.alerts {
		if tenant == "" || a.TenantID == tenant {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (m *Memory) Acknowledge(_ context.Context, tenant, id, reason string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.alerts {
		if a.ID == id && a.TenantID == tenant {
			return a.Acknowledge(), nil
		}
	}
	return false, nil
}
func (m *Memory) DueForRecovery(_ context.Context, now time.Time) ([]domain.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []domain.Alert{}
	for _, a := range m.alerts {
		if a.State == domain.Open || a.State == domain.Acknowledged || a.State == domain.Recovering {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (m *Memory) Transition(_ context.Context, a domain.Alert, to domain.AlertState, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.alerts {
		if x.ID == a.ID && x.State == a.State {
			now := time.Now().UTC()
			if to == domain.Recovering {
				x.State = to
			} else if to == domain.Resolved {
				x.State = to
				x.ResolvedAt = &now
			}
			return nil
		}
	}
	return nil
}
func (m *Memory) Policies(context.Context) ([]domain.AlertPolicy, error) { return m.policies, nil }
func (m *Memory) CreateSilence(_ context.Context, s port.Silence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.silences = append(m.silences, s)
	return nil
}
func (m *Memory) Silenced(_ context.Context, id string, now time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.silences {
		if s.AlertID == id && now.After(s.StartsAt) && now.Before(s.EndsAt) {
			return true, nil
		}
	}
	return false, nil
}
func (m *Memory) ClaimJobs(_ context.Context, n int, now time.Time) ([]port.NotificationJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []port.NotificationJob{}
	for i := range m.jobs {
		if len(out) >= n {
			break
		}
		if !m.jobs[i].AvailableAt.After(now) && m.jobs[i].Attempts < 8 {
			m.jobs[i].Attempts++
			m.jobs[i].AvailableAt = now.Add(time.Minute)
			out = append(out, m.jobs[i])
		}
	}
	return out, nil
}
func (m *Memory) CompleteJob(_ context.Context, j port.NotificationJob, _ string, delivery error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.jobs {
		if m.jobs[i].ID == j.ID {
			if delivery == nil {
				m.jobs = append(m.jobs[:i], m.jobs[i+1:]...)
			} else {
				m.jobs[i].AvailableAt = time.Now().Add(time.Duration(1<<min(j.Attempts, 6)) * time.Second)
			}
			break
		}
	}
	return nil
}
func (m *Memory) Ready(context.Context) error { return nil }
