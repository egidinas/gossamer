package commandcenter

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/egidinas/gossamer/internal/environmentalsim"
	"github.com/egidinas/signalforge/contracts"
	sharedgraph "github.com/egidinas/signalforge/graphwall"
)

// BuildFATBundle is the entry point called from generator.go.
// buildThermalProgramFn must be the parent synthetic.buildThermalProgram function.
func BuildFATBundle(
	env contracts.Envelope,
	fixedTime time.Time,
	campaignID string,
	buildThermalProgramFn func(campaignID, facility string, cycleCount int, coldTarget, hotTarget float64) *contracts.ThermalProgram,
) (contracts.CommandCenterFAT, []contracts.TelemetrySample, contracts.GraphModel) {
	baseProgram := buildThermalProgramFn(campaignID, "multi_chamber_fat", 4, -35, 65)
	baseSim := environmentalsim.Simulate(campaignID, baseProgram, fixedTime)
	baseStart := mustTime(baseProgram.Cycles[0].Start)
	baseEnd := mustTime(baseProgram.Cycles[len(baseProgram.Cycles)-1].End).Add(thermalContextDuration(baseProgram))
	runDuration := baseEnd.Sub(baseStart)
	model := buildCommandCenterFAT(env, fixedTime, campaignID, runDuration)
	samples, graph := buildCommandCenterGraphModel(env, fixedTime, campaignID, model, baseProgram, baseSim, baseStart)
	model.HeroGraph = graph.HeroGraph
	model.GraphWall = graph.GraphWall
	return model, samples, graph
}

func buildCommandCenterFAT(env contracts.Envelope, fixedTime time.Time, campaignID string, runDuration time.Duration) contracts.CommandCenterFAT {
	windowStart := time.Date(fixedTime.Year(), fixedTime.Month(), fixedTime.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -14)
	windowEnd := windowStart.AddDate(0, 0, 28)
	dataStart := windowStart.AddDate(0, 0, -28)
	dataEnd := windowEnd.AddDate(0, 0, 28)
	chambers := []struct {
		id, name, facility string
		offsetDays         int
	}{
		{"thermal_chamber_alpha", "Alpha", "TC-A", 0},
		{"thermal_chamber_bravo", "Bravo", "TC-B", 1},
		{"thermal_chamber_charlie", "Charlie", "TC-C", 2},
		{"thermal_chamber_delta", "Delta", "TC-D", 3},
	}
	lanes := make([]contracts.CommandCenterLane, 0, len(chambers))
	assignedStarts := []time.Time{}
	assignedFinishes := []time.Time{}
	assignedBreakdowns := []time.Time{}
	assignedResets := []time.Time{}
	for laneIndex, chamber := range chambers {
		start := commandCenterTestStartAfterPrep(workdayAt(dataStart.AddDate(0, 0, chamber.offsetDays), commandCenterPreferredStartHour(laneIndex, 0)), laneIndex, 0)
		runs := []contracts.CommandCenterRun{}
		for runIndex := 0; runIndex < 256; runIndex++ {
			start = commandCenterDeconflictStart(start, runDuration, laneIndex, runIndex, assignedStarts, assignedFinishes, assignedBreakdowns, assignedResets)
			runEnd := start.Add(runDuration)
			breakdownStart, breakdownEnd, resetStart, resetEnd := commandCenterOperatorWindows(runEnd, laneIndex, runIndex)
			state, result := commandCenterState(fixedTime, start, runEnd)
			if runEnd.After(dataStart) && start.Before(dataEnd) {
				runs = append(runs, buildCommandCenterRun(chamber.id, chamber.name, chamber.facility, laneIndex, runIndex, start, runEnd, breakdownStart, breakdownEnd, resetStart, resetEnd, state, result, fixedTime))
			}
			assignedStarts = append(assignedStarts, start)
			assignedFinishes = append(assignedFinishes, runEnd)
			assignedBreakdowns = append(assignedBreakdowns, breakdownStart)
			assignedResets = append(assignedResets, resetStart)
			if start.After(dataEnd.Add(runDuration)) {
				break
			}
			start = commandCenterTestStartAfterPrep(resetEnd, laneIndex, runIndex+1)
		}
		lanes = append(lanes, contracts.CommandCenterLane{
			ID:          chamber.id,
			ChamberName: chamber.name,
			Facility:    chamber.facility,
			Summary:     fmt.Sprintf("%s chamber ladder with completed, live, and forecast FAT slots.", chamber.name),
			GraphCardID: commandCenterLaneCardID(chamber.id),
			Runs:        runs,
		})
	}
	return contracts.CommandCenterFAT{
		Envelope:         env,
		ID:               "command_center_fat",
		Title:            "Command Center FAT",
		Summary:          "Four thermal chamber FAT swimlanes over a four-week operational window. The live cursor sits near the center with two weeks of as-run history and two weeks of forecast planning.",
		Now:              fixedTime.Format(time.RFC3339),
		WindowStart:      windowStart.Format(time.RFC3339),
		WindowEnd:        windowEnd.Format(time.RFC3339),
		DataStart:        dataStart.Format(time.RFC3339),
		DataEnd:          dataEnd.Format(time.RFC3339),
		SchedulePolicy:   "deterministic_tiled_fat_horizon_v1",
		WorkdayStartHour: 8,
		WorkdayEndHour:   18,
		WeekendBands:     weekendBands(dataStart, dataEnd),
		Lanes:            lanes,
		GraphCampaignID:  campaignID,
	}
}

