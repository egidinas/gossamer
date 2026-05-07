package synthetic

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egidinas/gossamer/internal/contracts"
	"github.com/egidinas/gossamer/internal/environmentalsim"
)

func TestBuildProducesValidContracts(t *testing.T) {
	set := Build()
	if err := contracts.ValidateManifest(set.Manifest); err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if err := contracts.ValidateSourceCatalogue(set.SourceCatalogue); err != nil {
		t.Fatalf("sources: %v", err)
	}
	for id, campaign := range set.Campaigns {
		if err := contracts.ValidateCampaign(campaign); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if len(set.Telemetry[id]) < 48 {
			t.Fatalf("%s telemetry count = %d, want at least 48", id, len(set.Telemetry[id]))
		}
	}
}

func TestThermalCampaignsExposeExplicitCyclePrograms(t *testing.T) {
	set := Build()
	cases := []struct {
		campaignID string
		wantCycles int
		wantKind   string
		minHours   float64
		maxHours   float64
	}{
		{campaignID: "thermal_acceptance_fat", wantCycles: 4, wantKind: "thermal_chamber_fat", minHours: 100, maxHours: 145},
		{campaignID: "tvac_qualification", wantCycles: 8, wantKind: "tvac_qualification", minHours: 230, maxHours: 350},
	}

	for _, tc := range cases {
		campaign := set.Campaigns[tc.campaignID]
		if campaign.ThermalProgram == nil {
			t.Fatalf("%s missing thermal program", tc.campaignID)
		}
		if got := campaign.ThermalProgram.Kind; got != tc.wantKind {
			t.Fatalf("%s thermal program kind = %q, want %q", tc.campaignID, got, tc.wantKind)
		}
		if got := campaign.ThermalProgram.CycleCount; got != tc.wantCycles {
			t.Fatalf("%s cycle count = %d, want %d", tc.campaignID, got, tc.wantCycles)
		}
		if got := len(campaign.ThermalProgram.Cycles); got != tc.wantCycles {
			t.Fatalf("%s cycles = %d, want %d", tc.campaignID, got, tc.wantCycles)
		}
		start := mustParseTime(t, campaign.ThermalProgram.Cycles[0].Start)
		end := mustParseTime(t, campaign.ThermalProgram.Cycles[len(campaign.ThermalProgram.Cycles)-1].End)
		hours := end.Sub(start).Hours()
		if hours < tc.minHours || hours > tc.maxHours {
			t.Fatalf("%s program duration = %.1fh, want %.1f..%.1fh", tc.campaignID, hours, tc.minHours, tc.maxHours)
		}
		for _, cycle := range campaign.ThermalProgram.Cycles {
			if cycle.Index < 1 || cycle.Index > tc.wantCycles {
				t.Fatalf("%s cycle index %d outside 1..%d", tc.campaignID, cycle.Index, tc.wantCycles)
			}
			if cycle.HotTargetDegC <= cycle.ColdTargetDegC {
				t.Fatalf("%s cycle %d hot %.1f must exceed cold %.1f", tc.campaignID, cycle.Index, cycle.HotTargetDegC, cycle.ColdTargetDegC)
			}
			if len(cycle.Phases) < 4 {
				t.Fatalf("%s cycle %d has %d phases, want hot/cold ramps and dwells", tc.campaignID, cycle.Index, len(cycle.Phases))
			}
			for _, phase := range cycle.Phases {
				start, err := time.Parse(time.RFC3339, phase.Start)
				if err != nil {
					t.Fatalf("%s phase %s invalid start %q: %v", tc.campaignID, phase.ID, phase.Start, err)
				}
				end, err := time.Parse(time.RFC3339, phase.End)
				if err != nil {
					t.Fatalf("%s phase %s invalid end %q: %v", tc.campaignID, phase.ID, phase.End, err)
				}
				if !end.After(start) {
					t.Fatalf("%s phase %s has non-positive duration: %s..%s", tc.campaignID, phase.ID, phase.Start, phase.End)
				}
			}
		}
		for _, dwell := range campaign.ThermalProgram.DwellWindows {
			start, err := time.Parse(time.RFC3339, dwell.Start)
			if err != nil {
				t.Fatalf("%s dwell %s invalid start %q: %v", tc.campaignID, dwell.ID, dwell.Start, err)
			}
			end, err := time.Parse(time.RFC3339, dwell.End)
			if err != nil {
				t.Fatalf("%s dwell %s invalid end %q: %v", tc.campaignID, dwell.ID, dwell.End, err)
			}
			if !end.After(start) {
				t.Fatalf("%s dwell %s has non-positive duration: %s..%s", tc.campaignID, dwell.ID, dwell.Start, dwell.End)
			}
			if end.Sub(start) < time.Duration(dwell.MinimumMinutes)*time.Minute {
				t.Fatalf("%s dwell %s duration %s shorter than declared minimum %d min", tc.campaignID, dwell.ID, end.Sub(start), dwell.MinimumMinutes)
			}
			phase := phaseByID(campaign.ThermalProgram, strings.TrimSuffix(dwell.ID, "-WINDOW"))
			if phase.ID == "" {
				t.Fatalf("%s dwell %s references missing phase", tc.campaignID, dwell.ID)
			}
			if !start.After(mustParseTime(t, phase.Start)) {
				t.Fatalf("%s dwell %s starts at %s before stability delay inside phase %s", tc.campaignID, dwell.ID, dwell.Start, phase.Start)
			}
		}
		if len(campaign.ThermalProgram.FunctionalGates) < 4 {
			t.Fatalf("%s functional gates = %d, want pre/cold/hot/post coverage", tc.campaignID, len(campaign.ThermalProgram.FunctionalGates))
		}
		expectedCycleDwellGates := tc.wantCycles * 2
		cycleDwellGates := 0
		for _, gate := range campaign.ThermalProgram.FunctionalGates {
			if gate.CycleIndex > 0 && (gate.Gate == "cold" || gate.Gate == "hot") {
				cycleDwellGates++
				phase := phaseByID(campaign.ThermalProgram, gate.PhaseID)
				gateAt := mustParseTime(t, gate.Timestamp)
				dwell := dwellByPhaseID(campaign.ThermalProgram, phase.ID)
				if dwell.ID == "" {
					t.Fatalf("%s gate %s references phase %s without dwell window", tc.campaignID, gate.ID, phase.ID)
				}
				dwellEnd := mustParseTime(t, dwell.End)
				if gateAt.Before(dwellEnd) {
					t.Fatalf("%s gate %s at %s occurs before fixed dwell ends at %s", tc.campaignID, gate.ID, gate.Timestamp, dwellEnd.Format(time.RFC3339))
				}
			}
		}
		if cycleDwellGates != expectedCycleDwellGates {
			t.Fatalf("%s hot/cold dwell functional gates = %d, want %d", tc.campaignID, cycleDwellGates, expectedCycleDwellGates)
		}
		stabilityMarkers := 0
		for _, marker := range campaign.ThermalProgram.EvidenceMarkers {
			if marker.Kind == "stability_achieved" {
				stabilityMarkers++
			}
		}
		if stabilityMarkers != len(campaign.ThermalProgram.DwellWindows) {
			t.Fatalf("%s stability markers = %d, want one per dwell window (%d)", tc.campaignID, stabilityMarkers, len(campaign.ThermalProgram.DwellWindows))
		}
		if len(campaign.ThermalProgram.EvidenceMarkers) == 0 {
			t.Fatalf("%s missing evidence markers", tc.campaignID)
		}
		firstCycle := campaign.ThermalProgram.Cycles[0]
		wantFirstPhaseKinds := []string{"ramp_hot", "hot_survival", "ramp_hot", "hot_operational", "ramp_cold", "cold_survival", "ramp_cold", "cold_operational"}
		if len(firstCycle.Phases) < len(wantFirstPhaseKinds) {
			t.Fatalf("%s first cycle phases = %d, want ECSS-style survival and operational sequence", tc.campaignID, len(firstCycle.Phases))
		}
		for i, wantKind := range wantFirstPhaseKinds {
			if got := firstCycle.Phases[i].Kind; got != wantKind {
				t.Fatalf("%s first cycle phase %d kind = %q, want %q", tc.campaignID, i, got, wantKind)
			}
		}
		if tc.campaignID == "tvac_qualification" {
			for _, wantGate := range []string{"ambient_pre", "vacuum_pre", "vacuum_post", "post"} {
				if !functionalGateNamed(campaign.ThermalProgram, wantGate) {
					t.Fatalf("%s missing TVac pressure/ambient gate %q", tc.campaignID, wantGate)
				}
			}
		}
	}
}

