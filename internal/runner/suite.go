package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/config"
	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/dataset"
	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/driver"
)

type FullBenchmarkReport struct {
	Timestamp         string             `json:"timestamp"`
	DatasetSummary    map[string]any     `json:"dataset_summary"`
	ResourceFootprint map[string]any     `json:"resource_footprint"`
	Ingestion         map[string]any     `json:"ingestion_metrics"`
	IndexCreation     map[string]any     `json:"index_creation"`
	Workloads         *WorkloadReport    `json:"read_workloads"`
	Concurrency1      *ConcurrencyReport `json:"concurrency_1_worker"`
	Concurrency10     *ConcurrencyReport `json:"concurrency_10_workers"`
	Concurrency40     *ConcurrencyReport `json:"concurrency_40_workers"`
}

// ExecuteSuite orchestrates the complete benchmark workflow following strict scientific methodology:
// 1. Raw Data Ingestion (isolated timing) -> 2. Count Verification -> 3. Index Creation -> 4. Warm-up -> 5. Read Benchmarks -> 6. Concurrency Stress Sweeps.
func ExecuteSuite(ctx context.Context, engines []driver.GraphEngine, data *dataset.CanonicalData, cfg *config.Config, skipLoad bool, batchSize int) map[string]FullBenchmarkReport {
	allResults := make(map[string]FullBenchmarkReport)

	expectedNodes := int64(len(data.Users))
	expectedRels := int64(len(data.Relationships))

	for _, engine := range engines {
		fmt.Printf("\n==========================================================\n")
		fmt.Printf("  BENCHMARKING TARGET: %s\n", engine.Name())
		fmt.Printf("==========================================================\n")

		report := FullBenchmarkReport{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			DatasetSummary: map[string]any{
				"total_nodes":         expectedNodes,
				"total_relationships": expectedRels,
			},
			Ingestion:     make(map[string]any),
			IndexCreation: make(map[string]any),
		}

		report.ResourceFootprint = map[string]any{
			"status":        "not_observable_via_driver",
			"notes":         "Managed cloud free-tier telemetry is restricted; instance specifications are recorded in README Section 3.",
			"instance_tier": "Cloud Free / Entry Tier",
		}

		if err := engine.Connect(ctx); err != nil {
			fmt.Printf("[%s] ❌ Connection failed: %v (skipping engine)\n", engine.Name(), err)
			continue
		}

		if !skipLoad {
			// Step 1: Raw Ingestion (timed before indexes exist)
			fmt.Printf("[%s] Step 1: Ingesting %d Nodes in batches of %d...\n", engine.Name(), expectedNodes, batchSize)
			startN := time.Now()
			nInserted, err := engine.BulkInsertNodes(ctx, data.Users, batchSize)
			if err != nil {
				fmt.Printf("[%s] ❌ Node ingestion failed: %v (skipping engine)\n", engine.Name(), err)
				_ = engine.Close(ctx)
				continue
			}
			durN := time.Since(startN)
			nPerSec := float64(nInserted) / durN.Seconds()

			fmt.Printf("[%s] Step 1: Ingesting %d Relationships in batches of %d...\n", engine.Name(), expectedRels, batchSize)
			startR := time.Now()
			rInserted, err := engine.BulkInsertRelationships(ctx, data.Relationships, batchSize)
			if err != nil {
				fmt.Printf("[%s] ❌ Relationship ingestion failed: %v (skipping engine)\n", engine.Name(), err)
				_ = engine.Close(ctx)
				continue
			}
			durR := time.Since(startR)
			rPerSec := float64(rInserted) / durR.Seconds()
			totalLoadTime := (durN + durR).Seconds()

			report.Ingestion = map[string]any{
				"nodes_inserted":      nInserted,
				"nodes_load_time_sec": durN.Seconds(),
				"nodes_per_sec":       nPerSec,
				"rels_inserted":       rInserted,
				"rels_load_time_sec":  durR.Seconds(),
				"rels_per_sec":        rPerSec,
				"total_load_time_sec": totalLoadTime,
			}
			fmt.Printf("⏱️ [%s] INGESTION COMPLETE: Nodes: %5.0f/s | Rels: %5.0f/s | Total Time: %5.2fs\n",
				engine.Name(), nPerSec, rPerSec, totalLoadTime)

			// Step 2: Create Index (timed separately)
			fmt.Printf("[%s] Step 2: Creating Index on User(id)...\n", engine.Name())
			idxStart := time.Now()
			if err := engine.CreateIndex(ctx); err != nil {
				fmt.Printf("[%s] ⚠️ Index creation warning: %v\n", engine.Name(), err)
			}
			idxDur := time.Since(idxStart)
			report.IndexCreation = map[string]any{
				"index_time_sec": idxDur.Seconds(),
				"indexed_field":  "User(id)",
			}
		} else {
			fmt.Printf("[%s] ⏩ Skipping ingestion as requested (--skip-load).\n", engine.Name())
		}

		// Step 3: Hard Count Verification
		vNodes, vRels, err := engine.VerifyCounts(ctx)
		if err != nil {
			fmt.Printf("[%s] ⚠️ Count verification error: %v\n", engine.Name(), err)
		} else {
			fmt.Printf("[%s] 🔍 Verification: %d Nodes, %d Relationships\n", engine.Name(), vNodes, vRels)
		}

		// Step 4: Complete Warm-up across ALL 6 workloads
		RunWarmup(ctx, engine, cfg.WarmupIterations, data.UserIDs)

		// Step 5: Read Benchmarks (100 runs each)
		wReport, err := RunAllWorkloads(ctx, engine, cfg.BenchmarkIterations, data.UserIDs)
		if err == nil {
			report.Workloads = wReport
		} else {
			fmt.Printf("[%s] ⚠️ Workload execution error: %v\n", engine.Name(), err)
		}

		// Step 6: Multi-Worker Concurrency Sweeps (1, 10, and 40 workers)
		c1, _ := RunConcurrencySweep(ctx, engine, 1, 15*time.Second, data.UserIDs)
		report.Concurrency1 = c1

		c10, _ := RunConcurrencySweep(ctx, engine, 10, 15*time.Second, data.UserIDs)
		report.Concurrency10 = c10

		c40, _ := RunConcurrencySweep(ctx, engine, 40, 15*time.Second, data.UserIDs)
		report.Concurrency40 = c40

		_ = engine.Close(ctx)
		allResults[engine.Name()] = report
	}

	return allResults
}

func SaveJSONReport(path string, newResults map[string]FullBenchmarkReport) error {
	_ = os.MkdirAll("results", 0755)

	existingMap := make(map[string]FullBenchmarkReport)

	if existingBytes, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(existingBytes, &existingMap)
	}

	for k, v := range newResults {
		existingMap[k] = v
	}

	data, err := json.MarshalIndent(existingMap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
