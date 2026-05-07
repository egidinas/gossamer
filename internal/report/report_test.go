package report

import (
	"testing"

	"github.com/egidinas/gossamer/internal/environmentalsim"
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
	for _, campaignID := range []string{"flatsat_derisking", "thermal_acceptance_fat", "tvac_qualification", "integrated_system_fat"} {
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
