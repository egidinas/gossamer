package tilebundle

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egidinas/gossamer/internal/contracts"
)

func TestBuildTileIncludesReplayDataForClientSideReveal(t *testing.T) {
	model := smallModel()
	tile, err := BuildTile(model, "thermal_program", "minute", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	observed := seriesByID(tile, "trace.actual.chamber_air")
	if len(observed.Points) == 0 || observed.Points[len(observed.Points)-1].Timestamp != "2026-01-01T04:00:00Z" {
		t.Fatalf("observed replay trace should be available for frontend reveal masking: %+v", observed.Points)
	}
	ghost := seriesByID(tile, "trace.ghost.profile")
	if len(ghost.Points) == 0 || ghost.Points[len(ghost.Points)-1].Timestamp != "2026-01-01T04:00:00Z" {
		t.Fatalf("ghost trace should remain available into planned future: %+v", ghost.Points)
	}
}

func TestBuildTilePreservesMinMaxEnvelopeSpike(t *testing.T) {
	model := smallModel()
	model.TileManifest.Levels[0].MaxPoints = 8
	trace := &model.HeroGraph.Traces[0]
	trace.Values = nil
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 200; i++ {
		value := 10.0
		if i == 91 {
			value = 80
		}
		if i == 92 {
			value = -40
		}
		trace.Values = append(trace.Values, contracts.GraphPoint{Timestamp: start.Add(time.Duration(i) * time.Minute).Format(time.RFC3339), Value: value})
	}
	tile, err := BuildTile(model, "thermal_program", "minute", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	actual := seriesByID(tile, "trace.actual.chamber_air")
	var sawHigh, sawLow bool
	for _, point := range actual.Points {
		sawHigh = sawHigh || point.Value == 80
		sawLow = sawLow || point.Value == -40
	}
	if !sawHigh || !sawLow {
		t.Fatalf("decimated envelope lost spike extrema: high=%v low=%v points=%+v", sawHigh, sawLow, actual.Points)
	}
}

func TestBuildTileCapsMaterializedPointsAndRoundsValues(t *testing.T) {
	model := smallModel()
	model.TileManifest.Levels[0].MaxPoints = materializedMaxPoints * 3
	trace := &model.HeroGraph.Traces[0]
	trace.Values = nil
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < materializedMaxPoints*3; i++ {
		trace.Values = append(trace.Values, contracts.GraphPoint{
			Timestamp: start.Add(time.Duration(i) * time.Second).Format(time.RFC3339),
			Value:     10 + float64(i)/3.0 + 0.000009,
		})
	}

	tile, err := BuildTile(model, "thermal_program", "minute", "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	actual := seriesByID(tile, "trace.actual.chamber_air")
	if len(actual.Points) > materializedMaxPoints {
		t.Fatalf("materialized tile has %d points, want <= %d", len(actual.Points), materializedMaxPoints)
	}
	for _, point := range actual.Points {
		if point.Value != math.Round(point.Value*1e4)/1e4 {
			t.Fatalf("point value was not rounded to display precision: %.12f", point.Value)
		}
	}
}

func TestSwimlaneTileEncodesSpans(t *testing.T) {
	tile, err := BuildTile(smallModel(), "state_change_swimlane", "minute", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	state := seriesByID(tile, "trace.dut_ready")
	if len(state.Spans) < 2 {
		t.Fatalf("swimlane series should expose state spans, got %+v", state)
	}
	if len(state.Points) != 0 {
		t.Fatalf("swimlane should not rely on fake analog points, got %+v", state.Points)
	}
	if state.Spans[0].End == "" || state.Spans[0].Start == state.Spans[0].End {
		t.Fatalf("bad span timing: %+v", state.Spans[0])
	}
}

func TestCommandCenterTileFiltersLaneLocalMarkersAndBands(t *testing.T) {
	model := smallModel()
	model.CampaignID = "command_center_fat"
	model.HeroGraph.PhaseBands = []contracts.GraphBand{
		{ID: "thermal_chamber_alpha-fat-01-window", Label: "Alpha FAT 01", Kind: "test_window", Start: "2026-01-01T00:00:00Z", End: "2026-01-01T02:00:00Z"},
		{ID: "thermal_chamber_bravo-fat-01-window", Label: "Bravo FAT 01", Kind: "test_window", Start: "2026-01-01T00:00:00Z", End: "2026-01-01T02:00:00Z"},
		{ID: "weekend-20260103", Label: "Weekend", Kind: "weekend", Start: "2026-01-01T00:00:00Z", End: "2026-01-01T02:00:00Z"},
	}
	model.HeroGraph.Markers = []contracts.GraphMarker{
		{ID: "thermal_chamber_alpha-fat-01-FUNC-C01-HOT", Label: "Alpha hot gate", Kind: "functional_gate", Timestamp: "2026-01-01T01:00:00Z", EvidenceRef: "reports/command_center_fat.json#thermal_chamber_alpha-fat-01"},
		{ID: "thermal_chamber_bravo-fat-01-FUNC-C01-HOT", Label: "Bravo hot gate", Kind: "functional_gate", Timestamp: "2026-01-01T01:00:00Z", EvidenceRef: "reports/command_center_fat.json#thermal_chamber_bravo-fat-01"},
	}
	model.HeroGraph.Traces = append(model.HeroGraph.Traces, contracts.GraphTrace{
		ID: "cc.alpha.command_deg_c", Label: "Alpha command", Role: "command", Units: "degC", AxisID: "temperature", Source: "plan",
		Values: []contracts.GraphPoint{
			{Timestamp: "2026-01-01T00:00:00Z", Value: 20},
			{Timestamp: "2026-01-01T01:00:00Z", Value: 40},
		},
	})
	alphaCard := contracts.GraphTileCardRef{
		CardID: "cc.alpha.lane", Title: "Alpha chamber FAT lane", RenderKind: "line", AxisPolicy: "shared_time", IncludeMarkers: true,
		Signals: []contracts.GraphWallSignal{
			{ID: "cc.alpha.command_deg_c", Label: "Command", Unit: "degC", Role: "command", Kind: "analog", Source: "plan", SourceFamily: "thermal", AxisID: "temperature"},
		},
	}
	model.TileManifest.Cards = append(model.TileManifest.Cards, alphaCard)

	tile, err := BuildTile(model, "cc.alpha.lane", "minute", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !hasBand(tile.Bands, "Alpha FAT 01") {
		t.Fatalf("alpha lane band missing: %+v", tile.Bands)
	}
	if !hasBand(tile.Bands, "Weekend") {
		t.Fatalf("shared weekend band missing: %+v", tile.Bands)
	}
	if hasBand(tile.Bands, "Bravo FAT 01") {
		t.Fatalf("bravo band leaked into alpha tile: %+v", tile.Bands)
	}
	if !hasMarker(tile.Markers, "Alpha hot gate") || !hasEvent(tile.Events, "Alpha hot gate") {
		t.Fatalf("alpha functional marker/event missing: markers=%+v events=%+v", tile.Markers, tile.Events)
	}
	if hasMarker(tile.Markers, "Bravo hot gate") || hasEvent(tile.Events, "Bravo hot gate") {
		t.Fatalf("bravo functional marker/event leaked into alpha tile: markers=%+v events=%+v", tile.Markers, tile.Events)
	}
}

func TestWriteBundleCreatesStaticManifestAndCompressedTiles(t *testing.T) {
	out := t.TempDir()
	manifest, err := WriteBundle([]contracts.GraphModel{smallModel()}, Options{
		DataVersion:  "test-data",
		OutputDir:    out,
		TileBasePath: "/data/current",
		Now:          time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.DataVersion != "test-data" || len(manifest.Campaigns) != 1 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	rootManifest := filepath.Join(out, "manifest.json")
	if _, err := os.Stat(rootManifest); err != nil {
		t.Fatalf("root manifest missing: %v", err)
	}
	campaign := manifest.Campaigns[0]
	if campaign.ManifestPath != "/data/current/campaigns/thermal_acceptance_fat/manifest.json" {
		t.Fatalf("manifest path = %q", campaign.ManifestPath)
	}
	if len(campaign.Cards[0].TileFiles) != 0 {
		t.Fatalf("card should not advertise legacy JSON tile files: %+v", campaign.Cards[0].TileFiles)
	}
	cardsDir := filepath.Join(out, "campaigns", "thermal_acceptance_fat", "cards")
	if _, err := os.Stat(cardsDir); !os.IsNotExist(err) {
		t.Fatalf("legacy JSON cards dir should not be generated, stat err=%v", err)
	}
	if campaign.CompressedBytes != 0 || campaign.UncompressedBytes != 0 {
		t.Fatalf("legacy tile byte counts should be zero: compressed=%d uncompressed=%d", campaign.CompressedBytes, campaign.UncompressedBytes)
	}
}

func seriesByID(tile contracts.GraphTile, id string) contracts.TileSeries {
	for _, series := range tile.Series {
		if series.ID == id {
			return series
		}
	}
	return contracts.TileSeries{}
}

func hasBand(bands []contracts.GraphBand, label string) bool {
	for _, band := range bands {
		if band.Label == label {
			return true
		}
	}
	return false
}

func hasMarker(markers []contracts.GraphMarker, label string) bool {
	for _, marker := range markers {
		if marker.Label == label {
			return true
		}
	}
	return false
}

func hasEvent(events []contracts.TileEvent, label string) bool {
	for _, event := range events {
		if event.Label == label {
			return true
		}
	}
	return false
}

func smallModel() contracts.GraphModel {
	start := "2026-01-01T00:00:00Z"
	end := "2026-01-01T04:00:00Z"
	now := "2026-01-01T02:00:00Z"
	points := []contracts.GraphPoint{
		{Timestamp: "2026-01-01T00:00:00Z", Value: 20},
		{Timestamp: "2026-01-01T01:00:00Z", Value: 30},
		{Timestamp: "2026-01-01T02:00:00Z", Value: 40},
		{Timestamp: "2026-01-01T03:00:00Z", Value: 50},
		{Timestamp: "2026-01-01T04:00:00Z", Value: 60},
	}
	statePoints := []contracts.GraphPoint{
		{Timestamp: "2026-01-01T00:00:00Z", Value: 0},
		{Timestamp: "2026-01-01T01:00:00Z", Value: 1},
		{Timestamp: "2026-01-01T02:00:00Z", Value: 1},
		{Timestamp: "2026-01-01T03:00:00Z", Value: 0},
	}
	tilePolicy := contracts.GraphTilePolicy{DefaultPoints: 40, MaxPoints: 80, SharedTimebaseRequired: true}
	levels := []contracts.TileLevel{{ID: "minute", Label: "1 min", Resolution: "PT1M", DurationMS: int64((2 * time.Hour).Milliseconds()), MaxPoints: 40, DecimationMode: "min_max_envelope"}}
	cards := []contracts.GraphTileCardRef{
		{
			CardID: "thermal_program", Title: "Thermal Program", RenderKind: "line", AxisPolicy: "shared_time",
			Signals: []contracts.GraphWallSignal{
				{ID: "trace.actual.chamber_air", Label: "Chamber air", Unit: "degC", Role: "actual", Kind: "analog", Source: "sim", SourceFamily: "thermal", AxisID: "temperature"},
				{ID: "trace.ghost.profile", Label: "Ghost", Unit: "degC", Role: "ghost", Kind: "analog", Source: "plan", SourceFamily: "thermal", AxisID: "temperature"},
			},
		},
		{
			CardID: "state_change_swimlane", Title: "State", RenderKind: "swimlane", AxisPolicy: "shared_time",
			Signals: []contracts.GraphWallSignal{
				{ID: "trace.dut_ready", Label: "DUT ready", Role: "state", Kind: "boolean", Source: "sim", SourceFamily: "dut", ValueTable: map[string]string{"0": "off", "1": "ready"}},
			},
		},
	}
	return contracts.GraphModel{
		Envelope:   contracts.NewEnvelope(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		CampaignID: "thermal_acceptance_fat",
		GraphWall: &contracts.GraphWallModel{
			ID: "wall", Title: "Wall", GeneratedAt: start, SourceMode: "fixture", GraphVersion: "test",
			TimeRange:  contracts.GraphWallTimeRange{Start: start, End: end, Anchor: now, RangeSeconds: 4 * 3600, Mode: "accelerated_replay", Source: "fixture"},
			TilePolicy: tilePolicy,
			Sections:   []contracts.GraphSection{{ID: "main", Title: "Main", Cards: []contracts.GraphWallCard{{ID: "thermal_program"}, {ID: "state_change_swimlane"}}}},
		},
		HeroGraph: &contracts.HeroGraphModel{
			ID: "hero", Title: "Hero",
			TimeAxis: contracts.GraphTimeAxis{Start: start, End: end, Now: now, RangeSeconds: 4 * 3600},
			Traces: []contracts.GraphTrace{
				{ID: "trace.actual.chamber_air", Label: "Chamber air", Role: "actual", Units: "degC", AxisID: "temperature", Source: "sim", Values: points},
				{ID: "trace.ghost.profile", Label: "Ghost", Role: "ghost", Units: "degC", AxisID: "temperature", Source: "plan", Values: points},
			},
			CompanionGroups: []contracts.CompanionGraphGroup{{ID: "states", Label: "States", Traces: []contracts.GraphTrace{{ID: "trace.dut_ready", Label: "DUT ready", Role: "state", Source: "sim", Values: statePoints}}}},
		},
		TileManifest: &contracts.GraphTileManifest{
			Envelope: contracts.NewEnvelope(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			ID:       "manifest", CampaignID: "thermal_acceptance_fat", GraphWallID: "wall", GeneratedAt: start,
			TimeRange:  contracts.GraphWallTimeRange{Start: start, End: end, Anchor: now, RangeSeconds: 4 * 3600, Mode: "accelerated_replay", Source: "fixture"},
			TilePolicy: tilePolicy, Levels: levels, Cards: cards,
		},
	}
}
