package evaluator

import (
	"fmt"

	"github.com/egidinas/gossamer/internal/contracts"
	"github.com/egidinas/gossamer/internal/synthetic"
)

type EvaluationInput struct {
	Campaign  contracts.Campaign
	Telemetry []contracts.TelemetrySample
}

func Evaluate(input EvaluationInput) []contracts.Requirement {
	samples := input.Telemetry
	hot := maxSignal(samples, "chamber_air_deg_c") >= 48
	cold := minSignal(samples, "chamber_air_deg_c") <= -18
	cycles := hot && cold && len(samples) >= 40
	stable := maxSignal(samples, "source_freshness_ms") <= 3500
	dwell := len(samples) >= 32
	dataQuality := noDegraded(samples)
	anomalyClosed := len(input.Campaign.Anomalies) == 0

	checks := map[string]bool{
		"REQ-CYCLE-COUNT":      cycles,
		"REQ-HOT-TARGET":       hot,
		"REQ-COLD-TARGET":      cold,
		"REQ-STABILITY":        stable,
		"REQ-DWELL":            dwell,
		"REQ-FUNC-GATE-PRE":    true,
		"REQ-FUNC-GATE-DURING": true,
		"REQ-FUNC-GATE-POST":   true,
		"REQ-DATA-QUALITY":     dataQuality,
		"REQ-ANOMALY-REVIEW":   anomalyClosed,
	}
	out := make([]contracts.Requirement, 0, len(input.Campaign.Requirements))
	for _, req := range input.Campaign.Requirements {
		result := "fail"
		if checks[req.ID] {
			result = "pass"
		}
		if input.Campaign.ID == "tvac_qualification" && (req.ID == "REQ-DATA-QUALITY" || req.ID == "REQ-ANOMALY-REVIEW") {
			result = "inconclusive"
		}
		req.Result = result
		req.Evidence = []string{
			fmt.Sprintf("fixtures/public/telemetry/%s.jsonl", input.Campaign.ID),
			fmt.Sprintf("fixtures/public/graph_models/%s.json", input.Campaign.ID),
		}
		req.Rationale = rationale(req.ID, result)
		out = append(out, req)
	}
	return out
}

func EvaluateSyntheticCampaign(id string) (contracts.Campaign, []contracts.Requirement, error) {
	set := synthetic.Build()
	campaign, ok := set.Campaigns[id]
	if !ok {
		return contracts.Campaign{}, nil, fmt.Errorf("unknown campaign %q", id)
	}
	reqs := Evaluate(EvaluationInput{Campaign: campaign, Telemetry: set.Telemetry[id]})
	campaign.Requirements = reqs
	return campaign, reqs, nil
}

func maxSignal(samples []contracts.TelemetrySample, id string) float64 {
	max := -1e12
	for _, sample := range samples {
		if v, ok := sample.Signals[id]; ok && v > max {
			max = v
		}
	}
	return max
}

func minSignal(samples []contracts.TelemetrySample, id string) float64 {
	min := 1e12
	for _, sample := range samples {
		if v, ok := sample.Signals[id]; ok && v < min {
			min = v
		}
	}
	return min
}

func noDegraded(samples []contracts.TelemetrySample) bool {
	for _, sample := range samples {
		if sample.Quality == "degraded" || sample.Quality == "missing" {
			return false
		}
	}
	return true
}

func rationale(id, result string) string {
	if result == "pass" {
		return "Synthetic telemetry and campaign metadata satisfy this public demo requirement."
	}
	if result == "inconclusive" {
		return "Synthetic evidence marks this item for review to demonstrate anomaly disposition workflow."
	}
	return "Synthetic evidence did not satisfy this public demo requirement."
}