func TestThermalDwellStartsOnlyAfterObservedStability(t *testing.T) {
	set := Build()
	for _, campaignID := range []string{"thermal_acceptance_fat", "tvac_qualification"} {
		campaign := set.Campaigns[campaignID]
		telemetry := set.Telemetry[campaignID]
		for _, dwell := range campaign.ThermalProgram.DwellWindows {
			start := mustParseTime(t, dwell.Start)
			windowStart := start.Add(-60 * time.Minute)
			for _, signalID := range []string{"chamber_air_deg_c", "interface_plate_deg_c", "thermal_zone_1_deg_c", "thermal_zone_2_deg_c"} {
				minValue := math.Inf(1)
				maxValue := math.Inf(-1)
				count := 0
				for _, sample := range telemetry {
					ts := mustParseTime(t, sample.Timestamp)
					if ts.Before(windowStart) || ts.After(start) {
						continue
					}
					value, ok := sample.Signals[signalID]
					if !ok {
						continue
					}
					minValue = math.Min(minValue, value)
					maxValue = math.Max(maxValue, value)
					count++
				}
				if count < 4 {
					t.Fatalf("%s dwell %s has only %d samples for %s in pre-stability window", campaignID, dwell.ID, count, signalID)
				}
				allowedDrift := dwell.StabilityBandC + 0.25
				if drift := maxValue - minValue; drift > allowedDrift {
					t.Fatalf("%s dwell %s signal %s drifts %.2f C in the hour before stability, want <= %.2f C", campaignID, dwell.ID, signalID, drift, allowedDrift)
				}
			}
		}
	}
}

func phaseByID(program *contracts.ThermalProgram, phaseID string) contracts.CyclePhase {
	for _, cycle := range program.Cycles {
		for _, phase := range cycle.Phases {
			if phase.ID == phaseID {
				return phase
			}
		}
	}
	return contracts.CyclePhase{}
}

func dwellByPhaseID(program *contracts.ThermalProgram, phaseID string) contracts.DwellWindow {
	for _, dwell := range program.DwellWindows {
		if strings.TrimSuffix(dwell.ID, "-WINDOW") == phaseID {
			return dwell
		}
	}
	return contracts.DwellWindow{}
}

func functionalGateNamed(program *contracts.ThermalProgram, gate string) bool {
	for _, functionalGate := range program.FunctionalGates {
		if functionalGate.Gate == gate {
			return true
		}
	}
	return false
}

