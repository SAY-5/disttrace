// Package trace models OpenTelemetry-shaped spans + the trace tree
// they form. We use only the fields the bottleneck analyzer needs:
// trace_id, span_id, parent_span_id, service, name, start/end ns.
package trace

import (
	"encoding/json"
	"errors"
	"sort"
)

type SpanID = string
type TraceID = string

type Span struct {
	TraceID      TraceID  `json:"trace_id"`
	SpanID       SpanID   `json:"span_id"`
	ParentSpanID SpanID   `json:"parent_span_id,omitempty"`
	Service      string   `json:"service"`
	Name         string   `json:"name"`
	StartNs      int64    `json:"start_ns"`
	EndNs        int64    `json:"end_ns"`
	StatusCode   string   `json:"status,omitempty"` // "ok" | "error"
}

func (s Span) DurationNs() int64 {
	if s.EndNs <= s.StartNs {
		return 0
	}
	return s.EndNs - s.StartNs
}

// Trace is a complete causal tree of spans sharing a trace_id.
type Trace struct {
	ID    TraceID
	Spans []Span
	root  *Span
}

func (t *Trace) Root() *Span {
	if t.root != nil {
		return t.root
	}
	for i := range t.Spans {
		if t.Spans[i].ParentSpanID == "" {
			t.root = &t.Spans[i]
			return t.root
		}
	}
	return nil
}

// Children returns spans whose parent is `id`. O(N) in trace size;
// production indexes by parent_id at ingest.
func (t *Trace) Children(id SpanID) []Span {
	out := []Span{}
	for _, s := range t.Spans {
		if s.ParentSpanID == id {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartNs < out[j].StartNs })
	return out
}

// ParseSpan reads one OTLP-shaped JSON span.
func ParseSpan(raw []byte) (Span, error) {
	var s Span
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, err
	}
	if s.TraceID == "" || s.SpanID == "" {
		return s, errors.New("trace: span missing trace_id or span_id")
	}
	return s, nil
}

// Group spans into per-trace bundles.
func GroupTraces(spans []Span) []*Trace {
	by := map[TraceID]*Trace{}
	for _, s := range spans {
		t := by[s.TraceID]
		if t == nil {
			t = &Trace{ID: s.TraceID}
			by[s.TraceID] = t
		}
		t.Spans = append(t.Spans, s)
	}
	keys := make([]TraceID, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*Trace, 0, len(by))
	for _, k := range keys {
		out = append(out, by[k])
	}
	return out
}
