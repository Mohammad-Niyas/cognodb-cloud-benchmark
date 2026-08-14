package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/config"
	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/dataset"
	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/driver"
	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/runner"
)

func main() {
	dbFlag := flag.String("db", "all", "Database target: cognodb, neo4j, memgraph, falkordb, arango, or all")
	skipLoad := flag.Bool("skip-load", false, "Skip data loading if already ingested")
	batchSize := flag.Int("batch-size", 5000, "Batch size for data ingestion")
	flag.Parse()

	fmt.Println("   CLOUD GRAPH DATABASE BENCHMARK         ")

	// 1. Config & Dataset
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[FATAL] Config error: %v", err)
	}

	data, err := dataset.LoadCanonical("data/nodes.csv", "data/relationships.csv")
	if err != nil {
		log.Fatalf("[FATAL] Dataset error: %v", err)
	}
	fmt.Printf("[Data] Loaded %d nodes and %d relationships from CSV.\n", len(data.Users), len(data.Relationships))

	// 2. Factory Registry
	engines := driver.BuildEngines(cfg, strings.ToLower(*dbFlag))
	if len(engines) == 0 {
		log.Fatalf("No valid database selected or configured. Check .env and --db flag.")
	}

	// 3. Execute Suite & Export
	results := runner.ExecuteSuite(context.Background(), engines, data, cfg, *skipLoad, *batchSize)
	_ = runner.SaveJSONReport("results/bench_results.json", results)

	fmt.Println("ALL BENCHMARKS COMPLETED!")
	fmt.Println(" Full JSON report written to: results/bench_results.json")
}