func TestThermalTelemetryCarriesCyclePhaseAndFacilitySafetySignals(t *testing.T) {
	set := Build()
	requiredSignals := []string{
		"thermal_cycle_index",
		"thermal_phase_code",
		"chamber_setpoint_deg_c",
		"chamber_air_deg_c",
		"thermal_zone_1_deg_c",
		"thermal_zone_2_deg_c",
		"huber_table_deg_c",
		"ln2_line_temp_deg_c",
		"ln2_valve_duty_pct",
		"cooling_water_freeze_margin_deg_c",
		"payload_sim_heater_w",
		"tm_packet_counter",
		"tc_packet_counter",
		"functional_gate_code",
		"facility_interlock_code",
		"dut_survival_mode",
		"stability_gate_reached",
		"dwell_active",
		"dwell_complete",
		"pressure_gate_reached",
		"overall_packet_counter",
		"cooling_water_temp_deg_c",
		"pressurized_air_supply_bar",
		"air_dewpoint_deg_c",
	}

	for _, campaignID := range []string{"thermal_acceptance_fat", "tvac_qualification"} {
		samples := set.Telemetry[campaignID]
		program := set.Campaigns[campaignID].ThermalProgram
		if len(samples) <= program.CycleCount*20 {
			t.Fatalf("%s telemetry count = %d, want high-detail cycle coverage beyond %d", campaignID, len(samples), program.CycleCount*20)
		}
		cycles := map[int]bool{}
		gates := map[string]bool{}
		for _, sample := range samples {
			for _, signal := range requiredSignals {
				if _, ok := sample.Signals[signal]; !ok {
					t.Fatalf("%s sample %s missing %s", campaignID, sample.Timestamp, signal)
				}
			}
			if campaignID == "tvac_qualification" {
				if _, ok := sample.Signals["tvac_pressure_pa"]; !ok {
					t.Fatalf("%s sample %s missing tvac_pressure_pa", campaignID, sample.Timestamp)
				}
			}
			cycle := int(sample.Signals["thermal_cycle_index"])
			if cycle > 0 {
				cycles[cycle] = true
			}
			if gate := sample.States["functional_gate"]; gate != "" && gate != "none" {
				gates[gate] = true
			}
			if phase := sample.States["thermal_phase"]; phase == "" {
				t.Fatalf("%s sample %s missing thermal_phase state", campaignID, sample.Timestamp)
			}
			if state := sample.States["facility_interlock_state"]; state == "" {
				t.Fatalf("%s sample %s missing facility_interlock_state", campaignID, sample.Timestamp)
			}
		}
		if got := len(cycles); got != program.CycleCount {
			t.Fatalf("%s observed cycles = %d, want %d", campaignID, got, program.CycleCount)
		}
		wantGates := []string{"pre", "cold", "hot", "post"}
		if campaignID == "tvac_qualification" {
			wantGates = []string{"ambient_pre", "vacuum_pre", "cold", "hot", "vacuum_post", "post"}
		}
		for _, wantGate := range wantGates {
			if !gates[wantGate] {
				t.Fatalf("%s missing observed functional gate %q", campaignID, wantGate)
			}
		}
	}
}

func TestTileManifestExposesBackendOwnedTileArchitecture(t *testing.T) {
	set := Build()
	for _, campaignID := range []string{"thermal_acceptance_fat", "tvac_qualification"} {
		model := set.GraphModels[campaignID]
		if err := contracts.ValidateGraphModel(model); err != nil {
			t.Fatalf("%s graph model invalid: %v", campaignID, err)
		}
		if model.TileManifest == nil {
			t.Fatalf("%s missing tile manifest", campaignID)
		}
		manifest := model.TileManifest
		requiredLevels := map[string]bool{"year": false, "month": false, "day": false, "hour": false, "minute": false, "second": false, "millisecond": false}
		for _, level := range manifest.Levels {
			if _, ok := requiredLevels[level.ID]; ok {
				requiredLevels[level.ID] = true
			}
		}
		for level, seen := range requiredLevels {
			if !seen {
				t.Fatalf("%s tile manifest missing %s level", campaignID, level)
			}
		}
		renderKinds := map[string]bool{}
		byCard := map[string]contracts.GraphTileCardRef{}
		for _, card := range manifest.Cards {
			renderKinds[card.RenderKind] = true
			byCard[card.CardID] = card
			if !card.Collapsible || !card.SupportsTimeZoom {
				t.Fatalf("%s card %s must be collapsible and time-zoomable", campaignID, card.CardID)
			}
			if card.TileEndpoint == "" || card.LatestEndpoint == "" {
				t.Fatalf("%s card %s missing tile endpoints", campaignID, card.CardID)
			}
		}
		for _, kind := range []string{"line", "counter", "swimlane", "event_rail"} {
			if !renderKinds[kind] {
				t.Fatalf("%s tile manifest missing render kind %s", campaignID, kind)
			}
		}
		hero := byCard["thermal_program"]
		heroSignals := map[string]bool{}
		for _, signal := range hero.Signals {
			heroSignals[signal.ID] = true
		}
		for _, signalID := range []string{"trace.command.chamber", "trace.ghost.profile", "trace.actual.chamber_air", "trace.dut_temp_a", "trace.dut_temp_b", "trace.table_loop"} {
			if !heroSignals[signalID] {
				t.Fatalf("%s hero tile missing signal %s", campaignID, signalID)
			}
		}
		if campaignID == "tvac_qualification" && !heroSignals["trace.tvac_pressure"] {
			t.Fatalf("%s hero tile missing pressure signal", campaignID)
		}
		if len(manifest.DataLensTranslations) < 3 {
			t.Fatalf("%s datalens translations = %d, want CSV, binary, and HDF5-like examples", campaignID, len(manifest.DataLensTranslations))
		}
		if len(manifest.EvidenceLinks) == 0 {
			t.Fatalf("%s missing evidence links", campaignID)
		}
	}
}

