package domain

import "testing"

func TestFingerprintStable(t *testing.T) {
	a := LogEvent{ID: "1", TenantID: "t", Service: "api", Environment: "prod", Message: "failed user 123"}
	b := LogEvent{ID: "2", TenantID: "t", Service: "api", Environment: "prod", Message: "failed user 456"}
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatal("expected normalized IDs")
	}
}
