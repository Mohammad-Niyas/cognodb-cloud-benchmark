package runner

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/driver"
	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/stats"
)

// WorkloadReport stores latency percentiles
type WorkloadReport struct {
	EngineName     string              `json:"engine_name"`
	OneHop         stats.LatencyResult `json:"one_hop"`
	TwoHop         stats.LatencyResult `json:"two_hop"`
	ThreeHop       stats.LatencyResult `json:"three_hop"`
	Lookup         stats.LatencyResult `json:"point_lookup"`
	FilteredLookup stats.LatencyResult `json:"filtered_lookup"`
	Aggregate      stats.LatencyResult `json:"aggregation"`
}

// RunAllWorkloads executes the full suite of 100-iteration read benchmarks
func RunAllWorkloads(ctx context.Context, engine driver.GraphEngine, iterations int, sampleNodeIDs []int64) (*WorkloadReport, error) {
	fmt.Printf("EXECUTING READ BENCHMARKS: %s (%d runs each)\n", engine.Name(), iterations)
	fmt.Println()

	rng := rand.New(rand.NewSource(42))
	getRandomNodeID := func() int64 {
		if len(sampleNodeIDs) == 0 {
			return 1
		}
		return sampleNodeIDs[rng.Intn(len(sampleNodeIDs))]
	}

	report := &WorkloadReport{
		EngineName: engine.Name(),
	}

	// 1. 1-Hop Traversal Benchmark
	fmt.Printf("[%s] Running 1-Hop Traversals...\n", engine.Name())
	report.OneHop = measureWorkload(iterations, func() error {
		nodeID := getRandomNodeID()
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, err := engine.OneHop(timeoutCtx, nodeID)
		return err
	})
	printMetricSummary("1-Hop Traversal", report.OneHop)

	// 2. 2-Hop Traversal Benchmark
	fmt.Printf("[%s] Running 2-Hop Traversals...\n", engine.Name())
	report.TwoHop = measureWorkload(iterations, func() error {
		nodeID := getRandomNodeID()
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, err := engine.TwoHop(timeoutCtx, nodeID)
		return err
	})
	printMetricSummary("2-Hop Traversal", report.TwoHop)

	// 3. 3-Hop Traversal Benchmark (Safety limit 1000)
	fmt.Printf("[%s] Running 3-Hop Traversals (Limit 1000)...\n", engine.Name())
	report.ThreeHop = measureWorkload(iterations, func() error {
		nodeID := getRandomNodeID()
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, err := engine.ThreeHop(timeoutCtx, nodeID, 1000)
		return err
	})
	printMetricSummary("3-Hop Traversal", report.ThreeHop)

	// 4. Point Lookup Benchmark
	fmt.Printf("[%s] Running Point Lookups by User ID...\n", engine.Name())
	report.Lookup = measureWorkload(iterations, func() error {
		nodeID := getRandomNodeID()
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, err := engine.PointLookup(timeoutCtx, nodeID)
		return err
	})
	printMetricSummary("Point Lookup", report.Lookup)

	// 5. Aggregation Benchmark
	fmt.Printf("[%s] Running Aggregations...\n", engine.Name())
	report.Aggregate = measureWorkload(iterations, func() error {
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, err := engine.Aggregation(timeoutCtx)
		return err
	})
	printMetricSummary("Aggregation", report.Aggregate)

	// 6. Filtered Property Lookup Benchmark (Age filter)
	fmt.Printf("[%s] Running Filtered Lookups (WHERE age = 25)...\n", engine.Name())
	report.FilteredLookup = measureWorkload(iterations, func() error {
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, err := engine.FilteredLookup(timeoutCtx, 25)
		return err
	})
	printMetricSummary("Filtered Lookup", report.FilteredLookup)

	return report, nil
}

func measureWorkload(iterations int, queryFn func() error) stats.LatencyResult {
	durations := make([]time.Duration, 0, iterations)
	errorCount := 0

	for i := 0; i < iterations; i++ {
		start := time.Now()
		err := queryFn()
		dur := time.Since(start)

		if err != nil {
			errorCount++
		} else {
			durations = append(durations, dur)
		}
	}

	return stats.CalculatePercentiles(durations, errorCount, iterations)
}

func printMetricSummary(name string, res stats.LatencyResult) {
	fmt.Printf("   📊 %-16s | p50: %6.2fms | p95: %6.2fms | p99: %6.2fms | Min: %6.2fms | Max: %6.2fms | Errors: %d\n",
		name, res.P50Ms, res.P95Ms, res.P99Ms, res.MinMs, res.MaxMs, res.ErrorCount)
}