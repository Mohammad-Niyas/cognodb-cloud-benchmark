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

	latenciesChan := make(chan time.Duration, 50000)
	stopChan := make(chan struct{})

	startTime := time.Now()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

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
						select {
						case latenciesChan <- dur:
						default:
						}
					}
				}
			}
		}(w)
	}

	time.Sleep(duration)
	close(stopChan)
	wg.Wait()
	close(latenciesChan)

	elapsedSec := time.Since(startTime).Seconds()

	var durations []time.Duration
	for d := range latenciesChan {
		durations = append(durations, d)
	}

	total := atomic.LoadInt64(&totalQueries)
	errs := atomic.LoadInt64(&errorCount)
	qps := float64(total) / elapsedSec

	latResult := stats.CalculatePercentiles(durations, int(errs), int(total))

	report := &ConcurrencyReport{
		EngineName:    engine.Name(),
		WorkerCount:   numWorkers,
		DurationSec:   elapsedSec,
		TotalQueries:  total,
		SustainedQPS:  qps,
		LatencyResult: latResult,
	}

	fmt.Printf("   🚀 %-12s | Workers: %2d | QPS: %7.1f | p50: %6.2fms | p95: %6.2fms | Errors: %d (%3.1f%%)\n",
		engine.Name(), numWorkers, qps, latResult.P50Ms, latResult.P95Ms, errs, latResult.ErrorRate)

	return report, nil
}