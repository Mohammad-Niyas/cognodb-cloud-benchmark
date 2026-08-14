package stats

import (
	"sort"
	"time"
)

type LatencyResult struct {
	Count      int     `json:"count"`
	MinMs      float64 `json:"min_ms"`
	P50Ms      float64 `json:"p50_ms"`
	P95Ms      float64 `json:"p95_ms"`
	P99Ms      float64 `json:"p99_ms"`
	MaxMs      float64 `json:"max_ms"`
	AvgMs      float64 `json:"avg_ms"`
	ErrorCount int     `json:"error_count"`
	ErrorRate  float64 `json:"error_rate_pct"`
}

// CalculatePercentiles
func CalculatePercentiles(durations []time.Duration, errorCount int, totalRuns int) LatencyResult {
	if len(durations) == 0 {
		return LatencyResult{
			Count:      0,
			ErrorCount: errorCount,
			ErrorRate:  100.0,
		}
	}

	//  Sort durations from fastest to slowest
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	n := len(durations)
	var total time.Duration
	for _, d := range durations {
		total += d
	}

	// convert time.Duration to millisecond float
	toMs := func(d time.Duration) float64 {
		return float64(d.Nanoseconds()) / 1e6
	}

	p50Idx := int(float64(n) * 0.50)
	p95Idx := int(float64(n) * 0.95)
	p99Idx := int(float64(n) * 0.99)

	if p50Idx >= n {
		p50Idx = n - 1
	}
	if p95Idx >= n {
		p95Idx = n - 1
	}
	if p99Idx >= n {
		p99Idx = n - 1
	}

	var errRate float64
	if totalRuns > 0 {
		errRate = (float64(errorCount) / float64(totalRuns)) * 100.0
	}

	return LatencyResult{
		Count:      n,
		MinMs:      toMs(durations[0]),
		P50Ms:      toMs(durations[p50Idx]),
		P95Ms:      toMs(durations[p95Idx]),
		P99Ms:      toMs(durations[p99Idx]),
		MaxMs:      toMs(durations[n-1]),
		AvgMs:      toMs(total / time.Duration(n)),
		ErrorCount: errorCount,
		ErrorRate:  errRate,
	}
}
