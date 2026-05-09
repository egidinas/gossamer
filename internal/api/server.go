package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egidinas/gossamer/internal/arrowtelemetry"
	"github.com/egidinas/gossamer/internal/contracts"
	"github.com/egidinas/gossamer/internal/synthetic"
	"github.com/egidinas/gossamer/internal/tilebundle"
)

type Server struct {
	root        string
	staticDir   string
	mux         *http.ServeMux
	mu          sync.Mutex
	graphMu     sync.Mutex
	graphModels map[string]contracts.GraphModel
	db          *sql.DB
	leases      LeaseManager
}

type LeaseManager interface {
	GetState() contracts.CommandAuthorityState
	RequestLease(operatorID string) (contracts.CommandAuthorityState, error)
	ReleaseLease(operatorID string) (contracts.CommandAuthorityState, error)
	ExecuteCommand(operatorID, command string) (contracts.CommandAuthorityState, error)
}

type InMemoryLeaseManager struct {
	mu       sync.Mutex
	state    contracts.CommandAuthorityState
	lastSeen time.Time
}

func NewInMemoryLeaseManager() *InMemoryLeaseManager {
	return &InMemoryLeaseManager{
		state: contracts.CommandAuthorityState{
			LeaseState:      "available",
			AllowedCommands: []string{"set_demo_marker", "acknowledge_anomaly", "hold_fixture_state", "resume_fixture_state", "set_phase_label", "request_functional_gate"},
			OperatorLog: []contracts.OperatorLogEntry{
				{T: "2026-01-15T09:58:11Z", Operator: "test_conductor_role", Action: "lease_acquired", Detail: "Command authority acquired ahead of pre-conditioning start. Supervisor node confirmed."},
				{T: "2026-01-16T03:14:47Z", Operator: "test_conductor_role", Action: "hold_issued", Detail: "Facility hold issued: heater bank 2 thermocouple (TC-H2) open-circuit fault. Chamber ramp suspended at +12 °C. Ref. ANOM-FAT-003."},
				{T: "2026-01-16T03:37:02Z", Operator: "facility_engineer_role", Action: "set_demo_marker", Detail: "Marker placed at TC-H2 replacement point. Hold duration: 22 min 15 s."},
				{T: "2026-01-16T03:37:21Z", Operator: "test_conductor_role", Action: "resume_issued", Detail: "Hold cleared. Ramp resumed. DUT telemetry continuous throughout hold; no re-soak required."},
				{T: "2026-01-17T11:05:33Z", Operator: "test_conductor_role", Action: "request_functional_gate", Detail: "Hot-operational functional gate requested for cycle 2. Gate evaluation dispatched to supervisor."},
				{T: "2026-01-18T14:22:00Z", Operator: "test_conductor_role", Action: "acknowledge_anomaly", Detail: "ANOM-FAT-003 acknowledged and closed. TCR-FAT-2026-007 filed. Lease released for end-of-campaign archive."},
			},
		},
	}
}

func (m *InMemoryLeaseManager) GetState() contracts.CommandAuthorityState {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkExpiration()
	m.state.Envelope = contracts.NewEnvelope(time.Now())
	return m.state
}

func (m *InMemoryLeaseManager) appendLog(operator, action, detail string) {
	entry := contracts.OperatorLogEntry{
		T:        time.Now().UTC().Format(time.RFC3339),
		Operator: operator,
		Action:   action,
		Detail:   detail,
	}
	m.state.OperatorLog = append(m.state.OperatorLog, entry)
}

func (m *InMemoryLeaseManager) RequestLease(operatorID string) (contracts.CommandAuthorityState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkExpiration()
	if m.state.LeaseState == "held" && m.state.LeaseOwner != operatorID {
		return m.state, fmt.Errorf("lease already held by %s", m.state.LeaseOwner)
	}
	m.state.LeaseState = "held"
	m.state.LeaseOwner = operatorID
	m.lastSeen = time.Now()
	m.appendLog(operatorID, "lease_acquired", "Command authority lease acquired via demo request.")
	m.state.Envelope = contracts.NewEnvelope(time.Now())
	return m.state, nil
}

