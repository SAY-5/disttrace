package analyze

import (
	"testing"

	"github.com/SAY-5/disttrace/trace"
)

func mkTrace(id string, spans ...trace.Span) *trace.Trace {
	for i := range spans {
		spans[i].TraceID = id
	}
	return &trace.Trace{ID: id, Spans: spans}
}

func TestPerServiceComputesQuantiles(t *testing.T) {
	tr := mkTrace("t",
		trace.Span{SpanID: "1", Service: "api", StartNs: 0, EndNs: 100},
		trace.Span{SpanID: "2", Service: "api", StartNs: 0, EndNs: 200},
		trace.Span{SpanID: "3", Service: "db", StartNs: 0, EndNs: 50},
	)
	stats := PerService([]*trace.Trace{tr})
	// Sorted by p99 desc → api first.
	if stats[0].Service != "api" {
		t.Errorf("expected api first, got %+v", stats)
	}
	if stats[0].N != 2 {
		t.Errorf("api n=%d want 2", stats[0].N)
	}
}

func TestBottlenecksFiltersByThreshold(t *testing.T) {
	stats := []ServiceStats{
		{Service: "fast", P99Ns: 50_000_000},
		{Service: "slow", P99Ns: 200_000_000},
	}
	bs := Bottlenecks(stats, 100_000_000)
	if len(bs) != 1 || bs[0].Service != "slow" {
		t.Errorf("expected only slow, got %+v", bs)
	}
}

func TestCriticalPathFollowsLongestChain(t *testing.T) {
	tr := mkTrace("t",
		trace.Span{SpanID: "root", StartNs: 0, EndNs: 1000, Service: "api"},
		trace.Span{SpanID: "fast", ParentSpanID: "root", StartNs: 100, EndNs: 200, Service: "x"},
		trace.Span{SpanID: "slow", ParentSpanID: "root", StartNs: 100, EndNs: 900, Service: "y"},
	)
	path := CriticalPath(tr)
	if len(path) != 2 {
		t.Fatalf("expected 2-span path, got %+v", path)
	}
	if path[1].SpanID != "slow" {
		t.Errorf("expected slow in critical path, got %+v", path)
	}
}

func TestEmptyTraceReturnsEmptyStats(t *testing.T) {
	stats := PerService(nil)
	if len(stats) != 0 {
		t.Errorf("empty input should give empty stats")
	}
}
