package synthetic

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egidinas/gossamer/internal/environmentalsim"
	"github.com/egidinas/gossamer/internal/synthetic/commandcenter"
	"github.com/egidinas/signalforge/arrowtelemetry"
	"github.com/egidinas/signalforge/contracts"
	sharedgraph "github.com/egidinas/signalforge/graphwall"
	"github.com/egidinas/signalforge/jsonfile"
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
				// Test article
				{ID: "reference_dut", Label: "Reference DUT", Kind: "test_article", Status: "in_test", Quality: "synthetic"},
				// Thermal chamber PLCs — one per chamber; thermal_supervisor_pc polls them all
				{ID: "thermal_chamber_a", Label: "Chamber Alpha PLC", Kind: "facility_plc", Status: "campaign_active", Quality: "fresh"},
				{ID: "thermal_chamber_b", Label: "Chamber Bravo PLC", Kind: "facility_plc", Status: "available", Quality: "fresh"},
				{ID: "thermal_chamber_c", Label: "Chamber Charlie PLC", Kind: "facility_plc", Status: "available", Quality: "fresh"},
				{ID: "thermal_chamber_d", Label: "Chamber Delta PLC", Kind: "facility_plc", Status: "available", Quality: "fresh"},
				// Thermal supervisor PC — gathers all four chamber PLCs and house PLC; serves as gateway for thermal data
				{ID: "thermal_supervisor_pc", Label: "Thermal Supervisor PC", Kind: "computer", Status: "active", Quality: "fresh"},
				// TVac — PLC controls cryo/vacuum; two computers relay transport streams and record data
				{ID: "tvac_chamber_q1", Label: "TVac Chamber Q1", Kind: "facility", Status: "campaign_active", Quality: "fresh"},
				{ID: "tvac_plc_q1", Label: "TVac PLC Q1", Kind: "facility_plc", Status: "campaign_active", Quality: "fresh"},
				{ID: "tvac_computer_1", Label: "TVac Computer 1 (Primary)", Kind: "computer", Status: "active", Quality: "fresh"},
				{ID: "tvac_computer_2", Label: "TVac Computer 2 (Backup)", Kind: "computer", Status: "active", Quality: "fresh"},
				// Flatsat bench
				{ID: "flatsat_rack_a", Label: "Flatsat Rack A", Kind: "facility", Status: "available", Quality: "fresh"},
				// Facility infrastructure
				{ID: "house_plc", Label: "House Control PLC", Kind: "facility_plc", Status: "active", Quality: "fresh"},
				// Live telemetry collection
				{ID: "archive_node_a", Label: "Archive Node A", Kind: "data_system", Status: "recording", Quality: "fresh"},
				// Long-term storage — finished tests exported as HDF5 files
				{ID: "nas_a", Label: "Network Storage (NAS A)", Kind: "storage", Status: "active", Quality: "fresh"},
				// Librarian — indexes NAS files and live-translates them for the gateway
				{ID: "librarian_a", Label: "Librarian", Kind: "service", Status: "active", Quality: "fresh"},
				// Gateway — backend API server; serves translated data to the web UI
				{ID: "gateway_a", Label: "Data Gateway", Kind: "gateway", Status: "active", Quality: "fresh"},
				// Test campaign supervisor
				{ID: "supervisor_a", Label: "Test Supervisor", Kind: "supervisor", Status: "active", Quality: "fresh"},
			},
			Links: []contracts.Link{
				// House PLC supplies utilities (power, water, air, LN2) to chambers and TVac
				{Source: "house_plc", Target: "thermal_chamber_a", Bus: "facility_utility_bus"},
				{Source: "house_plc", Target: "thermal_chamber_b", Bus: "facility_utility_bus"},
				{Source: "house_plc", Target: "thermal_chamber_c", Bus: "facility_utility_bus"},
				{Source: "house_plc", Target: "thermal_chamber_d", Bus: "facility_utility_bus"},
				{Source: "house_plc", Target: "tvac_plc_q1", Bus: "facility_utility_bus"},
				// Thermal supervisor PC polls all four chamber PLCs and house PLC
				{Source: "thermal_supervisor_pc", Target: "thermal_chamber_a", Bus: "facility_control_bus"},
				{Source: "thermal_supervisor_pc", Target: "thermal_chamber_b", Bus: "facility_control_bus"},
				{Source: "thermal_supervisor_pc", Target: "thermal_chamber_c", Bus: "facility_control_bus"},
				{Source: "thermal_supervisor_pc", Target: "thermal_chamber_d", Bus: "facility_control_bus"},
				{Source: "thermal_supervisor_pc", Target: "house_plc", Bus: "facility_control_bus"},
				// Supervisor issues test-profile commands via thermal supervisor PC
				{Source: "supervisor_a", Target: "thermal_supervisor_pc", Bus: "supervisor_bus"},
				{Source: "supervisor_a", Target: "tvac_computer_1", Bus: "supervisor_bus"},
				{Source: "supervisor_a", Target: "tvac_computer_2", Bus: "supervisor_bus"},
				{Source: "supervisor_a", Target: "archive_node_a", Bus: "supervisor_bus"},
				// TVac internal: PLC status feeds to both computers
				{Source: "tvac_plc_q1", Target: "tvac_computer_1", Bus: "facility_control_bus"},
				{Source: "tvac_plc_q1", Target: "tvac_computer_2", Bus: "facility_control_bus"},
				{Source: "tvac_plc_q1", Target: "tvac_chamber_q1", Bus: "facility_control_bus"},
				// Chambers connect to DUT (one active at a time; topology shows capability)
				{Source: "thermal_chamber_a", Target: "reference_dut", Bus: "facility_control_bus"},
				{Source: "thermal_chamber_b", Target: "reference_dut", Bus: "facility_control_bus"},
				{Source: "thermal_chamber_c", Target: "reference_dut", Bus: "facility_control_bus"},
				{Source: "thermal_chamber_d", Target: "reference_dut", Bus: "facility_control_bus"},
				{Source: "tvac_computer_1", Target: "reference_dut", Bus: "facility_control_bus"},
				{Source: "tvac_computer_2", Target: "reference_dut", Bus: "facility_control_bus"},
				// Flatsat bench command path
				{Source: "flatsat_rack_a", Target: "reference_dut", Bus: "command_bus"},
				// DUT telemetry to archive
				{Source: "reference_dut", Target: "archive_node_a", Bus: "telemetry_bus"},
				// Finished test exports: archive → NAS (HDF5), NAS → librarian → gateway
				{Source: "archive_node_a", Target: "nas_a", Bus: "storage_bus"},
				{Source: "nas_a", Target: "librarian_a", Bus: "storage_bus"},
				{Source: "librarian_a", Target: "gateway_a", Bus: "api_bus"},
				// Live telemetry path: supervisor + thermal gateway → gateway_a → web UI
				{Source: "thermal_supervisor_pc", Target: "gateway_a", Bus: "api_bus"},
				{Source: "supervisor_a", Target: "gateway_a", Bus: "api_bus"},
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
	if err := writeJSON(filepath.Join(base, "graph_wall_manifest.json"), buildGraphWallManifest(set.Manifest.Envelope)); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(base, "source_tree_config.json"), buildSourceTreeConfig(set.Manifest.Envelope)); err != nil {
		return err
	}
	for _, id := range CampaignIDs {
		if err := writeJSON(filepath.Join(base, "campaigns", id+".json"), set.Campaigns[id]); err != nil {
			return err
		}
		telemetry := telemetryWithGraphTraces(set.Telemetry[id], set.GraphModels[id])
		if err := arrowtelemetry.WriteCampaign(filepath.Join(base, "telemetry", id+".arrow"), id, sliceToChan(telemetry), arrowtelemetry.MetadataFromGraph(set.GraphModels[id])); err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(base, "graph_models", id+".json"), set.GraphModels[id]); err != nil {
			return err
		}
	}
	commandCenterTelemetry := telemetryWithGraphTraces(set.Telemetry[CommandCenterGraphCampaignID], set.GraphModels[CommandCenterGraphCampaignID])
	if err := arrowtelemetry.WriteCampaign(filepath.Join(base, "telemetry", CommandCenterGraphCampaignID+".arrow"), CommandCenterGraphCampaignID, sliceToChan(commandCenterTelemetry), arrowtelemetry.MetadataFromGraph(set.GraphModels[CommandCenterGraphCampaignID])); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(base, "graph_models", CommandCenterGraphCampaignID+".json"), set.GraphModels[CommandCenterGraphCampaignID]); err != nil {
		return err
	}
	return nil
}