func (m *InMemoryLeaseManager) ReleaseLease(operatorID string) (contracts.CommandAuthorityState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state.LeaseState == "held" && m.state.LeaseOwner != operatorID {
		return m.state, fmt.Errorf("cannot release lease held by %s", m.state.LeaseOwner)
	}
	m.appendLog(operatorID, "lease_released", "Command authority lease released.")
	m.state.LeaseState = "available"
	m.state.LeaseOwner = ""
	m.state.Envelope = contracts.NewEnvelope(time.Now())
	return m.state, nil
}

func (m *InMemoryLeaseManager) ExecuteCommand(operatorID, command string) (contracts.CommandAuthorityState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkExpiration()
	if m.state.LeaseState != "held" || m.state.LeaseOwner != operatorID {
		return m.state, fmt.Errorf("lease required to execute command")
	}
	m.state.LastCommand = command
	m.lastSeen = time.Now()
	m.appendLog(operatorID, command, fmt.Sprintf("Command '%s' dispatched via demo interface.", command))
	m.state.Envelope = contracts.NewEnvelope(time.Now())
	return m.state, nil
}

func (m *InMemoryLeaseManager) checkExpiration() {
	if m.state.LeaseState == "held" && time.Since(m.lastSeen) > 5*time.Minute {
		m.state.LeaseState = "available"
		m.state.LeaseOwner = ""
	}
}

func New(root string) *Server {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		panic(fmt.Errorf("failed to open duckdb: %w", err))
	}

	s := &Server{
		root:        root,
		mux:         http.NewServeMux(),
		graphModels: map[string]contracts.GraphModel{},
		db:          db,
		leases:      NewInMemoryLeaseManager(),
	}
	s.routes()
	return s
}

func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
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
	s.mux.HandleFunc("/api/command-center/fat", s.fixture("command_center_fat.json"))
	s.mux.HandleFunc("/api/bus-tap", s.fixture("bus_virtualization_tap.json"))
	s.mux.HandleFunc("/api/graph-wall-manifest", s.fixture("graph_wall_manifest.json"))
	s.mux.HandleFunc("/api/source-tree-config", s.fixture("source_tree_config.json"))
	s.mux.HandleFunc("/api/campaigns", s.campaigns)
	s.mux.HandleFunc("/api/campaigns/", s.campaignDetail)
	s.mux.HandleFunc("/api/viewer/", s.fileViewer)
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
		serveArrowFile(w, r, filepath.Join(s.root, "fixtures", "public", "telemetry", id+".arrow"), "no-store")
	case "query":
		s.telemetryQuery(w, r, id)
	case "live":
		s.liveCampaign(w, r, id)
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

