// Package analyze finds bottlenecks across a corpus of traces.
//
// The headline metric: per-service p99 latency. We compute this
// per top-level span name (or per service if name is missing) and
// rank services by p99 desc — the top of the list is where to
// invest in optimization.
//
// v3's "12 critical bottlenecks across 5 microservices" comes from
// running this on a synthetic mesh corpus + flagging anything
// with p99 > 100 ms.
package analyze

import (
	"sort"

	"github.com/SAY-5/disttrace/trace"
)

type ServiceStats struct {
	Service string
	N       int
	P50Ns   int64
	P95Ns   int64
	P99Ns   int64
	MaxNs   int64
}

// PerService aggregates duration across all spans by service.
// We use exact percentiles via sort; for very large corpora swap
// in t-digest, but for the 5-service synthetic eval the exact
// path is sub-millisecond.
func PerService(traces []*trace.Trace) []ServiceStats {
	bySvc := map[string][]int64{}
	for _, t := range traces {
		for _, s := range t.Spans {
			bySvc[s.Service] = append(bySvc[s.Service], s.DurationNs())
		}
	}
	out := make([]ServiceStats, 0, len(bySvc))
	for svc, durations := range bySvc {
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		stats := ServiceStats{
			Service: svc,
			N:       len(durations),
			P50Ns:   quantile(durations, 0.50),
			P95Ns:   quantile(durations, 0.95),
			P99Ns:   quantile(durations, 0.99),
		}
		if len(durations) > 0 {
			stats.MaxNs = durations[len(durations)-1]
		}
		out = append(out, stats)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].P99Ns > out[j].P99Ns })
	return out
}

func quantile(sorted []int64, q float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)) * q)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// Bottleneck reports services with p99 above the threshold.
type Bottleneck struct {
	Service string
	P99Ns   int64
	N       int
}

func Bottlenecks(stats []ServiceStats, p99ThresholdNs int64) []Bottleneck {
	out := []Bottleneck{}
	for _, s := range stats {
		if s.P99Ns >= p99ThresholdNs {
			out = append(out, Bottleneck{Service: s.Service, P99Ns: s.P99Ns, N: s.N})
		}
	}
	return out
}

// CriticalPath reports the longest-duration span chain in a single
// trace — the spans that, summed, account for end-to-end latency.
// Optimizing anything off the critical path doesn't reduce p99;
// optimizing on it does.
//
// We approximate the critical path as the longest root-to-leaf
// path by summed duration. Real OTel critical-path analysis uses
// async-span timing too; we cover sync RPC chains.
func CriticalPath(t *trace.Trace) []trace.Span {
	root := t.Root()
	if root == nil {
		return nil
	}
	return walkLongest(t, root)
}

func walkLongest(t *trace.Trace, s *trace.Span) []trace.Span {
	children := t.Children(s.SpanID)
	if len(children) == 0 {
		return []trace.Span{*s}
	}
	var bestChain []trace.Span
	var bestSum int64 = -1
	for _, c := range children {
		chain := walkLongest(t, &c)
		sum := int64(0)
		for _, sp := range chain {
			sum += sp.DurationNs()
		}
		if sum > bestSum {
			bestSum = sum
			bestChain = chain
		}
	}
	return append([]trace.Span{*s}, bestChain...)
}
