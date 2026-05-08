package tilebundle

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/egidinas/gossamer/internal/arrowtelemetry"
	"github.com/egidinas/gossamer/internal/contracts"
)

const (
	generatorVersion        = "static-tile-bundle-v1"
	materializedMaxPoints   = 900
	materializedPointDigits = 4
)

type Options struct {
	DataVersion  string
	UIVersion    string
	OutputDir    string
	TileBasePath string
	Levels       []string
	Now          time.Time
}

func LoadGraphModel(path string) (contracts.GraphModel, error) {
	var model contracts.GraphModel
	data, err := os.ReadFile(path)
	if err != nil {
		return model, err
	}
	if err := json.Unmarshal(data, &model); err != nil {
		return model, err
	}
	return model, nil
}

func WriteBundle(models []contracts.GraphModel, opts Options) (contracts.TileBundleManifest, error) {
	if len(models) == 0 {
		return contracts.TileBundleManifest{}, fmt.Errorf("no graph models supplied")
	}
	if opts.DataVersion == "" {
		opts.DataVersion = time.Now().UTC().Format("20060102T150405Z")
	}
	if opts.OutputDir == "" {
		return contracts.TileBundleManifest{}, fmt.Errorf("output dir is required")
	}
	if opts.TileBasePath == "" {
		opts.TileBasePath = "/data/current"
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return contracts.TileBundleManifest{}, err
	}
	rootRange := models[0].GraphWall.TimeRange
	manifest := contracts.TileBundleManifest{
		Envelope:             contracts.NewEnvelope(opts.Now),
		ID:                   "gossamer_tile_bundle_" + opts.DataVersion,
		DataVersion:          opts.DataVersion,
		UIVersion:            opts.UIVersion,
		GeneratedAt:          opts.Now.UTC().Format(time.RFC3339),
		SourceFixtureVersion: arrowtelemetry.SchemaName,
		TimeRange:            rootRange,
		ReplaySpeed:          replaySpeed(models[0]),
		PresentCursorPolicy:  "observed_until_cursor_planned_after_cursor",
		Provenance: contracts.TileBundleProvenance{
			Generator:              "cmd/gossamer-tiles",
			GeneratorVersion:       generatorVersion,
			GeneratedFrom:          modelIDs(models),
			HeavyComputationPolicy: "offline_on_build_host",
			RuntimePolicy:          "static_files_only",
		},
	}
	for _, model := range models {
		campaign, err := writeCampaignBundle(model, opts)
		if err != nil {
			return manifest, err
		}
		manifest.Campaigns = append(manifest.Campaigns, campaign)
		if model.TileManifest != nil {
			manifest.SourceNodes = append(manifest.SourceNodes, model.TileManifest.SourceNodes...)
			manifest.DataLensTranslations = append(manifest.DataLensTranslations, model.TileManifest.DataLensTranslations...)
			manifest.EvidenceLinks = append(manifest.EvidenceLinks, model.TileManifest.EvidenceLinks...)
		}
	}
	if err := writeJSON(filepath.Join(opts.OutputDir, "manifest.json"), manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

func writeCampaignBundle(model contracts.GraphModel, opts Options) (contracts.TileCampaignManifest, error) {
	if model.GraphWall == nil || model.HeroGraph == nil || model.TileManifest == nil {
		return contracts.TileCampaignManifest{}, fmt.Errorf("%s: missing graph wall, hero graph, or tile manifest", model.CampaignID)
	}
	campaignDir := filepath.Join(opts.OutputDir, "campaigns", model.CampaignID)
	if err := os.MkdirAll(campaignDir, 0o755); err != nil {
		return contracts.TileCampaignManifest{}, err
	}
	shell := model
	stripGraphModel(&shell)
	if err := writeJSON(filepath.Join(campaignDir, "graph-shell.json"), shell); err != nil {
		return contracts.TileCampaignManifest{}, err
	}
	cardRefs := append([]contracts.GraphTileCardRef{}, model.TileManifest.Cards...)
	for i := range cardRefs {
		card := &cardRefs[i]
		card.TileEndpoint = ""
		card.LatestEndpoint = ""
		card.TileFiles = nil
	}
	campaignManifest := contracts.TileCampaignManifest{
		CampaignID:           model.CampaignID,
		Title:                model.GraphWall.Title,
		GraphShellPath:       joinURL(opts.TileBasePath, "campaigns", model.CampaignID, "graph-shell.json"),
		ManifestPath:         joinURL(opts.TileBasePath, "campaigns", model.CampaignID, "manifest.json"),
		TimeRange:            model.GraphWall.TimeRange,
		ReplaySpeed:          replaySpeed(model),
		Levels:               levelsForModel(model, opts.Levels),
		Cards:                cardRefs,
		EvidenceLinks:        model.TileManifest.EvidenceLinks,
		SourceFixtureVersion: arrowtelemetry.SchemaName,
	}
	if err := writeJSON(filepath.Join(campaignDir, "manifest.json"), campaignManifest); err != nil {
		return contracts.TileCampaignManifest{}, err
	}
	return campaignManifest, nil
}

func BuildTile(model contracts.GraphModel, cardID, levelID, t0s, t1s string, latest bool) (contracts.GraphTile, error) {
	if model.GraphWall == nil || model.HeroGraph == nil || model.TileManifest == nil {
		return contracts.GraphTile{}, fmt.Errorf("campaign has no tile graph model")
	}
	if cardID == "" {
		cardID = "thermal_program"
	}
	if levelID == "" {
		levelID = "minute"
	}
	card, ok := findTileCard(*model.TileManifest, cardID)
	if !ok {
		return contracts.GraphTile{}, fmt.Errorf("unknown card_id %q", cardID)
	}
	start := mustParseTime(model.GraphWall.TimeRange.Start)
	end := mustParseTime(model.GraphWall.TimeRange.End)
	if t0s != "" {
		start = mustParseTime(t0s)
	}
	if t1s != "" {
		end = mustParseTime(t1s)
	}
	if latest && model.HeroGraph.TimeAxis.Now != "" {
		end = mustParseTime(model.HeroGraph.TimeAxis.Now)
		start = end.Add(-6 * time.Hour)
	}
	if !end.After(start) {
		return contracts.GraphTile{}, fmt.Errorf("t1 must be after t0")
	}
	maxPoints := min(maxPointsForLevel(*model.TileManifest, levelID), materializedMaxPoints)
	perSeriesMax := max(1, maxPoints)
	traceIndex := traceIndex(model.HeroGraph)
	series := make([]contracts.TileSeries, 0, len(card.Signals))
	rawCount := 0
	for _, signal := range card.Signals {
		points := traceIndex[signal.ID]
		if len(points) == 0 && card.RenderKind != "event_rail" {
			continue
		}
		filtered := filterPoints(points, start, end)
		rawCount += len(filtered)
		decimated := normalizePoints(decimate(filtered, perSeriesMax))
		tileSeries := contracts.TileSeries{
			ID: signal.ID, Label: signal.Label, Unit: signal.Unit, Role: signal.Role, Kind: signal.Kind, AxisID: signal.AxisID,
			Source: signal.Source, Step: card.RenderKind == "counter" || card.RenderKind == "swimlane", ValueTable: signal.ValueTable,
		}
		if card.RenderKind == "swimlane" {
			tileSeries.Spans = spansFromPoints(decimated, signal.ValueTable, end)
		} else {
			tileSeries.Points = decimated
		}
		series = append(series, tileSeries)
	}
	laneToken := commandCenterLaneToken(card)
	bands := filterCardBands(intersectBands(model.HeroGraph.PhaseBands, start, end), laneToken)
	bands = append(bands, filterCardBands(intersectBands(model.HeroGraph.DwellWindows, start, end), laneToken)...)
	markers := []contracts.GraphMarker{}
	events := []contracts.TileEvent{}
	if card.IncludeMarkers || card.CardID == "thermal_program" || card.RenderKind == "event_rail" || card.RenderKind == "swimlane" {
		markers = filterCardMarkers(intersectMarkers(model.HeroGraph.Markers, start, end), laneToken)
		events = make([]contracts.TileEvent, 0, len(markers))
		for _, marker := range markers {
			events = append(events, contracts.TileEvent{ID: marker.ID, Kind: marker.Kind, Label: marker.Label, Timestamp: marker.Timestamp, RequirementID: requirementID(marker.Kind), EvidenceRef: marker.EvidenceRef, Result: marker.Result, Value: marker.Value})
		}
	}
	pointCount := 0
	for _, s := range series {
		pointCount += len(s.Points) + len(s.Spans)
	}
	return contracts.GraphTile{
		Envelope:   contracts.NewEnvelope(time.Now()),
		ID:         fmt.Sprintf("%s_%s_%s_%d", model.CampaignID, cardID, levelID, start.Unix()),
		ManifestID: model.TileManifest.ID,
		CampaignID: model.CampaignID,
		CardID:     cardID,
		Level:      levelID,
		T0:         start.UTC().Format(time.RFC3339),
		T1:         end.UTC().Format(time.RFC3339),
		Diagnostics: contracts.TileDiagnostics{
			Source: "fixture_graph_model", Mode: "offline_static_tile", PointCount: pointCount, RawPointCount: rawCount,
			Decimated: rawCount > pointCount, Decimation: "min_max_envelope", TimeSpanMS: end.Sub(start).Milliseconds(), FreshnessMS: 250,
		},
		Provenance: contracts.TileProvenance{SourceNode: "fixture_backend", SourceFamily: sourceFamily(card), FixtureVersion: "graph_models", GenerationMode: "offline_tile_bundle", Synthetic: true},
		Series:     series,
		Bands:      bands,
		Markers:    markers,
		Events:     events,
	}, nil
}

func commandCenterLaneToken(card contracts.GraphTileCardRef) string {
	for _, signal := range card.Signals {
		if !strings.HasPrefix(signal.ID, "cc.") {
			continue
		}
		parts := strings.Split(signal.ID, ".")
		if len(parts) >= 3 {
			return strings.ToLower(parts[1])
		}
	}
	return ""
}

func filterCardBands(bands []contracts.GraphBand, laneToken string) []contracts.GraphBand {
	if laneToken == "" {
		return bands
	}
	filtered := make([]contracts.GraphBand, 0, len(bands))
	for _, band := range bands {
		if band.Kind == "weekend" || containsLaneToken(laneToken, band.ID, band.Label) {
			filtered = append(filtered, band)
		}
	}
	return filtered
}

func filterCardMarkers(markers []contracts.GraphMarker, laneToken string) []contracts.GraphMarker {
	if laneToken == "" {
		return markers
	}
	filtered := make([]contracts.GraphMarker, 0, len(markers))
	for _, marker := range markers {
		if containsLaneToken(laneToken, marker.ID, marker.Label, marker.EvidenceRef) {
			filtered = append(filtered, marker)
		}
	}
	return filtered
}

func containsLaneToken(laneToken string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), laneToken) {
			return true
		}
	}
	return false
}

