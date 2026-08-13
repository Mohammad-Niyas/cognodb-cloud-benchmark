package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Mohammad-Niyas/cognodb-cloud-benchmark/internal/dataset"
)

const (
	RelPath      = "data/soc-pokec-relationships.txt.gz"
	ProfilePath  = "data/soc-pokec-profiles.txt.gz"
	NodesCSV     = "data/nodes.csv"
	RelsCSV      = "data/relationships.csv"
	ManifestJSON = "data/manifest.json"
	TargetNodes  = 30000
	StartNodeID  = 1
)

func main() {
	fmt.Println("==========================================================")
	fmt.Println("  SNAP soc-Pokec Deterministic BFS Snowball Sampler       ")
	fmt.Println("==========================================================")
	fmt.Printf("Target Node Count: %d\n", TargetNodes)
	fmt.Printf("BFS Start Node ID: %d\n", StartNodeID)
	fmt.Println("----------------------------------------------------------")

	startTime := time.Now()

	graph, err := dataset.SampleBFSSubgraph(RelPath, ProfilePath, TargetNodes, StartNodeID)
	if err != nil {
		log.Fatalf("[FATAL] Sampling failed: %v", err)
	}

	// 2. Write CSV Files
	if err := dataset.WriteCanonicalCSVs(graph, NodesCSV, RelsCSV); err != nil {
		log.Fatalf("[FATAL] Failed to write CSVs: %v", err)
	}

	// 3. Generate SHA-256 Manifest
	if err := dataset.GenerateManifest(NodesCSV, RelsCSV, ManifestJSON, len(graph.Users), len(graph.Relationships), StartNodeID); err != nil {
		log.Fatalf("[FATAL] Failed to generate manifest: %v", err)
	}

	duration := time.Since(startTime)

	fmt.Println("==========================================================")
	fmt.Printf("✅ SAMPLING PIPELINE COMPLETE in %v\n", duration.Round(time.Millisecond))
	fmt.Printf("  • Total Nodes (Users):          %d\n", len(graph.Users))
	fmt.Printf("  • Total Relationships (Edges):  %d\n", len(graph.Relationships))
	fmt.Printf("  • Generated Files:              %s, %s, %s\n", NodesCSV, RelsCSV, ManifestJSON)
	fmt.Println("==========================================================")
}