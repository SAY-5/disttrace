// Package sample implements head-based sampling for high-volume
// trace ingest.
//
// Real production systems run at 10k+ spans/sec/service; storing
// every one is wasteful. Head sampling decides at the start of the
// trace (when the root span fires) whether to keep all spans for
// that trace or drop them all. The decision is based on a hash of
// the trace_id so all spans for the same trace get the same answer
// — collectors don't have to coordinate.
//
// Tail sampling (decide AFTER the trace completes, based on
// duration / error / etc.) requires buffering the whole trace; we
// punt that to a future v5.
package sample

import (
	"hash/fnv"
)

// Sampler returns true if a trace_id should be kept.
type Sampler interface {
	ShouldKeep(traceID string) bool
}

// RatioSampler keeps `ratio` of traces (0.0 = drop everything,
// 1.0 = keep everything). The decision is deterministic per
// trace_id so different collectors agree.
type RatioSampler struct {
	Ratio float64
}

func (s RatioSampler) ShouldKeep(traceID string) bool {
	if s.Ratio >= 1.0 {
		return true
	}
	if s.Ratio <= 0.0 {
		return false
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(traceID))
	bucket := float64(h.Sum64()%10000) / 10000.0
	return bucket < s.Ratio
}

// AlwaysSampler keeps everything. Useful as the default when no
// sampling is configured + as a sentinel in tests.
type AlwaysSampler struct{}

func (AlwaysSampler) ShouldKeep(string) bool { return true }

// PrioritySampler keeps everything for traces marked high-priority
// via a tag, samples the rest at `BackgroundRatio`. This is what
// production gateways use to keep all traces for paying customers
// while sampling free-tier at 1%.
type PrioritySampler struct {
	BackgroundRatio float64
	HighPriority    map[string]bool // trace_id → keep-no-matter-what
}

func (s *PrioritySampler) MarkHighPriority(traceID string) {
	if s.HighPriority == nil {
		s.HighPriority = map[string]bool{}
	}
	s.HighPriority[traceID] = true
}

func (s *PrioritySampler) ShouldKeep(traceID string) bool {
	if s.HighPriority[traceID] {
		return true
	}
	return RatioSampler{Ratio: s.BackgroundRatio}.ShouldKeep(traceID)
}