func buildCommandCenterRun(chamberID, chamberName, facility string, laneIndex, runIndex int, start, end, breakdownStart, breakdownEnd, resetStart, resetEnd time.Time, state, result string, fixedTime time.Time) contracts.CommandCenterRun {
	runID := fmt.Sprintf("%s-fat-%02d", chamberID, runIndex+1)
	article := fmt.Sprintf("DUT %02d-%s", 41+laneIndex*7+runIndex, strings.ToUpper(chamberName[:1]))
	serial := fmt.Sprintf("DUT-%s-%04d", strings.ToUpper(chamberName[:1]), 2600+laneIndex*100+runIndex*11)
	operatorNext := commandCenterOperatorNext(fixedTime, state, breakdownStart, breakdownEnd, resetEnd)
	manifest := contracts.CommandCenterTestItemManifest{
		ID:             runID + "-manifest",
		Label:          fmt.Sprintf("%s item manifest", chamberName),
		Article:        article,
		SerialNumber:   serial,
		Facility:       facility,
		ChamberName:    chamberName,
		CampaignID:     "thermal_acceptance_fat",
		OperatorNext:   operatorNext,
		State:          state,
		Result:         result,
		Start:          start.Format(time.RFC3339),
		End:            end.Format(time.RFC3339),
		BreakdownStart: breakdownStart.Format(time.RFC3339),
		BreakdownEnd:   breakdownEnd.Format(time.RFC3339),
		ResetStart:     resetStart.Format(time.RFC3339),
		ResetEnd:       resetEnd.Format(time.RFC3339),
	}
	return contracts.CommandCenterRun{
		ID:                 runID,
		CampaignID:         "thermal_acceptance_fat",
		Title:              fmt.Sprintf("%s FAT %02d", chamberName, runIndex+1),
		State:              state,
		Result:             result,
		Start:              start.Format(time.RFC3339),
		End:                end.Format(time.RFC3339),
		BreakdownStart:     breakdownStart.Format(time.RFC3339),
		BreakdownEnd:       breakdownEnd.Format(time.RFC3339),
		ResetStart:         resetStart.Format(time.RFC3339),
		ResetEnd:           resetEnd.Format(time.RFC3339),
		Manifest:           manifest,
		InteractionWindows: commandCenterInteractionWindows(runID, start, end, breakdownStart, breakdownEnd, resetStart, resetEnd),
		Events: []contracts.CommandCenterEvent{
			{ID: runID + "-start", Label: "FAT start", Kind: "start", Timestamp: start.Format(time.RFC3339), State: state},
			{ID: runID + "-end", Label: "FAT closeout", Kind: "closeout", Timestamp: end.Format(time.RFC3339), State: state},
			{ID: runID + "-breakdown-start", Label: "Breakdown start", Kind: "breakdown", Timestamp: breakdownStart.Format(time.RFC3339), State: "operator"},
			{ID: runID + "-reset-start", Label: "Reset start", Kind: "reset", Timestamp: resetStart.Format(time.RFC3339), State: "operator"},
			{ID: runID + "-reset-ready", Label: "Reset ready", Kind: "reset_ready", Timestamp: resetEnd.Format(time.RFC3339), State: "reset"},
		},
	}
}

