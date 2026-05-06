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
