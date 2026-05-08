package evaluator

import "testing"

func TestThermalFATRequirementsPass(t *testing.T) {
	campaign, reqs, err := EvaluateSyntheticCampaign("thermal_acceptance_fat")
	if err != nil {
		t.Fatal(err)
	}
	if campaign.ID != "thermal_acceptance_fat" {
		t.Fatalf("campaign id = %q", campaign.ID)
	}
	for _, req := range reqs {
		if req.Result != "pass" {
			t.Fatalf("%s result = %s, want pass", req.ID, req.Result)
		}
	}
}

func TestTVACQualificationHasInconclusiveReviewItems(t *testing.T) {
	_, reqs, err := EvaluateSyntheticCampaign("tvac_qualification")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, req := range reqs {
		got[req.ID] = req.Result
	}
	if got["REQ-DATA-QUALITY"] != "inconclusive" {
		t.Fatalf("data quality = %s, want inconclusive", got["REQ-DATA-QUALITY"])
	}
	if got["REQ-ANOMALY-REVIEW"] != "inconclusive" {
		t.Fatalf("anomaly review = %s, want inconclusive", got["REQ-ANOMALY-REVIEW"])
	}
}

func TestFlatSatRequirementsPass(t *testing.T) {
	_, reqs, err := EvaluateSyntheticCampaign("flatsat_derisking")
	if err != nil {
		t.Fatal(err)
	}
	for _, req := range reqs {
		if req.Result != "pass" {
			t.Fatalf("%s result = %s, want pass", req.ID, req.Result)
		}
	}
}

func TestIntegratedSystemFATRequirements(t *testing.T) {
	_, reqs, err := EvaluateSyntheticCampaign("integrated_system_fat")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, req := range reqs {
		got[req.ID] = req.Result
	}
	// The integrated FAT has a dispositioned EPS freshness degradation window.
	// REQ-DATA-QUALITY evaluates to fail because degraded samples are present.
	if got["REQ-DATA-QUALITY"] != "fail" {
		t.Fatalf("REQ-DATA-QUALITY = %s, want fail (degraded window present)", got["REQ-DATA-QUALITY"])
	}
	// All other requirements should pass.
	for _, req := range reqs {
		if req.ID == "REQ-DATA-QUALITY" {
			continue
		}
		if req.Result != "pass" {
			t.Fatalf("%s result = %s, want pass", req.ID, req.Result)
		}
	}
}

func TestRequirementsCarryExpression(t *testing.T) {
	_, reqs, err := EvaluateSyntheticCampaign("thermal_acceptance_fat")
	if err != nil {
		t.Fatal(err)
	}
	for _, req := range reqs {
		if req.Expression == "" {
			t.Fatalf("%s has no expression", req.ID)
		}
	}
}
