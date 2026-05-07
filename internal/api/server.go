package api

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/egidinas/gossamer/internal/contracts"
	"github.com/egidinas/gossamer/internal/tilebundle"
)

type Server struct {
	root        string
	staticDir   string
	mux         *http.ServeMux
	mu          sync.Mutex
	graphMu     sync.Mutex
	graphModels map[string]contracts.GraphModel
	commands    contracts.CommandAuthorityState
}

func New(root string) *Server {
	s := &Server{
		root:        root,
		mux:         http.NewServeMux(),
		graphModels: map[string]contracts.GraphModel{},
		commands: contracts.CommandAuthorityState{
			Envelope:        contracts.NewEnvelope(time.Now()),
			LeaseState:      "available",
			AllowedCommands: []string{"set_demo_marker", "acknowledge_anomaly", "hold_fixture_state"},
		},
	}
	s.routes()
	return s
}

func NewWithStatic(root, staticDir string) *Server {
	s := New(root)
	s.staticDir = staticDir
	if staticDir != "" {
		s.mux.HandleFunc("/", s.static)
	}
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.health)
	s.mux.HandleFunc("/api/manifest", s.fixture("manifest.json"))
	s.mux.HandleFunc("/api/topology", s.fixture("topology.json"))
	s.mux.HandleFunc("/api/sources", s.fixture("source_catalogue.json"))
	s.mux.HandleFunc("/api/supervisor", s.fixture("supervisor_overview.json"))
	s.mux.HandleFunc("/api/bus-tap", s.fixture("bus_virtualization_tap.json"))
	s.mux.HandleFunc("/api/campaigns", s.campaigns)
	s.mux.HandleFunc("/api/campaigns/", s.campaignDetail)
	s.mux.HandleFunc("/api/command-authority", s.commandState)
	s.mux.HandleFunc("/api/command-authority/request-lease", s.requestLease)
	s.mux.HandleFunc("/api/command-authority/release-lease", s.releaseLease)
	s.mux.HandleFunc("/api/command-authority/mock-command", s.mockCommand)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"schema_version": 1, "generated_at": time.Now().UTC().Format(time.RFC3339), "ok": true, "name": "Gossamer"})
}

func (s *Server) fixture(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		serveFile(w, filepath.Join(s.root, "fixtures", "public", name))
	}
}

func (s *Server) campaigns(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/campaigns" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	base := filepath.Join(s.root, "fixtures", "public", "campaigns")
	entries, err := os.ReadDir(base)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var campaigns []contracts.Campaign
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var campaign contracts.Campaign
		if err := readJSON(filepath.Join(base, entry.Name()), &campaign); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		campaigns = append(campaigns, campaign)
	}
	writeJSON(w, map[string]any{"schema_version": 1, "generated_at": time.Now().UTC().Format(time.RFC3339), "campaigns": campaigns})
}

func (s *Server) campaignDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/campaigns/"), "/")
	if len(parts) == 0 || parts[0] == "" || strings.Contains(parts[0], "..") {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		serveFile(w, filepath.Join(s.root, "fixtures", "public", "campaigns", id+".json"))
		return
	}
	switch parts[1] {
	case "telemetry":
		serveFile(w, filepath.Join(s.root, "fixtures", "public", "telemetry", id+".jsonl"))
	case "graph-model":
		serveFile(w, filepath.Join(s.root, "fixtures", "public", "graph_models", id+".json"))
	case "graph-shell":
		model, err := s.readGraphModel(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		stripGraphModel(&model)
		writeJSON(w, model)
	case "tile-manifest":
		model, err := s.readGraphModel(id)
		if err != nil || model.TileManifest == nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, model.TileManifest)
	case "tiles":
		model, err := s.readGraphModel(id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		tile, err := buildTile(model, r.URL.Query().Get("card_id"), r.URL.Query().Get("level"), r.URL.Query().Get("t0"), r.URL.Query().Get("t1"), false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, tile)
	case "latest-tile":
		model, err := s.readGraphModel(id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		tile, err := buildTile(model, r.URL.Query().Get("card_id"), r.URL.Query().Get("level"), r.URL.Query().Get("t0"), r.URL.Query().Get("t1"), true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, tile)
	case "requirements":
		var campaign contracts.Campaign
		path := filepath.Join(s.root, "fixtures", "public", "campaigns", id+".json")
		if err := readJSON(path, &campaign); err != nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, map[string]any{"schema_version": 1, "generated_at": time.Now().UTC().Format(time.RFC3339), "campaign_id": id, "requirements": campaign.Requirements})
	case "evidence-report":
		serveFile(w, filepath.Join(s.root, "fixtures", "public", "reports", id+"_report.json"))
	default:
		http.NotFound(w, r)
	}
}

