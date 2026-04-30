// Package api exposes a /spans ingest endpoint and a /traces
// readout, plus an SSE /stream that emits per-trace summaries.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/SAY-5/disttrace/analyze"
	"github.com/SAY-5/disttrace/trace"
)

type Server struct {
	mu    sync.Mutex
	spans []trace.Span
}

func New() *Server { return &Server{} }

func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/spans", s.handleSpans)
	mux.HandleFunc("/services", s.handleServices)
	mux.HandleFunc("/bottlenecks", s.handleBottlenecks)
	mux.HandleFunc("/stream", s.handleStream)
}

func (s *Server) handleSpans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var span trace.Span
	if err := json.NewDecoder(r.Body).Decode(&span); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if span.TraceID == "" || span.SpanID == "" {
		http.Error(w, "missing ids", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.spans = append(s.spans, span)
	s.mu.Unlock()
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleServices(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	traces := trace.GroupTraces(s.spans)
	s.mu.Unlock()
	stats := analyze.PerService(traces)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleBottlenecks(w http.ResponseWriter, r *http.Request) {
	threshold := int64(100_000_000) // 100ms default
	if v := r.URL.Query().Get("threshold_ns"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &threshold)
	}
	s.mu.Lock()
	traces := trace.GroupTraces(s.spans)
	s.mu.Unlock()
	stats := analyze.PerService(traces)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(analyze.Bottlenecks(stats, threshold))
}

func (s *Server) handleStream(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	flusher, _ := w.(http.Flusher)
	s.mu.Lock()
	traces := trace.GroupTraces(s.spans)
	s.mu.Unlock()
	emit := func(event string, payload any) {
		body, _ := json.Marshal(payload)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, body)
		if flusher != nil {
			flusher.Flush()
		}
	}
	emit("start", map[string]any{"n": len(traces)})
	for _, t := range traces {
		emit("trace", map[string]any{
			"trace_id": t.ID,
			"n_spans":  len(t.Spans),
		})
	}
	emit("done", map[string]any{})
}
