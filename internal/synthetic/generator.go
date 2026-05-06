package synthetic

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/egidinas/gossamer/internal/contracts"
)

var FixedTime = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

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
		samples := buildTelemetry(id)
		telemetry[id] = samples
		campaigns[id] = buildCampaign(env, id)
		graphs[id] = buildGraphModel(env, id)
	}
	return FixtureSet{
		Manifest: contracts.Manifest{
			Envelope:      env,
			Name:          "Gossamer",
			Description:   "Synthetic spacecraft environmental-test operating model demonstrator.",
			TestArticle:   "AuroraSat-1",
			Campaigns:     CampaignIDs,
			PublicDemo:    true,
			SyntheticOnly: true,
		},
		Topology: contracts.Topology{
			Envelope: env,
			Nodes: []contracts.Node{
				{ID: "aurorasat_1", Label: "AuroraSat-1", Kind: "test_article", Status: "in_test", Quality: "synthetic"},
				{ID: "thermal_chamber_a", Label: "Thermal Chamber A", Kind: "facility", Status: "available", Quality: "fresh"},
				{ID: "tvac_chamber_q1", Label: "TVAC Chamber Q1", Kind: "facility", Status: "campaign_active", Quality: "fresh"},
				{ID: "flatsat_rack_a", Label: "Flatsat Rack A", Kind: "facility", Status: "available", Quality: "fresh"},
				{ID: "archive_node_a", Label: "Archive Node A", Kind: "data_system", Status: "recording", Quality: "fresh"},
			},
			Links: []contracts.Link{
				{Source: "aurorasat_1", Target: "archive_node_a", Bus: "telemetry_bus"},
				{Source: "thermal_chamber_a", Target: "aurorasat_1", Bus: "facility_control_bus"},
				{Source: "tvac_chamber_q1", Target: "aurorasat_1", Bus: "facility_control_bus"},
				{Source: "flatsat_rack_a", Target: "aurorasat_1", Bus: "command_bus"},
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
		Supervisor:  buildSupervisorOverview(env, campaigns, telemetry),
		BusTap:      buildBusTap(env, telemetry["integrated_system_fat"]),
		Campaigns:   campaigns,
		Telemetry:   telemetry,
		GraphModels: graphs,
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
	if err := writeJSON(filepath.Join(base, "bus_virtualization_tap.json"), set.BusTap); err != nil {
		return err
	}
	for _, id := range CampaignIDs {
		if err := writeJSON(filepath.Join(base, "campaigns", id+".json"), set.Campaigns[id]); err != nil {
			return err
		}
		if err := writeJSONL(filepath.Join(base, "telemetry", id+".jsonl"), set.Telemetry[id]); err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(base, "graph_models", id+".json"), set.GraphModels[id]); err != nil {
			return err
		}
	}
	return nil
}

func buildSources(env contracts.Envelope) contracts.SourceCatalogue {
	return contracts.SourceCatalogue{Envelope: env, Sources: []contracts.Source{
		{ID: "spacecraft_power", Label: "Spacecraft Power", Owner: "egse_power_role", Bus: "telemetry_bus", Quality: "fresh", FreshnessMS: 250, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"eps_bus_voltage_v", "eps_bus_current_a"}},
		{ID: "spacecraft_cdh", Label: "Spacecraft CDH", Owner: "subsystem_test_role", Bus: "subsystem_event_bus", Quality: "fresh", FreshnessMS: 500, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"obc_boot_state", "obc_command_counter", "tc_packet_counter"}},
		{ID: "spacecraft_rf", Label: "Spacecraft RF Comms", Owner: "subsystem_test_role", Bus: "telemetry_bus", Quality: "fresh", FreshnessMS: 450, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"rf_link_margin_db", "tm_packet_counter"}},
		{ID: "facility_thermal", Label: "Facility Thermal", Owner: "facility_test_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"thermal_zone_1_deg_c", "thermal_zone_2_deg_c", "chamber_air_deg_c"}},
		{ID: "facility_pressure", Label: "Facility Pressure", Owner: "facility_test_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"tvac_pressure_pa"}},
		{ID: "demo_bus_virtualization", Label: "Demo Bus Virtualization Tap", Owner: "test_conductor_role", Bus: "telemetry_bus", Quality: "synthetic", FreshnessMS: 200, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"bus_latency_ms", "tm_packet_counter", "tc_packet_counter", "dropped_frame_count"}},
		{ID: "demo_quality", Label: "Demo Quality Monitor", Owner: "test_conductor_role", Bus: "telemetry_bus", Quality: "synthetic", FreshnessMS: 1000, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"source_freshness_ms", "facility_interlock_state"}},
	}}
}

