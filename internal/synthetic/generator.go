package synthetic

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egidinas/gossamer/internal/arrowtelemetry"
	"github.com/egidinas/gossamer/internal/contracts"
	"github.com/egidinas/gossamer/internal/environmentalsim"
	"github.com/egidinas/gossamer/internal/synthetic/commandcenter"
)

var FixedTime = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

const CommandCenterGraphCampaignID = "command_center_fat"

var CampaignIDs = []string{
	"flatsat_derisking",
	"thermal_acceptance_fat",
	"tvac_qualification",
	"integrated_system_fat",
}

type FixtureSet struct {
	Manifest         contracts.Manifest
	Topology         contracts.Topology
	SourceCatalogue  contracts.SourceCatalogue
	CommandAuthority contracts.CommandAuthorityState
	Supervisor       contracts.SupervisorOverview
	CommandCenterFAT contracts.CommandCenterFAT
	BusTap           contracts.BusVirtualizationTap
	Campaigns        map[string]contracts.Campaign
	Telemetry        map[string][]contracts.TelemetrySample
	GraphModels      map[string]contracts.GraphModel
}

func Build() FixtureSet {
	env := contracts.NewEnvelope(FixedTime)
	sources := buildSources(env)
	campaigns := map[string]contracts.Campaign{}
	telemetry := map[string][]contracts.TelemetrySample{}
	graphs := map[string]contracts.GraphModel{}
	for _, id := range CampaignIDs {
		campaign := buildCampaign(env, id)
		campaigns[id] = campaign
		samples := buildTelemetry(campaign)
		telemetry[id] = samples
		graphs[id] = buildGraphModel(env, campaign)
	}
	commandCenter, commandCenterTelemetry, commandCenterGraph := buildCommandCenterFATBundle(env)
	telemetry[CommandCenterGraphCampaignID] = commandCenterTelemetry
	graphs[CommandCenterGraphCampaignID] = commandCenterGraph
	return FixtureSet{
		Manifest: contracts.Manifest{
			Envelope:      env,
			Name:          "Gossamer",
			Description:   "Tile-backed environmental-test evidence and telemetry exploration.",
			TestArticle:   "Reference DUT",
			Campaigns:     CampaignIDs,
			PublicDemo:    true,
			SyntheticOnly: true,
		},
		Topology: contracts.Topology{
			Envelope: env,
			Nodes: []contracts.Node{
				{ID: "reference_dut", Label: "Reference DUT", Kind: "test_article", Status: "in_test", Quality: "synthetic"},
				{ID: "thermal_chamber_a", Label: "Thermal Chamber A", Kind: "facility", Status: "available", Quality: "fresh"},
				{ID: "tvac_chamber_q1", Label: "TVac Chamber Q1", Kind: "facility", Status: "campaign_active", Quality: "fresh"},
				{ID: "flatsat_rack_a", Label: "Flatsat Rack A", Kind: "facility", Status: "available", Quality: "fresh"},
				{ID: "archive_node_a", Label: "Archive Node A", Kind: "data_system", Status: "recording", Quality: "fresh"},
			},
			Links: []contracts.Link{
				{Source: "reference_dut", Target: "archive_node_a", Bus: "telemetry_bus"},
				{Source: "thermal_chamber_a", Target: "reference_dut", Bus: "facility_control_bus"},
				{Source: "tvac_chamber_q1", Target: "reference_dut", Bus: "facility_control_bus"},
				{Source: "flatsat_rack_a", Target: "reference_dut", Bus: "command_bus"},
			},
		},
		SourceCatalogue: sources,
		CommandAuthority: contracts.CommandAuthorityState{
			Envelope:        env,
			LeaseOwner:      "",
			LeaseState:      "available",
			AllowedCommands: []string{"set_demo_marker", "acknowledge_anomaly", "hold_fixture_state"},
			LastCommand:     "",
		},
		Supervisor:       buildSupervisorOverview(env, campaigns, telemetry),
		CommandCenterFAT: commandCenter,
		BusTap:           buildBusTap(env, telemetry["integrated_system_fat"]),
		Campaigns:        campaigns,
		Telemetry:        telemetry,
		GraphModels:      graphs,
	}
}

func WritePublicFixtures(root string) error {
	set := Build()
	base := filepath.Join(root, "fixtures", "public")
	dirs := []string{
		base,
		filepath.Join(base, "campaigns"),
		filepath.Join(base, "telemetry"),
		filepath.Join(base, "graph_models"),
		filepath.Join(base, "reports"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(base, "manifest.json"), set.Manifest); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(base, "topology.json"), set.Topology); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(base, "source_catalogue.json"), set.SourceCatalogue); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(base, "command_authority_state.json"), set.CommandAuthority); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(base, "supervisor_overview.json"), set.Supervisor); err != nil {
		return err
	}
	commandCenterOverview := set.CommandCenterFAT
	commandCenterOverview.HeroGraph = nil
	commandCenterOverview.GraphWall = nil
	if err := writeJSON(filepath.Join(base, "command_center_fat.json"), commandCenterOverview); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(base, "bus_virtualization_tap.json"), set.BusTap); err != nil {
		return err
	}
	for _, id := range CampaignIDs {
		if err := writeJSON(filepath.Join(base, "campaigns", id+".json"), set.Campaigns[id]); err != nil {
			return err
		}
		telemetry := telemetryWithGraphTraces(set.Telemetry[id], set.GraphModels[id])
		if err := arrowtelemetry.WriteCampaign(filepath.Join(base, "telemetry", id+".arrow"), id, telemetry, arrowtelemetry.MetadataFromGraph(set.GraphModels[id])); err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(base, "graph_models", id+".json"), set.GraphModels[id]); err != nil {
			return err
		}
	}
	commandCenterTelemetry := telemetryWithGraphTraces(set.Telemetry[CommandCenterGraphCampaignID], set.GraphModels[CommandCenterGraphCampaignID])
	if err := arrowtelemetry.WriteCampaign(filepath.Join(base, "telemetry", CommandCenterGraphCampaignID+".arrow"), CommandCenterGraphCampaignID, commandCenterTelemetry, arrowtelemetry.MetadataFromGraph(set.GraphModels[CommandCenterGraphCampaignID])); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(base, "graph_models", CommandCenterGraphCampaignID+".json"), set.GraphModels[CommandCenterGraphCampaignID]); err != nil {
		return err
	}
	return nil
}

func telemetryWithGraphTraces(samples []contracts.TelemetrySample, model contracts.GraphModel) []contracts.TelemetrySample {
	out := make([]contracts.TelemetrySample, len(samples))
	byTimestamp := make(map[string]int, len(samples))
	for i, sample := range samples {
		signals := make(map[string]float64, len(sample.Signals)+8)
		for id, value := range sample.Signals {
			signals[id] = value
		}
		states := make(map[string]string, len(sample.States))
		for id, value := range sample.States {
			states[id] = value
		}
		out[i] = sample
		out[i].Signals = signals
		out[i].States = states
		byTimestamp[sample.Timestamp] = i
	}
	if model.HeroGraph == nil {
		return out
	}
	for _, trace := range model.HeroGraph.Traces {
		if trace.ID == "" {
			continue
		}
		for _, point := range trace.Values {
			index, ok := byTimestamp[point.Timestamp]
			if !ok {
				continue
			}
			if _, exists := out[index].Signals[trace.ID]; !exists {
				out[index].Signals[trace.ID] = point.Value
			}
		}
	}
	return out
}

func buildSources(env contracts.Envelope) contracts.SourceCatalogue {
	return contracts.SourceCatalogue{Envelope: env, Sources: []contracts.Source{
		{ID: "dut_power", Label: "DUT Power", Owner: "egse_power_role", Bus: "telemetry_bus", Quality: "fresh", FreshnessMS: 250, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"eps_bus_voltage_v", "eps_bus_current_a", "dut_self_heat_w"}},
		{ID: "dut_control", Label: "DUT Control", Owner: "subsystem_test_role", Bus: "subsystem_event_bus", Quality: "fresh", FreshnessMS: 500, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"obc_boot_state", "obc_command_counter", "tc_packet_counter"}},
		{ID: "dut_link", Label: "DUT Link", Owner: "subsystem_test_role", Bus: "telemetry_bus", Quality: "fresh", FreshnessMS: 450, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"rf_link_margin_db", "tm_packet_counter"}},
		{ID: "dut_thermal", Label: "DUT Thermal Model", Owner: "test_conductor_role", Bus: "derived_model_bus", Quality: "synthetic", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"dut_fast_component_deg_c", "dut_lazy_component_deg_c", "dut_fast_air_flux_w", "dut_fast_interface_flux_w", "dut_fast_shroud_flux_w", "dut_lazy_air_flux_w", "dut_lazy_interface_flux_w", "dut_lazy_shroud_flux_w"}},
		{ID: "facility_thermal", Label: "Facility Thermal", Owner: "facility_test_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"thermal_cycle_index", "thermal_phase_code", "chamber_setpoint_deg_c", "thermal_zone_1_deg_c", "thermal_zone_2_deg_c", "chamber_air_deg_c", "interface_plate_deg_c", "thermal_shroud_deg_c", "thermal_shroud_inlet_deg_c", "thermal_shroud_outlet_deg_c", "thermal_shroud_gradient_deg_c", "huber_table_deg_c", "ln2_line_temp_deg_c", "ln2_valve_duty_pct", "tvac_cryo_exhaust_temp_deg_c"}},
		{ID: "facility_infrastructure", Label: "Building Infrastructure", Owner: "facility_test_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 600, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"cooling_water_freeze_margin_deg_c", "cooling_water_temp_deg_c", "pressurized_air_supply_bar", "air_dewpoint_deg_c", "tvac_scavenged_exhaust_temp_deg_c", "tvac_scavenger_cooling_water_return_deg_c", "tvac_exhaust_cold_recovery_pct", "tvac_exhaust_duct_safe"}},
		{ID: "facility_pressure", Label: "Facility Pressure", Owner: "facility_test_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"tvac_pressure_mbar"}},
		{ID: "demo_bus_virtualization", Label: "Demo Bus Virtualization Tap", Owner: "test_conductor_role", Bus: "telemetry_bus", Quality: "synthetic", FreshnessMS: 200, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"bus_latency_ms", "tm_packet_counter", "tc_packet_counter", "overall_packet_counter", "dropped_frame_count"}},
		{ID: "demo_quality", Label: "Demo Quality Monitor", Owner: "test_conductor_role", Bus: "telemetry_bus", Quality: "synthetic", FreshnessMS: 1000, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"source_freshness_ms", "facility_interlock_state"}},
	}}
}