type tileWindow struct{ Start, End time.Time }

func tileWindows(timeRange contracts.GraphWallTimeRange, level contracts.TileLevel) []tileWindow {
	start := mustParseTime(timeRange.Start)
	end := mustParseTime(timeRange.End)
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return nil
	}
	return []tileWindow{{Start: start, End: end}}
}

func spansFromPoints(points []contracts.GraphPoint, table map[string]string, fallbackEnd time.Time) []contracts.TileSpan {
	if len(points) == 0 {
		return nil
	}
	spans := make([]contracts.TileSpan, 0, len(points))
	for i, point := range points {
		start := mustParseTime(point.Timestamp)
		end := fallbackEnd
		if i+1 < len(points) {
			end = mustParseTime(points[i+1].Timestamp)
		}
		if !end.After(start) {
			continue
		}
		key := strconv.FormatFloat(point.Value, 'f', -1, 64)
		label := table[key]
		if len(spans) > 0 {
			last := &spans[len(spans)-1]
			if last.State == key && last.Label == label && last.End == start.UTC().Format(time.RFC3339) {
				last.End = end.UTC().Format(time.RFC3339)
				continue
			}
		}
		spans = append(spans, contracts.TileSpan{
			Start: start.UTC().Format(time.RFC3339),
			End:   end.UTC().Format(time.RFC3339),
			Value: point.Value,
			State: key,
			Label: label,
		})
	}
	return spans
}