func buildCampaign(env contracts.Envelope, id string) contracts.Campaign {
	c := contracts.Campaign{
		Envelope:      env,
		ID:            id,
		Article:       "AuroraSat-1",
		SyntheticNote: "Synthetic campaign generated for public demonstration only.",
		Start:         FixedTime.Format(time.RFC3339),
		End:           FixedTime.Add(11 * time.Hour).Format(time.RFC3339),
	}
	switch id {
	case "flatsat_derisking":
		c.Name, c.Level, c.State, c.Result, c.Facility = "Flatsat Derisking", "subsystem", "complete", "pass", "flatsat_rack_a"
	case "thermal_acceptance_fat":
		c.Name, c.Level, c.State, c.Result, c.Facility = "Thermal Acceptance FAT", "integrated_acceptance", "complete", "pass", "thermal_chamber_a"
	case "tvac_qualification":
		c.Name, c.Level, c.State, c.Result, c.Facility = "TVAC Qualification", "qualification", "review", "inconclusive", "tvac_chamber_q1"
		c.Anomalies = []contracts.Anomaly{{ID: "ANOM-TVAC-001", Title: "Synthetic pressure source degradation during cold dwell", Severity: "medium", Status: "needs_disposition", EvidenceRef: "telemetry/tvac_qualification.jsonl#sample-32", Disposition: "Review required before closure."}}
	case "integrated_system_fat":
		c.Name, c.Level, c.State, c.Result, c.Facility = "Integrated System FAT", "system", "complete", "pass", "thermal_chamber_a"
	default:
		c.Name, c.Level, c.State, c.Result, c.Facility = id, "unknown", "not_run", "not_run", "not_applicable"
	}
	c.Requirements = defaultRequirements(c.Result)
	return c
}

func defaultRequirements(result string) []contracts.Requirement {
	ids := []string{"REQ-CYCLE-COUNT", "REQ-HOT-TARGET", "REQ-COLD-TARGET", "REQ-STABILITY", "REQ-DWELL", "REQ-FUNC-GATE-PRE", "REQ-FUNC-GATE-DURING", "REQ-FUNC-GATE-POST", "REQ-DATA-QUALITY", "REQ-ANOMALY-REVIEW"}
	reqs := make([]contracts.Requirement, 0, len(ids))
	for _, id := range ids {
		r := "pass"
		if result == "inconclusive" && (id == "REQ-DATA-QUALITY" || id == "REQ-ANOMALY-REVIEW") {
			r = "inconclusive"
		}
		reqs = append(reqs, contracts.Requirement{ID: id, Title: requirementTitle(id), Description: "Synthetic public requirement used to demonstrate traceability.", Result: r, Evidence: []string{"telemetry", "graph_model"}, Rationale: "Evaluated from deterministic synthetic fixture data."})
	}
	return reqs
}

func requirementTitle(id string) string {
	titles := map[string]string{
		"REQ-CYCLE-COUNT":      "Required cycle count completed",
		"REQ-HOT-TARGET":       "Hot target reached",
		"REQ-COLD-TARGET":      "Cold target reached",
		"REQ-STABILITY":        "Stabilization window achieved",
		"REQ-DWELL":            "Dwell duration achieved",
		"REQ-FUNC-GATE-PRE":    "Pre-environment functional gate passed",
		"REQ-FUNC-GATE-DURING": "During-environment functional gate passed",
		"REQ-FUNC-GATE-POST":   "Post-environment functional gate passed",
		"REQ-DATA-QUALITY":     "Evidence data quality acceptable",
		"REQ-ANOMALY-REVIEW":   "Anomaly review disposition complete",
	}
	return titles[id]
}

