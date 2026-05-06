package synthetic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egidinas/gossamer/internal/contracts"
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
		if len(set.Telemetry[id]) != 48 {
			t.Fatalf("%s telemetry count = %d, want 48", id, len(set.Telemetry[id]))
		}
	}
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
			if got := sample.Signals["thermal_zone_1_deg_c"]; got < -60 || got > 85 {
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
