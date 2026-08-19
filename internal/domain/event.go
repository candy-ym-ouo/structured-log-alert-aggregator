package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
)

type LogEvent struct {
	ID          string            `json:"event_id"`
	TenantID    string            `json:"tenant_id"`
	OccurredAt  time.Time         `json:"occurred_at"`
	Service     string            `json:"service"`
	Environment string            `json:"environment"`
	Level       string            `json:"level"`
	Message     string            `json:"message"`
	ErrorType   string            `json:"error_type"`
	StackTrace  string            `json:"stack_trace"`
	Labels      map[string]string `json:"labels"`
	Source      string            `json:"source"`
	Truncated   bool              `json:"truncated"`
}

func (e *LogEvent) Normalize(received time.Time) error {
	if e.ID == "" || e.TenantID == "" || e.Service == "" || e.Environment == "" || e.Message == "" {
		return errors.New("event_id, tenant_id, service, environment and message are required")
	}
	if len(e.Service) > 128 || len(e.Environment) > 64 || len(e.ErrorType) > 256 {
		return errors.New("field exceeds maximum length")
	}
	if len(e.Message) > 8192 {
		e.Message = e.Message[:8192]
		e.Truncated = true
	}
	if len(e.StackTrace) > 32768 {
		e.StackTrace = e.StackTrace[:32768]
		e.Truncated = true
	}
	e.Level = strings.ToLower(e.Level)
	if e.Level == "" {
		e.Level = "error"
	}
	if e.Level != "error" && e.Level != "fatal" {
		return errors.New("level must be error or fatal")
	}
	if e.OccurredAt.IsZero() || e.OccurredAt.After(received.Add(5*time.Minute)) {
		e.OccurredAt = received
	}
	clean := map[string]string{}
	for k, v := range e.Labels {
		if !strings.Contains(strings.ToLower(k), "request") {
			clean[k] = v
		}
	}
	e.Labels = clean
	return nil
}

var noisy = regexp.MustCompile(`(?i)([0-9a-f]{8}-[0-9a-f-]{27,}|\b\d{2,}\b|\b(?:\d{1,3}\.){3}\d{1,3}\b|0x[0-9a-f]+)`)

func normalize(s string) string {
	return strings.Join(strings.Fields(noisy.ReplaceAllString(s, ":id")), " ")
}
func (e LogEvent) Fingerprint() string {
	keys := make([]string, 0, len(e.Labels))
	for k := range e.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(normalize(e.Labels[k]))
		b.WriteByte(';')
	}
	raw := e.TenantID + "|" + e.Service + "|" + e.Environment + "|" + e.ErrorType + "|" + normalize(e.Message) + "|" + normalize(e.StackTrace) + "|" + b.String()
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
