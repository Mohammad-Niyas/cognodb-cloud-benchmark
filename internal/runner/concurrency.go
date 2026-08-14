package runner

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/driver"
	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/stats"
)

type ConcurrencyReport struct {
	EngineName    string              `json:"engine_name"`
	WorkerCount   int                 `json:"worker_count"`
	DurationSec   float64             `json:"duration_sec"`
	TotalQueries  int64               `json:"total_queries"`
	SustainedQPS  float64             `json:"sustained_qps"`
	AttemptedQPS  float64             `json:"attempted_qps"`
	LatencyResult stats.LatencyResult `json:"latency_result"`
}

// RunConcurrencySweep executes a multi-client concurrent mixed read/write stress test
func RunConcurrencySweep(ctx context.Context, engine driver.GraphEngine, numWorkers int, duration time.Duration, sampleNodeIDs []int64) (*ConcurrencyReport, error) {
	fmt.Printf("\n==========================================================\n")
	fmt.Printf("  CONCURRENCY STRESS TEST: %s (%d Workers, %v Duration)\n", engine.Name(), numWorkers, duration)
	fmt.Printf("==========================================================\n")

	if len(sampleNodeIDs) == 0 {
		sampleNodeIDs = []int64{1}
	}

	var wg sync.WaitGroup
	var totalQueries int64
	var errorCount int64

	var mu sync.Mutex
	var durations []time.Duration

	stopChan := make(chan struct{})
	startTime := time.Now()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// Deterministic seed per worker + concurrency level
			seed := time.Now().UnixNano() + int64(workerID*1000) + int64(numWorkers)
			rng := rand.New(rand.NewSource(seed))

			for {
				select {
				case <-stopChan:
					return
				case <-ctx.Done():
					return
				default:
					nodeID := sampleNodeIDs[rng.Intn(len(sampleNodeIDs))]
					queryStart := time.Now()

					var err error
					timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
					if rng.Float64() < 0.80 {
						_, err = engine.OneHop(timeoutCtx, nodeID)
					} else {
						toID := sampleNodeIDs[rng.Intn(len(sampleNodeIDs))]
						err = engine.WriteRelationship(timeoutCtx, nodeID, toID)
					}
					cancel()

					dur := time.Since(queryStart)
					atomic.AddInt64(&totalQueries, 1)

					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						mu.Lock()
						durations = append(durations, dur)
						mu.Unlock()
					}
				}
			}
		}(w)
	}

	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	elapsedSec := time.Since(startTime).Seconds()
	total := atomic.LoadInt64(&totalQueries)
	errs := atomic.LoadInt64(&errorCount)
	successfulOps := int64(len(durations))

	sustainedQPS := float64(successfulOps) / elapsedSec
	attemptedQPS := float64(total) / elapsedSec

	latResult := stats.CalculatePercentiles(durations, int(errs), int(total))

	report := &ConcurrencyReport{
		EngineName:    engine.Name(),
		WorkerCount:   numWorkers,
		DurationSec:   elapsedSec,
		TotalQueries:  total,
		SustainedQPS:  sustainedQPS,
		AttemptedQPS:  attemptedQPS,
		LatencyResult: latResult,
	}

	fmt.Printf("   🚀 %-12s | Workers: %2d | Sustained QPS: %7.1f | Attempt QPS: %7.1f | p50: %6.2fms | p95: %6.2fms | Errors: %d (%3.1f%%)\n",
		engine.Name(), numWorkers, sustainedQPS, attemptedQPS, latResult.P50Ms, latResult.P95Ms, errs, latResult.ErrorRate)

	return report, nil
}
