package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egidinas/gossamer/internal/contracts"
)

type Server struct {
	root     string
	mux      *http.ServeMux
	mu       sync.Mutex
	commands contracts.CommandAuthorityState
}

func New(root string) *Server {
	s := &Server{
		root: root,
		mux:  http.NewServeMux(),
		commands: contracts.CommandAuthorityState{
			Envelope:        contracts.NewEnvelope(time.Now()),
			LeaseState:      "available",
			AllowedCommands: []string{"set_demo_marker", "acknowledge_anomaly", "hold_fixture_state"},
		},
	}
	s.routes()
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
