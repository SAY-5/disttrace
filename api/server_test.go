package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSpansEndpointAcceptsValidPayload(t *testing.T) {
	srv := New()
	mux := http.NewServeMux()
	srv.Routes(mux)
	body := strings.NewReader(`{"trace_id":"t","span_id":"s","service":"api","name":"x","start_ns":1,"end_ns":2}`)
	req := httptest.NewRequest("POST", "/spans", body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("status=%d", rec.Code)
	}
}

func TestSpansEndpointRejectsMissingIDs(t *testing.T) {
	srv := New()
	mux := http.NewServeMux()
	srv.Routes(mux)
	body := strings.NewReader(`{"service":"x"}`)
	req := httptest.NewRequest("POST", "/spans", body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStreamEmitsSSEFrames(t *testing.T) {
	srv := New()
	mux := http.NewServeMux()
	srv.Routes(mux)
	// Ingest a span first.
	body := strings.NewReader(`{"trace_id":"t","span_id":"s","service":"api","start_ns":0,"end_ns":100}`)
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/spans", body))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/stream", nil))
	body2 := rec.Body.String()
	for _, ev := range []string{"start", "trace", "done"} {
		if !strings.Contains(body2, "event: "+ev+"\n") {
			t.Errorf("missing event: %s", ev)
		}
	}
}