func buildGraphWallManifest(env contracts.Envelope) contracts.GraphWallManifest {
	return contracts.GraphWallManifest{
		Envelope: env,
		Targets: []contracts.GraphWallTarget{
			{TargetID: "graph_wall.thermal.chamber_a.air", Lane: "thermal", Role: "temperature_primary", SourceID: "chamber_thermal_fat", Timestamp: env.GeneratedAt},
			{TargetID: "graph_wall.tvac.pressure.main", Lane: "tvac", Role: "pressure_primary", SourceID: "chamber_pressure_tvac", Timestamp: env.GeneratedAt},
			{TargetID: "graph_wall.transport.tmtc.primary", Lane: "transport", Role: "decoded_tmtc", SourceID: "tvac_tmtc_primary", Timestamp: env.GeneratedAt},
			{TargetID: "graph_wall.archive.quality", Lane: "evidence", Role: "source_quality", SourceID: "archive_quality", Timestamp: env.GeneratedAt},
		},
	}
}

func buildSourceTreeConfig(env contracts.Envelope) contracts.SourceTreeConfig {
	views := []contracts.SourceTreeView{
		{
			// FAT: 4 thermal chambers only; infra listed last so it appears at bottom of table
			ID:    "thermal_acceptance_fat",
			Label: "Acceptance FAT",
			SourceIDs: []string{
				"chamber_thermal_fat", "chamber_thermal_b", "chamber_thermal_c", "chamber_thermal_d",
				"chamber_infra_fat",
			},
		},
		{
			ID:    "tvac_qualification",
			Label: "Qualification TVac",
			SourceIDs: []string{
				"dut_power", "dut_control", "dut_link", "dut_thermal",
				"chamber_thermal_tvac", "chamber_infra_tvac", "chamber_pressure_tvac",
				"tvac_plc", "archive_bus", "archive_quality",
			},
		},
		{
			// command center uses aggregate per-chamber lanes not mapped 1:1 to catalogue IDs
			ID:    "command_center_fat",
			Label: "Command Center",
			SourceIDs: []string{
				"chamber_thermal_fat", "chamber_thermal_b", "chamber_thermal_c", "chamber_thermal_d",
				"dut_thermal", "supervisor_state", "thermal_supervisor",
			},
		},
	}

	return contracts.SourceTreeConfig{Envelope: env, Views: views}
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
	for _, group := range model.HeroGraph.CompanionGroups {
		for _, trace := range group.Traces {
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
	}
	return out
}

func buildSources(env contracts.Envelope) contracts.SourceCatalogue {
	sources := []contracts.Source{
		// DUT — telemetry flows to archive_node_a, then librarian indexes and gateway serves to clients
		{ID: "dut_power", Label: "DUT Power", NodeID: "reference_dut", ServedBy: "archive_node_a", Owner: "egse_power_role", Bus: "telemetry_bus", Quality: "fresh", FreshnessMS: 250, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"eps_bus_voltage_v", "eps_bus_current_a", "dut_self_heat_w"}},
		{ID: "dut_control", Label: "DUT Control", NodeID: "reference_dut", ServedBy: "archive_node_a", Owner: "subsystem_test_role", Bus: "subsystem_event_bus", Quality: "fresh", FreshnessMS: 500, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"obc_boot_state", "obc_command_counter", "tc_packet_counter"}},
		{ID: "dut_link", Label: "DUT Link", NodeID: "reference_dut", ServedBy: "archive_node_a", Owner: "subsystem_test_role", Bus: "telemetry_bus", Quality: "fresh", FreshnessMS: 450, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"rf_link_margin_db", "tm_packet_counter"}},
		{ID: "dut_thermal", Label: "DUT Thermal Model", NodeID: "reference_dut", ServedBy: "archive_node_a", Owner: "test_conductor_role", Bus: "derived_model_bus", Quality: "synthetic", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"dut_fast_component_deg_c", "dut_lazy_component_deg_c", "dut_fast_air_flux_w", "dut_fast_interface_flux_w", "dut_fast_shroud_flux_w", "dut_lazy_air_flux_w", "dut_lazy_interface_flux_w", "dut_lazy_shroud_flux_w"}},
		// Thermal Chamber A PLC — polled by thermal_supervisor_pc which serves it to gateway_a
		{ID: "chamber_thermal_fat", Label: "Chamber Thermal Alpha", NodeID: "thermal_chamber_a", ServedBy: "thermal_supervisor_pc", Owner: "facility_operator_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"thermal_cycle_index", "thermal_phase_code", "chamber_setpoint_deg_c", "thermal_zone_1_deg_c", "thermal_zone_2_deg_c", "chamber_air_deg_c", "interface_plate_deg_c", "thermal_shroud_deg_c", "thermal_shroud_inlet_deg_c", "thermal_shroud_outlet_deg_c", "thermal_shroud_gradient_deg_c", "huber_table_deg_c", "ln2_valve_duty_pct"}},
		{ID: "chamber_infra_fat", Label: "Building Infrastructure Alpha", NodeID: "thermal_chamber_a", ServedBy: "thermal_supervisor_pc", Owner: "facility_operator_role", Bus: "facility_utility_bus", Quality: "fresh", FreshnessMS: 600, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"cooling_water_freeze_margin_deg_c", "cooling_water_temp_deg_c", "pressurized_air_supply_bar", "air_dewpoint_deg_c"}},
		// Thermal Chambers B/C/D PLCs — available; polled by thermal_supervisor_pc
		{ID: "chamber_thermal_b", Label: "Chamber Thermal Bravo", NodeID: "thermal_chamber_b", ServedBy: "thermal_supervisor_pc", Owner: "facility_operator_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"thermal_cycle_index", "thermal_phase_code", "chamber_setpoint_deg_c", "thermal_zone_1_deg_c", "thermal_zone_2_deg_c", "chamber_air_deg_c"}},
		{ID: "chamber_thermal_c", Label: "Chamber Thermal Charlie", NodeID: "thermal_chamber_c", ServedBy: "thermal_supervisor_pc", Owner: "facility_operator_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"thermal_cycle_index", "thermal_phase_code", "chamber_setpoint_deg_c", "thermal_zone_1_deg_c", "thermal_zone_2_deg_c", "chamber_air_deg_c"}},
		{ID: "chamber_thermal_d", Label: "Chamber Thermal Delta", NodeID: "thermal_chamber_d", ServedBy: "thermal_supervisor_pc", Owner: "facility_operator_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"thermal_cycle_index", "thermal_phase_code", "chamber_setpoint_deg_c", "thermal_zone_1_deg_c", "thermal_zone_2_deg_c", "chamber_air_deg_c"}},
		// Thermal Supervisor PC — aggregated view of all thermal chambers; served to gateway_a
		{ID: "thermal_supervisor", Label: "Thermal Supervisor PC State", NodeID: "thermal_supervisor_pc", ServedBy: "gateway_a", Owner: "facility_operator_role", Bus: "api_bus", Quality: "fresh", FreshnessMS: 500, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"active_chamber_count", "chamber_interlock_any", "thermal_schedule_state", "thermal_supervisor_uptime_s"}},
		// House Control PLC — facility-wide utilities; polled by thermal_supervisor_pc
		{ID: "house_utilities", Label: "House Control PLC", NodeID: "house_plc", ServedBy: "thermal_supervisor_pc", Owner: "facility_operator_role", Bus: "facility_utility_bus", Quality: "fresh", FreshnessMS: 1000, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"facility_mains_voltage_v", "facility_cooling_water_supply_bar", "facility_compressed_air_bar", "facility_ln2_supply_bar", "facility_power_ok", "facility_emergency_stop"}},
		// TVac PLC Q1 — controls LN2 cryo circuit and vacuum pumps; read by tvac_computer_1/2
		{ID: "tvac_plc", Label: "TVac PLC (Cryo & Vacuum)", NodeID: "tvac_plc_q1", ServedBy: "tvac_computer_1", Owner: "facility_operator_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"tvac_cryo_setpoint_deg_c", "tvac_cryo_actual_deg_c", "tvac_pump_state", "tvac_roughing_pressure_mbar", "tvac_turbo_speed_pct", "ln2_fill_valve_state", "ln2_vent_valve_state", "facility_interlock_state"}},
		// TVac Chamber Q1 — thermal and pressure sensors inside the chamber; collected by tvac_computer_1
		{ID: "chamber_thermal_tvac", Label: "Chamber Thermal (TVac)", NodeID: "tvac_chamber_q1", ServedBy: "tvac_computer_1", Owner: "tvac_computer_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"thermal_cycle_index", "thermal_phase_code", "chamber_setpoint_deg_c", "thermal_zone_1_deg_c", "thermal_zone_2_deg_c", "chamber_air_deg_c", "interface_plate_deg_c", "thermal_shroud_deg_c", "thermal_shroud_inlet_deg_c", "thermal_shroud_outlet_deg_c", "thermal_shroud_gradient_deg_c", "huber_table_deg_c", "ln2_line_temp_deg_c", "ln2_valve_duty_pct", "tvac_cryo_exhaust_temp_deg_c"}},
		{ID: "chamber_pressure_tvac", Label: "Facility Pressure (TVac)", NodeID: "tvac_chamber_q1", ServedBy: "tvac_computer_1", Owner: "tvac_computer_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 300, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"tvac_pressure_mbar", "tvac_pressure_pa"}},
		{ID: "chamber_infra_tvac", Label: "Building Infrastructure (TVac)", NodeID: "tvac_chamber_q1", ServedBy: "tvac_computer_1", Owner: "facility_operator_role", Bus: "facility_utility_bus", Quality: "fresh", FreshnessMS: 600, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"cooling_water_freeze_margin_deg_c", "cooling_water_temp_deg_c", "pressurized_air_supply_bar", "air_dewpoint_deg_c", "tvac_scavenged_exhaust_temp_deg_c", "tvac_scavenger_cooling_water_return_deg_c", "tvac_exhaust_cold_recovery_pct", "tvac_exhaust_duct_safe"}},
		// TVac Computer 1 — primary transport relay and test-script host; routes to supervisor_a then gateway_a
		{ID: "tvac_tmtc_primary", Label: "TVac Transport Primary (Computer 1)", NodeID: "tvac_computer_1", ServedBy: "supervisor_a", Owner: "tvac_computer_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 200, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"tmtc_link_state", "tc_queue_depth", "tm_frame_rate_hz", "last_tc_timestamp"}},
		// TVac Computer 2 — transport backup and independent data recorder; also routes to supervisor_a
		{ID: "tvac_tmtc_backup", Label: "TVac Transport Backup (Computer 2)", NodeID: "tvac_computer_2", ServedBy: "supervisor_a", Owner: "tvac_computer_role", Bus: "facility_control_bus", Quality: "fresh", FreshnessMS: 200, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"tmtc_link_state", "tc_queue_depth", "tm_frame_rate_hz", "backup_recording_state"}},
		// Archive node — logs live telemetry in Arrow IPC; supervisor or librarian triggers HDF5 export to NAS on test completion
		{ID: "archive_bus", Label: "Bus Virtualization Tap (Arrow IPC)", NodeID: "archive_node_a", ServedBy: "librarian_a", Owner: "test_conductor_role", Bus: "telemetry_bus", Quality: "synthetic", FreshnessMS: 200, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"bus_latency_ms", "tm_packet_counter", "tc_packet_counter", "overall_packet_counter", "dropped_frame_count"}},
		{ID: "archive_quality", Label: "Source Quality Monitor", NodeID: "archive_node_a", ServedBy: "librarian_a", Owner: "test_conductor_role", Bus: "telemetry_bus", Quality: "synthetic", FreshnessMS: 1000, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"source_freshness_ms", "facility_interlock_state"}},
		// NAS — long-term storage of completed tests as HDF5, plus legacy data (CSV, TXT, TDMS); all indexed by librarian_a
		{ID: "nas_exports", Label: "NAS Archival Store (HDF5 + legacy)", NodeID: "nas_a", ServedBy: "librarian_a", Owner: "test_conductor_role", Bus: "storage_bus", Quality: "fresh", FreshnessMS: 5000, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "primary", Signals: []string{"export_file_count", "export_total_size_gb", "latest_export_timestamp", "oldest_export_timestamp", "legacy_file_count"}},
		// Librarian — indexes NAS (HDF5, CSV, TXT, TDMS) and live Arrow IPC; any gateway can subscribe to translated streams on demand
		{ID: "librarian_index", Label: "Librarian (multi-format index & live translation)", NodeID: "librarian_a", ServedBy: "gateway_a", Owner: "test_conductor_role", Bus: "api_bus", Quality: "fresh", FreshnessMS: 2000, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"indexed_file_count", "live_translation_active", "translation_lag_ms", "last_index_scan_timestamp", "legacy_formats_active"}},
		// Gateway — backend API; serves all translated and live data to web UI clients
		{ID: "gateway_status", Label: "Data Gateway Status", NodeID: "gateway_a", ServedBy: "gateway_a", Owner: "test_conductor_role", Bus: "api_bus", Quality: "fresh", FreshnessMS: 1000, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"api_request_rate_hz", "connected_clients", "cache_hit_rate_pct", "gateway_uptime_s"}},
		// Supervisor — campaign orchestration; aggregates all feeds and routes to gateway_a
		{ID: "supervisor_state", Label: "Campaign Supervisor State", NodeID: "supervisor_a", ServedBy: "gateway_a", Owner: "test_conductor_role", Bus: "supervisor_bus", Quality: "fresh", FreshnessMS: 1000, Provenance: "synthetic_fixture_generator", EvidenceSuitability: "supporting", Signals: []string{"functional_gate_code", "facility_interlock_code"}},
	}
	enrichSourceOwnership(sources)
	return contracts.SourceCatalogue{
		Envelope: env,
		Sources:  sources,
		Tree: contracts.BuildSourceDiscoveryTree(sources, contracts.SourceDiscoveryTreeOptions{
			NodeLabeler:      nodeLabel,
			DeviceLabeler:    contracts.SourceDiscoveryDefaultLabel,
			SubsystemLabeler: contracts.SourceDiscoveryDefaultLabel,
		}),
	}
}

