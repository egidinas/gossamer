package contracts

import "testing"

func TestValidateManifestAcceptsPublicSyntheticManifest(t *testing.T) {
	manifest := Manifest{
		Envelope:      Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Name:          "Gossamer",
		TestArticle:   "Reference DUT",
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

func TestValidateSourceCatalogueRequiresOwnershipVocabulary(t *testing.T) {
	catalogue := SourceCatalogue{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Sources: []Source{{
			ID:               "thermal_source",
			Owner:            "test_conductor",
			Quality:          "fresh",
			OwnerMode:        "external_master",
			Use:              "primary",
			FormatPreference: "decoded",
			DiscoveryPath:    SourceDiscoveryPath{Node: "thermal_supervisor_pc", Device: "chamber_a_plc", Subsystem: "thermal", Stream: "thermal_source"},
		}},
		Tree: []SourceDiscoveryNode{{
			ID:    "thermal_supervisor_pc",
			Label: "Thermal Supervisor PC",
			Kind:  "node",
			Children: []SourceDiscoveryNode{{
				ID:    "chamber_a_plc",
				Label: "Chamber A PLC",
				Kind:  "device",
				Children: []SourceDiscoveryNode{{
					ID:    "thermal",
					Label: "Thermal",
					Kind:  "subsystem",
					Children: []SourceDiscoveryNode{{
						ID:       "thermal_source",
						Label:    "Thermal Source",
						Kind:     "stream",
						SourceID: "thermal_source",
					}},
				}},
			}},
		}},
	}

	if err := ValidateSourceCatalogue(catalogue); err != nil {
		t.Fatalf("expected valid ownership vocabulary: %v", err)
	}

	catalogue.Sources[0].OwnerMode = "unclear"
	if err := ValidateSourceCatalogue(catalogue); err == nil {
		t.Fatal("expected invalid owner_mode to fail")
	}
}

func TestValidateSourceCatalogueRequiresBackendAuthoredTreeLeaves(t *testing.T) {
	catalogue := SourceCatalogue{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Sources: []Source{{
			ID:               "archive_bus",
			Owner:            "test_conductor",
			Quality:          "synthetic",
			OwnerMode:        "shared_monitor",
			Use:              "shared",
			FormatPreference: "decoded",
			DiscoveryPath:    SourceDiscoveryPath{Node: "archive_node_a", Device: "arrow_tap", Subsystem: "transport", Stream: "archive_bus"},
		}},
		Tree: []SourceDiscoveryNode{{
			ID:    "archive_node_a",
			Label: "Archive Node A",
			Kind:  "node",
		}},
	}

	if err := ValidateSourceCatalogue(catalogue); err == nil {
		t.Fatal("expected tree without source leaf to fail")
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
		Name:     "Thermal Chamber FAT",
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

func TestValidateGraphWallManifestAcceptsValidTargets(t *testing.T) {
	m := GraphWallManifest{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Targets: []GraphWallTarget{
			{TargetID: "graph_wall.thermal.chamber_a.air", Lane: "thermal", Role: "temperature_primary", SourceID: "chamber_thermal_fat", Timestamp: "2026-01-15T10:00:00Z"},
			{TargetID: "graph_wall.tvac.pressure.main", Lane: "tvac", Role: "pressure_primary", SourceID: "chamber_pressure_tvac", Timestamp: "2026-01-15T10:00:00Z"},
		},
	}
	if err := ValidateGraphWallManifest(m); err != nil {
		t.Fatalf("expected valid manifest: %v", err)
	}
}

func TestValidateGraphWallManifestRejectsDuplicateTargetID(t *testing.T) {
	m := GraphWallManifest{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Targets: []GraphWallTarget{
			{TargetID: "graph_wall.thermal.a", Lane: "thermal", Role: "temperature_primary", SourceID: "chamber_thermal_fat", Timestamp: "2026-01-15T10:00:00Z"},
			{TargetID: "graph_wall.thermal.a", Lane: "thermal", Role: "temperature_secondary", SourceID: "chamber_thermal_b", Timestamp: "2026-01-15T10:00:00Z"},
		},
	}
	if err := ValidateGraphWallManifest(m); err == nil {
		t.Fatal("expected duplicate target_id to fail")
	}
}

func TestValidateGraphWallManifestRejectsEmptyTargets(t *testing.T) {
	m := GraphWallManifest{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
	}
	if err := ValidateGraphWallManifest(m); err == nil {
		t.Fatal("expected empty targets to fail")
	}
}

func TestValidateGraphWallManifestRejectsMissingEnvelope(t *testing.T) {
	m := GraphWallManifest{
		Targets: []GraphWallTarget{
			{TargetID: "graph_wall.thermal.a", Lane: "thermal", Role: "temperature_primary", SourceID: "chamber_thermal_fat", Timestamp: "2026-01-15T10:00:00Z"},
		},
	}
	if err := ValidateGraphWallManifest(m); err == nil {
		t.Fatal("expected missing envelope to fail")
	}
}

func TestValidateGraphWallManifestRejectsMissingRequiredField(t *testing.T) {
	base := GraphWallTarget{TargetID: "t1", Lane: "thermal", Role: "temperature_primary", SourceID: "chamber_thermal_fat", Timestamp: "2026-01-15T10:00:00Z"}
	env := Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"}
	cases := []struct {
		name   string
		mutate func(*GraphWallTarget)
	}{
		{"empty target_id", func(t *GraphWallTarget) { t.TargetID = "" }},
		{"empty lane", func(t *GraphWallTarget) { t.Lane = "" }},
		{"empty role", func(t *GraphWallTarget) { t.Role = "" }},
		{"empty source_id", func(t *GraphWallTarget) { t.SourceID = "" }},
		{"empty timestamp", func(t *GraphWallTarget) { t.Timestamp = "" }},
	}
	for _, tc := range cases {
		target := base
		tc.mutate(&target)
		m := GraphWallManifest{Envelope: env, Targets: []GraphWallTarget{target}}
		if err := ValidateGraphWallManifest(m); err == nil {
			t.Errorf("case %q: expected error but got nil", tc.name)
		}
	}
}

func TestValidateSourceTreeConfigAcceptsValidConfig(t *testing.T) {
	c := SourceTreeConfig{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Views: []SourceTreeView{
			{ID: "thermal_fat", Label: "Acceptance FAT", SourceIDs: []string{"chamber_thermal_fat", "chamber_thermal_b"}},
			{ID: "tvac", Label: "Qualification TVac", SourceIDs: []string{"chamber_thermal_tvac"}},
		},
	}
	if err := ValidateSourceTreeConfig(c); err != nil {
		t.Fatalf("expected valid config: %v", err)
	}
}

func TestValidateSourceTreeConfigRejectsMissingEnvelope(t *testing.T) {
	c := SourceTreeConfig{
		Views: []SourceTreeView{
			{ID: "thermal_fat", Label: "Acceptance FAT", SourceIDs: []string{"chamber_thermal_fat"}},
		},
	}
	if err := ValidateSourceTreeConfig(c); err == nil {
		t.Fatal("expected missing envelope to fail")
	}
}

func TestValidateSourceTreeConfigRejectsEmptyViews(t *testing.T) {
	c := SourceTreeConfig{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
	}
	if err := ValidateSourceTreeConfig(c); err == nil {
		t.Fatal("expected empty views to fail")
	}
}

func TestValidateSourceTreeConfigRejectsDuplicateViewID(t *testing.T) {
	c := SourceTreeConfig{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Views: []SourceTreeView{
			{ID: "thermal_fat", Label: "Acceptance FAT", SourceIDs: []string{"chamber_thermal_fat"}},
			{ID: "thermal_fat", Label: "Duplicate", SourceIDs: []string{"chamber_thermal_b"}},
		},
	}
	if err := ValidateSourceTreeConfig(c); err == nil {
		t.Fatal("expected duplicate view id to fail")
	}
}

func TestValidateSourceTreeConfigRejectsEmptySourceIDs(t *testing.T) {
	c := SourceTreeConfig{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Views: []SourceTreeView{
			{ID: "thermal_fat", Label: "Acceptance FAT", SourceIDs: []string{}},
		},
	}
	if err := ValidateSourceTreeConfig(c); err == nil {
		t.Fatal("expected empty source_ids to fail")
	}
}

func TestValidateSourceTreeConfigRejectsMissingViewIDOrLabel(t *testing.T) {
	env := Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"}
	cases := []struct {
		name string
		view SourceTreeView
	}{
		{"empty id", SourceTreeView{ID: "", Label: "FAT", SourceIDs: []string{"chamber_thermal_fat"}}},
		{"empty label", SourceTreeView{ID: "thermal_fat", Label: "", SourceIDs: []string{"chamber_thermal_fat"}}},
	}
	for _, tc := range cases {
		c := SourceTreeConfig{Envelope: env, Views: []SourceTreeView{tc.view}}
		if err := ValidateSourceTreeConfig(c); err == nil {
			t.Errorf("case %q: expected error but got nil", tc.name)
		}
	}
}

func TestValidateBusVirtualizationRequiresTMAndTCEvents(t *testing.T) {
	tap := BusVirtualizationTap{
		Envelope:     Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		ConnectionID: "bus_virtualization_demo",
		Streams: []BusStream{
			{ID: "tm_primary", Direction: "TM", SourceNode: "reference_dut", DestinationNode: "archive_node_a", Bus: "telemetry_bus", Quality: "fresh"},
			{ID: "tc_primary", Direction: "TC", SourceNode: "flatsat_rack_a", DestinationNode: "reference_dut", Bus: "command_bus", Quality: "fresh"},
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
