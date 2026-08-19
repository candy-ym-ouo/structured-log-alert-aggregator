//go:build mysql

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"structured-log-alert-aggregator/internal/domain"
	"structured-log-alert-aggregator/internal/port"
	"time"
)

type MySQL struct{ DB *sql.DB }

func mysqlMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func NewMySQL(ctx context.Context, dsn string) (*MySQL, error) {
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		return nil, e
	}
	if e = db.PingContext(ctx); e != nil {
		db.Close()
		return nil, e
	}
	return &MySQL{db}, nil
}
func (m *MySQL) Ready(c context.Context) error { return m.DB.PingContext(c) }
func (m *MySQL) Ingest(c context.Context, e domain.LogEvent, p domain.AlertPolicy) (port.IngestResult, error) {
	if x := e.Normalize(time.Now().UTC()); x != nil {
		return port.IngestResult{}, x
	}
	tx, x := m.DB.BeginTx(c, nil)
	if x != nil {
		return port.IngestResult{}, x
	}
	defer tx.Rollback()
	labels, _ := json.Marshal(e.Labels)
	r, x := tx.ExecContext(c, `INSERT IGNORE INTO log_events(event_id,tenant_id,occurred_at,service,environment,level,message,error_type,stack_trace,labels,fingerprint,truncated) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, e.ID, e.TenantID, e.OccurredAt, e.Service, e.Environment, e.Level, e.Message, e.ErrorType, e.StackTrace, labels, e.Fingerprint(), e.Truncated)
	if x != nil {
		return port.IngestResult{}, x
	}
	if n, _ := r.RowsAffected(); n == 0 {
		var id, owner, fingerprint string
		x = tx.QueryRowContext(c, `SELECT tenant_id,fingerprint FROM log_events WHERE event_id=?`, e.ID).Scan(&owner, &fingerprint)
		if x != nil {
			return port.IngestResult{}, x
		}
		if owner != e.TenantID {
			return port.IngestResult{}, fmt.Errorf("event_id conflict")
		}
		x = tx.QueryRowContext(c, `SELECT id FROM alert_groups WHERE tenant_id=? AND fingerprint=?`, owner, fingerprint).Scan(&id)
		if x != nil {
			return port.IngestResult{}, x
		}
		return port.IngestResult{Duplicate: true, AlertID: id}, tx.Commit()
	}
	id := e.Fingerprint()[:16]
	_, x = tx.ExecContext(c, `INSERT INTO alert_groups(id,tenant_id,fingerprint,service,environment,state,event_count,first_seen,last_seen) VALUES(?,?,?,?,?,'open',1,?,?) ON DUPLICATE KEY UPDATE event_count=event_count+1,last_seen=VALUES(last_seen),reopen_count=reopen_count+IF(state='resolved',1,0),resolved_at=IF(state='resolved',NULL,resolved_at),state=IF(state='resolved','open',state)`, id, e.TenantID, e.Fingerprint(), e.Service, e.Environment, e.OccurredAt, e.OccurredAt)
	if x != nil {
		return port.IngestResult{}, x
	}
	var count int
	x = tx.QueryRowContext(c, `SELECT id,event_count FROM alert_groups WHERE tenant_id=? AND fingerprint=?`, e.TenantID, e.Fingerprint()).Scan(&id, &count)
	if x != nil {
		return port.IngestResult{}, x
	}
	if p.TriggerCount > 0 && count == p.TriggerCount {
		for _, ch := range p.Channels {
			_, x = tx.ExecContext(c, `INSERT IGNORE INTO notification_jobs(id,alert_id,channel,kind,sequence,available_at) VALUES(?,?,?,'trigger',?,UTC_TIMESTAMP(6))`, fmt.Sprintf("%s-%s-%d", id, ch, count), id, ch, count)
			if x != nil {
				return port.IngestResult{}, x
			}
		}
	}
	return port.IngestResult{AlertID: id}, tx.Commit()
}
func (m *MySQL) ListAlerts(c context.Context, t string) ([]domain.Alert, error) {
	rows, e := m.DB.QueryContext(c, `SELECT id,tenant_id,fingerprint,service,environment,state,event_count,first_seen,last_seen,reopen_count,resolved_at FROM alert_groups WHERE tenant_id=? ORDER BY last_seen DESC`, t)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Alert{}
	for rows.Next() {
		var a domain.Alert
		if e = rows.Scan(&a.ID, &a.TenantID, &a.Fingerprint, &a.Service, &a.Environment, &a.State, &a.Count, &a.FirstSeen, &a.LastSeen, &a.ReopenCount, &a.ResolvedAt); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (m *MySQL) Acknowledge(c context.Context, t, id, reason string) (bool, error) {
	r, e := m.DB.ExecContext(c, `UPDATE alert_groups SET state='acknowledged' WHERE id=? AND tenant_id=? AND state='open'`, id, t)
	if e != nil {
		return false, e
	}
	n, _ := r.RowsAffected()
	return n == 1, nil
}
func (m *MySQL) DueForRecovery(c context.Context, now time.Time) ([]domain.Alert, error) {
	rows, e := m.DB.QueryContext(c, `SELECT id,tenant_id,fingerprint,service,environment,state,event_count,first_seen,last_seen,reopen_count,resolved_at FROM alert_groups WHERE state IN ('open','acknowledged','recovering')`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Alert{}
	for rows.Next() {
		var a domain.Alert
		if e = rows.Scan(&a.ID, &a.TenantID, &a.Fingerprint, &a.Service, &a.Environment, &a.State, &a.Count, &a.FirstSeen, &a.LastSeen, &a.ReopenCount, &a.ResolvedAt); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (m *MySQL) Transition(c context.Context, a domain.Alert, to domain.AlertState, reason string) error {
	tx, e := m.DB.BeginTx(c, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var result sql.Result
	result, e = tx.ExecContext(c, `UPDATE alert_groups SET state=?,resolved_at=IF(?='resolved',UTC_TIMESTAMP(6),resolved_at) WHERE id=? AND state=?`, to, to, a.ID, a.State)
	if e == nil {
		var rows int64
		rows, e = result.RowsAffected()
		if e == nil && rows != 1 {
			return fmt.Errorf("stale alert transition")
		}
	}
	if e == nil {
		_, e = tx.ExecContext(c, `INSERT INTO alert_transitions(alert_id,from_state,to_state,reason) VALUES(?,?,?,?)`, a.ID, a.State, to, reason)
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (m *MySQL) Policies(c context.Context) ([]domain.AlertPolicy, error) {
	rows, e := m.DB.QueryContext(c, `SELECT id,service,environment,error_type,trigger_count,window_seconds,recovery_window_seconds,severity,channels FROM alert_policies`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.AlertPolicy{}
	for rows.Next() {
		var p domain.AlertPolicy
		var w, r int
		var ch []byte
		if e = rows.Scan(&p.ID, &p.Service, &p.Environment, &p.ErrorType, &p.TriggerCount, &w, &r, &p.Severity, &ch); e != nil {
			return nil, e
		}
		p.Window = time.Duration(w) * time.Second
		p.RecoveryWindow = time.Duration(r) * time.Second
		_ = json.Unmarshal(ch, &p.Channels)
		out = append(out, p)
	}
	return out, rows.Err()
}
func (m *MySQL) CreateSilence(c context.Context, s port.Silence) error {
	_, e := m.DB.ExecContext(c, `INSERT INTO silences(id,alert_id,starts_at,ends_at) VALUES(?,?,?,?)`, s.ID, s.AlertID, s.StartsAt, s.EndsAt)
	return e
}
func (m *MySQL) Silenced(c context.Context, id string, n time.Time) (bool, error) {
	var x int
	e := m.DB.QueryRowContext(c, `SELECT EXISTS(SELECT 1 FROM silences WHERE alert_id=? AND starts_at<=? AND ends_at>?)`, id, n, n).Scan(&x)
	return x == 1, e
}
func (m *MySQL) ClaimJobs(c context.Context, n int, now time.Time) ([]port.NotificationJob, error) {
	tx, e := m.DB.BeginTx(c, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	rows, e := tx.QueryContext(c, `SELECT id,alert_id,channel,kind,sequence,attempts,available_at FROM notification_jobs WHERE available_at<=? AND attempts<8 ORDER BY available_at LIMIT ? FOR UPDATE SKIP LOCKED`, now, n)
	if e != nil {
		return nil, e
	}
	jobs := []port.NotificationJob{}
	for rows.Next() {
		var j port.NotificationJob
		if e = rows.Scan(&j.ID, &j.AlertID, &j.Channel, &j.Kind, &j.Sequence, &j.Attempts, &j.AvailableAt); e != nil {
			rows.Close()
			return nil, e
		}
		j.Attempts++
		jobs = append(jobs, j)
	}
	rows.Close()
	for _, j := range jobs {
		_, e = tx.ExecContext(c, `UPDATE notification_jobs SET attempts=attempts+1,available_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL 60 SECOND) WHERE id=?`, j.ID)
		if e != nil {
			return nil, e
		}
	}
	return jobs, tx.Commit()
}
func (m *MySQL) CompleteJob(c context.Context, j port.NotificationJob, external string, delivery error) error {
	tx, e := m.DB.BeginTx(c, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	ok := delivery == nil
	msg := ""
	if delivery != nil {
		msg = delivery.Error()
	}
	_, e = tx.ExecContext(c, `INSERT INTO notification_attempts(job_id,success,external_id,error_message) VALUES(?,?,?,?)`, j.ID, ok, external, msg)
	if e != nil {
		return e
	}
	if ok {
		_, e = tx.ExecContext(c, `DELETE FROM notification_jobs WHERE id=?`, j.ID)
	} else if j.Attempts >= 8 {
		_, e = tx.ExecContext(c, `INSERT INTO notification_dead_letters(job_id,reason,attempts) VALUES(?,?,?)`, j.ID, msg, j.Attempts)
		if e == nil {
			_, e = tx.ExecContext(c, `DELETE FROM notification_jobs WHERE id=?`, j.ID)
		}
	} else {
		delay := 1 << mysqlMin(j.Attempts, 6)
		_, e = tx.ExecContext(c, `UPDATE notification_jobs SET available_at=DATE_ADD(UTC_TIMESTAMP(6),INTERVAL ? SECOND) WHERE id=?`, delay, j.ID)
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}
