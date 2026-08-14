package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/driver"
)

// RunWarmup executes iterations of ALL workloads to warm up database memory buffers & connection pools
func RunWarmup(ctx context.Context, engine driver.GraphEngine, warmupRuns int, sampleNodeIDs []int64) {
	fmt.Printf("[%s] Warming up engine caches (%d iterations across all workloads)...\n", engine.Name(), warmupRuns)

	if len(sampleNodeIDs) == 0 {
		sampleNodeIDs = []int64{1}
	}

	for i := 0; i < warmupRuns; i++ {
		nodeID := sampleNodeIDs[i%len(sampleNodeIDs)]
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

		// Execute ALL workloads without recording metrics to prime memory & connection pools
		_, _ = engine.OneHop(timeoutCtx, nodeID)
		_, _ = engine.TwoHop(timeoutCtx, nodeID)
		_, _ = engine.ThreeHop(timeoutCtx, nodeID, 1000)
		_, _ = engine.PointLookup(timeoutCtx, nodeID)
		_, _ = engine.FilteredLookup(timeoutCtx, 25)
		_, _ = engine.Aggregation(timeoutCtx)

		cancel()
	}

	fmt.Printf("[%s] ✅ Warm-up complete for all workloads. Starting real measurements.\n", engine.Name())
}