func TestThermalGraphModelsExposeLoomGradeHeroContract(t *testing.T) {
	set := Build()
	for _, campaignID := range []string{"thermal_acceptance_fat", "tvac_qualification"} {
		model := set.GraphModels[campaignID]
		if model.SimulationProvenance == nil {
			t.Fatalf("%s missing simulation provenance", campaignID)
		}
		if model.SimulationProvenance.Model != environmentalsim.ModelName {
			t.Fatalf("%s simulation model = %q, want %s", campaignID, model.SimulationProvenance.Model, environmentalsim.ModelName)
		}
		if model.HeroGraph == nil {
			t.Fatalf("%s missing backend-owned hero graph", campaignID)
		}
		if model.HeroGraph.TimeAxis.Start == "" || model.HeroGraph.TimeAxis.End == "" {
			t.Fatalf("%s hero graph missing absolute time axis", campaignID)
		}
		if model.HeroGraph.Execution == nil {
			t.Fatalf("%s hero graph missing accelerated execution cursor", campaignID)
		}
		exec := model.HeroGraph.Execution
		if exec.Now == "" || model.HeroGraph.TimeAxis.Now != exec.Now {
			t.Fatalf("%s execution cursor not mirrored into graph time axis", campaignID)
		}
		now := mustParseTime(t, exec.Now)
		if !now.After(mustParseTime(t, model.HeroGraph.TimeAxis.Start)) || !now.Before(mustParseTime(t, model.HeroGraph.TimeAxis.End)) {
			t.Fatalf("%s execution cursor %s outside graph time range", campaignID, exec.Now)
		}
		if exec.PercentComplete < 55 || exec.PercentComplete > 65 {
			t.Fatalf("%s execution percent = %.1f, want around 60%%", campaignID, exec.PercentComplete)
		}
		if exec.TargetCycles != set.Campaigns[campaignID].ThermalProgram.CycleCount {
			t.Fatalf("%s execution target cycles = %d, want campaign cycle count", campaignID, exec.TargetCycles)
		}
		if len(exec.RequirementProgress) < 3 {
			t.Fatalf("%s requirement progress entries = %d, want cycle/dwell/gate progress", campaignID, len(exec.RequirementProgress))
		}
		cycleProgress := requirementProgress(exec.RequirementProgress, "REQ-CYCLE-COUNT")
		if cycleProgress == nil {
			t.Fatalf("%s missing cycle-count requirement progress", campaignID)
		}
		if cycleProgress.Completed <= 0 || cycleProgress.Completed >= cycleProgress.Target {
			t.Fatalf("%s cycle progress = %d/%d, want in-progress live execution", campaignID, cycleProgress.Completed, cycleProgress.Target)
		}
		if len(cycleProgress.Contributors) != cycleProgress.Completed {
			t.Fatalf("%s cycle progress contributors = %d, want one per completed cycle", campaignID, len(cycleProgress.Contributors))
		}
		requiredRoles := map[string]bool{
			"command":         false,
			"ghost":           false,
			"actual":          false,
			"acceptance_band": false,
			"event":           false,
			"interlock":       false,
			"evidence":        false,
		}
		for _, trace := range model.HeroGraph.Traces {
			if _, ok := requiredRoles[trace.Role]; ok {
				requiredRoles[trace.Role] = true
			}
			if len(trace.Values) < 160 {
				t.Fatalf("%s trace %s values = %d, want high-detail graph data", campaignID, trace.ID, len(trace.Values))
			}
		}
		for role, seen := range requiredRoles {
			if !seen {
				t.Fatalf("%s missing hero trace role %q", campaignID, role)
			}
		}
		if len(model.HeroGraph.PhaseBands) < set.Campaigns[campaignID].ThermalProgram.CycleCount*4 {
			t.Fatalf("%s phase bands = %d, want at least one per thermal phase", campaignID, len(model.HeroGraph.PhaseBands))
		}
		if len(model.HeroGraph.CompanionGroups) < 4 {
			t.Fatalf("%s companion groups = %d, want thermal, safety, power, bus/pressure groups", campaignID, len(model.HeroGraph.CompanionGroups))
		}
		dutGroup := companionGroup(model.HeroGraph.CompanionGroups, "dut_temperature_response")
		if dutGroup == nil {
			t.Fatalf("%s missing DUT temperature response companion group", campaignID)
		}
		for _, wantTrace := range []string{"Chamber air", "High-dissipation DUT node", "Vacuum-detached DUT node"} {
			if !companionTraceLabel(*dutGroup, wantTrace) {
				t.Fatalf("%s DUT temperature group missing %q context trace", campaignID, wantTrace)
			}
		}
		actuationGroup := companionGroup(model.HeroGraph.CompanionGroups, "facility_actuation")
		if actuationGroup == nil {
			t.Fatalf("%s missing facility actuation companion group", campaignID)
		}
		assertCompanionTraceUnits(t, campaignID, *actuationGroup, "%")
		safetyGroup := companionGroup(model.HeroGraph.CompanionGroups, "facility_temperature_safety")
		if campaignID == "tvac_qualification" {
			if safetyGroup == nil {
				t.Fatalf("%s missing facility temperature safety companion group", campaignID)
			}
			assertCompanionTraceUnits(t, campaignID, *safetyGroup, "degC")
		} else if safetyGroup != nil {
			t.Fatalf("%s must not publish freeze-margin safety graph on Thermal Chamber FAT", campaignID)
		}
		powerGroup := companionGroup(model.HeroGraph.CompanionGroups, "dut_power_response")
		if powerGroup == nil {
			t.Fatalf("%s missing DUT power budget companion group", campaignID)
		}
		assertCompanionTraceUnits(t, campaignID, *powerGroup, "W")
		for _, wantTrace := range []string{"Total power", "Subsystem budget", "Payload/FT load"} {
			if !companionTraceLabel(*powerGroup, wantTrace) {
				t.Fatalf("%s power group missing %q trace", campaignID, wantTrace)
			}
		}
		stateGroup := companionGroup(model.HeroGraph.CompanionGroups, "state_change_swimlane")
		if stateGroup == nil {
			t.Fatalf("%s missing state change swimlane companion group", campaignID)
		}
		if len(stateGroup.Axes) == 0 || stateGroup.Axes[0].Scale != "step" {
			t.Fatalf("%s state swimlane axis scale = %q, want step", campaignID, stateGroup.Axes[0].Scale)
		}
		for _, wantTrace := range []string{"Thermal phase", "Functional gate", "Stability reached", "Dwell active", "Dwell complete", "Interlock review", "Source degraded", "Evidence capture"} {
			if !companionTraceLabel(*stateGroup, wantTrace) {
				t.Fatalf("%s state swimlane missing %q trace", campaignID, wantTrace)
			}
		}
		if campaignID == "tvac_qualification" && !companionTraceLabel(*stateGroup, "Pressure gate") {
			t.Fatalf("%s state swimlane missing pressure gate trace", campaignID)
		}
		if campaignID == "tvac_qualification" {
			pressureGroup := companionGroup(model.HeroGraph.CompanionGroups, "tvac_pressure_response")
			if pressureGroup == nil {
				t.Fatalf("%s missing TVac pressure companion group", campaignID)
			}
			if len(pressureGroup.Axes) == 0 || pressureGroup.Axes[0].Scale != "log10" {
				t.Fatalf("%s TVac pressure axis scale = %q, want log10", campaignID, pressureGroup.Axes[0].Scale)
			}
		}
	}
}