func buildCampaign(env contracts.Envelope, id string) contracts.Campaign {
	c := contracts.Campaign{
		Envelope:      env,
		ID:            id,
		Article:       "Reference DUT",
		Anomalies:     []contracts.Anomaly{},
		SyntheticNote: "Reference campaign model with backend-owned telemetry, evidence, and graph contracts.",
		Start:         FixedTime.Format(time.RFC3339),
		End:           FixedTime.Add(11 * time.Hour).Format(time.RFC3339),
	}
	switch id {
	case "flatsat_derisking":
		c.Name, c.Level, c.State, c.Result, c.Facility = "Flatsat Derisking", "subsystem", "complete", "pass", "flatsat_rack_a"
	case "thermal_acceptance_fat":
		c.Name, c.Level, c.State, c.Result, c.Facility = "Thermal Chamber FAT", "integrated_acceptance", "complete", "pass", "thermal_chamber_a"
		c.ThermalProgram = buildThermalProgram(id, c.Facility, 4, -35, 65)
	case "tvac_qualification":
		c.Name, c.Level, c.State, c.Result, c.Facility = "TVac Qualification", "qualification", "review", "inconclusive", "tvac_chamber_q1"
		c.Anomalies = []contracts.Anomaly{{ID: "ANOM-TVAC-001", Title: "Pressure-source degradation during cold dwell", Severity: "medium", Status: "needs_disposition", EvidenceRef: "telemetry/tvac_qualification.arrow#sample-32", Disposition: "Review required before closure."}}
		c.ThermalProgram = buildThermalProgram(id, c.Facility, 8, -40, 70)
	case "integrated_system_fat":
		c.Name, c.Level, c.State, c.Result, c.Facility = "Integrated System FAT", "system", "complete", "pass", "thermal_chamber_a"
	default:
		c.Name, c.Level, c.State, c.Result, c.Facility = id, "unknown", "not_run", "not_run", "not_applicable"
	}
	if c.ThermalProgram != nil && len(c.ThermalProgram.Cycles) > 0 {
		lastCycle := c.ThermalProgram.Cycles[len(c.ThermalProgram.Cycles)-1]
		c.End = mustTime(lastCycle.End).Add(thermalContextDuration(c.ThermalProgram)).Format(time.RFC3339)
	}
	c.Requirements = defaultRequirements(c.Result)
	return c
}

func defaultRequirements(result string) []contracts.Requirement {
	ids := []string{"REQ-CYCLE-COUNT", "REQ-HOT-TARGET", "REQ-COLD-TARGET", "REQ-HOT-SURVIVAL", "REQ-COLD-SURVIVAL", "REQ-STABILITY", "REQ-DWELL", "REQ-FUNC-GATE-PRE", "REQ-FUNC-GATE-SURVIVAL", "REQ-FUNC-GATE-DURING", "REQ-FUNC-GATE-POST", "REQ-DATA-QUALITY", "REQ-ANOMALY-REVIEW"}
	reqs := make([]contracts.Requirement, 0, len(ids))
	for _, id := range ids {
		r := "pass"
		if result == "inconclusive" && (id == "REQ-DATA-QUALITY" || id == "REQ-ANOMALY-REVIEW") {
			r = "inconclusive"
		}
		reqs = append(reqs, contracts.Requirement{ID: id, Title: requirementTitle(id), Description: "Requirement used to demonstrate measurement-to-evidence traceability.", Result: r, Evidence: []string{"telemetry", "graph_model"}, Rationale: "Evaluated from deterministic fixture data."})
	}
	return reqs
}

func requirementTitle(id string) string {
	titles := map[string]string{
		"REQ-CYCLE-COUNT":        "Required cycle count completed",
		"REQ-HOT-TARGET":         "Hot target reached",
		"REQ-COLD-TARGET":        "Cold target reached",
		"REQ-HOT-SURVIVAL":       "Hot survival exposure completed",
		"REQ-COLD-SURVIVAL":      "Cold survival exposure completed",
		"REQ-STABILITY":          "Stabilization window achieved",
		"REQ-DWELL":              "Dwell duration achieved",
		"REQ-FUNC-GATE-PRE":      "Pre-environment functional gate passed",
		"REQ-FUNC-GATE-SURVIVAL": "Functional test after survival exposure passed",
		"REQ-FUNC-GATE-DURING":   "During-environment functional gate passed",
		"REQ-FUNC-GATE-POST":     "Post-environment functional gate passed",
		"REQ-DATA-QUALITY":       "Evidence data quality acceptable",
		"REQ-ANOMALY-REVIEW":     "Anomaly review disposition complete",
	}
	return titles[id]
}

type thermalTimingModel struct {
	kind              string
	slowNodeTauMin    float64
	survivalTauScale  float64
	gateBufferMinutes int
	fixedDwellMinutes int
	stabilityBandDegC float64
}

func (m thermalTimingModel) stabilizationMinutes(transitionDegC float64, survival bool) int {
	band := m.stabilityBandDegC
	if band <= 0 {
		band = 2
	}
	tau := m.slowNodeTauMin
	if survival {
		tau *= m.survivalTauScale
	}
	delta := math.Max(transitionDegC, band*1.25)
	minutes := int(math.Ceil(tau * math.Log(delta/band)))
	if minutes < 90 {
		minutes = 90
	}
	if survival && minutes < 150 {
		minutes = 150
	}
	if m.kind == "vacuum" && minutes < 180 {
		minutes = 180
	}
	return minutes
}