func stripGraphModel(model *contracts.GraphModel) {
	if model.HeroGraph == nil {
		return
	}
	hero := *model.HeroGraph
	hero.Traces = append([]contracts.GraphTrace{}, hero.Traces...)
	for i := range hero.Traces {
		hero.Traces[i].Values = nil
	}
	hero.CompanionGroups = append([]contracts.CompanionGraphGroup{}, hero.CompanionGroups...)
	for i := range hero.CompanionGroups {
		hero.CompanionGroups[i].Traces = append([]contracts.GraphTrace{}, hero.CompanionGroups[i].Traces...)
		for j := range hero.CompanionGroups[i].Traces {
			hero.CompanionGroups[i].Traces[j].Values = nil
		}
	}
	model.HeroGraph = &hero
}

func replayCursor(hero *contracts.HeroGraphModel) time.Time {
	if hero == nil {
		return time.Time{}
	}
	if hero.TimeAxis.Now != "" {
		return mustParseTime(hero.TimeAxis.Now)
	}
	if hero.Execution != nil && hero.Execution.Now != "" {
		return mustParseTime(hero.Execution.Now)
	}
	return time.Time{}
}

func filterFuturePoints(points []contracts.GraphPoint, role string, cursor time.Time) []contracts.GraphPoint {
	if cursor.IsZero() || plannedRole(role) {
		return points
	}
	out := points[:0]
	for _, point := range points {
		if !mustParseTime(point.Timestamp).After(cursor) {
			out = append(out, point)
		}
	}
	return out
}

