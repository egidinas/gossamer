package tilebundle

import (
	"compress/gzip"
	"encoding/json"
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
	if len(campaign.Cards[0].TileFiles) == 0 {
		t.Fatalf("card has no tile files: %+v", campaign.Cards[0])
	}
	tilePath := filepath.Join(out, "campaigns", "thermal_acceptance_fat", campaign.Cards[0].TileFiles[0].Path[len("/data/current/campaigns/thermal_acceptance_fat/"):])
	if _, err := os.Stat(tilePath); err != nil {
		t.Fatalf("tile file missing: %v", err)
	}
	gzPath := tilePath + ".gz"
	file, err := os.Open(gzPath)
	if err != nil {
		t.Fatalf("compressed tile missing: %v", err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("compressed tile is not gzip: %v", err)
	}
	defer reader.Close()
	var tile contracts.GraphTile
	if err := json.NewDecoder(reader).Decode(&tile); err != nil {
		t.Fatalf("decode compressed tile: %v", err)
	}
	if tile.CardID == "" || tile.Diagnostics.PointCount == 0 {
		t.Fatalf("bad compressed tile: %+v", tile)
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