func buildThermalProgram(campaignID, facility string, cycleCount int, coldTarget, hotTarget float64) *contracts.ThermalProgram {
	kind := "thermal_chamber_fat"
	label := "Thermal Chamber FAT - 4 cycle acceptance profile"
	commandedRampRateDegCMin := 1.0
	fixedDwellMinutes := 50
	survivalMarginK := 5.0
	preContextMinutes := 720
	rampRate := commandedRampRateDegCMin
	dwellMinutes := fixedDwellMinutes
	timingModel := thermalTimingModel{kind: "air", slowNodeTauMin: 135, survivalTauScale: 1.18, gateBufferMinutes: 45, fixedDwellMinutes: fixedDwellMinutes, stabilityBandDegC: 2.0}
	if campaignID == "tvac_qualification" {
		kind = "tvac_qualification"
		label = "TVac Qualification - 8 cycle vacuum thermal profile"
		fixedDwellMinutes = 55
		survivalMarginK = 10
		preContextMinutes = 1560
		rampRate = commandedRampRateDegCMin
		dwellMinutes = fixedDwellMinutes
		timingModel = thermalTimingModel{kind: "vacuum", slowNodeTauMin: 205, survivalTauScale: 1.35, gateBufferMinutes: 60, fixedDwellMinutes: fixedDwellMinutes, stabilityBandDegC: 2.0}
	}
	hotSurvivalTarget := hotTarget + survivalMarginK
	coldSurvivalTarget := coldTarget - survivalMarginK

	program := &contracts.ThermalProgram{
		ID:              campaignID + "_program",
		Kind:            kind,
		Label:           label,
		Facility:        facility,
		CycleCount:      cycleCount,
		ColdTargetDegC:  coldTarget,
		HotTargetDegC:   hotTarget,
		RampRateDegCMin: rampRate,
		DwellMinutes:    dwellMinutes,
	}
	t := FixedTime.Add(time.Duration(preContextMinutes) * time.Minute)
	rampMinutes := func(from, to float64) int {
		minutes := int(math.Ceil(math.Abs(to-from) / commandedRampRateDegCMin))
		if minutes < 1 {
			return 1
		}
		return minutes
	}
	currentTarget := 22.0
	lastTransitionDegC := 0.0
	type dwellTiming struct {
		stableAt       time.Time
		dwellEnd       time.Time
		minimumMinutes int
	}
	dwellByPhase := map[string]dwellTiming{}
	addPhase := func(cycle int, suffix, label, kind string, minutes int, target float64) contracts.CyclePhase {
		start := t
		t = t.Add(time.Duration(minutes) * time.Minute)
		return contracts.CyclePhase{ID: fmt.Sprintf("C%02d-%s", cycle, suffix), Label: label, Kind: kind, Start: start.Format(time.RFC3339), End: t.Format(time.RFC3339), TargetDegC: target}
	}
	addRamp := func(cycle int, suffix, label, kind string, target float64) contracts.CyclePhase {
		lastTransitionDegC = math.Abs(target - currentTarget)
		phase := addPhase(cycle, suffix, label, kind, rampMinutes(currentTarget, target), target)
		currentTarget = target
		return phase
	}
	addDwell := func(cycle int, suffix, label, kind string, target float64) contracts.CyclePhase {
		stabilizeMinutes := timingModel.stabilizationMinutes(lastTransitionDegC, isSurvivalPhase(kind))
		stabilityGuardMinutes := timingModel.gateBufferMinutes
		minutes := stabilizeMinutes + stabilityGuardMinutes + timingModel.fixedDwellMinutes
		phase := addPhase(cycle, suffix, label, kind, minutes, target)
		stableAt := mustTime(phase.Start).Add(time.Duration(stabilizeMinutes+stabilityGuardMinutes) * time.Minute)
		dwellEnd := stableAt.Add(time.Duration(timingModel.fixedDwellMinutes) * time.Minute)
		phaseEnd := mustTime(phase.End)
		if !dwellEnd.Before(phaseEnd) {
			dwellEnd = phaseEnd.Add(-15 * time.Minute)
		}
		dwellByPhase[phase.ID] = dwellTiming{stableAt: stableAt, dwellEnd: dwellEnd, minimumMinutes: int(dwellEnd.Sub(stableAt).Minutes())}
		currentTarget = target
		return phase
	}
	for cycle := 1; cycle <= cycleCount; cycle++ {
		cycleStart := t
		phases := []contracts.CyclePhase{}
		if cycle == 1 {
			phases = append(phases,
				addRamp(cycle, "RAMP-HOT-SURVIVAL", "Ramp to hot survival", "ramp_hot", hotSurvivalTarget),
				addDwell(cycle, "HOT-SURVIVAL", "Hot survival dwell", "hot_survival", hotSurvivalTarget),
				addRamp(cycle, "RAMP-HOT-OPERATIONAL", "Cool down to hot operational", "ramp_hot", hotTarget),
				addDwell(cycle, "HOT-OPERATIONAL", "Hot operational dwell", "hot_operational", hotTarget),
				addRamp(cycle, "RAMP-COLD-SURVIVAL", "Ramp to cold survival", "ramp_cold", coldSurvivalTarget),
				addDwell(cycle, "COLD-SURVIVAL", "Cold survival dwell", "cold_survival", coldSurvivalTarget),
				addRamp(cycle, "RAMP-COLD-OPERATIONAL", "Warm up to cold operational", "ramp_cold", coldTarget),
				addDwell(cycle, "COLD-OPERATIONAL", "Cold operational dwell", "cold_operational", coldTarget),
				addRamp(cycle, "RAMP-AMBIENT", "Return to ambient", "ambient_recovery", 22),
			)
		} else {
			phases = append(phases,
				addRamp(cycle, "RAMP-HOT", "Ramp to hot operational", "ramp_hot", hotTarget),
				addDwell(cycle, "HOT-OPERATIONAL", "Hot operational dwell", "hot_operational", hotTarget),
				addRamp(cycle, "RAMP-COLD", "Ramp to cold operational", "ramp_cold", coldTarget),
				addDwell(cycle, "COLD-OPERATIONAL", "Cold operational dwell", "cold_operational", coldTarget),
				addRamp(cycle, "RAMP-AMBIENT", "Return to ambient", "ambient_recovery", 22),
			)
		}
		program.Cycles = append(program.Cycles, contracts.ThermalCycle{
			Index: cycle, Label: fmt.Sprintf("Cycle %d", cycle), Start: cycleStart.Format(time.RFC3339), End: t.Format(time.RFC3339), ColdTargetDegC: coldTarget, HotTargetDegC: hotTarget, Phases: phases,
		})
		for _, phase := range phases {
			if isThermalDwellPhase(phase.Kind) {
				timing := dwellByPhase[phase.ID]
				program.DwellWindows = append(program.DwellWindows, contracts.DwellWindow{ID: phase.ID + "-WINDOW", Label: phase.Label, CycleIndex: cycle, Kind: phase.Kind, Start: timing.stableAt.Format(time.RFC3339), End: timing.dwellEnd.Format(time.RFC3339), TargetDegC: phase.TargetDegC, StabilityBandC: 2.0, MinimumMinutes: timing.minimumMinutes, EvidenceRef: fmt.Sprintf("telemetry/%s.arrow#cycle-%02d-%s-stable", campaignID, cycle, phase.Kind)})
				program.EvidenceMarkers = append(program.EvidenceMarkers, contracts.EvidenceMarker{ID: fmt.Sprintf("%s-STABLE-C%02d-%s", campaignID, cycle, phase.Kind), Label: fmt.Sprintf("Stable %s confirmed", phase.Label), Timestamp: timing.stableAt.Format(time.RFC3339), Kind: "stability_achieved", Result: "pass", EvidenceRef: fmt.Sprintf("reports/%s.json#stable-c%02d-%s", campaignID, cycle, phase.Kind)})
			}
		}
	}
	firstCycle := program.Cycles[0]
	lastCycle := program.Cycles[len(program.Cycles)-1]
	postEnd := mustTime(lastCycle.End).Add(thermalContextDuration(program))
	preGateTime := mustTime(firstCycle.Start).Add(-30 * time.Minute)
	postGateTime := postEnd.Add(-30 * time.Minute)
	gates := []struct {
		id, label, gate, phaseID string
		cycle                    int
		timestamp                time.Time
	}{
		{"PRE", "Pre-environment functional test", "pre", "ambient_precheck", 0, preGateTime},
		{"POST", "Post-environment functional test", "post", "ambient_postcheck", 0, postGateTime},
	}
	if campaignID == "tvac_qualification" {
		gates = []struct {
			id, label, gate, phaseID string
			cycle                    int
			timestamp                time.Time
		}{
			{"AMBIENT-PRE", "Ambient-pressure functional test before pumpdown", "ambient_pre", "ambient_precheck", 0, FixedTime.Add(90 * time.Minute)},
			{"VACUUM-PRE", "Vacuum ambient functional test after pressure gate", "vacuum_pre", "ambient_precheck", 0, preGateTime},
			{"VACUUM-POST", "Final ambient-temperature functional test under vacuum", "vacuum_post", "ambient_postcheck_vacuum", 0, mustTime(lastCycle.End).Add(60 * time.Minute)},
			{"POST", "Ambient-pressure functional test after vent", "post", "ambient_postcheck", 0, postGateTime},
		}
		program.EvidenceMarkers = append(program.EvidenceMarkers,
			contracts.EvidenceMarker{ID: campaignID + "-PRESSURE-GATE-PRE", Label: "Vacuum pressure gate reached", Timestamp: preGateTime.Add(-30 * time.Minute).Format(time.RFC3339), Kind: "pressure_gate", Result: "pass", EvidenceRef: fmt.Sprintf("reports/%s.json#pressure-gate-pre", campaignID)},
			contracts.EvidenceMarker{ID: campaignID + "-PRESSURE-GATE-POST", Label: "Vacuum pressure gate held for final test", Timestamp: mustTime(lastCycle.End).Add(30 * time.Minute).Format(time.RFC3339), Kind: "pressure_gate", Result: "pass", EvidenceRef: fmt.Sprintf("reports/%s.json#pressure-gate-post", campaignID)},
		)
	}
	for _, cycle := range program.Cycles {
		for _, phase := range cycle.Phases {
			if !isOperationalDwellPhase(phase.Kind) {
				continue
			}
			gateName := "cold"
			if phase.Kind == "hot_dwell" || phase.Kind == "hot_operational" {
				gateName = "hot"
			}
			timing := dwellByPhase[phase.ID]
			delay := deterministicOperatorDelay(campaignID, cycle.Index, gateName, timing.dwellEnd, program)
			gateAt := timing.dwellEnd.Add(delay)
			latest := mustTime(phase.End).Add(-15 * time.Minute)
			if gateAt.After(latest) {
				gateAt = latest
			}
			label := fmt.Sprintf("Cycle %d %s dwell functional test", cycle.Index, gateName)
			if cycle.Index == 1 && gateName == "hot" {
				label = "Hot survival recovery functional test"
			}
			if cycle.Index == 1 && gateName == "cold" {
				label = "Cold survival recovery functional test"
			}
			gates = append(gates, struct {
				id, label, gate, phaseID string
				cycle                    int
				timestamp                time.Time
			}{
				fmt.Sprintf("C%02d-%s", cycle.Index, strings.ToUpper(gateName)),
				label,
				gateName,
				phase.ID,
				cycle.Index,
				gateAt,
			})
		}
	}
	for _, gate := range gates {
		result := "pass"
		if campaignID == "tvac_qualification" && gate.gate == "hot" {
			result = "inconclusive"
		}
		program.FunctionalGates = append(program.FunctionalGates, contracts.FunctionalGate{ID: fmt.Sprintf("%s-FUNC-%s", campaignID, gate.id), Label: gate.label, Gate: gate.gate, CycleIndex: gate.cycle, PhaseID: gate.phaseID, Timestamp: gate.timestamp.Format(time.RFC3339), Result: result, EvidenceRef: fmt.Sprintf("telemetry/%s.arrow#functional-%s", campaignID, gate.gate)})
		program.EvidenceMarkers = append(program.EvidenceMarkers, contracts.EvidenceMarker{ID: fmt.Sprintf("%s-EVID-%s", campaignID, gate.id), Label: gate.label, Timestamp: gate.timestamp.Format(time.RFC3339), Kind: "functional_gate", Result: result, EvidenceRef: fmt.Sprintf("reports/%s.json#functional-%s", campaignID, gate.gate)})
	}
	program.InterlockWindows = []contracts.InterlockWindow{{ID: campaignID + "-INTERLOCK-NOMINAL", Label: "Facility interlocks closed", Start: FixedTime.Format(time.RFC3339), End: mustTime(lastCycle.End).Add(thermalContextDuration(program)).Format(time.RFC3339), State: "closed", Severity: "info", EvidenceRef: fmt.Sprintf("telemetry/%s.arrow#interlocks", campaignID)}}
	if campaignID == "tvac_qualification" {
		reviewPhase := phaseByKind(program.Cycles[5], "cold_operational")
		reviewStart := mustTime(reviewPhase.Start).Add(3 * time.Hour)
		program.InterlockWindows = append(program.InterlockWindows, contracts.InterlockWindow{ID: "TVAC-LN2-VALVE-REVIEW", Label: "LN2 valve duty review window", Start: reviewStart.Format(time.RFC3339), End: reviewStart.Add(50 * time.Minute).Format(time.RFC3339), State: "review", Severity: "medium", EvidenceRef: "telemetry/tvac_qualification.arrow#ln2-review"})
	}
	return program
}