func stripGraphModel(model *contracts.GraphModel) {
	if model.HeroGraph != nil {
		hero := *model.HeroGraph
		hero.Traces = stripTraceValues(hero.Traces)
		hero.CompanionGroups = append([]contracts.CompanionGraphGroup{}, hero.CompanionGroups...)
		for i := range hero.CompanionGroups {
			hero.CompanionGroups[i].Traces = stripTraceValues(hero.CompanionGroups[i].Traces)
		}
		model.HeroGraph = &hero
	}
}

func stripTraceValues(traces []contracts.GraphTrace) []contracts.GraphTrace {
	out := append([]contracts.GraphTrace{}, traces...)
	for i := range out {
		out[i].Values = nil
	}
	return out
}

func (s *Server) readGraphModel(id string) (contracts.GraphModel, error) {
	s.graphMu.Lock()
	defer s.graphMu.Unlock()
	if model, ok := s.graphModels[id]; ok {
		return model, nil
	}
	var model contracts.GraphModel
	err := readJSON(filepath.Join(s.root, "fixtures", "public", "graph_models", id+".json"), &model)
	if err != nil {
		return contracts.GraphModel{}, err
	}
	s.graphModels[id] = model
	return model, nil
}

func buildTile(model contracts.GraphModel, cardID, levelID, t0s, t1s string, latest bool) (contracts.GraphTile, error) {
	return tilebundle.BuildTile(model, cardID, levelID, t0s, t1s, latest)
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
		return points
	}
	sort.SliceStable(points, func(i, j int) bool {
		return points[i].Timestamp < points[j].Timestamp
	})
	out := make([]contracts.GraphPoint, 0, len(points))
	for _, p := range points {
		if len(out) > 0 && out[len(out)-1].Timestamp == p.Timestamp {
			out[len(out)-1] = p
			continue
		}
		out = append(out, p)
	}
	return out
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

func mustParseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func (s *Server) commandState(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/command-authority" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands.Envelope = contracts.NewEnvelope(time.Now())
	writeJSON(w, s.commands)
}

func (s *Server) requestLease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.commands.LeaseState == "held" {
		http.Error(w, "lease already held", http.StatusConflict)
		return
	}
	s.commands.LeaseState = "held"
	s.commands.LeaseOwner = "demo_operator"
	s.commands.Envelope = contracts.NewEnvelope(time.Now())
	writeJSON(w, s.commands)
}

func (s *Server) releaseLease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands.LeaseState = "available"
	s.commands.LeaseOwner = ""
	s.commands.Envelope = contracts.NewEnvelope(time.Now())
	writeJSON(w, s.commands)
}

func (s *Server) mockCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.commands.LeaseState != "held" {
		http.Error(w, "lease required", http.StatusConflict)
		return
	}
	s.commands.LastCommand = "set_demo_marker"
	s.commands.Envelope = contracts.NewEnvelope(time.Now())
	writeJSON(w, s.commands)
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.staticDir == "" || strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/healthz/") {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/data/") {
		rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/data/")
		candidate := filepath.Join(s.root, "fixtures", "public_tiles", filepath.FromSlash(rel))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			if strings.HasSuffix(candidate, "manifest.json") {
				w.Header().Set("Cache-Control", "no-cache")
			} else {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFile(w, r, candidate)
			return
		}
		http.NotFound(w, r)
		return
	}
	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if rel == "" || strings.HasSuffix(r.URL.Path, "/") {
		rel = "index.html"
	}
	candidate := filepath.Join(s.staticDir, filepath.FromSlash(rel))
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, candidate)
		return
	}
	index := filepath.Join(s.staticDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, index)
}

func serveFile(w http.ResponseWriter, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	headers(w)
	if strings.HasSuffix(path, ".jsonl") {
		w.Header().Set("Content-Type", "application/x-ndjson")
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, v any) {
	headers(w)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func headers(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5179")
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
