package transport

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBug02NilServiceRejectsIngestion(t *testing.T) {
	h := New(nil, map[string]string{"token": "tenant"})
	r := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString(`{"event_id":"event-1","tenant_id":"tenant","service":"api","environment":"prod","message":"failure"}`))
	r.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code < 500 {
		t.Fatalf("nil service accepted the event with status %d", w.Code)
	}
}
