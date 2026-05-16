package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egidinas/gossamer/internal/environmentalsim"
	"github.com/egidinas/gossamer/internal/synthetic"
)

func TestBuildThermalFATReport(t *testing.T) {
	report, err := Build("thermal_acceptance_fat")
	if err != nil {
		t.Fatal(err)
	}
	if report.Result != "pass" {
		t.Fatalf("result = %s, want pass", report.Result)
	}
	if report.SyntheticDataNote == "" {
		t.Fatal("expected synthetic data note")
	}
}

func TestBuildTVACReportCarriesAnomaly(t *testing.T) {
	report, err := Build("tvac_qualification")
	if err != nil {
		t.Fatal(err)
	}
	if report.Result != "inconclusive" {
		t.Fatalf("result = %s, want inconclusive", report.Result)
	}
	if len(report.Anomalies) == 0 {
		t.Fatal("expected anomaly")
	}
}

func TestBuildThermalReportsCarryProgramAndPhaseEvidence(t *testing.T) {
	cases := []struct {
		campaignID string
		wantCycles int
	}{
		{campaignID: "thermal_acceptance_fat", wantCycles: 4},
		{campaignID: "tvac_qualification", wantCycles: 8},
	}
	for _, tc := range cases {
		report, err := Build(tc.campaignID)
		if err != nil {
			t.Fatal(err)
		}
		if report.ThermalProgram == nil {
			t.Fatalf("%s missing thermal program in report", tc.campaignID)
		}
		if report.SimulationProvenance == nil {
			t.Fatalf("%s missing simulation provenance in report", tc.campaignID)
		}
		if report.SimulationProvenance.Model != environmentalsim.ModelName {
			t.Fatalf("%s simulation model = %q, want %s", tc.campaignID, report.SimulationProvenance.Model, environmentalsim.ModelName)
		}
		if got := report.ThermalProgram.CycleCount; got != tc.wantCycles {
			t.Fatalf("%s report cycle count = %d, want %d", tc.campaignID, got, tc.wantCycles)
		}
		if len(report.ThermalProgram.DwellWindows) < tc.wantCycles*2 {
			t.Fatalf("%s report dwell windows = %d, want hot/cold windows per cycle", tc.campaignID, len(report.ThermalProgram.DwellWindows))
		}
		if len(report.ThermalProgram.FunctionalGates) < 4 {
			t.Fatalf("%s report functional gates = %d, want pre/cold/hot/post", tc.campaignID, len(report.ThermalProgram.FunctionalGates))
		}
		requirements := map[string]string{}
		for _, req := range report.Requirements {
			requirements[req.ID] = req.Rationale
		}
		for _, reqID := range []string{"REQ-CYCLE-COUNT", "REQ-HOT-TARGET", "REQ-COLD-TARGET", "REQ-STABILITY", "REQ-DWELL", "REQ-FUNC-GATE-PRE", "REQ-FUNC-GATE-DURING", "REQ-FUNC-GATE-POST", "REQ-DATA-QUALITY", "REQ-ANOMALY-REVIEW"} {
			if requirements[reqID] == "" {
				t.Fatalf("%s requirement %s missing rationale", tc.campaignID, reqID)
			}
		}
	}
}

func TestBuildReportForEveryCampaign(t *testing.T) {
	for _, campaignID := range CampaignIDs() {
		report, err := Build(campaignID)
		if err != nil {
			t.Fatalf("%s: %v", campaignID, err)
		}
		if report.CampaignID != campaignID {
			t.Fatalf("campaign id = %s, want %s", report.CampaignID, campaignID)
		}
		if len(report.GraphEvidence) < 2 {
			t.Fatalf("%s missing graph evidence", campaignID)
		}
		if report.Requirements == nil {
			t.Fatalf("%s requirements is nil", campaignID)
		}
		if report.Sources == nil {
			t.Fatalf("%s sources is nil", campaignID)
		}
		if report.GraphEvidence == nil {
			t.Fatalf("%s graph evidence is nil", campaignID)
		}
		if report.Anomalies == nil {
			t.Fatalf("%s anomalies is nil", campaignID)
		}
		if report.Reproducibility == nil {
			t.Fatalf("%s reproducibility is nil", campaignID)
		}
	}
}

func TestWriteCreatesReportNamesUsedByGraphEvidence(t *testing.T) {
	dir := t.TempDir()
	for _, campaignID := range CampaignIDs() {
		if err := Write(dir, campaignID); err != nil {
			t.Fatalf("%s: %v", campaignID, err)
		}
		for _, name := range []string{campaignID + ".json", campaignID + "_report.json"} {
			path := filepath.Join(dir, "fixtures", "public", "reports", name)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("%s missing: %v", path, err)
			}
		}
	}
}

func TestGeneratedGraphEvidenceReportRefsResolve(t *testing.T) {
	dir := t.TempDir()
	if err := synthetic.WritePublicFixtures(dir); err != nil {
		t.Fatal(err)
	}
	for _, campaignID := range CampaignIDs() {
		if err := Write(dir, campaignID); err != nil {
			t.Fatalf("%s: %v", campaignID, err)
		}
	}

	graphDir := filepath.Join(dir, "fixtures", "public", "graph_models")
	entries, err := os.ReadDir(graphDir)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(graphDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var document any
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		for _, ref := range evidenceRefs(document) {
			refPath, _, _ := strings.Cut(ref, "#")
			if !strings.HasPrefix(refPath, "reports/") {
				continue
			}
			checked++
			fullPath := filepath.Join(dir, "fixtures", "public", filepath.FromSlash(refPath))
			if _, err := os.Stat(fullPath); err != nil {
				t.Fatalf("%s references missing report %s: %v", entry.Name(), refPath, err)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no report evidence refs checked")
	}
}

func evidenceRefs(value any) []string {
	switch typed := value.(type) {
	case map[string]any:
		refs := []string{}
		for key, child := range typed {
			if key == "evidence_ref" {
				if ref, ok := child.(string); ok && ref != "" {
					refs = append(refs, ref)
				}
				continue
			}
			refs = append(refs, evidenceRefs(child)...)
		}
		return refs
	case []any:
		refs := []string{}
		for _, child := range typed {
			refs = append(refs, evidenceRefs(child)...)
		}
		return refs
	default:
		return nil
	}
}