func TestThermalGraphModelsExposeOperatorGraphWallContract(t *testing.T) {
	set := Build()
	cases := []struct {
		campaignID       string
		wantPressureCard bool
	}{
		{campaignID: "thermal_acceptance_fat"},
		{campaignID: "tvac_qualification", wantPressureCard: true},
	}

	for _, tc := range cases {
		model := set.GraphModels[tc.campaignID]
		if model.GraphWall == nil {
			t.Fatalf("%s missing backend-owned graph wall", tc.campaignID)
		}
		wall := model.GraphWall
		if wall.GraphVersion == "" || wall.SourceMode == "" {
			t.Fatalf("%s graph wall missing version/source mode", tc.campaignID)
		}
		if wall.TimeRange.Start == "" || wall.TimeRange.End == "" {
			t.Fatalf("%s graph wall missing absolute time range", tc.campaignID)
		}
		if wall.TilePolicy.DefaultPoints <= 0 || wall.TilePolicy.MaxPoints < wall.TilePolicy.DefaultPoints {
			t.Fatalf("%s graph wall tile policy is not usable: %+v", tc.campaignID, wall.TilePolicy)
		}
		if len(wall.GraphGroups) == 0 {
			t.Fatalf("%s graph wall missing graph groups", tc.campaignID)
		}
		if len(wall.Sections) < 3 {
			t.Fatalf("%s graph wall sections = %d, want thermal/power/tmtc/state coverage", tc.campaignID, len(wall.Sections))
		}

		cards := graphWallCards(*wall)
		requiredCards := map[string]string{
			"thermal_program":       "line",
			"dut_temperature":       "line",
			"facility_actuation":    "line",
			"dut_power":             "line",
			"tmtc_health":           "line",
			"tmtc_counters":         "counter",
			"state_change_swimlane": "state",
			"functional_events":     "event",
		}
		for cardID, wantKind := range requiredCards {
			card, ok := cards[cardID]
			if !ok {
				t.Fatalf("%s graph wall missing card %q", tc.campaignID, cardID)
			}
			if card.Kind != wantKind {
				t.Fatalf("%s graph wall card %s kind = %q, want %q", tc.campaignID, cardID, card.Kind, wantKind)
			}
			if card.Placement.SectionID == "" || card.Placement.GroupID == "" {
				t.Fatalf("%s graph wall card %s missing backend placement", tc.campaignID, cardID)
			}
			if card.AxisPolicy == "" {
				t.Fatalf("%s graph wall card %s missing axis policy", tc.campaignID, cardID)
			}
			if len(card.Signals) == 0 {
				t.Fatalf("%s graph wall card %s missing signals", tc.campaignID, cardID)
			}
		}
		if _, ok := cards["facility_temperature_safety"]; ok != tc.wantPressureCard {
			t.Fatalf("%s freeze-margin card presence = %v, want %v", tc.campaignID, ok, tc.wantPressureCard)
		}
		if _, ok := cards["source_quality"]; !ok {
			t.Fatalf("%s graph wall must split freshness ms away from packet counters", tc.campaignID)
		}
		if _, ok := cards["tvac_pressure"]; ok != tc.wantPressureCard {
			t.Fatalf("%s pressure card presence = %v, want %v", tc.campaignID, ok, tc.wantPressureCard)
		}
		power := cards["dut_power"]
		if power.Unit != "W" || power.AxisPolicy != "power_w" {
			t.Fatalf("%s DUT power card unit/axis = %s/%s, want W/power_w", tc.campaignID, power.Unit, power.AxisPolicy)
		}
		for _, wantSignal := range []string{"trace.power_total", "trace.power_subsystem", "trace.power_payload"} {
			if !graphCardSignal(power, wantSignal) {
				t.Fatalf("%s DUT power card missing signal %q", tc.campaignID, wantSignal)
			}
		}
		state := cards["state_change_swimlane"]
		for _, signal := range state.Signals {
			if signal.Kind != "enum" && signal.Kind != "bool" && signal.Kind != "fault" {
				t.Fatalf("%s state swimlane signal %s kind = %q, want enum/bool/fault", tc.campaignID, signal.ID, signal.Kind)
			}
		}
		counter := cards["tmtc_counters"]
		for _, signal := range counter.Signals {
			if signal.Kind != "counter" {
				t.Fatalf("%s counter card signal %s kind = %q, want counter", tc.campaignID, signal.ID, signal.Kind)
			}
		}
	}
}

