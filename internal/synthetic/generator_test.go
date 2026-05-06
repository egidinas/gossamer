package synthetic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egidinas/gossamer/internal/contracts"
)

func TestBuildProducesValidContracts(t *testing.T) {
	set := Build()
	if err := contracts.ValidateManifest(set.Manifest); err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if err := contracts.ValidateSourceCatalogue(set.SourceCatalogue); err != nil {
		t.Fatalf("sources: %v", err)
	}
	for id, campaign := range set.Campaigns {
		if err := contracts.ValidateCampaign(campaign); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if len(set.Telemetry[id]) != 48 {
			t.Fatalf("%s telemetry count = %d, want 48", id, len(set.Telemetry[id]))
		}
	}
}

func TestIntegratedSystemFATIncludesDegradedSourceInterval(t *testing.T) {
	set := Build()
	found := false
	for _, sample := range set.Telemetry["integrated_system_fat"] {
		if sample.Quality == "degraded" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected degraded interval")
	}
}

func TestWritePublicFixturesCreatesDeterministicFiles(t *testing.T) {
	dir := t.TempDir()
	if err := WritePublicFixtures(dir); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "fixtures", "public", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := WritePublicFixtures(dir); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(dir, "fixtures", "public", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("fixture generation changed between runs")
	}
}
