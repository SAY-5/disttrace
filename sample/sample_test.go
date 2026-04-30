package sample

import (
	"fmt"
	"testing"
)

func TestRatioOneKeepsEverything(t *testing.T) {
	s := RatioSampler{Ratio: 1.0}
	for i := 0; i < 100; i++ {
		if !s.ShouldKeep(fmt.Sprintf("t-%d", i)) {
			t.Errorf("expected keep at ratio=1.0")
		}
	}
}

func TestRatioZeroDropsEverything(t *testing.T) {
	s := RatioSampler{Ratio: 0.0}
	for i := 0; i < 100; i++ {
		if s.ShouldKeep(fmt.Sprintf("t-%d", i)) {
			t.Errorf("expected drop at ratio=0.0")
		}
	}
}

func TestRatioHalfApproximatesHalf(t *testing.T) {
	s := RatioSampler{Ratio: 0.5}
	kept := 0
	n := 5000
	for i := 0; i < n; i++ {
		if s.ShouldKeep(fmt.Sprintf("t-%d", i)) {
			kept++
		}
	}
	// 50% ± 3-sigma: kept should be 2500 ± 110.
	if kept < 2350 || kept > 2650 {
		t.Errorf("kept=%d outside 3-sigma band", kept)
	}
}

func TestRatioIsDeterministicPerTraceID(t *testing.T) {
	s := RatioSampler{Ratio: 0.5}
	a := s.ShouldKeep("trace-abc")
	b := s.ShouldKeep("trace-abc")
	if a != b {
		t.Errorf("not deterministic")
	}
}

func TestPrioritySamplerKeepsHighPriorityRegardlessOfRatio(t *testing.T) {
	s := &PrioritySampler{BackgroundRatio: 0.0}
	s.MarkHighPriority("vip-trace")
	if !s.ShouldKeep("vip-trace") {
		t.Errorf("vip trace must be kept")
	}
	if s.ShouldKeep("regular-trace") {
		t.Errorf("regular trace should drop at ratio 0")
	}
}

func TestAlwaysSamplerKeepsEverything(t *testing.T) {
	s := AlwaysSampler{}
	if !s.ShouldKeep("anything") {
		t.Errorf("AlwaysSampler should keep everything")
	}
}
