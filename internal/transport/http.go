package transport

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"structured-log-alert-aggregator/internal/app"
	"structured-log-alert-aggregator/internal/domain"
	"time"
)

type Server struct {
	app    *app.Service
	tokens map[string]string
}

func New(a *app.Service, tokens map[string]string) http.Handler {
	if a == nil {
		a = app.NewWithRepository(nil)
	}
	return &Server{a, tokens}
}

type apiError struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
		Details   any    `json:"details"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, msg string, details any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	x := apiError{}
	x.Error.Code = code
	x.Error.Message = msg
	x.Error.RequestID = r.Header.Get("X-Request-ID")
	x.Error.Details = details
	_ = json.NewEncoder(w).Encode(x)
}
func (s *Server) auth(w http.ResponseWriter, r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		writeError(w, r, 401, "unauthorized", "bearer token required", nil)
		return ""
	}
	t := s.tokens[strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))]
	if t == "" {
		writeError(w, r, 401, "unauthorized", "invalid bearer token", nil)
	}
	return t
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	r.Header.Set("X-Request-ID", requestID)
	w.Header().Set("X-Request-ID", requestID)
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/v1/healthz":
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	case r.URL.Path == "/v1/readyz":
		if e := s.app.Ready(r.Context()); e != nil {
			writeError(w, r, 503, "not_ready", e.Error(), nil)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	default:
		tenant := s.auth(w, r)
		if tenant == "" {
			return
		}
		switch {
		case r.Method == "POST" && r.URL.Path == "/v1/events":
			s.event(w, r, tenant)
		case r.Method == "GET" && r.URL.Path == "/v1/alerts":
			a, e := s.app.Alerts(r.Context(), tenant)
			if e != nil {
				writeError(w, r, 500, "storage_error", e.Error(), nil)
				return
			}
			json.NewEncoder(w).Encode(a)
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/ack"):
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/alerts/"), "/ack")
			ok, e := s.app.Acknowledge(r.Context(), tenant, id, r.Header.Get("X-Reason"))
			if e != nil {
				writeError(w, r, 500, "storage_error", e.Error(), nil)
				return
			}
			if !ok {
				writeError(w, r, 404, "not_found", "alert not found", nil)
				return
			}
			json.NewEncoder(w).Encode(map[string]bool{"acknowledged": true})
		default:
			http.NotFound(w, r)
		}
	}
}
func (s *Server) event(w http.ResponseWriter, r *http.Request, tenant string) {
	var e domain.LogEvent
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&e) != nil {
		writeError(w, r, 400, "invalid_json", "invalid event JSON", nil)
		return
	}
	if e.TenantID == "" {
		e.TenantID = tenant
	}
	if e.TenantID != tenant {
		writeError(w, r, 403, "tenant_mismatch", "event tenant does not match token", nil)
		return
	}
	result, err := s.app.Ingest(r.Context(), e)
	if err != nil {
		writeError(w, r, 422, "invalid_event", err.Error(), nil)
		return
	}
	if result.Duplicate {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"duplicate": true, "alert_id": result.AlertID})
	} else {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"accepted": true, "alert_id": result.AlertID})
	}
}
