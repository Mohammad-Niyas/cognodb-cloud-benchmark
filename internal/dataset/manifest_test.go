package dataset

import (
	"os"
	"testing"
)

func TestManifestValidation(t *testing.T) {
	manifestPath := "/home/mohammad_niyas/cognodb-cloud-benchmark/data/manifest.json"
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Skip("manifest.json not present in local test environment")
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if manifest.TotalNodes != 30000 {
		t.Errorf("expected 30000 nodes, got %d", manifest.TotalNodes)
	}
	if manifest.TotalRelationships != 393090 {
		t.Errorf("expected 393090 relationships, got %d", manifest.TotalRelationships)
	}
}