type sourceOwnershipProfile struct {
	ownerMode        string
	use              string
	formatPreference string
	device           string
	subsystem        string
}

func enrichSourceOwnership(sources []contracts.Source) {
	profiles := map[string]sourceOwnershipProfile{
		"dut_power":             {ownerMode: "exclusive_connection", use: "primary", formatPreference: "decoded", device: "dut_eps", subsystem: "power"},
		"dut_control":           {ownerMode: "external_master", use: "primary", formatPreference: "decoded", device: "dut_obc", subsystem: "command"},
		"dut_link":              {ownerMode: "shared_monitor", use: "shared", formatPreference: "decoded", device: "dut_tm_link", subsystem: "telemetry"},
		"dut_thermal":           {ownerMode: "derived", use: "derivative", formatPreference: "decoded", device: "thermal_model", subsystem: "thermal_model"},
		"chamber_thermal_fat":   {ownerMode: "external_master", use: "primary", formatPreference: "decoded", device: "chamber_a_plc", subsystem: "thermal"},
		"chamber_infra_fat":     {ownerMode: "shared_monitor", use: "shared", formatPreference: "decoded", device: "building_services", subsystem: "utilities"},
		"chamber_thermal_b":     {ownerMode: "external_master", use: "primary", formatPreference: "decoded", device: "chamber_b_plc", subsystem: "thermal"},
		"chamber_thermal_c":     {ownerMode: "external_master", use: "primary", formatPreference: "decoded", device: "chamber_c_plc", subsystem: "thermal"},
		"chamber_thermal_d":     {ownerMode: "external_master", use: "primary", formatPreference: "decoded", device: "chamber_d_plc", subsystem: "thermal"},
		"thermal_supervisor":    {ownerMode: "exclusive_connection", use: "primary", formatPreference: "decoded", device: "thermal_supervisor_api", subsystem: "supervision"},
		"house_utilities":       {ownerMode: "external_master", use: "shared", formatPreference: "decoded", device: "house_plc", subsystem: "utilities"},
		"tvac_plc":              {ownerMode: "external_master", use: "primary", formatPreference: "decoded", device: "tvac_plc_q1", subsystem: "cryo_vacuum"},
		"chamber_thermal_tvac":  {ownerMode: "external_master", use: "primary", formatPreference: "decoded", device: "tvac_chamber_q1", subsystem: "thermal"},
		"chamber_pressure_tvac": {ownerMode: "external_master", use: "primary", formatPreference: "decoded", device: "tvac_chamber_q1", subsystem: "pressure"},
		"chamber_infra_tvac":    {ownerMode: "shared_monitor", use: "shared", formatPreference: "decoded", device: "tvac_services", subsystem: "utilities"},
		"tvac_tmtc_primary":     {ownerMode: "exclusive_connection", use: "primary", formatPreference: "decoded", device: "tmtc_primary_console", subsystem: "tmtc"},
		"tvac_tmtc_backup":      {ownerMode: "fallback", use: "fallback", formatPreference: "raw_legacy", device: "tmtc_backup_recorder", subsystem: "tmtc"},
		"archive_bus":           {ownerMode: "shared_monitor", use: "shared", formatPreference: "decoded", device: "arrow_tap", subsystem: "transport"},
		"archive_quality":       {ownerMode: "derived", use: "derivative", formatPreference: "decoded", device: "quality_monitor", subsystem: "observability"},
		"nas_exports":           {ownerMode: "external_master", use: "fallback", formatPreference: "raw_legacy", device: "nas_archive", subsystem: "storage"},
		"librarian_index":       {ownerMode: "derived", use: "derivative", formatPreference: "decoded", device: "librarian_indexer", subsystem: "catalogue"},
		"gateway_status":        {ownerMode: "exclusive_connection", use: "shared", formatPreference: "decoded", device: "gateway_api", subsystem: "serving"},
		"supervisor_state":      {ownerMode: "exclusive_connection", use: "primary", formatPreference: "decoded", device: "campaign_supervisor", subsystem: "campaign"},
	}
	for i := range sources {
		profile := profiles[sources[i].ID]
		if profile.ownerMode == "" {
			profile = sourceOwnershipProfile{ownerMode: "external_master", use: "primary", formatPreference: "decoded", device: sources[i].NodeID, subsystem: sources[i].Bus}
		}
		sources[i].OwnerMode = profile.ownerMode
		sources[i].Use = profile.use
		sources[i].FormatPreference = profile.formatPreference
		sources[i].DiscoveryPath = contracts.SourceDiscoveryPath{
			Node:      sources[i].NodeID,
			Device:    profile.device,
			Subsystem: profile.subsystem,
			Stream:    sources[i].ID,
		}
	}
}

