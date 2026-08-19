package domain

import "time"

type AlertPolicy struct {
	ID, Service, Environment, ErrorType         string
	TriggerCount                                int
	Window                                      time.Duration
	NotifyAfter, RepeatInterval, RecoveryWindow time.Duration
	Severity                                    string
	Channels                                    []string
}

func (p AlertPolicy) Matches(e LogEvent) bool {
	return (p.Service == "" || p.Service == e.Service) && (p.Environment == "" || p.Environment == e.Environment) && (p.ErrorType == "" || p.ErrorType == e.ErrorType)
}
func SelectPolicy(ps []AlertPolicy, e LogEvent) (AlertPolicy, bool) {
	var selected AlertPolicy
	score := -1
	for _, p := range ps {
		if !p.Matches(e) {
			continue
		}
		s := 0
		if p.Service != "" {
			s += 2
		}
		if p.Environment != "" {
			s += 2
		}
		if p.ErrorType != "" {
			s += 4
		}
		if s > score {
			selected, score = p, s
		}
	}
	return selected, score >= 0
}