func TestThermalSimulationBehavesLikeThermalHardware(t *testing.T) {
	set := Build()
	fat := set.Telemetry["thermal_acceptance_fat"]
	tvac := set.Telemetry["tvac_qualification"]

	var rampLagSeen, dwellSettlingSeen, ln2SaturationSeen, gateLoadSeen, gateThermalSeen bool
	var previousDwellError float64
	for i, sample := range fat {
		setpoint := sample.Signals["chamber_setpoint_deg_c"]
		air := sample.Signals["chamber_air_deg_c"]
		zone := sample.Signals["thermal_zone_1_deg_c"]
		if sample.States["thermal_phase"] == "ramp_hot" && setpoint-air > 1.5 && air-zone > 1.0 {
			rampLagSeen = true
		}
		if sample.States["thermal_phase"] == "hot_operational" && sample.States["stability_state"] == "stable" {
			err := abs(setpoint - zone)
			if previousDwellError > 0 && err < previousDwellError {
				dwellSettlingSeen = true
			}
			previousDwellError = err
		}
		if sample.States["functional_gate"] != "none" && sample.Signals["eps_bus_current_a"] > 5.0 && sample.Signals["bus_latency_ms"] > 28 {
			gateLoadSeen = true
		}
		if sample.States["functional_gate"] != "none" && i > 0 && sample.Signals["payload_sim_heater_w"] > fat[i-1].Signals["payload_sim_heater_w"] {
			gateThermalSeen = true
		}
		if i > 0 && sample.States["thermal_phase"] == "ramp_cold" && sample.Signals["ln2_valve_duty_pct"] > 80 {
			ln2SaturationSeen = true
		}
	}
	if !rampLagSeen {
		t.Fatal("expected chamber/article lag during a hot ramp")
	}
	if !dwellSettlingSeen {
		t.Fatal("expected article dwell error to settle toward the target")
	}
	if !ln2SaturationSeen {
		t.Fatal("expected LN2 duty saturation during a cold ramp")
	}
	if !gateLoadSeen {
		t.Fatal("expected functional gates to increase electrical load and bus latency")
	}
	if !gateThermalSeen {
		t.Fatal("expected functional gates to create visible DUT thermal/load response")
	}
	assertAnalogContinuity(t, "thermal_acceptance_fat", fat, "chamber_setpoint_deg_c", 8.0)
	assertAnalogContinuity(t, "thermal_acceptance_fat", fat, "chamber_air_deg_c", 8.0)
	assertAnalogContinuity(t, "thermal_acceptance_fat", fat, "thermal_zone_1_deg_c", 5.0)

	if len(tvac) < 12 {
		t.Fatal("tvac telemetry too short")
	}
	startPressure := tvac[0].Signals["tvac_pressure_pa"]
	midPressure := tvac[len(tvac)/3].Signals["tvac_pressure_pa"]
	var vacuumPostPressure float64
	for _, sample := range tvac {
		if sample.States["functional_gate"] == "vacuum_post" {
			vacuumPostPressure = sample.Signals["tvac_pressure_pa"]
			break
		}
	}
	endPressure := tvac[len(tvac)-2].Signals["tvac_pressure_pa"]
	if !(startPressure > 100000 && midPressure < 10 && vacuumPostPressure < 0.05 && endPressure > 90000) {
		t.Fatalf("unexpected TVac pumpdown profile: start %.4f mid %.4f end %.4f", startPressure, midPressure, endPressure)
	}
	var pressureBurstSeen bool
	for i := 1; i < len(tvac); i++ {
		if tvac[i].Signals["tvac_pressure_pa"] > tvac[i-1].Signals["tvac_pressure_pa"]*1.7 && tvac[i].Signals["tvac_pressure_pa"] < 5 {
			pressureBurstSeen = true
			break
		}
	}
	if !pressureBurstSeen {
		t.Fatal("expected thermal-transition/outgassing pressure burst in TVac data")
	}
	assertAnalogContinuity(t, "tvac_qualification", tvac, "chamber_setpoint_deg_c", 8.0)
	assertAnalogContinuity(t, "tvac_qualification", tvac, "chamber_air_deg_c", 8.0)
	assertAnalogContinuity(t, "tvac_qualification", tvac, "thermal_zone_1_deg_c", 5.0)
}

func TestThermalGraphWallExposesSubsystemStatusSwimlanes(t *testing.T) {
	set := Build()
	for _, campaignID := range []string{"thermal_acceptance_fat", "tvac_qualification"} {
		model := set.GraphModels[campaignID]
		stateGroup := companionGroup(model.HeroGraph.CompanionGroups, "state_change_swimlane")
		if stateGroup == nil {
			t.Fatalf("%s missing state change swimlane", campaignID)
		}
		for _, wantTrace := range []string{"DUT ready", "DUT operative", "Payload active", "RF link locked", "Fault flag"} {
			if !companionTraceLabel(*stateGroup, wantTrace) {
				t.Fatalf("%s state swimlane missing %q trace", campaignID, wantTrace)
			}
		}
		cards := graphWallCards(*model.GraphWall)
		stateCard := cards["state_change_swimlane"]
		for _, wantSignal := range []string{"trace.dut_ready", "trace.dut_operative", "trace.payload_active", "trace.rf_link_locked", "trace.fault_flag", "trace.stability_reached", "trace.dwell_active", "trace.dwell_complete"} {
			if !graphCardSignal(stateCard, wantSignal) {
				t.Fatalf("%s state graph card missing swimlane signal %q", campaignID, wantSignal)
			}
		}
	}
}

