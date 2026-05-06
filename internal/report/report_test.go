package report

import "testing"

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