func buildCommandCenterGraphModel(env contracts.Envelope, fixedTime time.Time, campaignID string, commandCenter contracts.CommandCenterFAT, baseProgram *contracts.ThermalProgram, baseSim environmentalsim.Result, baseStart time.Time) ([]contracts.TelemetrySample, contracts.GraphModel) {
	windowStart := mustTime(commandCenter.DataStart)
	windowEnd := mustTime(commandCenter.DataEnd)
	hero := contracts.HeroGraphModel{
		ID:         campaignID + "_hero",
		Title:      "Command Center FAT chamber lanes",
		Owner:      "test_conductor_role",
		Provenance: baseSim.Provenance.Source,
		TimeAxis: contracts.GraphTimeAxis{
			Start:              commandCenter.DataStart,
			End:                commandCenter.DataEnd,
			Anchor:             commandCenter.Now,
			Now:                commandCenter.Now,
			DefaultWindowStart: commandCenter.WindowStart,
			DefaultWindowEnd:   commandCenter.WindowEnd,
			RangeSeconds:       int(windowEnd.Sub(windowStart).Seconds()),
			Clamp:              true,
			LatestPolicy:       "tiled as-run physics traces across complete command-center horizon",
		},
		Axes: []contracts.GraphYAxis{
			{ID: "temperature_c", Label: "Temperature", Units: "degC", Scale: "linear", Min: -55, Max: 82, Side: "left", Format: "fixed_1"},
		},
		Execution: &contracts.ExecutionState{
			Mode:             "simulated_live_command_center",
			Now:              commandCenter.Now,
			PercentComplete:  round(100 * windowPercent(fixedTime, windowStart, windowEnd)),
			Acceleration:     "wall clock replay cursor",
			PastDataPolicy:   "physics as-run traces are tiled from the source FAT simulation",
			FutureDataPolicy: "future chamber-air physics is shown only as a dashed ghost trace; DUT actuals stop at the live cursor",
			CompletedCycles:  completedCommandCenterRuns(commandCenter),
			TargetCycles:     totalCommandCenterRuns(commandCenter),
			CurrentCycle:     0,
			CurrentPhase:     "multi_chamber_ladder",
		},
	}
	samples := []contracts.TelemetrySample{}
	graphLanes := []contracts.GraphLane{}
	now := mustTime(commandCenter.Now)
	for laneIndex, lane := range commandCenter.Lanes {
		prefix := CommandCenterSignalPrefix(lane.ID)
		commandID := prefix + ".command_deg_c"
		ghostID := prefix + ".ghost_deg_c"
		chamberID := prefix + ".chamber_deg_c"
		dutID := prefix + ".dut_deg_c"
		commandTrace := contracts.GraphTrace{ID: commandID, Label: lane.ChamberName + " command", Role: "command", Units: "degC", AxisID: "temperature_c", Source: "thermal_program"}
		ghostTrace := contracts.GraphTrace{ID: ghostID, Label: lane.ChamberName + " chamber forecast", Role: "ghost", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal"}
		chamberTrace := contracts.GraphTrace{ID: chamberID, Label: lane.ChamberName + " chamber air", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal"}
		dutTrace := contracts.GraphTrace{ID: dutID, Label: lane.ChamberName + " DUT", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "dut_thermal"}
		graphLanes = append(graphLanes, contracts.GraphLane{
			ID: lane.ID, Label: lane.ChamberName, Series: []contracts.GraphSeries{
				{ID: commandID, Label: "Command", Role: "command", Units: "degC", Source: "thermal_program", Min: -55, Max: 82},
				{ID: ghostID, Label: "Chamber forecast", Role: "ghost", Units: "degC", Source: "facility_thermal", Min: -55, Max: 82},
				{ID: chamberID, Label: "Chamber air", Role: "facility_environment", Units: "degC", Source: "facility_thermal", Min: -55, Max: 82},
				{ID: dutID, Label: "DUT", Role: "article_temperature", Units: "degC", Source: "dut_thermal", Min: -55, Max: 82},
			},
		})
		for runIndex, run := range lane.Runs {
			runStart := mustTime(run.Start)
			runEnd := mustTime(run.End)
			hero.PhaseBands = append(hero.PhaseBands, contracts.GraphBand{ID: run.ID + "-window", Label: run.Title, Kind: "test_window", Start: run.Start, End: run.End, Result: run.Result})
			hero.PhaseBands = append(hero.PhaseBands, contracts.GraphBand{ID: run.ID + "-breakdown", Label: lane.ChamberName + " breakdown", Kind: "breakdown", Start: run.BreakdownStart, End: run.BreakdownEnd, Result: "operator"})
			hero.PhaseBands = append(hero.PhaseBands, contracts.GraphBand{ID: run.ID + "-reset", Label: lane.ChamberName + " reset", Kind: "reset", Start: run.ResetStart, End: run.ResetEnd, Result: "operator"})
			hero.Markers = append(hero.Markers,
				contracts.GraphMarker{
					ID:        run.ID + "-operator-breakdown-start",
					Label:     lane.ChamberName + " breakdown start",
					Kind:      "operator_breakdown",
					Role:      "operator_interaction",
					Timestamp: run.BreakdownStart,
					Result:    "action_required",
					Severity:  "operator",
				},
				contracts.GraphMarker{
					ID:        run.ID + "-operator-reset-start",
					Label:     lane.ChamberName + " reset start",
					Kind:      "operator_reset",
					Role:      "operator_interaction",
					Timestamp: run.ResetStart,
					Result:    "action_required",
					Severity:  "operator",
				},
				contracts.GraphMarker{
					ID:        run.ID + "-operator-reset-ready",
					Label:     lane.ChamberName + " reset ready",
					Kind:      "operator_reset_ready",
					Role:      "operator_interaction",
					Timestamp: run.ResetEnd,
					Result:    "ready",
					Severity:  "operator",
				},
			)
			for _, gate := range baseProgram.FunctionalGates {
				ts := shiftTime(mustTime(gate.Timestamp), runStart, baseStart)
				if ts.Before(runStart) || ts.After(runEnd) {
					continue
				}
				hero.Markers = append(hero.Markers, contracts.GraphMarker{
					ID:         fmt.Sprintf("%s-%s", run.ID, strings.TrimPrefix(gate.ID, campaignID+"-")),
					Label:      fmt.Sprintf("%s %s", lane.ChamberName, gate.Label),
					Kind:       "functional_gate",
					Role:       "event",
					Timestamp:  ts.Format(time.RFC3339),
					CycleIndex: gate.CycleIndex + laneIndex*10 + runIndex,
					Result:     "pass",
				})
			}
			for _, sample := range baseSim.Samples {
				t := shiftTime(mustTime(sample.Timestamp), runStart, baseStart)
				if t.Before(windowStart) || t.After(windowEnd) || t.Before(runStart) || t.After(runEnd) {
					continue
				}
				command := sample.Signals["chamber_setpoint_deg_c"]
				chamber := sample.Signals["chamber_air_deg_c"] + float64(laneIndex)*0.22
				dut := sample.Signals["thermal_zone_1_deg_c"] + float64(laneIndex)*0.18
				commandTrace.Values = append(commandTrace.Values, graphPointAt(t, command))
				signalValues := map[string]float64{
					commandID: round(command),
				}
				if t.After(now) {
					ghostTrace.Values = append(ghostTrace.Values, graphPointAt(t, chamber))
					signalValues[ghostID] = round(chamber)
				} else {
					chamberTrace.Values = append(chamberTrace.Values, graphPointAt(t, chamber))
					dutTrace.Values = append(dutTrace.Values, graphPointAt(t, dut))
					signalValues[chamberID] = round(chamber)
					signalValues[dutID] = round(dut)
				}
				samples = append(samples, contracts.TelemetrySample{Timestamp: t.Format(time.RFC3339), Quality: sample.Quality, Signals: signalValues, States: map[string]string{"command_center_lane": lane.ID, "command_center_run": run.ID}})
			}
		}
		hero.Traces = append(hero.Traces, commandTrace, ghostTrace, chamberTrace, dutTrace)
	}
	for _, weekend := range commandCenter.WeekendBands {
		hero.PhaseBands = append(hero.PhaseBands, contracts.GraphBand{ID: weekend.ID, Label: weekend.Label, Kind: weekend.Kind, Start: weekend.Start, End: weekend.End})
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].Timestamp < samples[j].Timestamp })
	sort.Slice(hero.Markers, func(i, j int) bool { return hero.Markers[i].Timestamp < hero.Markers[j].Timestamp })
	wall := buildCommandCenterGraphWall(env, campaignID, commandCenter, hero, baseSim.Provenance)
	manifest := buildTileManifest(env, campaignID, wall, hero)
	return samples, contracts.GraphModel{
		Envelope:             env,
		CampaignID:           campaignID,
		Lanes:                graphLanes,
		ThermalProgram:       baseProgram,
		SimulationProvenance: &baseSim.Provenance,
		HeroGraph:            &hero,
		GraphWall:            &wall,
		TileManifest:         &manifest,
	}
}

