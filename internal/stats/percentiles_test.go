package stats

import (
	"testing"
	"time"
)

func TestCalculatePercentiles(t *testing.T) {
	durations := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		durations[i] = time.Duration(i+1) * time.Millisecond
	}

	res := CalculatePercentiles(durations, 0, 100)

	if res.Count != 100 {
		t.Errorf("expected count 100, got %d", res.Count)
	}
	if res.MinMs != 1.0 {
		t.Errorf("expected min 1.0ms, got %f", res.MinMs)
	}
	if res.P50Ms != 50.0 {
		t.Errorf("expected p50 50.0ms, got %f", res.P50Ms)
	}
	if res.P95Ms != 95.0 {
		t.Errorf("expected p95 95.0ms, got %f", res.P95Ms)
	}
	if res.P99Ms != 99.0 {
		t.Errorf("expected p99 99.0ms, got %f", res.P99Ms)
	}
	if res.MaxMs != 100.0 {
		t.Errorf("expected max 100.0ms, got %f", res.MaxMs)
	}
}
