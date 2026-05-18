package api

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egidinas/gossamer/internal/synthetic"
	"github.com/egidinas/signalforge/arrowtelemetry"
	"github.com/egidinas/signalforge/contracts"
	"github.com/egidinas/signalforge/safepath"
	"github.com/egidinas/signalforge/tilebundle"
)

const cacheControlNoStore = "no-store"

type commandAuthorityLeaseResponse struct {
	contracts.CommandAuthorityState
	LeaseToken string `json:"lease_token"`
}

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
	RequestLease(operatorID string) (contracts.CommandAuthorityState, string, error)
	ReleaseLease(operatorID, leaseToken string) (contracts.CommandAuthorityState, error)
	ExecuteCommand(operatorID, leaseToken, command string) (contracts.CommandAuthorityState, error)
}

type InMemoryLeaseManager struct {
	mu         sync.Mutex
	state      contracts.CommandAuthorityState
	lastSeen   time.Time
	leaseToken string
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
	return cloneCommandAuthorityState(m.state)
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

func (m *InMemoryLeaseManager) RequestLease(operatorID string) (contracts.CommandAuthorityState, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkExpiration()
	if m.state.LeaseState == "held" && m.state.LeaseOwner != operatorID {
		return cloneCommandAuthorityState(m.state), "", fmt.Errorf("lease already held by %s", m.state.LeaseOwner)
	}
	leaseToken, err := generateLeaseToken()
	if err != nil {
		return cloneCommandAuthorityState(m.state), "", err
	}
	m.state.LeaseState = "held"
	m.state.LeaseOwner = operatorID
	m.leaseToken = leaseToken
	m.lastSeen = time.Now()
	m.appendLog(operatorID, "lease_acquired", "Command authority lease acquired via demo request.")
	m.state.Envelope = contracts.NewEnvelope(time.Now())
	return cloneCommandAuthorityState(m.state), leaseToken, nil
}

func (m *InMemoryLeaseManager) ReleaseLease(operatorID, leaseToken string) (contracts.CommandAuthorityState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkExpiration()
	if m.state.LeaseState != "held" {
		return cloneCommandAuthorityState(m.state), fmt.Errorf("lease required to release command authority")
	}
	if m.state.LeaseOwner != operatorID || !m.validLeaseToken(leaseToken) {
		return cloneCommandAuthorityState(m.state), fmt.Errorf("valid lease token required for %s", m.state.LeaseOwner)
	}
	m.appendLog(operatorID, "lease_released", "Command authority lease released.")
	m.state.LeaseState = "available"
	m.state.LeaseOwner = ""
	m.leaseToken = ""
	m.state.Envelope = contracts.NewEnvelope(time.Now())
	return cloneCommandAuthorityState(m.state), nil
}

func (m *InMemoryLeaseManager) ExecuteCommand(operatorID, leaseToken, command string) (contracts.CommandAuthorityState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkExpiration()
	if m.state.LeaseState != "held" || m.state.LeaseOwner != operatorID || !m.validLeaseToken(leaseToken) {
		return cloneCommandAuthorityState(m.state), fmt.Errorf("lease required to execute command")
	}
	m.state.LastCommand = command
	m.lastSeen = time.Now()
	m.appendLog(operatorID, command, fmt.Sprintf("Command '%s' dispatched via demo interface.", command))
	m.state.Envelope = contracts.NewEnvelope(time.Now())
	return cloneCommandAuthorityState(m.state), nil
}

func cloneCommandAuthorityState(state contracts.CommandAuthorityState) contracts.CommandAuthorityState {
	state.AllowedCommands = append([]string(nil), state.AllowedCommands...)
	state.OperatorLog = append([]contracts.OperatorLogEntry(nil), state.OperatorLog...)
	return state
}

func (m *InMemoryLeaseManager) validLeaseToken(token string) bool {
	if m.leaseToken == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(m.leaseToken), []byte(token)) == 1
}

func generateLeaseToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate lease token: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