func discoveryLabel(id string) string {
	overrides := map[string]string{
		"chamber_a_plc": "Chamber Alpha PLC",
		"chamber_b_plc": "Chamber Bravo PLC",
		"chamber_c_plc": "Chamber Charlie PLC",
		"chamber_d_plc": "Chamber Delta PLC",
	}
	if label, ok := overrides[id]; ok {
		return label
	}
	return contracts.SourceDiscoveryDefaultLabel(id)
}

func nodeLabel(id string) string {
	labels := map[string]string{
		"reference_dut":         "Reference DUT",
		"thermal_chamber_a":     "Chamber Alpha PLC",
		"thermal_chamber_b":     "Chamber Bravo PLC",
		"thermal_chamber_c":     "Chamber Charlie PLC",
		"thermal_chamber_d":     "Chamber Delta PLC",
		"thermal_supervisor_pc": "Thermal Supervisor PC",
		"tvac_chamber_q1":       "TVac Chamber Q1",
		"tvac_plc_q1":           "TVac PLC Q1",
		"tvac_computer_1":       "TVac Computer 1",
		"tvac_computer_2":       "TVac Computer 2",
		"house_plc":             "House Control PLC",
		"archive_node_a":        "Archive Node A",
		"nas_a":                 "NAS A",
		"librarian_a":           "Librarian",
		"gateway_a":             "Gateway A",
		"supervisor_a":          "Supervisor A",
	}
	if label, ok := labels[id]; ok {
		return label
	}
	return discoveryLabel(id)
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
	c.Requirements = defaultRequirements(c)
	return c
}

func defaultRequirements(campaign contracts.Campaign) []contracts.Requirement {
	ids := []string{"REQ-STABILITY", "REQ-FUNC-GATE-PRE", "REQ-FUNC-GATE-DURING", "REQ-FUNC-GATE-POST", "REQ-DATA-QUALITY", "REQ-ANOMALY-REVIEW"}
	if campaign.ThermalProgram != nil {
		ids = append([]string{"REQ-CYCLE-COUNT", "REQ-HOT-TARGET", "REQ-COLD-TARGET", "REQ-HOT-SURVIVAL", "REQ-COLD-SURVIVAL", "REQ-DWELL", "REQ-FUNC-GATE-SURVIVAL"}, ids...)
	}
	reqs := make([]contracts.Requirement, 0, len(ids))
	for _, id := range ids {
		r := "pass"
		if campaign.Result == "inconclusive" && (id == "REQ-DATA-QUALITY" || id == "REQ-ANOMALY-REVIEW") {
			r = "inconclusive"
		}
		reqs = append(reqs, contracts.Requirement{
			ID:          id,
			Title:       requirementTitle(id),
			Description: "Requirement used to demonstrate measurement-to-evidence traceability.",
			Expression:  getExpression(id, campaign),
			Result:      r,
			Evidence:    []string{"telemetry", "graph_model"},
			Rationale:   "Evaluated from deterministic fixture data.",
		})
	}
	return reqs
}