func buildTelemetry(campaign string) []contracts.TelemetrySample {
	start := FixedTime
	out := make([]contracts.TelemetrySample, 0, 48)
	for i := 0; i < 48; i++ {
		t := start.Add(time.Duration(i) * 15 * time.Minute)
		phase := float64(i) / 47
		chamber := -20.0 + 70*math.Sin(phase*math.Pi)
		if campaign == "tvac_qualification" {
			chamber = -35.0 + 95*math.Sin(phase*math.Pi)
		}
		pressure := 101325.0
		if campaign == "tvac_qualification" {
			pressure = math.Max(0.001, 101325*math.Exp(-phase*12))
		}
		quality := "fresh"
		freshness := 250.0
		if campaign == "integrated_system_fat" && i >= 18 && i <= 24 {
			quality = "degraded"
			freshness = 3200
		}
		if campaign == "tvac_qualification" && i >= 30 && i <= 34 {
			quality = "degraded"
			freshness = 5100
		}
		out = append(out, contracts.TelemetrySample{
			Timestamp: t.Format(time.RFC3339),
			Quality:   quality,
			Signals: map[string]float64{
				"eps_bus_voltage_v":       28.0 + 0.3*math.Sin(phase*2*math.Pi),
				"eps_bus_current_a":       4.0 + 0.8*math.Sin(phase*4*math.Pi),
				"obc_command_counter":     float64(1000 + i*3),
				"payload_sim_heater_w":    math.Max(0, 25+20*math.Cos(phase*2*math.Pi)),
				"thermal_zone_1_deg_c":    chamber + 2.4,
				"thermal_zone_2_deg_c":    chamber - 1.8,
				"chamber_air_deg_c":       chamber,
				"tvac_pressure_pa":        pressure,
				"source_freshness_ms":     freshness,
				"facility_interlock_code": 1,
				"bus_latency_ms":          18 + 8*math.Sin(phase*2*math.Pi),
				"tm_packet_counter":       float64(6000 + i*17),
				"tc_packet_counter":       float64(120 + i*2),
				"dropped_frame_count":     droppedFrames(campaign, i),
				"rf_link_margin_db":       8.5 + 1.5*math.Cos(phase*2*math.Pi),
			},
			States: map[string]string{
				"obc_boot_state":           "nominal",
				"rf_link_sim_state":        "locked",
				"facility_interlock_state": "closed",
			},
		})
	}
	return out
}

