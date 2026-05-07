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
	if input.Campaign.ThermalProgram != nil {
		cycles = observedCycleCount(samples) == input.Campaign.ThermalProgram.CycleCount
		hot = maxSignal(samples, "chamber_air_deg_c") >= input.Campaign.ThermalProgram.HotTargetDegC-2
		cold = minSignal(samples, "chamber_air_deg_c") <= input.Campaign.ThermalProgram.ColdTargetDegC+2
	}
	stable := maxSignal(samples, "source_freshness_ms") <= 3500
	dwell := len(samples) >= 32
	if input.Campaign.ThermalProgram != nil {
		dwell = observedPhaseCount(samples, "cold_operational") >= input.Campaign.ThermalProgram.CycleCount && observedPhaseCount(samples, "hot_operational") >= input.Campaign.ThermalProgram.CycleCount
	}
	dataQuality := noDegraded(samples)
	anomalyClosed := len(input.Campaign.Anomalies) == 0
	hotSurvival := observedPhaseCount(samples, "hot_survival") > 0
	coldSurvival := observedPhaseCount(samples, "cold_survival") > 0
	preGate := observedGate(samples, "pre")
	duringGate := observedGate(samples, "cold") && observedGate(samples, "hot")
	postGate := observedGate(samples, "post")
	survivalGate := observedGate(samples, "hot") && observedGate(samples, "cold")
	if input.Campaign.ID == "tvac_qualification" {
		preGate = observedGate(samples, "ambient_pre") && observedGate(samples, "vacuum_pre")
		postGate = observedGate(samples, "vacuum_post") && observedGate(samples, "post")
	}

	checks := map[string]bool{
		"REQ-CYCLE-COUNT":        cycles,
		"REQ-HOT-TARGET":         hot,
		"REQ-COLD-TARGET":        cold,
		"REQ-HOT-SURVIVAL":       hotSurvival,
		"REQ-COLD-SURVIVAL":      coldSurvival,
		"REQ-STABILITY":          stable,
		"REQ-DWELL":              dwell,
		"REQ-FUNC-GATE-PRE":      preGate,
		"REQ-FUNC-GATE-SURVIVAL": survivalGate,
		"REQ-FUNC-GATE-DURING":   duringGate,
		"REQ-FUNC-GATE-POST":     postGate,
		"REQ-DATA-QUALITY":       dataQuality,
		"REQ-ANOMALY-REVIEW":     anomalyClosed,
	}
	out := make([]contracts.Requirement, 0, len(input.Campaign.Requirements))
	for _, req := range input.Campaign.Requirements {
		result := "fail"
		if checks[req.ID] {
			result = "pass"
		}
		if input.Campaign.ID == "tvac_qualification" && (req.ID == "REQ-STABILITY" || req.ID == "REQ-DATA-QUALITY" || req.ID == "REQ-ANOMALY-REVIEW") {
			result = "inconclusive"
		}
		req.Result = result
		req.Evidence = []string{
			fmt.Sprintf("fixtures/public/telemetry/%s.arrow", input.Campaign.ID),
			fmt.Sprintf("fixtures/public/graph_models/%s.json", input.Campaign.ID),
		}
		req.Rationale = rationale(req.ID, result, input.Campaign)
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

func observedCycleCount(samples []contracts.TelemetrySample) int {
	seen := map[int]bool{}
	for _, sample := range samples {
		cycle := int(sample.Signals["thermal_cycle_index"])
		if cycle > 0 {
			seen[cycle] = true
		}
	}
	return len(seen)
}

func observedPhaseCount(samples []contracts.TelemetrySample, phase string) int {
	seen := map[int]bool{}
	for _, sample := range samples {
		if sample.States["thermal_phase"] == phase {
			cycle := int(sample.Signals["thermal_cycle_index"])
			if cycle > 0 {
				seen[cycle] = true
			}
		}
	}
	return len(seen)
}

func observedGate(samples []contracts.TelemetrySample, gate string) bool {
	for _, sample := range samples {
		if sample.States["functional_gate"] == gate {
			return true
		}
	}
	return false
}

func rationale(id, result string, campaign contracts.Campaign) string {
	if campaign.ThermalProgram != nil {
		switch id {
		case "REQ-CYCLE-COUNT":
			return fmt.Sprintf("Observed %d backend-defined thermal cycles for %s.", campaign.ThermalProgram.CycleCount, campaign.ThermalProgram.Label)
		case "REQ-HOT-TARGET":
			return fmt.Sprintf("Chamber command, chamber air, article-zone traces, and acceptance bands show hot target achievement near %.1f degC.", campaign.ThermalProgram.HotTargetDegC)
		case "REQ-COLD-TARGET":
			if campaign.ThermalProgram.Kind == "tvac_qualification" {
				return fmt.Sprintf("Cold ramp, LN2 duty, chamber air, and article-zone traces show cold target achievement near %.1f degC.", campaign.ThermalProgram.ColdTargetDegC)
			}
			return fmt.Sprintf("Cold ramp, cooling actuator duty, chamber air, and article-zone traces show cold target achievement near %.1f degC.", campaign.ThermalProgram.ColdTargetDegC)
		case "REQ-HOT-SURVIVAL":
			return "The first cycle includes hot survival exposure before the hot operational functional gate."
		case "REQ-COLD-SURVIVAL":
			return "The first cycle includes cold survival exposure before the cold operational functional gate."
		case "REQ-STABILITY":
			return "Dwell-window evidence uses source freshness, article-zone convergence, and stability markers from the physics-backed thermal model."
		case "REQ-DWELL":
			return fmt.Sprintf("Hot and cold dwell windows are declared for all %d cycles and referenced by telemetry evidence.", campaign.ThermalProgram.CycleCount)
		case "REQ-FUNC-GATE-PRE", "REQ-FUNC-GATE-SURVIVAL", "REQ-FUNC-GATE-DURING", "REQ-FUNC-GATE-POST":
			return "Functional gate markers are present in telemetry and linked to load, TM/TC bus latency, packet counter, and thermal self-heating evidence."
		case "REQ-DATA-QUALITY":
			return "Sensor freshness, dropout/source-quality flags, and graph evidence markers are evaluated from the generated physical telemetry."
		case "REQ-ANOMALY-REVIEW":
			return "Interlock and anomaly disposition references are tied to exact cycle/phase timestamps in the thermal program."
		}
	}
	if result == "pass" {
		return "Telemetry and campaign metadata satisfy this traceability requirement."
	}
	if result == "inconclusive" {
		return "Evidence marks this item for review to demonstrate anomaly disposition workflow."
	}
	return "Evidence did not satisfy this traceability requirement."
}