func getExpression(id string, c contracts.Campaign) string {
	hasThermal := c.ThermalProgram != nil
	switch id {
	case "REQ-CYCLE-COUNT":
		if hasThermal {
			return `observed_cycle_count() == campaign_cycle_count`
		}
		return `max_signal("chamber_air_deg_c") >= 48.0 && min_signal("chamber_air_deg_c") <= -18.0 && sample_count() >= 40`
	case "REQ-HOT-TARGET":
		if hasThermal {
			return `max_signal("chamber_air_deg_c") >= campaign_hot_target - 2.0`
		}
		return `max_signal("chamber_air_deg_c") >= 48.0`
	case "REQ-COLD-TARGET":
		if hasThermal {
			return `min_signal("chamber_air_deg_c") <= campaign_cold_target + 2.0`
		}
		return `min_signal("chamber_air_deg_c") <= -18.0`
	case "REQ-HOT-SURVIVAL":
		return `observed_phase_count("hot_survival") > 0`
	case "REQ-COLD-SURVIVAL":
		return `observed_phase_count("cold_survival") > 0`
	case "REQ-STABILITY":
		return `max_signal("source_freshness_ms") <= 3500.0`
	case "REQ-DWELL":
		if hasThermal {
			return `observed_phase_count("cold_operational") >= campaign_cycle_count && observed_phase_count("hot_operational") >= campaign_cycle_count`
		}
		return `sample_count() >= 32`
	case "REQ-FUNC-GATE-PRE":
		if c.ID == "tvac_qualification" {
			return `observed_gate("ambient_pre") && observed_gate("vacuum_pre")`
		}
		return `observed_gate("pre")`
	case "REQ-FUNC-GATE-SURVIVAL":
		return `observed_gate("hot") && observed_gate("cold")`
	case "REQ-FUNC-GATE-DURING":
		if hasThermal {
			return `observed_gate("cold") && observed_gate("hot")`
		}
		return `observed_gate("during")`
	case "REQ-FUNC-GATE-POST":
		if c.ID == "tvac_qualification" {
			return `observed_gate("vacuum_post") && observed_gate("post")`
		}
		return `observed_gate("post")`
	case "REQ-DATA-QUALITY":
		return `no_degraded()`
	case "REQ-ANOMALY-REVIEW":
		return `anomaly_closed()`
	}
	return "true"
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
				"functional_gate":          ambientFunctionalGate(i),
			},
		})
	}
	return out
}

func ambientFunctionalGate(i int) string {
	if i <= 3 {
		return "pre"
	}
	if i >= 44 {
		return "post"
	}
	return "during"
}

func buildThermalTelemetry(campaign contracts.Campaign) []contracts.TelemetrySample {
	return environmentalsim.Simulate(campaign.ID, campaign.ThermalProgram, FixedTime).Samples
}

