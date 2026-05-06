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