func (m *InMemoryLeaseManager) checkExpiration() {
	if m.state.LeaseState == "held" && time.Since(m.lastSeen) > 5*time.Minute {
		m.state.LeaseState = "available"
		m.state.LeaseOwner = ""
		m.leaseToken = ""
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
	s.mux.HandleFunc("/data/", s.dataBundle)
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
		serveArrowFile(w, r, filepath.Join(s.root, "fixtures", "public", "telemetry", id+".arrow"), cacheControlNoStore)
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
	w.Header().Set("Cache-Control", cacheControlNoStore)
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
	state, leaseToken, err := s.leases.RequestLease(opID)
	if err != nil {
		status := http.StatusConflict
		if strings.Contains(err.Error(), "generate lease token") {
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, commandAuthorityLeaseResponse{CommandAuthorityState: state, LeaseToken: leaseToken})
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
	state, err := s.leases.ReleaseLease(opID, r.Header.Get("X-Lease-Token"))
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
	state, err := s.leases.ExecuteCommand(opID, r.Header.Get("X-Lease-Token"), "set_demo_marker")
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
		s.dataBundle(w, r)
		return
	}
	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if rel == "" || strings.HasSuffix(r.URL.Path, "/") {
		rel = "index.html"
	}
	if candidate, ok := resolveExistingStaticFile(s.staticDir, rel); ok {
		w.Header().Set("Cache-Control", cacheControlNoStore)
		http.ServeFile(w, r, candidate)
		return
	}
	if isStaticAssetPath(rel) {
		http.NotFound(w, r)
		return
	}
	index, ok := resolveExistingStaticFile(s.staticDir, "index.html")
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", cacheControlNoStore)
	http.ServeFile(w, r, index)
}

func isStaticAssetPath(rel string) bool {
	if strings.HasPrefix(rel, "assets/") {
		return true
	}
	switch rel {
	case "favicon.ico", "manifest.webmanifest", "robots.txt":
		return true
	default:
		return path.Ext(rel) != ""
	}
}

func resolveExistingStaticFile(root, rel string) (string, bool) {
	candidate, err := safepath.ResolveUnderRoot(root, rel)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return "", false
	}
	rootResolved, err := resolveRootSymlinks(root)
	if err != nil {
		return "", false
	}
	candidateResolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", false
	}
	relative, err := filepath.Rel(rootResolved, candidateResolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", false
	}
	return candidateResolved, true
}

func (s *Server) dataBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rel := strings.TrimPrefix(r.URL.Path, "/data/")
	root := filepath.Join(s.root, "fixtures", "public_tiles")
	candidate, ok := resolveExistingDataFile(root, rel)
	if !ok && strings.HasSuffix(rel, ".arrow") {
		if gzipCandidate, gzipOK := resolveExistingDataFile(root, rel+".gz"); gzipOK {
			candidate = strings.TrimSuffix(gzipCandidate, ".gz")
			ok = true
		}
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
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
}

func resolveExistingDataFile(root, rel string) (string, bool) {
	candidate, err := safepath.ResolveUnderRoot(root, rel)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return "", false
	}
	rootResolved, err := resolveRootSymlinks(root)
	if err != nil {
		return "", false
	}
	candidateResolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", false
	}
	relative, err := filepath.Rel(rootResolved, candidateResolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", false
	}
	return candidateResolved, true
}

func resolveRootSymlinks(root string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(rootAbs)
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
	if query != "preview" {
		http.Error(w, "unsupported query", http.StatusBadRequest)
		return
	}
	limit, err := boundedQueryLimit(r.URL.Query().Get("limit"), 100, 10000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	parquetPath := filepath.Join(s.root, "fixtures", "public", "telemetry", id+".parquet")
	if _, err := os.Stat(parquetPath); err != nil {
		http.Error(w, "telemetry data not found for campaign "+id, http.StatusNotFound)
		return
	}

	fullQuery := fmt.Sprintf("SELECT * FROM read_parquet('%s') LIMIT %d", duckdbString(parquetPath), limit)

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
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"schema_version": 1,
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
		"campaign_id":    id,
		"query":          query,
		"results":        results,
	})
}

func boundedQueryLimit(raw string, defaultLimit, maxLimit int) (int, error) {
	if raw == "" {
		return defaultLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > maxLimit {
		return 0, fmt.Errorf("limit must be between 1 and %d", maxLimit)
	}
	return limit, nil
}

func duckdbString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func headers(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", cacheControlNoStore)
	w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5179")
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
