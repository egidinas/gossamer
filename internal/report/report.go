package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/egidinas/gossamer/internal/evaluator"
	"github.com/egidinas/gossamer/internal/synthetic"
	"github.com/egidinas/signalforge/contracts"
)

func Build(campaignID string) (contracts.EvidenceReport, error) {
	set := synthetic.Build()
	campaign, ok := set.Campaigns[campaignID]
	if !ok {
		if campaignID == synthetic.CommandCenterGraphCampaignID {
			return buildCommandCenterReport(set), nil
		}
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
	anomalies := campaign.Anomalies
	if anomalies == nil {
		anomalies = []contracts.Anomaly{}
	}
	var simulationProvenance *contracts.SimulationProvenance
	if graphModel, ok := set.GraphModels[campaignID]; ok && graphModel.SimulationProvenance != nil {
		provenance := *graphModel.SimulationProvenance
		simulationProvenance = &provenance
	}
	return contracts.EvidenceReport{
		Envelope:             contracts.NewEnvelope(synthetic.FixedTime),
		CampaignID:           campaignID,
		Summary:              fmt.Sprintf("%s evidence package for a generic reference DUT.", campaign.Name),
		Result:               result,
		Requirements:         reqs,
		Sources:              set.SourceCatalogue.Sources,
		GraphEvidence:        []string{fmt.Sprintf("fixtures/public/graph_models/%s.json", campaignID), fmt.Sprintf("fixtures/public/telemetry/%s.arrow", campaignID)},
		Anomalies:            anomalies,
		ThermalProgram:       campaign.ThermalProgram,
		SimulationProvenance: simulationProvenance,
		Reproducibility:      []string{"go run ./cmd/gossamer-fixtures", fmt.Sprintf("go run ./cmd/gossamer-report --campaign %s", campaignID)},
		SyntheticDataNote:    "Generated from deterministic physics-backed fixture data.",
	}, nil
}

func CampaignIDs() []string {
	set := synthetic.Build()
	ids := make([]string, 0, len(set.Campaigns)+1)
	for id := range set.Campaigns {
		ids = append(ids, id)
	}
	ids = append(ids, synthetic.CommandCenterGraphCampaignID)
	sort.Strings(ids)
	return ids
}

func buildCommandCenterReport(set synthetic.FixtureSet) contracts.EvidenceReport {
	graphModel := set.GraphModels[synthetic.CommandCenterGraphCampaignID]
	result := commandCenterResult(set.CommandCenterFAT)
	anomalies := []contracts.Anomaly{}
	var simulationProvenance *contracts.SimulationProvenance
	if graphModel.SimulationProvenance != nil {
		provenance := *graphModel.SimulationProvenance
		simulationProvenance = &provenance
	}
	return contracts.EvidenceReport{
		Envelope:             contracts.NewEnvelope(synthetic.FixedTime),
		CampaignID:           synthetic.CommandCenterGraphCampaignID,
		Summary:              set.CommandCenterFAT.Summary,
		Result:               result,
		Requirements:         []contracts.Requirement{},
		Sources:              set.SourceCatalogue.Sources,
		GraphEvidence:        []string{fmt.Sprintf("fixtures/public/graph_models/%s.json", synthetic.CommandCenterGraphCampaignID), fmt.Sprintf("fixtures/public/telemetry/%s.arrow", synthetic.CommandCenterGraphCampaignID)},
		Anomalies:            anomalies,
		ThermalProgram:       graphModel.ThermalProgram,
		SimulationProvenance: simulationProvenance,
		Reproducibility:      []string{"go run ./cmd/gossamer-fixtures", fmt.Sprintf("go run ./cmd/gossamer-report --campaign %s", synthetic.CommandCenterGraphCampaignID)},
		SyntheticDataNote:    "Generated from deterministic physics-backed fixture data.",
	}
}

func commandCenterResult(model contracts.CommandCenterFAT) string {
	result := "pass"
	for _, lane := range model.Lanes {
		for _, run := range lane.Runs {
			switch run.Result {
			case "fail":
				return "fail"
			case "in_progress", "pending":
				if result == "pass" {
					result = run.Result
				}
			case "inconclusive":
				if result != "fail" {
					result = "inconclusive"
				}
			}
		}
	}
	return result
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
	data = append(data, '\n')
	for _, name := range []string{campaignID + ".json", campaignID + "_report.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
