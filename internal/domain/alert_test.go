package domain

import (
	"testing"
	"time"
)

func TestAlertTransitions(t *testing.T) {
	now := time.Now()
	a := Alert{State: Open, LastSeen: now.Add(-time.Hour)}
	if !a.Scan(now, time.Minute) || a.State != Recovering {
		t.Fatal("scan")
	}
	if !a.Resolve(now) || a.State != Resolved {
		t.Fatal("resolve")
	}
}
