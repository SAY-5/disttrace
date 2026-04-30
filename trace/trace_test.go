package trace

import "testing"

func TestParseSpanRoundTrips(t *testing.T) {
	raw := []byte(`{"trace_id":"t1","span_id":"s1","service":"api","name":"GET /x","start_ns":1000,"end_ns":2000}`)
	s, err := ParseSpan(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.DurationNs() != 1000 {
		t.Errorf("duration=%d want 1000", s.DurationNs())
	}
}

func TestParseSpanRejectsMissingIDs(t *testing.T) {
	_, err := ParseSpan([]byte(`{"trace_id":"","span_id":"s","service":"x"}`))
	if err == nil {
		t.Errorf("expected error on empty trace_id")
	}
}

func TestGroupTracesByID(t *testing.T) {
	spans := []Span{
		{TraceID: "a", SpanID: "1", Service: "x"},
		{TraceID: "a", SpanID: "2", Service: "x", ParentSpanID: "1"},
		{TraceID: "b", SpanID: "3", Service: "y"},
	}
	traces := GroupTraces(spans)
	if len(traces) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(traces))
	}
	if traces[0].ID != "a" || len(traces[0].Spans) != 2 {
		t.Errorf("trace a wrong: %+v", traces[0])
	}
}

func TestRootIsSpanWithoutParent(t *testing.T) {
	tr := &Trace{
		ID: "x",
		Spans: []Span{
			{TraceID: "x", SpanID: "child", ParentSpanID: "root"},
			{TraceID: "x", SpanID: "root"},
		},
	}
	root := tr.Root()
	if root == nil || root.SpanID != "root" {
		t.Errorf("expected root span, got %+v", root)
	}
}

func TestChildrenSortedByStart(t *testing.T) {
	tr := &Trace{
		Spans: []Span{
			{SpanID: "1"},
			{SpanID: "2", ParentSpanID: "1", StartNs: 200},
			{SpanID: "3", ParentSpanID: "1", StartNs: 100},
		},
	}
	cs := tr.Children("1")
	if len(cs) != 2 || cs[0].SpanID != "3" {
		t.Errorf("children not sorted: %+v", cs)
	}
}
