package dataset

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

// Manifest represents the metadata and checksums of the canonical dataset
type Manifest struct {
	GeneratedAt        string `json:"generated_at"`
	SamplingMethod     string `json:"sampling_method"`
	StartNodeID        int64  `json:"start_node_id"`
	TotalNodes         int    `json:"total_nodes"`
	TotalRelationships int    `json:"total_relationships"`
	NodesFileSHA256    string `json:"nodes_file_sha256"`
	RelsFileSHA256     string `json:"relationships_file_sha256"`
}

// WriteCanonicalCSVs writes the sampled graph into standard CSV files
func WriteCanonicalCSVs(graph *SampledGraph, nodesPath, relsPath string) error {
	fmt.Printf("[Writer] Writing %d nodes to %s...\n", len(graph.Users), nodesPath)

	nodeFile, err := os.Create(nodesPath)
	if err != nil {
		return fmt.Errorf("failed to create nodes file: %w", err)
	}
	defer nodeFile.Close()

	nodeWriter := csv.NewWriter(nodeFile)
	// Header
	if err := nodeWriter.Write([]string{"id", "age", "gender"}); err != nil {
		return err
	}

	for _, u := range graph.Users {
		record := []string{
			strconv.FormatInt(u.ID, 10),
			strconv.Itoa(u.Age),
			u.Gender,
		}
		if err := nodeWriter.Write(record); err != nil {
			return err
		}
	}
	nodeWriter.Flush()

	fmt.Printf("[Writer] Writing %d relationships to %s...\n", len(graph.Relationships), relsPath)

	relFile, err := os.Create(relsPath)
	if err != nil {
		return fmt.Errorf("failed to create relationships file: %w", err)
	}
	defer relFile.Close()

	relWriter := csv.NewWriter(relFile)
	if err := relWriter.Write([]string{"from_user", "to_user"}); err != nil {
		return err
	}

	for _, r := range graph.Relationships {
		record := []string{
			strconv.FormatInt(r.FromUser, 10),
			strconv.FormatInt(r.ToUser, 10),
		}
		if err := relWriter.Write(record); err != nil {
			return err
		}
	}
	relWriter.Flush()

	return nil
}

func ComputeSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GenerateManifest writes data/manifest.json containing dataset stats and checksums
func GenerateManifest(nodesPath, relsPath, manifestPath string, nodeCount, relCount int, startNodeID int64) error {
	fmt.Printf("[Manifest] Computing SHA-256 checksums...\n")

	nodeHash, err := ComputeSHA256(nodesPath)
	if err != nil {
		return fmt.Errorf("failed to hash nodes file: %w", err)
	}

	relHash, err := ComputeSHA256(relsPath)
	if err != nil {
		return fmt.Errorf("failed to hash relationships file: %w", err)
	}

	manifest := Manifest{
		GeneratedAt:        time.Now().UTC().Format(time.RFC3339),
		SamplingMethod:     "BFS Snowball Sampling (Giant Component)",
		StartNodeID:        startNodeID,
		TotalNodes:         nodeCount,
		TotalRelationships: relCount,
		NodesFileSHA256:    nodeHash,
		RelsFileSHA256:     relHash,
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest json: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	fmt.Printf("[Manifest] Successfully generated %s\n", manifestPath)
	return nil
}
