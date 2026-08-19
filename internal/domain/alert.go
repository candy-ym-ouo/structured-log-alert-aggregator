package domain

import "time"

type AlertState string

const (
	Open         AlertState = "open"
	Acknowledged AlertState = "acknowledged"
	Recovering   AlertState = "recovering"
	Resolved     AlertState = "resolved"
	Suppressed   AlertState = "suppressed"
)

type Alert struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Fingerprint string     `json:"fingerprint"`
	Service     string     `json:"service"`
	Environment string     `json:"environment"`
	State       AlertState `json:"state"`
	Count       int        `json:"count"`
	FirstSeen   time.Time  `json:"first_seen"`
	LastSeen    time.Time  `json:"last_seen"`
	ReopenCount int        `json:"reopen_count"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}
type Transition struct {
	AlertID  string
	From, To AlertState
	At       time.Time
	Reason   string
}

func (a *Alert) Add(e LogEvent, now time.Time) {
	a.Count++
	a.LastSeen = now
	if a.State == Resolved {
		a.State = Open
		a.ReopenCount++
		a.ResolvedAt = nil
	}
}
func (a *Alert) Acknowledge() bool {
	if a.State != Open {
		return false
	}
	a.State = Acknowledged
	return true
}
func (a *Alert) Scan(now time.Time, quiet time.Duration) bool {
	if a.State == Open || a.State == Acknowledged {
		if now.Sub(a.LastSeen) >= quiet {
			a.State = Recovering
			return true
		}
	}
	return false
}
func (a *Alert) Resolve(now time.Time) bool {
	if a.State != Recovering {
		return false
	}
	a.State = Resolved
	a.ResolvedAt = &now
	return true
}