func buildCommandCenterGraphWall(env contracts.Envelope, campaignID string, commandCenter contracts.CommandCenterFAT, hero contracts.HeroGraphModel, provenance contracts.SimulationProvenance) contracts.GraphWallModel {
	groupID := campaignID + "_operator_wall"
	wall := contracts.GraphWallModel{
		ID:           campaignID + "_graph_wall",
		Title:        "Command Center FAT operator graph wall",
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
			Source:       "command_center_fat",
		},
		TilePolicy: denseOperatorGraphTilePolicy(1200, 4200, 128, 760, 256),
		GraphGroups: []contracts.GraphGroup{{
			ID:              groupID,
			Title:           "Four-chamber FAT command center",
			Mode:            "multi_chamber_command_center",
			BehaviorProfile: "loom_dense_operator_wall",
			Application:     "environmental_test_command_center",
			SectionIDs:      []string{"command_center_lanes"},
			Interaction: contracts.GraphInteraction{
				SharedTimeline: true, SharedCrosshair: true, VerticalGrid: true, SingleTimeAxis: true,
				CursorMode: "inspect", CrosshairScope: "all_cards", TimelineGridMode: "absolute_workday_ladder",
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
	section := contracts.GraphSection{ID: "command_center_lanes", Title: "Chamber lanes", GroupID: groupID, Transport: "arrow_ipc", Direction: "derived", Status: "fresh"}
	for i, lane := range commandCenter.Lanes {
		prefix := CommandCenterSignalPrefix(lane.ID)
		card := graphCard(commandCenterLaneCardID(lane.ID), lane.ChamberName+" chamber FAT lane", "line", "primary_hero", "degC", "temperature_c", "facility_thermal", []contracts.GraphWallSignal{
			graphSignal(prefix+".command_deg_c", "Command", "degC", "thermal_program", "command", "command", "facility", "temperature_c", "command_center_lanes"),
			graphSignal(prefix+".ghost_deg_c", "Chamber forecast", "degC", "facility_thermal", "ghost", "projection", "facility", "temperature_c", "command_center_lanes"),
			graphSignal(prefix+".chamber_deg_c", "Chamber air", "degC", "facility_thermal", "actual", "measurement", "facility", "temperature_c", "command_center_lanes"),
			graphSignal(prefix+".dut_deg_c", "DUT", "degC", "dut_thermal", "actual", "measurement", "dut", "temperature_c", "command_center_lanes"),
		})
		card.IncludeMarkers = true
		card.Placement.SectionID = section.ID
		card.Placement.GroupID = groupID
		card.Placement.Order = i + 1
		card.Placement.Pinned = i == 0
		card.Placement.DefaultVisible = true
		card.Placement.ResizePolicy = "fixed_plot_area"
		section.Cards = append(section.Cards, card)
	}
	wall.Sections = []contracts.GraphSection{section}
	return wall
}

func denseOperatorGraphTilePolicy(defaultPoints, maxPoints, historyTileMaxCount, viewportPrefetchPX, tileBufferMaxEntries int) contracts.GraphTilePolicy {
	policy := sharedgraph.DenseOperatorTilePolicy()
	policy.DefaultPoints = defaultPoints
	policy.MaxPoints = maxPoints
	policy.HistoryTileMaxCount = historyTileMaxCount
	policy.ViewportPrefetchPX = viewportPrefetchPX
	policy.TileBufferMaxEntries = tileBufferMaxEntries
	return contracts.GraphTilePolicy{
		DefaultPoints:               policy.DefaultPoints,
		MaxPoints:                   policy.MaxPoints,
		LiveTileMinRefreshMS:        policy.LiveTileMinRefreshMS,
		HistoryTileMaxCount:         policy.HistoryTileMaxCount,
		ViewportPrefetchPX:          policy.ViewportPrefetchPX,
		TileBufferMaxEntries:        policy.TileBufferMaxEntries,
		TileBufferTTLMS:             policy.TileBufferTTLMS,
		ResolutionLevels:            append([]string(nil), policy.ResolutionLevels...),
		SubscriberRole:              policy.SubscriberRole,
		SharedTimebaseRequired:      policy.SharedTimebaseRequired,
		LegendMayAffectPlotWidth:    policy.LegendMayAffectPlotWidth,
		MalformedSVGPathHardFailure: policy.MalformedSVGPathHardFailure,
	}
}

func shiftTime(t, runStart, baseStart time.Time) time.Time {
	return runStart.Add(t.Sub(baseStart))
}

func graphPointAt(t time.Time, value float64) contracts.GraphPoint {
	return contracts.GraphPoint{Timestamp: t.Format(time.RFC3339), Value: round(value)}
}

// CommandCenterSignalPrefix returns the telemetry signal prefix for a chamber ID.
func CommandCenterSignalPrefix(chamberID string) string {
	return "cc." + strings.TrimPrefix(chamberID, "thermal_chamber_")
}

func commandCenterLaneCardID(chamberID string) string {
	return CommandCenterSignalPrefix(chamberID) + ".lane"
}

func completedCommandCenterRuns(model contracts.CommandCenterFAT) int {
	count := 0
	for _, lane := range model.Lanes {
		for _, run := range lane.Runs {
			if run.State == "complete" {
				count++
			}
		}
	}
	return count
}

func totalCommandCenterRuns(model contracts.CommandCenterFAT) int {
	count := 0
	for _, lane := range model.Lanes {
		count += len(lane.Runs)
	}
	return count
}

func windowPercent(t, start, end time.Time) float64 {
	if !end.After(start) {
		return 0
	}
	return math.Max(0, math.Min(1, float64(t.Sub(start))/float64(end.Sub(start))))
}

func commandCenterInteractionWindows(runID string, start, end, breakdownStart, breakdownEnd, resetStart, resetEnd time.Time) []contracts.CommandCenterBand {
	duration := end.Sub(start)
	points := []struct {
		suffix, label string
		offset        float64
	}{
		{"setup", "Setup review", 0.02},
		{"cold", "Cold dwell gate", 0.46},
		{"hot", "Hot dwell gate", 0.76},
		{"closeout", "Closeout evidence", 0.94},
	}
	windows := make([]contracts.CommandCenterBand, 0, len(points)+2)
	for _, point := range points {
		center := start.Add(time.Duration(point.offset * float64(duration)))
		windows = append(windows, contracts.CommandCenterBand{ID: runID + "-" + point.suffix, Label: point.label, Kind: "operator_gate", Start: center.Add(-45 * time.Minute).Format(time.RFC3339), End: center.Add(45 * time.Minute).Format(time.RFC3339)})
	}
	windows = append(windows,
		contracts.CommandCenterBand{ID: runID + "-breakdown-window", Label: "Breakdown", Kind: "operator_breakdown", Start: breakdownStart.Format(time.RFC3339), End: breakdownEnd.Format(time.RFC3339)},
		contracts.CommandCenterBand{ID: runID + "-reset-window", Label: "Reset", Kind: "operator_reset", Start: resetStart.Format(time.RFC3339), End: resetEnd.Format(time.RFC3339)},
	)
	return windows
}

func commandCenterState(fixedTime time.Time, start, end time.Time) (string, string) {
	switch {
	case fixedTime.Before(start):
		return "scheduled", "pending"
	case fixedTime.After(end):
		return "complete", "pass"
	default:
		return "running", "in_progress"
	}
}

func commandCenterOperatorNext(fixedTime time.Time, state string, breakdownStart, breakdownEnd, resetEnd time.Time) string {
	switch state {
	case "complete":
		if fixedTime.Before(breakdownEnd) {
			return "breakdown in progress"
		}
		if fixedTime.Before(resetEnd) {
			return "reset in progress"
		}
		return "archive evidence"
	case "running":
		return "monitor dwell gate"
	default:
		return fmt.Sprintf("prepare breakdown slot %s", breakdownStart.Format("Mon 15:04"))
	}
}

// mustTime parses an RFC3339 timestamp or panics.
func mustTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}

// round rounds a float64 to a sensible number of significant digits.
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

// thermalContextDuration returns the post-cycle context window for a thermal program.
func thermalContextDuration(program *contracts.ThermalProgram) time.Duration {
	if program == nil || len(program.Cycles) == 0 {
		return 8 * time.Hour
	}
	return 8 * time.Hour
}

// graphCard builds a GraphWallCard with standard defaults.
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

func heightWeight(kind, role string) float64 {
	if role == "primary_hero" {
		return 2.8
	}
	if kind == "state" || kind == "event" {
		return 0.85
	}
	return 1.0
}

func graphSignal(id, label, unit, source, role, kind, subsystem, axisID, sectionID string) contracts.GraphWallSignal {
	return contracts.GraphWallSignal{ID: id, Label: label, Unit: unit, Source: source, SourceFamily: source, Kind: kind, Category: role, Role: role, Subsystem: subsystem, AxisID: axisID, SectionID: sectionID}
}

// buildTileManifest builds a GraphTileManifest from a wall and hero graph.
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
			{ID: "legacy_csv_environment", Label: "Legacy CSV environment import", SourceFormat: "legacy_csv", TargetSchema: arrowtelemetrySchemaName, Mode: "translated_fixture", Confidence: "high", Provenance: "synthetic DataLens translation demo"},
			{ID: "binary_tmtc_log", Label: "Binary transport log import", SourceFormat: "binary_log", TargetSchema: arrowtelemetrySchemaName, Mode: "translated_fixture", Confidence: "medium", Provenance: "synthetic DataLens translation demo"},
			{ID: "hdf5_evidence_archive", Label: "HDF5-like evidence archive", SourceFormat: "hdf5", TargetSchema: arrowtelemetrySchemaName, Mode: "translated_fixture", Confidence: "high", Provenance: "synthetic DataLens translation demo"},
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

// arrowtelemetrySchemaName mirrors the constant from the arrowtelemetry package
// to avoid a dependency on that package from commandcenter.
const arrowtelemetrySchemaName = "gossamer.telemetry.arrow.v2"