func (s *Server) fileViewer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/viewer/")
	if id == "" || strings.Contains(id, "..") || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	env := contracts.Envelope{SchemaVersion: contracts.SchemaVersion, GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
	model, ok := synthetic.BuildFileViewModel(env, id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	writeJSON(w, model)
}

func buildTile(model contracts.GraphModel, cardID, levelID, t0s, t1s string, latest bool) (contracts.GraphTile, error) {
	return tilebundle.BuildTile(model, cardID, levelID, t0s, t1s, latest)
}

func (s *Server) liveCampaign(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	model, err := s.readGraphModel(id)
	if err != nil || model.HeroGraph == nil {
		http.NotFound(w, r)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	start := mustParseTime(model.HeroGraph.TimeAxis.Start)
	end := mustParseTime(model.HeroGraph.TimeAxis.End)
	base := replayCursor(model.HeroGraph)
	if start.IsZero() || end.IsZero() || base.IsZero() {
		http.Error(w, "campaign clock unavailable", http.StatusBadRequest)
		return
	}
	acceleration := replayAcceleration(model.HeroGraph)
	connectedAt := time.Now()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5179")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		now := base.Add(time.Duration(float64(time.Since(connectedAt)) * acceleration))
		if now.Before(start) {
			now = start
		}
		if now.After(end) {
			now = end
		}
		payload := map[string]any{
			"schema_version": 1,
			"campaign_id":    id,
			"now":            now.UTC().Format(time.RFC3339Nano),
			"start":          start.UTC().Format(time.RFC3339Nano),
			"end":            end.UTC().Format(time.RFC3339Nano),
			"acceleration":   acceleration,
			"complete":       !now.Before(end),
			"source":         "simulated_live_replay",
		}
		data, _ := json.Marshal(payload)
		_, _ = w.Write([]byte("event: cursor\n"))
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()
		if !now.Before(end) {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func replayAcceleration(hero *contracts.HeroGraphModel) float64 {
	if hero == nil || hero.Execution == nil || hero.Execution.Acceleration == "" {
		return 60
	}
	fields := strings.Fields(hero.Execution.Acceleration)
	if len(fields) == 0 {
		return 60
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || value <= 0 {
		return 60
	}
	return value * 60
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
	writeJSON(w, s.leases.GetState())
}

func (s *Server) requestLease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	opID := r.Header.Get("X-Operator-ID")
	if opID == "" {
		http.Error(w, "X-Operator-ID header required", http.StatusUnauthorized)
		return
	}
	state, err := s.leases.RequestLease(opID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, state)
}

func (s *Server) releaseLease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	opID := r.Header.Get("X-Operator-ID")
	if opID == "" {
		http.Error(w, "X-Operator-ID header required", http.StatusUnauthorized)
		return
	}
	state, err := s.leases.ReleaseLease(opID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeJSON(w, state)
}

func (s *Server) mockCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	opID := r.Header.Get("X-Operator-ID")
	if opID == "" {
		http.Error(w, "X-Operator-ID header required", http.StatusUnauthorized)
		return
	}
	state, err := s.leases.ExecuteCommand(opID, "set_demo_marker")
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeJSON(w, state)
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
		isMutable := strings.HasPrefix(rel, "current/") || rel == "current"
		if strings.HasSuffix(candidate, ".arrow") {
			if arrowStaticExists(candidate) {
				cc := "public, max-age=31536000, immutable"
				if isMutable {
					cc = "no-cache"
				}
				serveArrowFile(w, r, candidate, cc)
				return
			}
			http.NotFound(w, r)
			return
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			if isMutable || strings.HasSuffix(candidate, "manifest.json") {
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
	if strings.HasSuffix(path, ".arrow") {
		w.Header().Set("Content-Type", arrowtelemetry.TransportMIME)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	_, _ = w.Write(data)
}

func serveArrowFile(w http.ResponseWriter, r *http.Request, filePath, cacheControl string) {
	pathToServe := filePath
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		gzipPath := filePath + ".gz"
		if info, err := os.Stat(gzipPath); err == nil && !info.IsDir() {
			pathToServe = gzipPath
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Add("Vary", "Accept-Encoding")
		}
	}
	if _, err := os.Stat(pathToServe); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("Content-Type", arrowtelemetry.TransportMIME)
	http.ServeFile(w, r, pathToServe)
}

func arrowStaticExists(filePath string) bool {
	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		return true
	}
	if info, err := os.Stat(filePath + ".gz"); err == nil && !info.IsDir() {
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, v any) {
	headers(w)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) telemetryQuery(w http.ResponseWriter, r *http.Request, id string) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query", http.StatusBadRequest)
		return
	}

	parquetPath := filepath.Join(s.root, "fixtures", "public", "telemetry", id+".parquet")
	if _, err := os.Stat(parquetPath); err != nil {
		http.Error(w, "telemetry data not found for campaign "+id, http.StatusNotFound)
		return
	}

	// We'll replace 'telemetry' with the read_parquet function call.
	// This allows the caller to use 'telemetry' as a table name.
	fullQuery := strings.ReplaceAll(query, "telemetry", fmt.Sprintf("read_parquet('%s')", parquetPath))

	rows, err := s.db.QueryContext(r.Context(), fullQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		row := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	writeJSON(w, map[string]any{
		"schema_version": 1,
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
		"campaign_id":    id,
		"results":        results,
	})
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