func buildGraphModel(env contracts.Envelope, campaign contracts.Campaign) contracts.GraphModel {
	isTVac := campaign.ThermalProgram != nil && campaign.ThermalProgram.Kind == "tvac_qualification"
	actuatorLabel := "Cooling Actuator Duty"
	chamberThermalSrc := "chamber_thermal_fat"
	chamberInfraSrc := "chamber_infra_fat"
	chamberNode := "thermal_chamber_a"
	if isTVac {
		actuatorLabel = "LN2 Valve Duty"
		chamberThermalSrc = "chamber_thermal_tvac"
		chamberInfraSrc = "chamber_infra_tvac"
		chamberNode = "tvac_chamber_q1"
	}
	model := contracts.GraphModel{
		Envelope:   env,
		CampaignID: campaign.ID,
		Lanes: []contracts.GraphLane{
			{ID: "thermal", Label: "Thermal", Series: []contracts.GraphSeries{
				{ID: "chamber_setpoint_deg_c", Label: "Chamber Setpoint", Role: "facility_setpoint", Units: "degC", Source: chamberThermalSrc, NodeID: chamberNode, Min: -45, Max: 75},
				{ID: "chamber_air_deg_c", Label: "Chamber Air", Role: "facility_environment", Units: "degC", Source: chamberThermalSrc, NodeID: chamberNode, Min: -45, Max: 75},
				{ID: "thermal_zone_1_deg_c", Label: "Thermal Zone 1", Role: "article_temperature", Units: "degC", Source: "dut_thermal", NodeID: "reference_dut", Min: -45, Max: 75},
			}},
			{ID: "pressure", Label: "Pressure", Series: []contracts.GraphSeries{
				{ID: "tvac_pressure_mbar", Label: "TVac pressure", Role: "facility_environment", Units: "mbar", Source: "chamber_pressure_tvac", NodeID: "tvac_chamber_q1", Min: 0.00000001, Max: 1013.25},
			}},
			{ID: "facility_safety", Label: "Facility Safety", Series: []contracts.GraphSeries{
				{ID: "ln2_valve_duty_pct", Label: actuatorLabel, Role: "facility_interlock", Units: "%", Source: chamberThermalSrc, NodeID: chamberNode, Min: 0, Max: 100},
				{ID: "cooling_water_freeze_margin_deg_c", Label: "Water scavenger freeze margin", Role: "facility_interlock", Units: "degC", Source: chamberInfraSrc, NodeID: chamberNode, Min: 0, Max: 25},
			}},
			{ID: "power", Label: "Power", Series: []contracts.GraphSeries{
				{ID: "eps_bus_voltage_v", Label: "Bus Voltage", Role: "dut_power", Units: "V", Source: "dut_power", NodeID: "reference_dut", Min: 26, Max: 30},
				{ID: "eps_bus_current_a", Label: "Bus Current", Role: "dut_power", Units: "A", Source: "dut_power", NodeID: "reference_dut", Min: 0, Max: 8},
			}},
			{ID: "bus", Label: "Virtual Bus", Series: []contracts.GraphSeries{
				{ID: "bus_latency_ms", Label: "Bus Latency", Role: "virtual_bus_health", Units: "ms", Source: "archive_bus", NodeID: "archive_node_a", Min: 0, Max: 250},
				{ID: "tm_packet_counter", Label: "TM Counter", Role: "telemetry_counter", Units: "count", Source: "archive_bus", NodeID: "archive_node_a", Min: 0, Max: 8000},
			}},
			{ID: "quality", Label: "Source Quality", Series: []contracts.GraphSeries{
				{ID: "source_freshness_ms", Label: "Source Freshness", Role: "data_quality", Units: "ms", Source: "archive_quality", NodeID: "archive_node_a", Min: 0, Max: 6000},
			}},
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
	isTVac := campaignID == "tvac_qualification"
	chamberThermalSrc := "chamber_thermal_fat"
	chamberPressureSrc := "chamber_pressure_tvac"
	chamberInfraSrc := "chamber_infra_fat"
	if isTVac {
		chamberThermalSrc = "chamber_thermal_tvac"
		chamberInfraSrc = "chamber_infra_tvac"
	}
	_ = chamberPressureSrc
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
		TilePolicy: denseOperatorGraphTilePolicy(900, 3600, 96, 640, 192),
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
		graphSignal("trace.table_loop", "Fluid interface", "degC", chamberThermalSrc, "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
	}
	if campaignID != "tvac_qualification" {
		thermalProgramSignals = append(thermalProgramSignals[:3], append([]contracts.GraphWallSignal{
			graphSignal("trace.actual.chamber_air", "Chamber air actual", "degC", chamberThermalSrc, "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
		}, thermalProgramSignals[3:]...)...)
	}
	if campaignID == "tvac_qualification" {
		thermalProgramSignals = append(thermalProgramSignals,
			graphSignal("trace.tvac_pressure", "TVac pressure", "mbar", "chamber_pressure_tvac", "actual", "measurement", "facility", "pressure_mbar", "thermal_environment"),
			graphSignal("trace.tvac_pressure_target", "Vacuum target", "mbar", "requirements", "ghost", "target", "requirements", "pressure_mbar", "thermal_environment"),
			graphSignal("trace.shroud_inlet", "Shroud inlet", "degC", chamberThermalSrc, "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
			graphSignal("trace.shroud_outlet", "Shroud outlet", "degC", chamberThermalSrc, "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
		)
	}
	thermalTitle := "Chamber command, ghost, acceptance, actual"
	if campaignID == "tvac_qualification" {
		thermalTitle = "TVac command, pressure, shroud, interface, DUT"
	}
	add("thermal_environment", graphCard("thermal_program", thermalTitle, "line", "primary_hero", "degC", "temperature_c", chamberThermalSrc, thermalProgramSignals))
	dutTemperatureSignals := []contracts.GraphWallSignal{}
	if campaignID != "tvac_qualification" {
		dutTemperatureSignals = append(dutTemperatureSignals,
			graphSignal("trace.context.chamber_air", "Chamber air", "degC", chamberThermalSrc, "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
		)
	}
	dutTemperatureSignals = append(dutTemperatureSignals,
		graphSignal("trace.dut_temp_a", "High-dissipation DUT node", "degC", "dut_thermal", "actual", "measurement", "dut", "temperature_c", "thermal_environment"),
		graphSignal("trace.dut_temp_b", "Vacuum-detached DUT node", "degC", "dut_thermal", "actual", "measurement", "dut", "temperature_c", "thermal_environment"),
		graphSignal("trace.table_loop", "Fluid interface", "degC", chamberThermalSrc, "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
	)
	if campaignID == "tvac_qualification" {
		dutTemperatureSignals = append(dutTemperatureSignals,
			graphSignal("trace.shroud_inlet", "Shroud inlet", "degC", chamberThermalSrc, "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
			graphSignal("trace.shroud_outlet", "Shroud outlet", "degC", chamberThermalSrc, "actual", "measurement", "facility", "temperature_c", "thermal_environment"),
			graphSignal("trace.shroud_gradient", "Shroud gradient", "degC", chamberThermalSrc, "source_quality", "measurement", "facility", "temperature_c", "thermal_environment"),
		)
	}
	dutTemperatureTitle := "DUT temperatures in chamber context"
	if campaignID == "tvac_qualification" {
		dutTemperatureTitle = "DUT temperatures in TVac context"
	}
	add("thermal_environment", graphCard("dut_temperature", dutTemperatureTitle, "line", "companion", "degC", "temperature_c", "dut_thermal", dutTemperatureSignals))
	if campaignID == "tvac_qualification" {
		add("facility_response", graphCard("tvac_pressure", "TVac pressure pumpdown and bursts", "line", "companion", "mbar", "log_pressure_mbar", "chamber_pressure_tvac", []contracts.GraphWallSignal{
			graphSignal("trace.tvac_pressure", "Pressure", "mbar", "chamber_pressure_tvac", "actual", "measurement", "facility", "pressure_mbar", "facility_response"),
			graphSignal("trace.tvac_pressure_target", "Vacuum target", "mbar", "requirements", "ghost", "target", "requirements", "pressure_mbar", "facility_response"),
		}))
		add("facility_response", graphCard("tvac_pressure_sources", "Pump, leak, and outgassing balance", "line", "companion", "mixed", "pressure_balance", "chamber_pressure_tvac", []contracts.GraphWallSignal{
			graphSignal("trace.tvac_outgassing", "Temperature outgassing", "mbar/min", "chamber_pressure_tvac", "actual", "measurement", "facility", "pressure_rate", "facility_response"),
			graphSignal("trace.tvac_virtual_leak", "Virtual leak", "mbar/min", "chamber_pressure_tvac", "acceptance_band", "limit", "facility", "pressure_rate", "facility_response"),
			graphSignal("trace.tvac_roughing_pump", "Roughing pump", "mbar/min", "chamber_pressure_tvac", "source_quality", "measurement", "facility", "pressure_rate", "facility_response"),
			graphSignal("trace.tvac_turbo_pump", "Turbo pump", "mbar/min", "chamber_pressure_tvac", "actual", "measurement", "facility", "pressure_rate", "facility_response"),
			graphSignal("trace.tvac_pump_removal", "Pump removal", "mbar/min", "chamber_pressure_tvac", "source_quality", "measurement", "facility", "pressure_rate", "facility_response"),
			graphSignal("trace.tvac_volatile_inventory", "Volatile inventory", "%", "chamber_pressure_tvac", "ghost", "derived", "facility", "percent", "facility_response"),
		}))
	}
	actuationTitle := "Cooling actuator duty"
	actuationLabel := "Cooling demand"
	if campaignID == "tvac_qualification" {
		actuationTitle = "LN2 valve duty"
		actuationLabel = "LN2 valve"
	}
	add("facility_response", graphCard("facility_actuation", actuationTitle, "line", "companion", "%", "percent", chamberThermalSrc, []contracts.GraphWallSignal{
		graphSignal("trace.ln2_duty", actuationLabel, "%", chamberThermalSrc, "actual", "actuator", "facility", "percent", "facility_response"),
	}))
	if campaignID == "tvac_qualification" {
		add("facility_response", graphCard("facility_temperature_safety", "Heat-exchanger freeze margin", "line", "companion", "degC", "temperature_c", chamberInfraSrc, []contracts.GraphWallSignal{
			graphSignal("trace.freeze_margin", "Water scavenger freeze margin", "degC", chamberInfraSrc, "interlock", "safety_margin", "facility", "temperature_c", "facility_response"),
		}))
		add("facility_response", graphCard("tvac_exhaust_scavenger", "Exhaust cold scavenger", "line", "companion", "mixed", "facility_exhaust_scavenger", chamberInfraSrc, []contracts.GraphWallSignal{
			graphSignal("trace.tvac_cryo_exhaust", "Cryogenic exhaust", "degC", chamberThermalSrc, "actual", "measurement", "facility", "temperature_c", "facility_response"),
			graphSignal("trace.tvac_scavenged_exhaust", "After water scavenger", "degC", chamberInfraSrc, "actual", "measurement", "facility", "temperature_c", "facility_response"),
			graphSignal("trace.tvac_scavenger_water_return", "Scavenger water return", "degC", chamberInfraSrc, "actual", "measurement", "facility", "temperature_c", "facility_response"),
			graphSignal("trace.tvac_exhaust_cold_recovery", "Cold recovery", "%", chamberInfraSrc, "source_quality", "derived", "facility", "percent", "facility_response"),
		}))
	}
	add("facility_response", graphCard("building_infrastructure", "Building infrastructure", "line", "companion", "mixed", "facility_infra", chamberInfraSrc, []contracts.GraphWallSignal{
		graphSignal("trace.cooling_water_temp", "Cooling water temp", "degC", chamberInfraSrc, "actual", "measurement", "facility", "temperature_c", "facility_response"),
		graphSignal("trace.pressurized_air_supply", "Pressurized air supply", "bar", chamberInfraSrc, "actual", "measurement", "facility", "pressure_bar", "facility_response"),
		graphSignal("trace.air_dewpoint", "Air dew point", "degC", chamberInfraSrc, "actual", "measurement", "facility", "temperature_c", "facility_response"),
	}))
	add("dut_response", graphCard("dut_power", "DUT power budgets", "line", "companion", "W", "power_w", "dut_power", []contracts.GraphWallSignal{
		graphSignal("trace.power_total", "Total power", "W", "dut_power", "actual", "measurement", "dut_power", "power_w", "dut_response"),
		graphSignal("trace.power_subsystem", "Subsystem budget", "W", "dut_power", "actual", "measurement", "dut_power", "power_w", "dut_response"),
		graphSignal("trace.power_payload", "Payload/FT load", "W", "dut_power", "actual", "measurement", "dut_power", "power_w", "dut_response"),
		graphSignal("trace.power_avionics", "Avionics", "W", "dut_power", "actual", "measurement", "dut_power", "power_w", "dut_response"),
		graphSignal("trace.power_link", "Link subsystem", "W", "dut_power", "actual", "measurement", "dut_power", "power_w", "dut_response"),
	}))
	add("tmtc_response", graphCard("tmtc_health", "Transport latency and freshness", "line", "companion", "ms", "bus_ms", "archive_bus", []contracts.GraphWallSignal{
		graphSignal("trace.bus_latency", "Bus latency", "ms", "archive_bus", "source_quality", "measurement", "tmtc", "bus_ms", "tmtc_response"),
		graphSignal("trace.source_freshness", "Freshness", "ms", "archive_quality", "source_quality", "measurement", "tmtc", "bus_ms", "tmtc_response"),
	}))
	add("tmtc_response", graphCard("tmtc_counters", "Transport packet counters", "counter", "companion", "count", "counter", "archive_bus", []contracts.GraphWallSignal{
		graphSignal("trace.overall_packet_counter", "Overall packet counter", "count", "archive_bus", "counter", "counter", "tmtc", "counter", "tmtc_response"),
		graphSignal("trace.tm_packet_counter", "Downlink packet counter", "count", "archive_bus", "counter", "counter", "tmtc", "counter", "tmtc_response"),
		graphSignal("trace.tc_packet_counter", "Uplink packet counter", "count", "archive_bus", "counter", "counter", "tmtc", "counter", "tmtc_response"),
		graphSignal("trace.dropped_frame_count", "Dropped frames", "count", "archive_bus", "counter", "counter", "tmtc", "counter", "tmtc_response"),
	}))
	add("tmtc_response", graphCard("source_quality", "Source freshness quality", "line", "companion", "ms", "bus_ms", "archive_quality", []contracts.GraphWallSignal{
		graphSignal("trace.source_freshness", "Source freshness", "ms", "archive_quality", "source_quality", "measurement", "tmtc", "bus_ms", "tmtc_response"),
	}))
	stateSignals := []contracts.GraphWallSignal{
		enumSignal("trace.phase_enum", "Thermal phase", "thermal_program", "enum", "testbed_dynamics", map[string]string{"0": "ambient pre", "1": "ramp cold", "2": "cold op", "3": "ramp hot", "4": "hot op", "5": "hot survival", "6": "cold survival", "7": "vacuum ambient", "8": "ambient recovery", "9": "ambient post"}),
		enumSignal("trace.functional_gate_active", "Functional gate active", "test_conductor", "bool", "testbed_dynamics", map[string]string{"0": "inactive", "1": "active"}),
		enumSignal("trace.stability_reached", "Stability reached", "thermal_program", "bool", "testbed_dynamics", map[string]string{"0": "stabilizing", "1": "stable"}),
		enumSignal("trace.dwell_active", "Dwell active", "thermal_program", "bool", "testbed_dynamics", map[string]string{"0": "idle", "1": "dwelling"}),
		enumSignal("trace.dwell_complete", "Dwell complete", "thermal_program", "bool", "testbed_dynamics", map[string]string{"0": "open", "1": "complete"}),
		enumSignal("trace.interlock_review", "Facility interlock state", "facility_safety", "fault", "testbed_dynamics", map[string]string{"1": "closed", "2": "review"}),
		enumSignal("trace.source_degraded", "Source degraded", "archive_quality", "bool", "testbed_dynamics", map[string]string{"0": "fresh", "1": "degraded"}),
		enumSignal("trace.evidence_capture", "Evidence capture", "evidence_report", "bool", "testbed_dynamics", map[string]string{"0": "idle", "1": "capture"}),
		enumSignal("trace.dut_ready", "DUT ready", "dut_control", "bool", "testbed_dynamics", map[string]string{"0": "not ready", "1": "ready"}),
		enumSignal("trace.dut_operative", "DUT operative", "dut_control", "bool", "testbed_dynamics", map[string]string{"0": "inhibited", "1": "operative"}),
		enumSignal("trace.payload_active", "Payload active", "dut_power", "bool", "testbed_dynamics", map[string]string{"0": "standby", "1": "active"}),
		enumSignal("trace.rf_link_locked", "RF link locked", "dut_link", "bool", "testbed_dynamics", map[string]string{"0": "searching", "1": "locked"}),
		enumSignal("trace.fault_flag", "Fault flag", "archive_quality", "fault", "testbed_dynamics", map[string]string{"0": "nominal", "1": "fault"}),
	}
	if campaignID == "tvac_qualification" {
		stateSignals = append(stateSignals[:5], append([]contracts.GraphWallSignal{enumSignal("trace.pressure_gate", "Pressure gate", "chamber_pressure_tvac", "bool", "testbed_dynamics", map[string]string{"0": "waiting", "1": "reached"})}, stateSignals[5:]...)...)
		stateSignals = append(stateSignals[:6], append([]contracts.GraphWallSignal{enumSignal("trace.pump_mode", "Pump mode", "chamber_pressure_tvac", "enum", "testbed_dynamics", map[string]string{"0": "ambient", "1": "roughing", "2": "crossover", "3": "turbo"})}, stateSignals[6:]...)...)
		stateSignals = append(stateSignals[:7], append([]contracts.GraphWallSignal{enumSignal("trace.exhaust_duct_safe", "Exhaust duct safe", chamberInfraSrc, "bool", "testbed_dynamics", map[string]string{"0": "scavenger warming", "1": "duct safe"})}, stateSignals[7:]...)...)
	}
	add("testbed_dynamics", graphCard("state_change_swimlane", "Testbed state swimlanes", "state", "swimlane", "state", "state_lane", "thermal_program", stateSignals))
	add("testbed_dynamics", graphCard("functional_events", "Functional gates and evidence events", "event", "event_rail", "event", "event", "test_conductor", []contracts.GraphWallSignal{
		graphSignal("functional_gates", "Functional gates", "event", "test_conductor", "event", "event", "test_conductor", "event", "testbed_dynamics"),
		graphSignal("evidence_markers", "Evidence markers", "event", "evidence_report", "evidence", "event", "evidence", "event", "testbed_dynamics"),
		graphSignal("interlock_windows", "Interlock windows", "event", "facility_safety", "interlock", "event", "facility", "event", "testbed_dynamics"),
	}))
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
			{ID: "binary_tmtc_log", Label: "Binary transport log import", SourceFormat: "binary_log", TargetSchema: arrowtelemetry.SchemaName, Mode: "translated_fixture", Confidence: "medium", Provenance: "synthetic DataLens translation demo"},
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
		return "Transport Response"
	case "testbed_dynamics":
		return "Testbed Dynamics"
	default:
		return id
	}
}

func heightWeight(kind, role string) float64 {
	if role == "primary_hero" {
		return 2.0
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
		supervisorLane("thermal_fat", "Thermal Chamber FAT", "thermal_chamber_a", campaigns["thermal_acceptance_fat"], "4 cycle chamber FAT with cold/hot dwell gates", "facility_control_bus", "4 cycles, hot/cold dwell, pre/cold/hot/post functional gates", "fresh", []string{"Thermal chamber profile is synchronized with DUT temperature."}, telemetry["thermal_acceptance_fat"], []heroSpec{{"chamber_setpoint_deg_c", "Setpoint", "facility_setpoint", "degC", "chamber_thermal_fat", -45, 75}, {"chamber_air_deg_c", "Chamber Air", "facility_environment", "degC", "chamber_thermal_fat", -45, 75}, {"thermal_zone_1_deg_c", "DUT Node 1", "article_temperature", "degC", "dut_thermal", -45, 75}, {"bus_latency_ms", "Transport Latency", "virtual_bus_health", "ms", "archive_bus", 0, 250}}),
		supervisorLane("eps_load_step", "EPS Load Step", "flatsat_rack_a", campaigns["integrated_system_fat"], "Power load and command script", "command_bus", "REQ-FUNC-GATE-DURING pass", "degraded", []string{"Synthetic freshness degradation demonstrates disposition workflow."}, telemetry["integrated_system_fat"], []heroSpec{{"eps_bus_voltage_v", "Bus Voltage", "dut_power", "V", "dut_power", 24, 32}, {"eps_bus_current_a", "Bus Current", "dut_power", "A", "dut_power", 0, 8}}),
		supervisorLane("payload_thermal", "Payload Thermal Cycle", "thermal_chamber_a", campaigns["integrated_system_fat"], "Payload simulator heater cycling", "telemetry_bus", "REQ-STABILITY pass", "fresh", []string{"Payload heater response is fictional and bounded for demo use."}, telemetry["integrated_system_fat"], []heroSpec{{"payload_sim_heater_w", "Payload Heater", "payload_thermal_control", "W", "chamber_thermal_fat", 0, 60}, {"thermal_zone_2_deg_c", "Article Zone 2", "article_temperature", "degC", "dut_thermal", -45, 70}}),
		supervisorLane("tvac_qualification", "TVac Qualification", "tvac_chamber_q1", campaigns["tvac_qualification"], "8 cycle TVac qualification with pumpdown and thermal-source review", "facility_control_bus", "8 cycles, pressure plateau, safety interlock review open", "degraded", []string{"Pressure-source degradation remains open for review."}, telemetry["tvac_qualification"], []heroSpec{{"chamber_setpoint_deg_c", "Setpoint", "facility_setpoint", "degC", "chamber_thermal_tvac", -45, 75}, {"thermal_zone_1_deg_c", "DUT Node 1", "article_temperature", "degC", "dut_thermal", -45, 75}, {"tvac_pressure_mbar", "TVac pressure", "facility_environment", "mbar", "chamber_pressure_tvac", 0.00000001, 1013.25}, {"ln2_valve_duty_pct", "TVac cooling valve duty", "facility_interlock", "%", "chamber_thermal_tvac", 0, 100}, {"cooling_water_freeze_margin_deg_c", "Water scavenger freeze margin", "facility_interlock", "degC", "chamber_infra_tvac", 0, 25}}),
		supervisorLane("archive_capture", "Archive Evidence Capture", "archive_node_a", campaigns["integrated_system_fat"], "Transport capture and evidence packaging", "telemetry_bus", "REQ-DATA-QUALITY pass with review note", "synthetic", []string{"Archive node receives virtualized transport events from the replay bus."}, telemetry["integrated_system_fat"], []heroSpec{{"bus_latency_ms", "Bus Latency", "virtual_bus_health", "ms", "archive_bus", 0, 250}, {"tm_packet_counter", "Packet counter", "telemetry_counter", "count", "archive_bus", 0, 8000}}),
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

var writeJSON = jsonfile.WriteIndent

// BuildFileViewModel builds a FileViewModel for the given campaign, deriving
// signal groups by source and node so the file viewer can show data provenance.
func BuildFileViewModel(env contracts.Envelope, campaignID string) (contracts.FileViewModel, bool) {
	set := Build()
	campaign, ok := set.Campaigns[campaignID]
	if !ok {
		return contracts.FileViewModel{}, false
	}
	model, ok := set.GraphModels[campaignID]
	if !ok {
		return contracts.FileViewModel{}, false
	}

	// Collect signal groups: one per (source, node) pair from lanes.
	type key struct{ source, node string }
	bySource := make(map[key][]contracts.GraphSeries)
	for _, lane := range model.Lanes {
		for _, s := range lane.Series {
			k := key{s.Source, s.NodeID}
			bySource[k] = append(bySource[k], s)
		}
	}

	// Build a source lookup map from the source catalogue.
	srcMap := make(map[string]contracts.Source)
	for _, s := range set.SourceCatalogue.Sources {
		srcMap[s.ID] = s
	}
	// Node label lookup.
	nodeLabels := map[string]string{
		"reference_dut":         "Reference DUT",
		"thermal_chamber_a":     "Chamber Alpha PLC",
		"thermal_chamber_b":     "Chamber Bravo PLC",
		"thermal_chamber_c":     "Chamber Charlie PLC",
		"thermal_chamber_d":     "Chamber Delta PLC",
		"thermal_supervisor_pc": "Thermal Supervisor PC",
		"tvac_chamber_q1":       "TVac Chamber Q1",
		"tvac_plc_q1":           "TVac PLC Q1",
		"tvac_computer_1":       "TVac Computer 1 (Primary)",
		"tvac_computer_2":       "TVac Computer 2 (Backup)",
		"flatsat_rack_a":        "Flatsat Rack A",
		"house_plc":             "House Control PLC",
		"archive_node_a":        "Archive Node A",
		"nas_a":                 "NAS A",
		"librarian_a":           "Librarian",
		"gateway_a":             "Data Gateway",
		"supervisor_a":          "Test Supervisor",
	}

	groups := make([]contracts.FileSignalGroup, 0, len(bySource))
	for k, series := range bySource {
		src := srcMap[k.source]
		nl := nodeLabels[k.node]
		if nl == "" {
			nl = k.node
		}
		groups = append(groups, contracts.FileSignalGroup{
			NodeID:      k.node,
			NodeLabel:   nl,
			SourceID:    k.source,
			SourceLabel: src.Label,
			Bus:         src.Bus,
			Series:      series,
		})
	}

	fileKind := "ambient_fat"
	if campaign.ThermalProgram != nil && campaign.ThermalProgram.Kind == "tvac_qualification" {
		fileKind = "tvac_qualification"
	} else if campaign.ThermalProgram != nil {
		fileKind = "thermal_fat"
	}

	return contracts.FileViewModel{
		Envelope:     env,
		CampaignID:   campaignID,
		CampaignName: campaign.Name,
		FileRef:      "telemetry/" + campaignID + ".arrow",
		FileKind:     fileKind,
		TimeStart:    campaign.Start,
		TimeEnd:      campaign.End,
		SignalGroups: groups,
		Lanes:        model.Lanes,
	}, true
}

func sliceToChan(samples []contracts.TelemetrySample) <-chan contracts.TelemetrySample {
	ch := make(chan contracts.TelemetrySample, len(samples))
	for _, s := range samples {
		ch <- s
	}
	close(ch)
	return ch
}