func isThermalDwellPhase(kind string) bool {
	return kind == "hot_survival" || kind == "cold_survival" || isOperationalDwellPhase(kind)
}

func isSurvivalPhase(kind string) bool {
	return kind == "hot_survival" || kind == "cold_survival"
}

func isOperationalDwellPhase(kind string) bool {
	return kind == "hot_dwell" || kind == "cold_dwell" || kind == "hot_operational" || kind == "cold_operational"
}

func thermalContextDuration(program *contracts.ThermalProgram) time.Duration {
	if program == nil || len(program.Cycles) == 0 {
		return 8 * time.Hour
	}
	return 8 * time.Hour
}

func deterministicOperatorDelay(campaignID string, cycle int, gate string, stableAt time.Time, program *contracts.ThermalProgram) time.Duration {
	start := mustTime(program.Cycles[0].Start)
	end := mustTime(program.Cycles[len(program.Cycles)-1].End)
	cursor := start.Add(time.Duration(float64(end.Sub(start)) * 0.60))
	if stableAt.After(cursor) {
		return 0
	}
	base := cycle*17 + len(gate)*11
	if campaignID == "tvac_qualification" {
		base += 23
	}
	return time.Duration(8+base%47) * time.Minute
}

func phaseByKind(cycle contracts.ThermalCycle, kind string) contracts.CyclePhase {
	for _, phase := range cycle.Phases {
		if phase.Kind == kind {
			return phase
		}
	}
	return cycle.Phases[0]
}

func midpoint(start, end string) time.Time {
	startTime := mustTime(start)
	endTime := mustTime(end)
	return startTime.Add(endTime.Sub(startTime) / 2)
}

func mustTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func buildTelemetry(campaign contracts.Campaign) []contracts.TelemetrySample {
	if campaign.ThermalProgram != nil {
		return buildThermalTelemetry(campaign)
	}
	start := FixedTime
	out := make([]contracts.TelemetrySample, 0, 48)
	for i := 0; i < 48; i++ {
		t := start.Add(time.Duration(i) * 15 * time.Minute)
		phase := float64(i) / 47
		chamber := -20.0 + 70*math.Sin(phase*math.Pi)
		pressure := 101325.0
		quality := "fresh"
		freshness := 250.0
		if campaign.ID == "integrated_system_fat" && i >= 18 && i <= 24 {
			quality = "degraded"
			freshness = 3200
		}
		out = append(out, contracts.TelemetrySample{
			Timestamp: t.Format(time.RFC3339),
			Quality:   quality,
			Signals: map[string]float64{
				"eps_bus_voltage_v":                         28.0 + 0.3*math.Sin(phase*2*math.Pi),
				"eps_bus_current_a":                         4.0 + 0.8*math.Sin(phase*4*math.Pi),
				"obc_command_counter":                       float64(1000 + i*3),
				"payload_sim_heater_w":                      math.Max(0, 25+20*math.Cos(phase*2*math.Pi)),
				"thermal_zone_1_deg_c":                      chamber + 2.4,
				"thermal_zone_2_deg_c":                      chamber - 1.8,
				"chamber_air_deg_c":                         chamber,
				"chamber_setpoint_deg_c":                    chamber,
				"thermal_cycle_index":                       0,
				"thermal_phase_code":                        0,
				"huber_table_deg_c":                         chamber - 0.6,
				"ln2_line_temp_deg_c":                       22,
				"ln2_valve_duty_pct":                        0,
				"cooling_water_freeze_margin_deg_c":         18,
				"cooling_water_temp_deg_c":                  15.5,
				"tvac_cryo_exhaust_temp_deg_c":              20,
				"tvac_scavenged_exhaust_temp_deg_c":         20,
				"tvac_scavenger_cooling_water_return_deg_c": 15.8,
				"tvac_exhaust_cold_recovery_pct":            0,
				"tvac_exhaust_duct_safe":                    0,
				"pressurized_air_supply_bar":                6.2,
				"air_dewpoint_deg_c":                        -41.5,
				"functional_gate_code":                      0,
				"tvac_pressure_pa":                          pressure,
				"tvac_pressure_mbar":                        pressure * 0.01,
				"source_freshness_ms":                       freshness,
				"facility_interlock_code":                   1,
				"bus_latency_ms":                            18 + 8*math.Sin(phase*2*math.Pi),
				"tm_packet_counter":                         float64(6000 + i*17),
				"tc_packet_counter":                         float64(120 + i*2),
				"overall_packet_counter":                    float64(6120 + i*19),
				"dropped_frame_count":                       droppedFrames(campaign.ID, i),
				"rf_link_margin_db":                         8.5 + 1.5*math.Cos(phase*2*math.Pi),
			},
			States: map[string]string{
				"obc_boot_state":           "nominal",
				"rf_link_sim_state":        "locked",
				"facility_interlock_state": "closed",
				"thermal_phase":            "ambient",
				"functional_gate":          "none",
			},
		})
	}
	return out
}

func buildThermalTelemetry(campaign contracts.Campaign) []contracts.TelemetrySample {
	return environmentalsim.Simulate(campaign.ID, campaign.ThermalProgram, FixedTime).Samples
}

func buildGraphModel(env contracts.Envelope, campaign contracts.Campaign) contracts.GraphModel {
	actuatorLabel := "Cooling Actuator Duty"
	if campaign.ThermalProgram != nil && campaign.ThermalProgram.Kind == "tvac_qualification" {
		actuatorLabel = "LN2 Valve Duty"
	}
	model := contracts.GraphModel{
		Envelope:   env,
		CampaignID: campaign.ID,
		Lanes: []contracts.GraphLane{
			{ID: "thermal", Label: "Thermal", Series: []contracts.GraphSeries{{ID: "chamber_setpoint_deg_c", Label: "Chamber Setpoint", Role: "facility_setpoint", Units: "degC", Source: "facility_thermal", Min: -45, Max: 75}, {ID: "chamber_air_deg_c", Label: "Chamber Air", Role: "facility_environment", Units: "degC", Source: "facility_thermal", Min: -45, Max: 75}, {ID: "thermal_zone_1_deg_c", Label: "Thermal Zone 1", Role: "article_temperature", Units: "degC", Source: "facility_thermal", Min: -45, Max: 75}}},
			{ID: "pressure", Label: "Pressure", Series: []contracts.GraphSeries{{ID: "tvac_pressure_mbar", Label: "TVac pressure", Role: "facility_environment", Units: "mbar", Source: "facility_pressure", Min: 0.00000001, Max: 1013.25}}},
			{ID: "facility_safety", Label: "Facility Safety", Series: []contracts.GraphSeries{{ID: "ln2_valve_duty_pct", Label: actuatorLabel, Role: "facility_interlock", Units: "%", Source: "facility_thermal", Min: 0, Max: 100}, {ID: "cooling_water_freeze_margin_deg_c", Label: "Water scavenger freeze margin", Role: "facility_interlock", Units: "degC", Source: "facility_infrastructure", Min: 0, Max: 25}}},
			{ID: "power", Label: "Power", Series: []contracts.GraphSeries{{ID: "eps_bus_voltage_v", Label: "Bus Voltage", Role: "dut_power", Units: "V", Source: "dut_power", Min: 26, Max: 30}, {ID: "eps_bus_current_a", Label: "Bus Current", Role: "dut_power", Units: "A", Source: "dut_power", Min: 0, Max: 8}}},
			{ID: "bus", Label: "Virtual Bus", Series: []contracts.GraphSeries{{ID: "bus_latency_ms", Label: "Bus Latency", Role: "virtual_bus_health", Units: "ms", Source: "demo_bus_virtualization", Min: 0, Max: 250}, {ID: "tm_packet_counter", Label: "TM Counter", Role: "telemetry_counter", Units: "count", Source: "demo_bus_virtualization", Min: 0, Max: 8000}}},
			{ID: "quality", Label: "Source Quality", Series: []contracts.GraphSeries{{ID: "source_freshness_ms", Label: "Source Freshness", Role: "data_quality", Units: "ms", Source: "demo_quality", Min: 0, Max: 6000}}},
		},
	}
	if campaign.ThermalProgram != nil {
		sim := environmentalsim.Simulate(campaign.ID, campaign.ThermalProgram, FixedTime)
		model.ThermalProgram = campaign.ThermalProgram
		model.SimulationProvenance = &sim.Provenance
		model.HeroGraph = &sim.HeroGraph
		wall := buildGraphWall(env, campaign.ID, sim.HeroGraph, sim.Provenance)
		model.GraphWall = &wall
		manifest := buildTileManifest(env, campaign.ID, wall, sim.HeroGraph)
		model.TileManifest = &manifest
		for _, cycle := range campaign.ThermalProgram.Cycles {
			model.Annotations = append(model.Annotations, contracts.GraphAnnotation{ID: fmt.Sprintf("%s-cycle-%02d", campaign.ID, cycle.Index), Label: cycle.Label, Kind: "cycle", Start: cycle.Start, End: cycle.End, CycleIndex: cycle.Index})
		}
		for _, gate := range campaign.ThermalProgram.FunctionalGates {
			model.Annotations = append(model.Annotations, contracts.GraphAnnotation{ID: gate.ID, Label: gate.Label, Kind: "functional_gate", Timestamp: gate.Timestamp, CycleIndex: gate.CycleIndex, Result: gate.Result})
		}
	}
	return model
}

