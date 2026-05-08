package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egidinas/gossamer/internal/arrowtelemetry"
	"github.com/egidinas/gossamer/internal/contracts"
	"github.com/egidinas/gossamer/internal/tilebundle"
)

func main() {
	var (
		root        = flag.String("root", ".", "repository or fixture root")
		out         = flag.String("out", "fixtures/public_tiles", "tile bundle output root")
		dataVersion = flag.String("data-version", "", "immutable data bundle version (default: content-addressed hash)")
		current     = flag.Bool("current", true, "also refresh fixtures/public_tiles/current")
		campaigns   = flag.String("campaigns", "thermal_acceptance_fat,tvac_qualification,command_center_fat", "comma-separated campaign IDs")
		levels      = flag.String("levels", "minute", "comma-separated tile levels to materialize")
	)
	flag.Parse()

	models, err := loadModels(*root, *campaigns)
	if err != nil {
		log.Fatal(err)
	}

	version := *dataVersion
	if version == "" {
		version = contentAddressedVersion(models)
	}

	versionedOut := filepath.Join(*root, *out, version)
	if err := os.RemoveAll(versionedOut); err != nil {
		log.Fatal(err)
	}
	manifest, err := tilebundle.WriteBundle(models, tilebundle.Options{
		DataVersion:  version,
		OutputDir:    versionedOut,
		TileBasePath: "/data/current",
		Levels:       splitCSV(*levels),
		Now:          time.Now().UTC(),
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := copyTelemetry(*root, versionedOut, manifest.Campaigns); err != nil {
		log.Fatal(err)
	}
	if err := copyStaticFixtures(*root, versionedOut, []string{"command_center_fat.json"}); err != nil {
		log.Fatal(err)
	}
	if *current {
		currentOut := filepath.Join(*root, *out, "current")
		if err := replaceDir(currentOut, versionedOut); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("wrote tile bundle %s with %d campaigns to %s\n", manifest.DataVersion, len(manifest.Campaigns), versionedOut)
}

// contentAddressedVersion produces a stable 8-hex-char version derived from
// campaign IDs, simulation model versions, and the Arrow schema version.
// Identical inputs → identical version → no noisy git diffs on every rebuild.
func contentAddressedVersion(models []contracts.GraphModel) string {
	h := sha256.New()
	h.Write([]byte(arrowtelemetry.SchemaName))
	for _, model := range models {
		h.Write([]byte(model.CampaignID))
		if model.SimulationProvenance != nil {
			h.Write([]byte(model.SimulationProvenance.ModelVersion))
		}
	}
	return "v" + hex.EncodeToString(h.Sum(nil))[:8]
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func loadModels(root, csv string) ([]contracts.GraphModel, error) {
	var models []contracts.GraphModel
	for _, id := range strings.Split(csv, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		model, err := tilebundle.LoadGraphModel(filepath.Join(root, "fixtures", "public", "graph_models", id+".json"))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", id, err)
		}
		models = append(models, model)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("no campaigns selected")
	}
	return models, nil
}

func copyTelemetry(root string, out string, campaigns []contracts.TileCampaignManifest) error {
	for _, campaign := range campaigns {
		src := filepath.Join(root, "fixtures", "public", "telemetry", campaign.CampaignID+".arrow")
		dst := filepath.Join(out, "campaigns", campaign.CampaignID, "telemetry.arrow")
		if err := copyFile(src+".gz", dst+".gz"); err != nil {
			return err
		}
	}
	return nil
}

func copyStaticFixtures(root string, out string, names []string) error {
	for _, name := range names {
		src := filepath.Join(root, "fixtures", "public", name)
		dst := filepath.Join(out, name)
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("%s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func replaceDir(dst, src string) error {
	tmp := dst + ".tmp"
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	if err := copyDir(tmp, src); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	if err := os.RemoveAll(dst); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

func copyDir(dst, src string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