func buildGraphModel(env contracts.Envelope, campaign string) contracts.GraphModel {
	return contracts.GraphModel{
		Envelope:   env,
		CampaignID: campaign,
		Lanes: []contracts.GraphLane{
			{ID: "thermal", Label: "Thermal", Series: []contracts.GraphSeries{{ID: "chamber_air_deg_c", Label: "Chamber Air", Role: "facility_environment", Units: "degC", Source: "facility_thermal", Min: -45, Max: 70}, {ID: "thermal_zone_1_deg_c", Label: "Thermal Zone 1", Role: "article_temperature", Units: "degC", Source: "facility_thermal", Min: -45, Max: 70}}},
			{ID: "pressure", Label: "Pressure", Series: []contracts.GraphSeries{{ID: "tvac_pressure_pa", Label: "TVAC Pressure", Role: "facility_environment", Units: "Pa", Source: "facility_pressure", Min: 0.001, Max: 101325}}},
			{ID: "power", Label: "Power", Series: []contracts.GraphSeries{{ID: "eps_bus_voltage_v", Label: "Bus Voltage", Role: "spacecraft_power", Units: "V", Source: "spacecraft_power", Min: 26, Max: 30}, {ID: "eps_bus_current_a", Label: "Bus Current", Role: "spacecraft_power", Units: "A", Source: "spacecraft_power", Min: 0, Max: 8}}},
			{ID: "bus", Label: "Virtual Bus", Series: []contracts.GraphSeries{{ID: "bus_latency_ms", Label: "Bus Latency", Role: "virtual_bus_health", Units: "ms", Source: "demo_bus_virtualization", Min: 0, Max: 250}, {ID: "tm_packet_counter", Label: "TM Counter", Role: "telemetry_counter", Units: "count", Source: "demo_bus_virtualization", Min: 0, Max: 8000}}},
			{ID: "quality", Label: "Source Quality", Series: []contracts.GraphSeries{{ID: "source_freshness_ms", Label: "Source Freshness", Role: "data_quality", Units: "ms", Source: "demo_quality", Min: 0, Max: 6000}}},
		},
	}
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
		supervisorLane("thermal_fat", "Thermal Chamber FAT", "thermal_chamber_a", campaigns["thermal_acceptance_fat"], "Thermal ramp and hot/cold dwell", "facility_control_bus", "REQ-HOT-TARGET pass, REQ-COLD-TARGET pass", "fresh", []string{"Thermal chamber profile is synchronized with article temperature."}, telemetry["thermal_acceptance_fat"], []heroSpec{{"chamber_air_deg_c", "Chamber Air", "facility_environment", "degC", "facility_thermal", -45, 70}, {"thermal_zone_1_deg_c", "Article Zone 1", "article_temperature", "degC", "facility_thermal", -45, 70}}),
		supervisorLane("eps_load_step", "EPS Load Step", "flatsat_rack_a", campaigns["integrated_system_fat"], "Power load and command script", "command_bus", "REQ-FUNC-GATE-DURING pass", "degraded", []string{"Synthetic freshness degradation demonstrates disposition workflow."}, telemetry["integrated_system_fat"], []heroSpec{{"eps_bus_voltage_v", "Bus Voltage", "spacecraft_power", "V", "spacecraft_power", 24, 32}, {"eps_bus_current_a", "Bus Current", "spacecraft_power", "A", "spacecraft_power", 0, 8}}),
		supervisorLane("payload_thermal", "Payload Thermal Cycle", "thermal_chamber_a", campaigns["integrated_system_fat"], "Payload simulator heater cycling", "telemetry_bus", "REQ-STABILITY pass", "fresh", []string{"Payload heater response is fictional and bounded for demo use."}, telemetry["integrated_system_fat"], []heroSpec{{"payload_sim_heater_w", "Payload Heater", "payload_thermal_control", "W", "facility_thermal", 0, 60}, {"thermal_zone_2_deg_c", "Article Zone 2", "article_temperature", "degC", "facility_thermal", -45, 70}}),
		supervisorLane("tvac_qualification", "TVAC Qualification", "tvac_chamber_q1", campaigns["tvac_qualification"], "Pressure pumpdown and cold dwell", "facility_control_bus", "REQ-ANOMALY-REVIEW inconclusive", "degraded", []string{"Synthetic pressure-source degradation remains open for review."}, telemetry["tvac_qualification"], []heroSpec{{"tvac_pressure_pa", "TVAC Pressure", "facility_environment", "Pa", "facility_pressure", 0.001, 101325}, {"thermal_zone_1_deg_c", "Cold Dwell Temperature", "article_temperature", "degC", "facility_thermal", -45, 70}}),
		supervisorLane("archive_capture", "Archive Evidence Capture", "archive_node_a", campaigns["integrated_system_fat"], "TM/TC capture and evidence packaging", "telemetry_bus", "REQ-DATA-QUALITY pass with review note", "synthetic", []string{"Archive node receives virtualized TM and TC events from the replay bus."}, telemetry["integrated_system_fat"], []heroSpec{{"bus_latency_ms", "Bus Latency", "virtual_bus_health", "ms", "demo_bus_virtualization", 0, 250}, {"tm_packet_counter", "TM Counter", "telemetry_counter", "count", "demo_bus_virtualization", 0, 8000}}),
	}
	return contracts.SupervisorOverview{
		Envelope:    env,
		TestArticle: "AuroraSat-1",
		Summary:     "Parallel synthetic FAT and qualification activities for a public-safe spacecraft environmental-test demo.",
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
			Values: graphPoints(samples, spec.signal, 12),
		})
	}
	return contracts.SupervisorLane{
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
	}
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
		{ID: "tm_primary", Label: "AuroraSat-1 TM to Archive", Direction: "TM", SourceNode: "aurorasat_1", DestinationNode: "archive_node_a", Bus: "telemetry_bus", Quality: "fresh", LatencyMS: 22, PacketCounter: 6748, DroppedFrames: 0},
		{ID: "tc_primary", Label: "Flatsat Rack TC to AuroraSat-1", Direction: "TC", SourceNode: "flatsat_rack_a", DestinationNode: "aurorasat_1", Bus: "command_bus", Quality: "fresh", LatencyMS: 31, PacketCounter: 214, DroppedFrames: 0},
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
			SourceNode:      "aurorasat_1",
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
				DestinationNode: "aurorasat_1",
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
		ReplayCursor: "cursor-aurorasat-demo-001",
		Streams:      streams,
		Events:       events,
	}
}

func round(value float64) float64 {
	return math.Round(value*100) / 100
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func writeJSONL(path string, samples []contracts.TelemetrySample) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for i, sample := range samples {
		if err := enc.Encode(sample); err != nil {
			return fmt.Errorf("sample %d: %w", i, err)
		}
	}
	return nil
}
