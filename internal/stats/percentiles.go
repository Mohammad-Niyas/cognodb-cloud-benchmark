package stats

import (
	"math"
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

// CalculatePercentiles computes latency percentiles using the standard NIST/Nearest-Rank method.
func CalculatePercentiles(durations []time.Duration, errorCount int, totalRuns int) LatencyResult {
	if len(durations) == 0 {
		errRate := 0.0
		if totalRuns > 0 {
			errRate = (float64(errorCount) / float64(totalRuns)) * 100.0
		}
		return LatencyResult{
			Count:      0,
			ErrorCount: errorCount,
			ErrorRate:  errRate,
		}
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	n := len(durations)
	var total time.Duration
	for _, d := range durations {
		total += d
	}

	toMs := func(d time.Duration) float64 {
		return float64(d.Nanoseconds()) / 1e6
	}

	// Nearest-rank method: rank = ceil(p * n)
	getPercentile := func(p float64) float64 {
		if n == 1 {
			return toMs(durations[0])
		}
		rank := int(math.Ceil(p * float64(n)))
		if rank < 1 {
			rank = 1
		}
		if rank > n {
			rank = n
		}
		return toMs(durations[rank-1])
	}

	errRate := 0.0
	if totalRuns > 0 {
		errRate = (float64(errorCount) / float64(totalRuns)) * 100.0
	}

	return LatencyResult{
		Count:      n,
		MinMs:      toMs(durations[0]),
		P50Ms:      getPercentile(0.50),
		P95Ms:      getPercentile(0.95),
		P99Ms:      getPercentile(0.99),
		MaxMs:      toMs(durations[n-1]),
		AvgMs:      toMs(total / time.Duration(n)),
		ErrorCount: errorCount,
		ErrorRate:  errRate,
	}
}
