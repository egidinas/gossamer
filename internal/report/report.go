package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/egidinas/gossamer/internal/contracts"
	"github.com/egidinas/gossamer/internal/evaluator"
	"github.com/egidinas/gossamer/internal/synthetic"
)

func Build(campaignID string) (contracts.EvidenceReport, error) {
	set := synthetic.Build()
	campaign, ok := set.Campaigns[campaignID]
	if !ok {
		return contracts.EvidenceReport{}, fmt.Errorf("unknown campaign %q", campaignID)
	}
	reqs := evaluator.Evaluate(evaluator.EvaluationInput{Campaign: campaign, Telemetry: set.Telemetry[campaignID]})
	result := "pass"
	for _, req := range reqs {
		if req.Result == "fail" {
			result = "fail"
		}
		if req.Result == "inconclusive" && result != "fail" {
			result = "inconclusive"
		}
	}
	return contracts.EvidenceReport{
		Envelope:          contracts.NewEnvelope(synthetic.FixedTime),
		CampaignID:        campaignID,
		Summary:           fmt.Sprintf("%s evidence package for AuroraSat-1.", campaign.Name),
		Result:            result,
		Requirements:      reqs,
		Sources:           set.SourceCatalogue.Sources,
		GraphEvidence:     []string{fmt.Sprintf("fixtures/public/graph_models/%s.json", campaignID), fmt.Sprintf("fixtures/public/telemetry/%s.jsonl", campaignID)},
		Anomalies:         campaign.Anomalies,
		Reproducibility:   []string{"go run ./cmd/gossamer-fixtures", fmt.Sprintf("go run ./cmd/gossamer-report --campaign %s", campaignID)},
		SyntheticDataNote: "This report is generated from deterministic synthetic data for public demonstration only.",
	}, nil
}

func Write(root, campaignID string) error {
	report, err := Build(campaignID)
	if err != nil {
		return err
	}
	dir := filepath.Join(root, "fixtures", "public", "reports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, campaignID+"_report.json"), append(data, '\n'), 0o644)
}
