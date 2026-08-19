package transport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"structured-log-alert-aggregator/internal/app"
	"structured-log-alert-aggregator/internal/store"
	"testing"
)

func testServer() http.Handler {
	return New(app.NewWithRepository(store.NewMemory()), map[string]string{"token-a": "tenant-a", "token-b": "tenant-b"})
}
func req(s http.Handler, method, path, token string, body []byte) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestRequiresBearerToken(t *testing.T) {
	w := req(testServer(), "GET", "/v1/alerts", "", nil)
	if w.Code != 401 {
		t.Fatalf("got %d", w.Code)
	}
	var v map[string]any
	if json.Unmarshal(w.Body.Bytes(), &v) != nil || v["error"] == nil {
		t.Fatal("expected structured error")
	}
}
func TestTenantIsolation(t *testing.T) {
	s := testServer()
	body := []byte(`{"event_id":"e1","tenant_id":"tenant-a","service":"api","environment":"prod","message":"boom"}`)
	if w := req(s, "POST", "/v1/events", "token-a", body); w.Code != 202 {
		t.Fatalf("ingest %d: %s", w.Code, w.Body.String())
	}
	if w := req(s, "POST", "/v1/events", "token-b", body); w.Code != 403 {
		t.Fatalf("tenant mismatch got %d", w.Code)
	}
	a := req(s, "GET", "/v1/alerts", "token-a", nil)
	b := req(s, "GET", "/v1/alerts", "token-b", nil)
	if !bytes.Contains(a.Body.Bytes(), []byte(`"tenant_id":"tenant-a"`)) {
		t.Fatal("tenant a alert missing")
	}
	if bytes.Contains(b.Body.Bytes(), []byte(`tenant-a`)) {
		t.Fatal("tenant leaked")
	}
}