func plannedRole(role string) bool {
	switch role {
	case "ghost", "command", "acceptance_band":
		return true
	default:
		return false
	}
}

func findTileCard(manifest contracts.GraphTileManifest, cardID string) (contracts.GraphTileCardRef, bool) {
	for _, card := range manifest.Cards {
		if card.CardID == cardID {
			return card, true
		}
	}
	return contracts.GraphTileCardRef{}, false
}

func maxPointsForLevel(manifest contracts.GraphTileManifest, levelID string) int {
	for _, level := range manifest.Levels {
		if level.ID == levelID && level.MaxPoints > 0 {
			return level.MaxPoints
		}
	}
	if manifest.TilePolicy.DefaultPoints > 0 {
		return manifest.TilePolicy.DefaultPoints
	}
	return 900
}

func traceIndex(hero *contracts.HeroGraphModel) map[string][]contracts.GraphPoint {
	out := map[string][]contracts.GraphPoint{}
	if hero == nil {
		return out
	}
	for _, trace := range hero.Traces {
		out[trace.ID] = trace.Values
	}
	for _, group := range hero.CompanionGroups {
		for _, trace := range group.Traces {
			out[trace.ID] = trace.Values
		}
	}
	return out
}

func filterPoints(points []contracts.GraphPoint, start, end time.Time) []contracts.GraphPoint {
	out := make([]contracts.GraphPoint, 0, len(points))
	for _, point := range points {
		t := mustParseTime(point.Timestamp)
		if !t.Before(start) && !t.After(end) {
			out = append(out, point)
		}
	}
	return out
}

func decimate(points []contracts.GraphPoint, maxPoints int) []contracts.GraphPoint {
	if maxPoints <= 0 || len(points) <= maxPoints {
		return points
	}
	buckets := max(1, maxPoints/4)
	step := float64(len(points)) / float64(buckets)
	out := make([]contracts.GraphPoint, 0, maxPoints)
	for b := 0; b < buckets; b++ {
		from := int(math.Floor(float64(b) * step))
		to := int(math.Floor(float64(b+1) * step))
		if to > len(points) {
			to = len(points)
		}
		if from >= to {
			continue
		}
		minPoint, maxPoint := points[from], points[from]
		for _, p := range points[from:to] {
			if p.Value < minPoint.Value {
				minPoint = p
			}
			if p.Value > maxPoint.Value {
				maxPoint = p
			}
		}
		out = appendUniquePoints(out, points[from], minPoint, maxPoint, points[to-1])
	}
	return out
}

func normalizePoints(points []contracts.GraphPoint) []contracts.GraphPoint {
	if len(points) < 2 {
		return roundPoints(points)
	}
	sort.SliceStable(points, func(i, j int) bool {
		return points[i].Timestamp < points[j].Timestamp
	})
	out := make([]contracts.GraphPoint, 0, len(points))
	for _, p := range points {
		p.Value = roundFloat(p.Value, materializedPointDigits)
		if len(out) > 0 && out[len(out)-1].Timestamp == p.Timestamp {
			out[len(out)-1] = p
			continue
		}
		out = append(out, p)
	}
	return out
}

func roundPoints(points []contracts.GraphPoint) []contracts.GraphPoint {
	out := append([]contracts.GraphPoint(nil), points...)
	for i := range out {
		out[i].Value = roundFloat(out[i].Value, materializedPointDigits)
	}
	return out
}

func roundFloat(value float64, digits int) float64 {
	if value == 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return value
	}
	scale := math.Pow10(digits)
	return math.Round(value*scale) / scale
}

func appendUniquePoints(out []contracts.GraphPoint, points ...contracts.GraphPoint) []contracts.GraphPoint {
	for _, p := range points {
		if len(out) == 0 || out[len(out)-1].Timestamp != p.Timestamp || out[len(out)-1].Value != p.Value {
			out = append(out, p)
		}
	}
	return out
}