func buildGraphWall(env contracts.Envelope, campaignID string, hero contracts.HeroGraphModel, provenance contracts.SimulationProvenance) contracts.GraphWallModel {
	groupID := campaignID + "_operator_wall"
	sectionIDs := []string{"thermal_environment", "facility_response", "dut_response", "tmtc_response", "testbed_dynamics"}
	wall := contracts.GraphWallModel{
		ID:           campaignID + "_graph_wall",
		Title:        hero.Title + " operator graph wall",
		GeneratedAt:  env.GeneratedAt,
		SourceMode:   "arrow_backend_owned",
		GraphVersion: "gossamer.graph_wall.v1",
		Owner:        "gossamer_backend_fixture_generator",
		Provenance:   provenance.Source,
		TimeRange: contracts.GraphWallTimeRange{
			Start:        hero.TimeAxis.Start,
			End:          hero.TimeAxis.End,
			Anchor:       hero.TimeAxis.Anchor,
			RangeSeconds: hero.TimeAxis.RangeSeconds,
			Mode:         "absolute_fixture",
			Source:       "thermal_program",
		},
		TilePolicy: contracts.GraphTilePolicy{
			DefaultPoints:               900,
			MaxPoints:                   3600,
			LiveTileMinRefreshMS:        1000,
			HistoryTileMaxCount:         96,
			ViewportPrefetchPX:          640,
			TileBufferMaxEntries:        192,
			TileBufferTTLMS:             90000,
			ResolutionLevels:            []string{"raw", "1m", "5m", "15m"},
			SubscriberRole:              "operator_supervisor",
			SharedTimebaseRequired:      true,
			LegendMayAffectPlotWidth:    false,
			MalformedSVGPathHardFailure: true,
		},
		GraphGroups: []contracts.GraphGroup{{
			ID:              groupID,
			Title:           "Synchronized environmental execution wall",
			Mode:            "single_chamber_supervisor",
			BehaviorProfile: "loom_dense_operator_wall",
			Application:     "environmental_test_execution",
			SectionIDs:      sectionIDs,
			Interaction: contracts.GraphInteraction{
				SharedTimeline: true, SharedCrosshair: true, VerticalGrid: true, SingleTimeAxis: true,
				CursorMode: "inspect", CrosshairScope: "all_cards", TimelineGridMode: "phase_cycle_aligned",
			},
			Layout: contracts.GraphLayoutContract{
				PinnedCardsSeparate: true,
				OverflowMode:        "vertical_scroll_only",
				AxisRail:            "fixed_left_and_right",
				LegendRail:          "fixed_right_outside_plot",
				LabelRail:           "fixed_left_outside_plot",
				PlotAreaPolicy:      "same_width_all_cards",
			},
		}},
	}
	sections := map[string]*contracts.GraphSection{}
	wall.Sections = make([]contracts.GraphSection, len(sectionIDs))
	for i, id := range sectionIDs {
		wall.Sections[i] = contracts.GraphSection{ID: id, Title: graphSectionTitle(id), GroupID: groupID, Transport: "arrow_ipc", Direction: "derived", Status: "fresh"}
		sections[id] = &wall.Sections[i]
	}
	add := func(sectionID string, card contracts.GraphWallCard) {
		card.Placement.SectionID = sectionID
		card.Placement.GroupID = groupID
		card.Placement.Order = len(sections[sectionID].Cards) + 1
		if card.Placement.HeightWeight == 0 {
			card.Placement.HeightWeight = 1
		}
		card.Placement.DefaultVisible = true
		if card.Placement.ResizePolicy == "" {
			card.Placement.ResizePolicy = "fixed_plot_area"
		}
		sections[sectionID].Cards = append(sections[sectionID].Cards, card)
	}
	thermalProgramSignals := []contracts.GraphWallSignal{
		graphSignal("trace.command.chamber", "Chamber command", "degC", "thermal_program", "command", "command", "facility", "temperature_c", "thermal_environment"),
		graphSignal("trace.ghost.profile", "Cycle ghost profile", "degC", "thermal_program", "ghost", "ghost", "facility", "temperature_c", "thermal_environment"),
		graphSignal("trace.acceptance.temperature", "Acceptance band center", "degC", "requirements", "acceptance_band", "acceptance", "requirements", "temperature_c", "thermal_environment"),
		graphSignal("trace.dut_temp_a", "High-dissipation DUT node", "degC", "dut_thermal", "actual", "measurement", "dut", "temperature_c", "thermal_environment"),
		graphSignal("trace.dut_temp_b", "Vacuum-detached DUT node", "degC", "dut_thermal", "actual", "measurement", "dut", "temperature_c", "thermal_environment"),
		graphSignal("trace.table_loop", "Fluid interface", "degC", "facility_thermal", "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
	}
	if campaignID != "tvac_qualification" {
		thermalProgramSignals = append(thermalProgramSignals[:3], append([]contracts.GraphWallSignal{
			graphSignal("trace.actual.chamber_air", "Chamber air actual", "degC", "facility_thermal", "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
		}, thermalProgramSignals[3:]...)...)
	}
	if campaignID == "tvac_qualification" {
		thermalProgramSignals = append(thermalProgramSignals,
			graphSignal("trace.tvac_pressure", "TVac pressure", "mbar", "facility_pressure", "actual", "measurement", "facility", "pressure_mbar", "thermal_environment"),
			graphSignal("trace.tvac_pressure_target", "Vacuum target", "mbar", "requirements", "ghost", "target", "requirements", "pressure_mbar", "thermal_environment"),
			graphSignal("trace.shroud_inlet", "Shroud inlet", "degC", "facility_thermal", "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
			graphSignal("trace.shroud_outlet", "Shroud outlet", "degC", "facility_thermal", "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
		)
	}
	thermalTitle := "Chamber command, ghost, acceptance, actual"
	if campaignID == "tvac_qualification" {
		thermalTitle = "TVac command, pressure, shroud, interface, DUT"
	}
	add("thermal_environment", graphCard("thermal_program", thermalTitle, "line", "primary_hero", "degC", "temperature_c", "facility_thermal", thermalProgramSignals))
	dutTemperatureSignals := []contracts.GraphWallSignal{}
	if campaignID != "tvac_qualification" {
		dutTemperatureSignals = append(dutTemperatureSignals,
			graphSignal("trace.context.chamber_air", "Chamber air", "degC", "facility_thermal", "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
		)
	}
	dutTemperatureSignals = append(dutTemperatureSignals,
		graphSignal("trace.dut_temp_a", "High-dissipation DUT node", "degC", "dut_thermal", "actual", "measurement", "dut", "temperature_c", "thermal_environment"),
		graphSignal("trace.dut_temp_b", "Vacuum-detached DUT node", "degC", "dut_thermal", "actual", "measurement", "dut", "temperature_c", "thermal_environment"),
		graphSignal("trace.table_loop", "Fluid interface", "degC", "facility_thermal", "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
	)
	if campaignID == "tvac_qualification" {
		dutTemperatureSignals = append(dutTemperatureSignals,
			graphSignal("trace.shroud_inlet", "Shroud inlet", "degC", "facility_thermal", "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
			graphSignal("trace.shroud_outlet", "Shroud outlet", "degC", "facility_thermal", "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
			graphSignal("trace.shroud_gradient", "Shroud gradient", "degC", "facility_thermal", "source_quality", "measurement", "facility", "temperature_c", "thermal_environment"),
		)
	}
	dutTemperatureTitle := "DUT temperatures in chamber context"
	if campaignID == "tvac_qualification" {
		dutTemperatureTitle = "DUT temperatures in TVac context"
	}
	add("thermal_environment", graphCard("dut_temperature", dutTemperatureTitle, "line", "companion", "degC", "temperature_c", "dut_thermal", dutTemperatureSignals))
	if campaignID == "tvac_qualification" {
		add("facility_response", graphCard("tvac_pressure", "TVac pressure pumpdown and bursts", "line", "companion", "mbar", "log_pressure_mbar", "facility_pressure", []contracts.GraphWallSignal{
			graphSignal("trace.tvac_pressure", "Pressure", "mbar", "facility_pressure", "actual", "measurement", "facility", "pressure_mbar", "facility_response"),
			graphSignal("trace.tvac_pressure_target", "Vacuum target", "mbar", "requirements", "ghost", "target", "requirements", "pressure_mbar", "facility_response"),
		}))
		add("facility_response", graphCard("tvac_pressure_sources", "Pump, leak, and outgassing balance", "line", "companion", "mixed", "pressure_balance", "facility_pressure", []contracts.GraphWallSignal{
			graphSignal("trace.tvac_outgassing", "Temperature outgassing", "mbar/min", "facility_pressure", "actual", "measurement", "facility", "pressure_rate", "facility_response"),
			graphSignal("trace.tvac_virtual_leak", "Virtual leak", "mbar/min", "facility_pressure", "acceptance_band", "limit", "facility", "pressure_rate", "facility_response"),
			graphSignal("trace.tvac_roughing_pump", "Roughing pump", "mbar/min", "facility_pressure", "source_quality", "measurement", "facility", "pressure_rate", "facility_response"),
			graphSignal("trace.tvac_turbo_pump", "Turbo pump", "mbar/min", "facility_pressure", "actual", "measurement", "facility", "pressure_rate", "facility_response"),
			graphSignal("trace.tvac_pump_removal", "Pump removal", "mbar/min", "facility_pressure", "source_quality", "measurement", "facility", "pressure_rate", "facility_response"),
			graphSignal("trace.tvac_volatile_inventory", "Volatile inventory", "%", "facility_pressure", "ghost", "derived", "facility", "percent", "facility_response"),
		}))
	}
	actuationTitle := "Cooling actuator duty"
	actuationLabel := "Cooling demand"
	if campaignID == "tvac_qualification" {
		actuationTitle = "LN2 valve duty"
		actuationLabel = "LN2 valve"
	}
	add("facility_response", graphCard("facility_actuation", actuationTitle, "line", "companion", "%", "percent", "facility_thermal", []contracts.GraphWallSignal{
		graphSignal("trace.ln2_duty", actuationLabel, "%", "facility_thermal", "actual", "actuator", "facility", "percent", "facility_response"),
	}))
	if campaignID == "tvac_qualification" {
		add("facility_response", graphCard("facility_temperature_safety", "Heat-exchanger freeze margin", "line", "companion", "degC", "temperature_c", "facility_infrastructure", []contracts.GraphWallSignal{
			graphSignal("trace.freeze_margin", "Water scavenger freeze margin", "degC", "facility_infrastructure", "interlock", "safety_margin", "facility", "temperature_c", "facility_response"),
		}))
		add("facility_response", graphCard("tvac_exhaust_scavenger", "Exhaust cold scavenger", "line", "companion", "mixed", "facility_exhaust_scavenger", "facility_infrastructure", []contracts.GraphWallSignal{
			graphSignal("trace.tvac_cryo_exhaust", "Cryogenic exhaust", "degC", "facility_thermal", "actual", "measurement", "facility", "temperature_c", "facility_response"),
			graphSignal("trace.tvac_scavenged_exhaust", "After water scavenger", "degC", "facility_infrastructure", "actual", "measurement", "facility", "temperature_c", "facility_response"),
			graphSignal("trace.tvac_scavenger_water_return", "Scavenger water return", "degC", "facility_infrastructure", "actual", "measurement", "facility", "temperature_c", "facility_response"),
			graphSignal("trace.tvac_exhaust_cold_recovery", "Cold recovery", "%", "facility_infrastructure", "source_quality", "derived", "facility", "percent", "facility_response"),
		}))
	}
	add("facility_response", graphCard("building_infrastructure", "Building infrastructure", "line", "companion", "mixed", "facility_infra", "facility_infrastructure", []contracts.GraphWallSignal{
		graphSignal("trace.cooling_water_temp", "Cooling water temp", "degC", "facility_infrastructure", "actual", "measurement", "facility", "temperature_c", "facility_response"),
		graphSignal("trace.pressurized_air_supply", "Pressurized air supply", "bar", "facility_infrastructure", "actual", "measurement", "facility", "pressure_bar", "facility_response"),
		graphSignal("trace.air_dewpoint", "Air dew point", "degC", "facility_infrastructure", "actual", "measurement", "facility", "temperature_c", "facility_response"),
	}))
	add("dut_response", graphCard("dut_power", "DUT power budgets", "line", "companion", "W", "power_w", "dut_power", []contracts.GraphWallSignal{
		graphSignal("trace.power_total", "Total power", "W", "dut_power", "actual", "measurement", "dut_power", "power_w", "dut_response"),
		graphSignal("trace.power_subsystem", "Subsystem budget", "W", "dut_power", "actual", "measurement", "dut_power", "power_w", "dut_response"),
		graphSignal("trace.power_payload", "Payload/FT load", "W", "dut_power", "actual", "measurement", "dut_power", "power_w", "dut_response"),
		graphSignal("trace.power_avionics", "Avionics", "W", "dut_power", "actual", "measurement", "dut_power", "power_w", "dut_response"),
		graphSignal("trace.power_link", "Link subsystem", "W", "dut_power", "actual", "measurement", "dut_power", "power_w", "dut_response"),
	}))
	add("tmtc_response", graphCard("tmtc_health", "TM/TC latency and freshness", "line", "companion", "ms", "bus_ms", "demo_bus_virtualization", []contracts.GraphWallSignal{
		graphSignal("trace.bus_latency", "Bus latency", "ms", "demo_bus_virtualization", "source_quality", "measurement", "tmtc", "bus_ms", "tmtc_response"),
		graphSignal("trace.source_freshness", "Freshness", "ms", "demo_quality", "source_quality", "measurement", "tmtc", "bus_ms", "tmtc_response"),
	}))
	add("tmtc_response", graphCard("tmtc_counters", "TM/TC packet counters", "counter", "companion", "count", "counter", "demo_bus_virtualization", []contracts.GraphWallSignal{
		graphSignal("trace.overall_packet_counter", "Overall packet counter", "count", "demo_bus_virtualization", "counter", "counter", "tmtc", "counter", "tmtc_response"),
		graphSignal("trace.tm_packet_counter", "TM packet counter", "count", "demo_bus_virtualization", "counter", "counter", "tmtc", "counter", "tmtc_response"),
		graphSignal("trace.tc_packet_counter", "TC packet counter", "count", "demo_bus_virtualization", "counter", "counter", "tmtc", "counter", "tmtc_response"),
		graphSignal("trace.dropped_frame_count", "Dropped frames", "count", "demo_bus_virtualization", "counter", "counter", "tmtc", "counter", "tmtc_response"),
	}))
	add("tmtc_response", graphCard("source_quality", "Source freshness quality", "line", "companion", "ms", "bus_ms", "demo_quality", []contracts.GraphWallSignal{
		graphSignal("trace.source_freshness", "Source freshness", "ms", "demo_quality", "source_quality", "measurement", "tmtc", "bus_ms", "tmtc_response"),
	}))
	stateSignals := []contracts.GraphWallSignal{
		enumSignal("trace.phase_enum", "Thermal phase", "thermal_program", "enum", "testbed_dynamics", map[string]string{"0": "ambient pre", "1": "ramp cold", "2": "cold op", "3": "ramp hot", "4": "hot op", "5": "hot survival", "6": "cold survival", "7": "vacuum ambient", "8": "ambient recovery", "9": "ambient post"}),
		enumSignal("trace.functional_gate_active", "Functional gate active", "test_conductor", "bool", "testbed_dynamics", map[string]string{"0": "inactive", "1": "active"}),
		enumSignal("trace.stability_reached", "Stability reached", "thermal_program", "bool", "testbed_dynamics", map[string]string{"0": "stabilizing", "1": "stable"}),
		enumSignal("trace.dwell_active", "Dwell active", "thermal_program", "bool", "testbed_dynamics", map[string]string{"0": "idle", "1": "dwelling"}),
		enumSignal("trace.dwell_complete", "Dwell complete", "thermal_program", "bool", "testbed_dynamics", map[string]string{"0": "open", "1": "complete"}),
		enumSignal("trace.interlock_review", "Facility interlock state", "facility_safety", "fault", "testbed_dynamics", map[string]string{"1": "closed", "2": "review"}),
		enumSignal("trace.source_degraded", "Source degraded", "demo_quality", "bool", "testbed_dynamics", map[string]string{"0": "fresh", "1": "degraded"}),
		enumSignal("trace.evidence_capture", "Evidence capture", "evidence_report", "bool", "testbed_dynamics", map[string]string{"0": "idle", "1": "capture"}),
		enumSignal("trace.dut_ready", "DUT ready", "dut_control", "bool", "testbed_dynamics", map[string]string{"0": "not ready", "1": "ready"}),
		enumSignal("trace.dut_operative", "DUT operative", "dut_control", "bool", "testbed_dynamics", map[string]string{"0": "inhibited", "1": "operative"}),
		enumSignal("trace.payload_active", "Payload active", "dut_power", "bool", "testbed_dynamics", map[string]string{"0": "standby", "1": "active"}),
		enumSignal("trace.rf_link_locked", "RF link locked", "dut_link", "bool", "testbed_dynamics", map[string]string{"0": "searching", "1": "locked"}),
		enumSignal("trace.fault_flag", "Fault flag", "demo_quality", "fault", "testbed_dynamics", map[string]string{"0": "nominal", "1": "fault"}),
	}
	if campaignID == "tvac_qualification" {
		stateSignals = append(stateSignals[:5], append([]contracts.GraphWallSignal{enumSignal("trace.pressure_gate", "Pressure gate", "facility_pressure", "bool", "testbed_dynamics", map[string]string{"0": "waiting", "1": "reached"})}, stateSignals[5:]...)...)
		stateSignals = append(stateSignals[:6], append([]contracts.GraphWallSignal{enumSignal("trace.pump_mode", "Pump mode", "facility_pressure", "enum", "testbed_dynamics", map[string]string{"0": "ambient", "1": "roughing", "2": "crossover", "3": "turbo"})}, stateSignals[6:]...)...)
		stateSignals = append(stateSignals[:7], append([]contracts.GraphWallSignal{enumSignal("trace.exhaust_duct_safe", "Exhaust duct safe", "facility_infrastructure", "bool", "testbed_dynamics", map[string]string{"0": "scavenger warming", "1": "duct safe"})}, stateSignals[7:]...)...)
	}
	add("testbed_dynamics", graphCard("state_change_swimlane", "Testbed state swimlanes", "state", "swimlane", "state", "state_lane", "thermal_program", stateSignals))
	add("testbed_dynamics", graphCard("functional_events", "Functional gates and evidence events", "event", "event_rail", "event", "event", "test_conductor", []contracts.GraphWallSignal{
		graphSignal("functional_gates", "Functional gates", "event", "test_conductor", "event", "event", "test_conductor", "event", "testbed_dynamics"),
		graphSignal("evidence_markers", "Evidence markers", "event", "evidence_report", "evidence", "event", "evidence", "event", "testbed_dynamics"),
		graphSignal("interlock_windows", "Interlock windows", "event", "facility_safety", "interlock", "event", "facility", "event", "testbed_dynamics"),
	}))
	return wall
}

func graphCard(id, title, kind, role, unit, axisPolicy, source string, signals []contracts.GraphWallSignal) contracts.GraphWallCard {
	return contracts.GraphWallCard{
		ID: id, Title: title, Kind: kind, Role: role, Transport: "arrow_ipc", Direction: "derived",
		Unit: unit, AxisPolicy: axisPolicy, SourceFamily: source, Signals: signals,
		RenderKind: renderKind(kind), TileEndpoint: "/data/current/campaigns/{campaign_id}/telemetry.arrow.gz", LatestEndpoint: "/data/current/campaigns/{campaign_id}/telemetry.arrow.gz",
		Collapsible: true, DefaultExpanded: true, SupportsTimeZoom: true, SupportsYZoom: kind == "line" || kind == "counter",
		Placement: contracts.GraphCardPlacement{Pinned: role == "primary_hero", HeightWeight: heightWeight(kind, role)},
	}
}

func renderKind(kind string) string {
	switch kind {
	case "counter":
		return "counter"
	case "state":
		return "swimlane"
	case "event":
		return "event_rail"
	default:
		return "line"
	}
}

func buildTileManifest(env contracts.Envelope, campaignID string, wall contracts.GraphWallModel, hero contracts.HeroGraphModel) contracts.GraphTileManifest {
	manifest := contracts.GraphTileManifest{
		Envelope:    env,
		ID:          campaignID + "_tile_manifest",
		CampaignID:  campaignID,
		GraphWallID: wall.ID,
		GeneratedAt: env.GeneratedAt,
		SourceMode:  "arrow_telemetry_backend",
		TimeRange:   wall.TimeRange,
		TilePolicy:  wall.TilePolicy,
		Levels:      tileLevels(),
		SourceNodes: sourceNodes(),
		DataLensTranslations: []contracts.DataLensTranslation{
			{ID: "legacy_csv_environment", Label: "Legacy CSV environment import", SourceFormat: "legacy_csv", TargetSchema: arrowtelemetry.SchemaName, Mode: "translated_fixture", Confidence: "high", Provenance: "synthetic DataLens translation demo"},
			{ID: "binary_tmtc_log", Label: "Binary TM/TC log import", SourceFormat: "binary_log", TargetSchema: arrowtelemetry.SchemaName, Mode: "translated_fixture", Confidence: "medium", Provenance: "synthetic DataLens translation demo"},
			{ID: "hdf5_evidence_archive", Label: "HDF5-like evidence archive", SourceFormat: "hdf5", TargetSchema: arrowtelemetry.SchemaName, Mode: "translated_fixture", Confidence: "high", Provenance: "synthetic DataLens translation demo"},
		},
	}
	for _, section := range wall.Sections {
		for _, card := range section.Cards {
			ref := contracts.GraphTileCardRef{
				CardID: card.ID, Title: card.Title, RenderKind: card.RenderKind, Unit: card.Unit, AxisPolicy: card.AxisPolicy,
				TileEndpoint: strings.ReplaceAll(card.TileEndpoint, "{campaign_id}", campaignID), LatestEndpoint: strings.ReplaceAll(card.LatestEndpoint, "{campaign_id}", campaignID),
				Collapsible: card.Collapsible, DefaultExpanded: card.DefaultExpanded, SupportsTimeZoom: card.SupportsTimeZoom, SupportsYZoom: card.SupportsYZoom,
				IncludeMarkers: card.IncludeMarkers,
				Signals:        card.Signals,
			}
			manifest.Cards = append(manifest.Cards, ref)
		}
	}
	for _, marker := range hero.Markers {
		if marker.EvidenceRef == "" {
			continue
		}
		manifest.EvidenceLinks = append(manifest.EvidenceLinks, contracts.EvidenceLink{ID: "manifest-" + marker.ID, RequirementID: requirementForMarker(marker), CardID: "thermal_program", MarkerID: marker.ID, Timestamp: marker.Timestamp, Status: marker.Result, Label: marker.Label})
	}
	return manifest
}

func tileLevels() []contracts.TileLevel {
	return []contracts.TileLevel{
		{ID: "year", Label: "Year", Resolution: "year", DurationMS: int64((365 * 24 * time.Hour) / time.Millisecond), MaxPoints: 480, DecimationMode: "min_max_envelope"},
		{ID: "month", Label: "Month", Resolution: "month", DurationMS: int64((31 * 24 * time.Hour) / time.Millisecond), MaxPoints: 600, DecimationMode: "min_max_envelope"},
		{ID: "day", Label: "Day", Resolution: "day", DurationMS: int64((24 * time.Hour) / time.Millisecond), MaxPoints: 720, DecimationMode: "min_max_envelope"},
		{ID: "hour", Label: "Hour", Resolution: "hour", DurationMS: int64(time.Hour / time.Millisecond), MaxPoints: 900, DecimationMode: "min_max_envelope"},
		{ID: "minute", Label: "Minute", Resolution: "minute", DurationMS: int64(time.Minute / time.Millisecond), MaxPoints: 4200, DecimationMode: "viewport_lttb_seed"},
		{ID: "second", Label: "Second", Resolution: "second", DurationMS: int64(time.Second / time.Millisecond), MaxPoints: 1800, DecimationMode: "raw_or_envelope"},
		{ID: "millisecond", Label: "Millisecond", Resolution: "millisecond", DurationMS: 1, MaxPoints: 3600, DecimationMode: "raw_or_envelope"},
	}
}

func sourceNodes() []contracts.SourceNode {
	return []contracts.SourceNode{
		{ID: "fixture_backend", Label: "Fixture backend", Kind: "synthetic", Mode: "deterministic", Confidence: "high", Provenance: "gossamer internal simulation"},
		{ID: "facility_control", Label: "Facility control node", Kind: "live_capable", Mode: "simulated", Confidence: "high", Provenance: "tile source contract"},
		{ID: "legacy_import", Label: "Legacy import node", Kind: "translated", Mode: "fixture", Confidence: "medium", Provenance: "DataLens translation demo"},
	}
}

func requirementForMarker(marker contracts.GraphMarker) string {
	switch marker.Kind {
	case "functional_gate":
		return "REQ-FUNC-GATE"
	case "stability_achieved":
		return "REQ-STABILITY"
	case "interlock":
		return "REQ-ANOMALY-REVIEW"
	default:
		return "REQ-DATA-QUALITY"
	}
}

func graphSignal(id, label, unit, source, role, kind, subsystem, axisID, sectionID string) contracts.GraphWallSignal {
	return contracts.GraphWallSignal{ID: id, Label: label, Unit: unit, Source: source, SourceFamily: source, Kind: kind, Category: role, Role: role, Subsystem: subsystem, AxisID: axisID, SectionID: sectionID}
}

func enumSignal(id, label, source, kind, sectionID string, table map[string]string) contracts.GraphWallSignal {
	return contracts.GraphWallSignal{ID: id, Label: label, Unit: "state", Source: source, SourceFamily: source, Kind: kind, Category: "state", Role: "state", Subsystem: "operator_state", AxisID: "state_lane", SectionID: sectionID, ValueTable: table}
}

func graphSectionTitle(id string) string {
	switch id {
	case "thermal_environment":
		return "Thermal Environment"
	case "facility_response":
		return "Facility Response"
	case "dut_response":
		return "DUT Response"
	case "tmtc_response":
		return "Telecommand/Telemetry Response"
	case "testbed_dynamics":
		return "Testbed Dynamics"
	default:
		return id
	}
}

func heightWeight(kind, role string) float64 {
	if role == "primary_hero" {
		return 2.8
	}
	if kind == "state" || kind == "event" {
		return 0.85
	}
	return 1.0
}

func droppedFrames(campaign string, index int) float64 {
	if campaign == "integrated_system_fat" && index >= 18 && index <= 24 {
		return float64(index - 17)
	}
	if campaign == "tvac_qualification" && index >= 30 && index <= 34 {
		return float64(index - 29)
	}
	return 0
}

func buildSupervisorOverview(env contracts.Envelope, campaigns map[string]contracts.Campaign, telemetry map[string][]contracts.TelemetrySample) contracts.SupervisorOverview {
	lanes := []contracts.SupervisorLane{
		supervisorLane("thermal_fat", "Thermal Chamber FAT", "thermal_chamber_a", campaigns["thermal_acceptance_fat"], "4 cycle chamber FAT with cold/hot dwell gates", "facility_control_bus", "4 cycles, hot/cold dwell, pre/cold/hot/post functional gates", "fresh", []string{"Thermal chamber profile is synchronized with DUT temperature."}, telemetry["thermal_acceptance_fat"], []heroSpec{{"chamber_setpoint_deg_c", "Setpoint", "facility_setpoint", "degC", "facility_thermal", -45, 75}, {"chamber_air_deg_c", "Chamber Air", "facility_environment", "degC", "facility_thermal", -45, 75}, {"thermal_zone_1_deg_c", "DUT Node 1", "article_temperature", "degC", "facility_thermal", -45, 75}, {"bus_latency_ms", "TM/TC Latency", "virtual_bus_health", "ms", "demo_bus_virtualization", 0, 250}}),
		supervisorLane("eps_load_step", "EPS Load Step", "flatsat_rack_a", campaigns["integrated_system_fat"], "Power load and command script", "command_bus", "REQ-FUNC-GATE-DURING pass", "degraded", []string{"Synthetic freshness degradation demonstrates disposition workflow."}, telemetry["integrated_system_fat"], []heroSpec{{"eps_bus_voltage_v", "Bus Voltage", "dut_power", "V", "dut_power", 24, 32}, {"eps_bus_current_a", "Bus Current", "dut_power", "A", "dut_power", 0, 8}}),
		supervisorLane("payload_thermal", "Payload Thermal Cycle", "thermal_chamber_a", campaigns["integrated_system_fat"], "Payload simulator heater cycling", "telemetry_bus", "REQ-STABILITY pass", "fresh", []string{"Payload heater response is fictional and bounded for demo use."}, telemetry["integrated_system_fat"], []heroSpec{{"payload_sim_heater_w", "Payload Heater", "payload_thermal_control", "W", "facility_thermal", 0, 60}, {"thermal_zone_2_deg_c", "Article Zone 2", "article_temperature", "degC", "facility_thermal", -45, 70}}),
		supervisorLane("tvac_qualification", "TVac Qualification", "tvac_chamber_q1", campaigns["tvac_qualification"], "8 cycle TVac qualification with pumpdown and thermal-source review", "facility_control_bus", "8 cycles, pressure plateau, safety interlock review open", "degraded", []string{"Pressure-source degradation remains open for review."}, telemetry["tvac_qualification"], []heroSpec{{"chamber_setpoint_deg_c", "Setpoint", "facility_setpoint", "degC", "facility_thermal", -45, 75}, {"thermal_zone_1_deg_c", "DUT Node 1", "article_temperature", "degC", "facility_thermal", -45, 75}, {"tvac_pressure_mbar", "TVac pressure", "facility_environment", "mbar", "facility_pressure", 0.00000001, 1013.25}, {"ln2_valve_duty_pct", "TVac cooling valve duty", "facility_interlock", "%", "facility_thermal", 0, 100}, {"cooling_water_freeze_margin_deg_c", "Water scavenger freeze margin", "facility_interlock", "degC", "facility_infrastructure", 0, 25}}),
		supervisorLane("archive_capture", "Archive Evidence Capture", "archive_node_a", campaigns["integrated_system_fat"], "TM/TC capture and evidence packaging", "telemetry_bus", "REQ-DATA-QUALITY pass with review note", "synthetic", []string{"Archive node receives virtualized TM and TC events from the replay bus."}, telemetry["integrated_system_fat"], []heroSpec{{"bus_latency_ms", "Bus Latency", "virtual_bus_health", "ms", "demo_bus_virtualization", 0, 250}, {"tm_packet_counter", "TM Counter", "telemetry_counter", "count", "demo_bus_virtualization", 0, 8000}}),
	}
	return contracts.SupervisorOverview{
		Envelope:    env,
		TestArticle: "Reference DUT",
		Summary:     "Parallel FAT and qualification campaign views with shared telemetry and evidence contracts.",
		Lanes:       lanes,
	}
}

type heroSpec struct {
	signal string
	label  string
	role   string
	units  string
	source string
	min    float64
	max    float64
}

func supervisorLane(id, label, facility string, campaign contracts.Campaign, activity, primaryBus, requirementSummary, quality string, notes []string, samples []contracts.TelemetrySample, specs []heroSpec) contracts.SupervisorLane {
	graphs := make([]contracts.SupervisorHeroGraph, 0, len(specs))
	for _, spec := range specs {
		graphs = append(graphs, contracts.SupervisorHeroGraph{
			ID:     spec.signal,
			Label:  spec.label,
			Signal: spec.signal,
			Units:  spec.units,
			Role:   spec.role,
			Source: spec.source,
			Min:    spec.min,
			Max:    spec.max,
			Values: graphPoints(samples, spec.signal, 80),
		})
	}
	lane := contracts.SupervisorLane{
		ID:                 id,
		Label:              label,
		Facility:           facility,
		Campaign:           campaign.ID,
		Activity:           activity,
		State:              campaign.State,
		Result:             campaign.Result,
		PrimaryBus:         primaryBus,
		RequirementSummary: requirementSummary,
		SourceQuality:      quality,
		HeroGraphs:         graphs,
		Notes:              notes,
		ThermalProgram:     campaign.ThermalProgram,
	}
	if campaign.ThermalProgram != nil {
		lane.FunctionalGates = campaign.ThermalProgram.FunctionalGates
		lane.InterlockWindows = campaign.ThermalProgram.InterlockWindows
		lane.EvidenceMarkers = campaign.ThermalProgram.EvidenceMarkers
	}
	return lane
}

func buildCommandCenterFATBundle(env contracts.Envelope) (contracts.CommandCenterFAT, []contracts.TelemetrySample, contracts.GraphModel) {
	return commandcenter.BuildFATBundle(env, FixedTime, CommandCenterGraphCampaignID, buildThermalProgram)
}

// isBusinessHour, addBusinessDuration, and commandCenterSignalPrefix are forwarding shims
// so that tests in this package can call them directly.
func isBusinessHour(t time.Time) bool {
	return commandcenter.IsBusinessHour(t)
}

func addBusinessDuration(start time.Time, duration time.Duration) time.Time {
	return commandcenter.AddBusinessDuration(start, duration)
}

func commandCenterSignalPrefix(chamberID string) string {
	return commandcenter.CommandCenterSignalPrefix(chamberID)
}

func graphPoints(samples []contracts.TelemetrySample, signal string, limit int) []contracts.GraphPoint {
	step := len(samples) / limit
	if step < 1 {
		step = 1
	}
	points := []contracts.GraphPoint{}
	for i := 0; i < len(samples) && len(points) < limit; i += step {
		if value, ok := samples[i].Signals[signal]; ok {
			points = append(points, contracts.GraphPoint{Timestamp: samples[i].Timestamp, Value: round(value)})
		}
	}
	return points
}

func buildBusTap(env contracts.Envelope, samples []contracts.TelemetrySample) contracts.BusVirtualizationTap {
	streams := []contracts.BusStream{
		{ID: "tm_primary", Label: "Reference DUT TM to Archive", Direction: "TM", SourceNode: "reference_dut", DestinationNode: "archive_node_a", Bus: "telemetry_bus", Quality: "fresh", LatencyMS: 22, PacketCounter: 6748, DroppedFrames: 0},
		{ID: "tc_primary", Label: "Flatsat Rack TC to Reference DUT", Direction: "TC", SourceNode: "flatsat_rack_a", DestinationNode: "reference_dut", Bus: "command_bus", Quality: "fresh", LatencyMS: 31, PacketCounter: 214, DroppedFrames: 0},
		{ID: "facility_tm", Label: "Facility Control TM to Archive", Direction: "TM", SourceNode: "thermal_chamber_a", DestinationNode: "archive_node_a", Bus: "facility_control_bus", Quality: "fresh", LatencyMS: 28, PacketCounter: 4080, DroppedFrames: 0},
	}
	events := []contracts.BusEvent{}
	for i := 0; i < 14 && i < len(samples); i++ {
		sample := samples[i]
		tmID := fmt.Sprintf("BUS-TM-%04d", i+1)
		events = append(events, contracts.BusEvent{
			ID:              tmID,
			StreamID:        "tm_primary",
			Direction:       "TM",
			Timestamp:       sample.Timestamp,
			SourceNode:      "reference_dut",
			DestinationNode: "archive_node_a",
			EventClass:      "telemetry_sample",
			Authority:       "read_only_tap",
			Quality:         sample.Quality,
			LatencyMS:       int(math.Round(sample.Signals["bus_latency_ms"])),
			PacketCounter:   int(sample.Signals["tm_packet_counter"]),
			Fields: map[string]float64{
				"thermal_zone_1_deg_c": round(sample.Signals["thermal_zone_1_deg_c"]),
				"eps_bus_voltage_v":    round(sample.Signals["eps_bus_voltage_v"]),
				"bus_latency_ms":       round(sample.Signals["bus_latency_ms"]),
			},
			States:  map[string]string{"rf_link_sim_state": sample.States["rf_link_sim_state"]},
			Summary: "Synthetic telemetry envelope mirrored into the archive tap.",
		})
		if i%3 == 0 {
			events = append(events, contracts.BusEvent{
				ID:              fmt.Sprintf("BUS-TC-%04d", i/3+1),
				StreamID:        "tc_primary",
				Direction:       "TC",
				Timestamp:       sample.Timestamp,
				SourceNode:      "flatsat_rack_a",
				DestinationNode: "reference_dut",
				EventClass:      "command_request",
				Authority:       "demo_operator_lease",
				Quality:         "fresh",
				LatencyMS:       30 + i,
				PacketCounter:   int(sample.Signals["tc_packet_counter"]),
				Fields:          map[string]float64{"requested_step": float64(i / 3), "expected_ack_ms": 120},
				States:          map[string]string{"authorization": "accepted", "execution_ack": "complete"},
				Summary:         "Fictional command request accepted by the mocked authority lease.",
			})
		}
	}
	return contracts.BusVirtualizationTap{
		Envelope:     env,
		ConnectionID: "bus_virtualization_demo",
		Description:  "Polling replay of fictional TM and TC envelopes between generic test nodes.",
		ReplayCursor: "cursor-reference-dut-demo-001",
		Streams:      streams,
		Events:       events,
	}
}

func round(value float64) float64 {
	abs := math.Abs(value)
	switch {
	case abs == 0:
		return 0
	case abs < 0.000001:
		return math.Round(value*1e12) / 1e12
	case abs < 0.001:
		return math.Round(value*1e9) / 1e9
	case abs < 1:
		return math.Round(value*1e6) / 1e6
	case abs < 1000:
		return math.Round(value*1000) / 1000
	default:
		return math.Round(value*100) / 100
	}
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
