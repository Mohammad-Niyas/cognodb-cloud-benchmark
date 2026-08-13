package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/driver"
)

// RunWarmup executes 10 iterations of each query
func RunWarmup(ctx context.Context, engine driver.GraphEngine, warmupRuns int, sampleNodeIDs []int64) {
	fmt.Printf("[%s] Warming up engine caches (%d iterations)...\n", engine.Name(), warmupRuns)

	if len(sampleNodeIDs) == 0 {
		sampleNodeIDs = []int64{1}
	}

	for i := 0; i < warmupRuns; i++ {
		nodeID := sampleNodeIDs[i%len(sampleNodeIDs)]
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

		// Execute queries without recording metrics
		_, _ = engine.OneHop(timeoutCtx, nodeID)
		_, _ = engine.TwoHop(timeoutCtx, nodeID)
		_, _ = engine.PointLookup(timeoutCtx, nodeID)

		cancel()
	}

	fmt.Printf("[%s] ✅ Warm-up complete. Starting real measurements.\n", engine.Name())
}