func TestTVacModelsExhaustColdScavengerAndFreezeRisk(t *testing.T) {
	set := Build()
	samples := set.Telemetry["tvac_qualification"]
	if len(samples) == 0 {
		t.Fatal("tvac telemetry missing")
	}

	var (
		minCryoExhaust     = math.Inf(1)
		minSafeExhaust     = math.Inf(1)
		maxWaterReturn     float64
		maxRecovery        float64
		ductSafeObserved   bool
		ductUnsafeObserved bool
	)
	for _, sample := range samples {
		cryo, ok := sample.Signals["tvac_cryo_exhaust_temp_deg_c"]
		if !ok {
			t.Fatalf("sample %s missing cryogenic exhaust temperature", sample.Timestamp)
		}
		safe, ok := sample.Signals["tvac_scavenged_exhaust_temp_deg_c"]
		if !ok {
			t.Fatalf("sample %s missing scavenged exhaust temperature", sample.Timestamp)
		}
		waterReturn, ok := sample.Signals["tvac_scavenger_cooling_water_return_deg_c"]
		if !ok {
			t.Fatalf("sample %s missing scavenger cooling-water return temperature", sample.Timestamp)
		}
		recovery, ok := sample.Signals["tvac_exhaust_cold_recovery_pct"]
		if !ok {
			t.Fatalf("sample %s missing exhaust cold recovery", sample.Timestamp)
		}
		ductSafe, ok := sample.Signals["tvac_exhaust_duct_safe"]
		if !ok {
			t.Fatalf("sample %s missing exhaust duct-safe state", sample.Timestamp)
		}
		minCryoExhaust = math.Min(minCryoExhaust, cryo)
		minSafeExhaust = math.Min(minSafeExhaust, safe)
		maxWaterReturn = math.Max(maxWaterReturn, waterReturn)
		maxRecovery = math.Max(maxRecovery, recovery)
		ductSafeObserved = ductSafeObserved || ductSafe > 0
		ductUnsafeObserved = ductUnsafeObserved || ductSafe == 0
	}
	if minCryoExhaust > -45 {
		t.Fatalf("cryogenic exhaust min = %.1f degC, want visibly cryogenic before scavenger", minCryoExhaust)
	}
	if minSafeExhaust < 4 {
		t.Fatalf("scavenged exhaust min = %.1f degC, want normal duct-safe exhaust after water scavenger", minSafeExhaust)
	}
	if maxWaterReturn < 18 || maxRecovery < 35 {
		t.Fatalf("weak cold scavenger response: water return %.1f degC recovery %.1f%%", maxWaterReturn, maxRecovery)
	}
	if !ductSafeObserved || !ductUnsafeObserved {
		t.Fatalf("duct-safe swimlane did not transition: safe=%v unsafe=%v", ductSafeObserved, ductUnsafeObserved)
	}

	model := set.GraphModels["tvac_qualification"]
	exhaustGroup := companionGroup(model.HeroGraph.CompanionGroups, "tvac_exhaust_scavenger")
	if exhaustGroup == nil {
		t.Fatal("missing TVac exhaust cold-scavenger companion group")
	}
	for _, wantTrace := range []string{"Cryogenic exhaust", "After water scavenger", "Scavenger water return", "Cold recovery"} {
		if !companionTraceLabel(*exhaustGroup, wantTrace) {
			t.Fatalf("exhaust scavenger group missing %q trace", wantTrace)
		}
	}
	stateGroup := companionGroup(model.HeroGraph.CompanionGroups, "state_change_swimlane")
	if stateGroup == nil || !companionTraceLabel(*stateGroup, "Exhaust duct safe") {
		t.Fatal("state swimlane missing exhaust duct-safe trace")
	}
	cards := graphWallCards(*model.GraphWall)
	exhaustCard, ok := cards["tvac_exhaust_scavenger"]
	if !ok {
		t.Fatal("graph wall missing exhaust cold-scavenger card")
	}
	for _, wantSignal := range []string{"trace.tvac_cryo_exhaust", "trace.tvac_scavenged_exhaust", "trace.tvac_scavenger_water_return", "trace.tvac_exhaust_cold_recovery"} {
		if !graphCardSignal(exhaustCard, wantSignal) {
			t.Fatalf("exhaust graph card missing signal %q", wantSignal)
		}
	}
	stateCard := cards["state_change_swimlane"]
	if !graphCardSignal(stateCard, "trace.exhaust_duct_safe") {
		t.Fatal("state graph card missing exhaust duct-safe swimlane signal")
	}
}

