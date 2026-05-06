package contracts

import "testing"

func TestValidateManifestAcceptsPublicSyntheticManifest(t *testing.T) {
	manifest := Manifest{
		Envelope:      Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Name:          "Gossamer",
		TestArticle:   "AuroraSat-1",
		Campaigns:     []string{"thermal_acceptance_fat"},
		PublicDemo:    true,
		SyntheticOnly: true,
	}

	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("expected valid manifest: %v", err)
	}
}

func TestValidateSourceCatalogueRejectsUnknownQuality(t *testing.T) {
	catalogue := SourceCatalogue{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Sources: []Source{{
			ID:      "thermal_source",
			Owner:   "test_conductor",
			Quality: "mystery",
		}},
	}

	if err := ValidateSourceCatalogue(catalogue); err == nil {
		t.Fatal("expected invalid source quality to fail")
	}
}

func TestValidateGraphModelRejectsSeriesWithoutUnitsOrRole(t *testing.T) {
	model := GraphModel{
		Envelope:   Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		CampaignID: "thermal_acceptance_fat",
		Lanes: []GraphLane{{
			ID: "thermal",
			Series: []GraphSeries{{
				ID: "chamber_air_deg_c",
			}},
		}},
	}

	if err := ValidateGraphModel(model); err == nil {
		t.Fatal("expected missing units and role to fail")
	}
}

func TestValidateCampaignRejectsUnknownRequirementResult(t *testing.T) {
	campaign := Campaign{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		ID:       "thermal_acceptance_fat",
		Name:     "Thermal Acceptance FAT",
		Result:   "pass",
		Requirements: []Requirement{{
			ID:     "REQ-DATA-QUALITY",
			Result: "maybe",
		}},
	}

	if err := ValidateCampaign(campaign); err == nil {
		t.Fatal("expected unknown requirement result to fail")
	}
}

func TestValidateSupervisorOverviewRequiresHeroTemperatureLane(t *testing.T) {
	overview := SupervisorOverview{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Lanes: []SupervisorLane{
			testSupervisorLane("thermal_fat", "thermal_chamber_a", "thermal_acceptance_fat", "thermal_zone_1_deg_c", "degC"),
			testSupervisorLane("eps_load", "flatsat_rack_a", "integrated_system_fat", "eps_bus_voltage_v", "V"),
			testSupervisorLane("payload_thermal", "thermal_chamber_a", "integrated_system_fat", "payload_sim_heater_w", "W"),
			testSupervisorLane("archive_capture", "archive_node_a", "integrated_system_fat", "bus_latency_ms", "ms"),
		},
	}

	if err := ValidateSupervisorOverview(overview); err != nil {
		t.Fatalf("expected valid supervisor overview: %v", err)
	}
}

func testSupervisorLane(id, facility, campaign, signal, units string) SupervisorLane {
	return SupervisorLane{
		ID:       id,
		Label:    id,
		Facility: facility,
		Campaign: campaign,
		State:    "running",
		HeroGraphs: []SupervisorHeroGraph{{
			ID:     signal,
			Label:  signal,
			Signal: signal,
			Units:  units,
			Role:   "demo_role",
			Source: "demo_source",
			Values: []GraphPoint{{Timestamp: "2026-01-15T10:00:00Z", Value: 21.5}},
		}},
	}
}

func TestValidateBusVirtualizationRequiresTMAndTCEvents(t *testing.T) {
	tap := BusVirtualizationTap{
		Envelope:     Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		ConnectionID: "bus_virtualization_demo",
		Streams: []BusStream{
			{ID: "tm_primary", Direction: "TM", SourceNode: "aurorasat_1", DestinationNode: "archive_node_a", Bus: "telemetry_bus", Quality: "fresh"},
			{ID: "tc_primary", Direction: "TC", SourceNode: "flatsat_rack_a", DestinationNode: "aurorasat_1", Bus: "command_bus", Quality: "fresh"},
		},
		Events: []BusEvent{
			{ID: "BUS-TM-0001", StreamID: "tm_primary", Direction: "TM", Timestamp: "2026-01-15T10:00:00Z", EventClass: "telemetry_sample", Quality: "fresh"},
			{ID: "BUS-TC-0001", StreamID: "tc_primary", Direction: "TC", Timestamp: "2026-01-15T10:00:05Z", EventClass: "command_request", Quality: "fresh"},
		},
	}

	if err := ValidateBusVirtualizationTap(tap); err != nil {
		t.Fatalf("expected valid bus tap: %v", err)
	}
}