func intersectBands(bands []contracts.GraphBand, start, end time.Time) []contracts.GraphBand {
	out := []contracts.GraphBand{}
	for _, band := range bands {
		if mustParseTime(band.End).Before(start) || mustParseTime(band.Start).After(end) {
			continue
		}
		out = append(out, band)
	}
	return out
}

func intersectMarkers(markers []contracts.GraphMarker, start, end time.Time) []contracts.GraphMarker {
	out := []contracts.GraphMarker{}
	for _, marker := range markers {
		t := mustParseTime(marker.Timestamp)
		if !t.Before(start) && !t.After(end) {
			out = append(out, marker)
		}
	}
	return out
}

func filterFutureMarkers(markers []contracts.GraphMarker, cursor time.Time) []contracts.GraphMarker {
	if cursor.IsZero() {
		return markers
	}
	out := markers[:0]
	for _, marker := range markers {
		if !mustParseTime(marker.Timestamp).After(cursor) {
			out = append(out, marker)
		}
	}
	return out
}

func filterFutureBands(bands []contracts.GraphBand, cursor time.Time) []contracts.GraphBand {
	if cursor.IsZero() {
		return bands
	}
	out := bands[:0]
	for _, band := range bands {
		start := mustParseTime(band.Start)
		end := mustParseTime(band.End)
		if start.After(cursor) {
			continue
		}
		if end.After(cursor) {
			band.End = cursor.UTC().Format(time.RFC3339)
		}
		out = append(out, band)
	}
	return out
}

func requirementID(kind string) string {
	switch kind {
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

func levelsForModel(model contracts.GraphModel, allowed []string) []contracts.TileLevel {
	allowedSet := map[string]bool{}
	for _, id := range allowed {
		id = strings.TrimSpace(id)
		if id != "" {
			allowedSet[id] = true
		}
	}
	if model.TileManifest == nil || len(model.TileManifest.Levels) == 0 {
		return []contracts.TileLevel{{ID: "minute", Label: "1 min", Resolution: "PT1M", DurationMS: int64(time.Hour.Milliseconds()), MaxPoints: 900, DecimationMode: "min_max_envelope"}}
	}
	if len(allowedSet) == 0 {
		return model.TileManifest.Levels
	}
	out := make([]contracts.TileLevel, 0, len(allowedSet))
	for _, level := range model.TileManifest.Levels {
		if allowedSet[level.ID] {
			out = append(out, level)
		}
	}
	if len(out) == 0 {
		return []contracts.TileLevel{{ID: "minute", Label: "Minute", Resolution: "minute", DurationMS: int64(time.Hour.Milliseconds()), MaxPoints: 900, DecimationMode: "min_max_envelope"}}
	}
	return out
}

func replaySpeed(model contracts.GraphModel) string {
	if model.HeroGraph != nil && model.HeroGraph.Execution != nil && model.HeroGraph.Execution.Acceleration != "" {
		return model.HeroGraph.Execution.Acceleration
	}
	return "static_replay_cursor"
}

func sourceFamily(card contracts.GraphTileCardRef) string {
	if len(card.Signals) == 0 {
		return "fixture"
	}
	return card.Signals[0].SourceFamily
}

func modelIDs(models []contracts.GraphModel) []string {
	out := make([]string, 0, len(models))
	for _, model := range models {
		out = append(out, model.CampaignID)
	}
	sort.Strings(out)
	return out
}

func tileFileID(cardID, levelID string, start time.Time) string {
	return safePath(cardID) + "_" + safePath(levelID) + "_" + start.UTC().Format("20060102T150405Z")
}

func safePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unnamed"
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func gzipJSON(path string) (int64, error) {
	in, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	out, err := os.Create(path + ".gz")
	if err != nil {
		return 0, err
	}
	gz := gzip.NewWriter(out)
	if _, err := gz.Write(in); err != nil {
		_ = out.Close()
		return 0, err
	}
	if err := gz.Close(); err != nil {
		_ = out.Close()
		return 0, err
	}
	if err := out.Close(); err != nil {
		return 0, err
	}
	info, err := os.Stat(path + ".gz")
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func joinURL(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			clean = append(clean, strings.TrimRight(part, "/"))
		} else {
			clean = append(clean, strings.Trim(part, "/"))
		}
	}
	if len(clean) == 0 {
		return "/"
	}
	return strings.ReplaceAll(strings.Join(clean, "/"), "//", "/")
}

func mustParseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