func TestSupervisorLanesExposeThermalProgramsAndDenseHeroGraphs(t *testing.T) {
	set := Build()
	required := map[string]int{
		"thermal_fat":        4,
		"tvac_qualification": 8,
	}
	for _, lane := range set.Supervisor.Lanes {
		wantCycles, ok := required[lane.ID]
		if !ok {
			continue
		}
		if lane.ThermalProgram == nil {
			t.Fatalf("lane %s missing thermal program", lane.ID)
		}
		if got := lane.ThermalProgram.CycleCount; got != wantCycles {
			t.Fatalf("lane %s cycle count = %d, want %d", lane.ID, got, wantCycles)
		}
		if len(lane.HeroGraphs) < 4 {
			t.Fatalf("lane %s hero graphs = %d, want synchronized multi-hero set", lane.ID, len(lane.HeroGraphs))
		}
		for _, graph := range lane.HeroGraphs {
			if len(graph.Values) < 40 {
				t.Fatalf("lane %s graph %s values = %d, want dense hero graph", lane.ID, graph.ID, len(graph.Values))
			}
		}
		if len(lane.FunctionalGates) < 4 {
			t.Fatalf("lane %s functional gates = %d, want pre/cold/hot/post", lane.ID, len(lane.FunctionalGates))
		}
		if len(lane.EvidenceMarkers) == 0 {
			t.Fatalf("lane %s missing evidence markers", lane.ID)
		}
		if lane.HeroGraph == nil {
			t.Fatalf("lane %s missing backend-owned hero graph contract", lane.ID)
		}
		if lane.HeroGraph.Owner != "gossamer_backend_fixture_generator" {
			t.Fatalf("lane %s hero owner = %q", lane.ID, lane.HeroGraph.Owner)
		}
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func assertAnalogContinuity(t *testing.T, campaignID string, samples []contracts.TelemetrySample, signal string, maxStep float64) {
	t.Helper()
	for i := 1; i < len(samples); i++ {
		delta := abs(samples[i].Signals[signal] - samples[i-1].Signals[signal])
		if delta > maxStep {
			t.Fatalf("%s %s jump at %s: %.2f -> %.2f delta %.2f > %.2f", campaignID, signal, samples[i].Timestamp, samples[i-1].Signals[signal], samples[i].Signals[signal], delta, maxStep)
		}
	}
}

func graphCardSignal(card contracts.GraphWallCard, id string) bool {
	for _, signal := range card.Signals {
		if signal.ID == id {
			return true
		}
	}
	return false
}

func companionGroup(groups []contracts.CompanionGraphGroup, id string) *contracts.CompanionGraphGroup {
	for i := range groups {
		if groups[i].ID == id {
			return &groups[i]
		}
	}
	return nil
}

func companionTraceLabel(group contracts.CompanionGraphGroup, label string) bool {
	for _, trace := range group.Traces {
		if trace.Label == label {
			return true
		}
	}
	return false
}

func assertCompanionTraceUnits(t *testing.T, campaignID string, group contracts.CompanionGraphGroup, units string) {
	t.Helper()
	for _, trace := range group.Traces {
		if trace.Units != units {
			t.Fatalf("%s group %s trace %s units = %q, want %q", campaignID, group.ID, trace.ID, trace.Units, units)
		}
	}
}

func graphWallCards(wall contracts.GraphWallModel) map[string]contracts.GraphWallCard {
	cards := map[string]contracts.GraphWallCard{}
	for _, section := range wall.Sections {
		for _, card := range section.Cards {
			cards[card.ID] = card
		}
	}
	return cards
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func requirementProgress(progress []contracts.RequirementProgress, id string) *contracts.RequirementProgress {
	for i := range progress {
		if progress[i].ID == id {
			return &progress[i]
		}
	}
	return nil
}

func TestIntegratedSystemFATIncludesDegradedSourceInterval(t *testing.T) {
	set := Build()
	found := false
	for _, sample := range set.Telemetry["integrated_system_fat"] {
		if sample.Quality == "degraded" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected degraded interval")
	}
}

func TestBuildProducesSupervisorOverviewWithHeroTemperatureGraphs(t *testing.T) {
	set := Build()
	if err := contracts.ValidateSupervisorOverview(set.Supervisor); err != nil {
		t.Fatalf("supervisor overview: %v", err)
	}
	if len(set.Supervisor.Lanes) < 4 {
		t.Fatalf("supervisor lanes = %d, want at least 4", len(set.Supervisor.Lanes))
	}
	foundTemperature := false
	for _, lane := range set.Supervisor.Lanes {
		if len(lane.HeroGraphs) == 0 {
			t.Fatalf("lane %s has no hero graphs", lane.ID)
		}
		for _, graph := range lane.HeroGraphs {
			if graph.Units == "degC" && len(graph.Values) > 0 {
				foundTemperature = true
			}
		}
	}
	if !foundTemperature {
		t.Fatal("expected at least one temperature hero graph")
	}
}

func TestBuildProducesBusVirtualizationWithTMAndTCEvents(t *testing.T) {
	set := Build()
	if err := contracts.ValidateBusVirtualizationTap(set.BusTap); err != nil {
		t.Fatalf("bus tap: %v", err)
	}
	var tm, tc bool
	for _, event := range set.BusTap.Events {
		if event.Direction == "TM" {
			tm = true
		}
		if event.Direction == "TC" {
			tc = true
		}
	}
	if !tm || !tc {
		t.Fatalf("expected both TM and TC bus events, got TM=%t TC=%t", tm, tc)
	}
}

func TestTelemetryIncludesRicherBusAndThermalSignalsWithinPlausibleBounds(t *testing.T) {
	set := Build()
	for campaignID, samples := range set.Telemetry {
		for _, sample := range samples {
			for _, signal := range []string{"bus_latency_ms", "tm_packet_counter", "tc_packet_counter"} {
				if _, ok := sample.Signals[signal]; !ok {
					t.Fatalf("%s sample %s missing %s", campaignID, sample.Timestamp, signal)
				}
			}
			if got := sample.Signals["thermal_zone_1_deg_c"]; got < -60 || got > 95 {
				t.Fatalf("%s thermal_zone_1_deg_c = %.2f outside plausible demo bounds", campaignID, got)
			}
			if got := sample.Signals["eps_bus_voltage_v"]; got < 24 || got > 32 {
				t.Fatalf("%s eps_bus_voltage_v = %.2f outside plausible demo bounds", campaignID, got)
			}
			if got := sample.Signals["bus_latency_ms"]; got < 0 || got > 250 {
				t.Fatalf("%s bus_latency_ms = %.2f outside plausible demo bounds", campaignID, got)
			}
		}
	}
}

func TestWritePublicFixturesCreatesDeterministicFiles(t *testing.T) {
	dir := t.TempDir()
	if err := WritePublicFixtures(dir); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "fixtures", "public", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := WritePublicFixtures(dir); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(dir, "fixtures", "public", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("fixture generation changed between runs")
	}
	for _, rel := range []string{
		"fixtures/public/supervisor_overview.json",
		"fixtures/public/bus_virtualization_tap.json",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("expected %s to be written: %v", rel, err)
		}
	}
}
