package contracts

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const SchemaVersion = 1

var SourceQualityStates = map[string]bool{
	"fresh":          true,
	"stale":          true,
	"missing":        true,
	"degraded":       true,
	"synthetic":      true,
	"not_applicable": true,
}

var CampaignResultStates = map[string]bool{
	"pass":         true,
	"fail":         true,
	"inconclusive": true,
	"blocked":      true,
	"not_run":      true,
}

type Envelope struct {
	SchemaVersion int    `json:"schema_version"`
	GeneratedAt   string `json:"generated_at"`
}

func NewEnvelope(t time.Time) Envelope {
	return Envelope{SchemaVersion: SchemaVersion, GeneratedAt: t.UTC().Format(time.RFC3339)}
}

type Manifest struct {
	Envelope
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	TestArticle   string   `json:"test_article"`
	Campaigns     []string `json:"campaigns"`
	PublicDemo    bool     `json:"public_demo"`
	SyntheticOnly bool     `json:"synthetic_only"`
}

type Node struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Status  string `json:"status"`
	Quality string `json:"quality"`
}

type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Bus    string `json:"bus"`
}

type Topology struct {
	Envelope
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

type Source struct {
	ID                  string   `json:"id"`
	Label               string   `json:"label"`
	Owner               string   `json:"owner"`
	Bus                 string   `json:"bus"`
	Quality             string   `json:"quality"`
	FreshnessMS         int      `json:"freshness_ms"`
	Provenance          string   `json:"provenance"`
	EvidenceSuitability string   `json:"evidence_suitability"`
	Signals             []string `json:"signals"`
}

type SourceCatalogue struct {
	Envelope
	Sources []Source `json:"sources"`
}

type Requirement struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Result      string   `json:"result"`
	Evidence    []string `json:"evidence"`
	Rationale   string   `json:"rationale"`
}

type Campaign struct {
	Envelope
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Level         string        `json:"level"`
	State         string        `json:"state"`
	Result        string        `json:"result"`
	Article       string        `json:"article"`
	Facility      string        `json:"facility"`
	Start         string        `json:"start"`
	End           string        `json:"end"`
	Requirements  []Requirement `json:"requirements"`
	Anomalies     []Anomaly     `json:"anomalies"`
	SyntheticNote string        `json:"synthetic_note"`
}

type TelemetrySample struct {
	Timestamp string             `json:"timestamp"`
	Signals   map[string]float64 `json:"signals"`
	States    map[string]string  `json:"states"`
	Quality   string             `json:"quality"`
}

type GraphSeries struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Role   string  `json:"role"`
	Units  string  `json:"units"`
	Source string  `json:"source"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

type GraphPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type GraphLane struct {
	ID     string        `json:"id"`
	Label  string        `json:"label"`
	Series []GraphSeries `json:"series"`
}

type GraphModel struct {
	Envelope
	CampaignID string      `json:"campaign_id"`
	Lanes      []GraphLane `json:"lanes"`
}

type SupervisorHeroGraph struct {
	ID     string       `json:"id"`
	Label  string       `json:"label"`
	Signal string       `json:"signal"`
	Units  string       `json:"units"`
	Role   string       `json:"role"`
	Source string       `json:"source"`
	Min    float64      `json:"min"`
	Max    float64      `json:"max"`
	Values []GraphPoint `json:"values"`
}

type SupervisorLane struct {
	ID                 string                `json:"id"`
	Label              string                `json:"label"`
	Facility           string                `json:"facility"`
	Campaign           string                `json:"campaign"`
	Activity           string                `json:"activity"`
	State              string                `json:"state"`
	Result             string                `json:"result"`
	PrimaryBus         string                `json:"primary_bus"`
	RequirementSummary string                `json:"requirement_summary"`
	SourceQuality      string                `json:"source_quality"`
	HeroGraphs         []SupervisorHeroGraph `json:"hero_graphs"`
	Notes              []string              `json:"notes"`
}

type SupervisorOverview struct {
	Envelope
	TestArticle string           `json:"test_article"`
	Summary     string           `json:"summary"`
	Lanes       []SupervisorLane `json:"lanes"`
}

type BusStream struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	Direction       string `json:"direction"`
	SourceNode      string `json:"source_node"`
	DestinationNode string `json:"destination_node"`
	Bus             string `json:"bus"`
	Quality         string `json:"quality"`
	LatencyMS       int    `json:"latency_ms"`
	PacketCounter   int    `json:"packet_counter"`
	DroppedFrames   int    `json:"dropped_frames"`
}

type BusEvent struct {
	ID              string             `json:"id"`
	StreamID        string             `json:"stream_id"`
	Direction       string             `json:"direction"`
	Timestamp       string             `json:"timestamp"`
	SourceNode      string             `json:"source_node"`
	DestinationNode string             `json:"destination_node"`
	EventClass      string             `json:"event_class"`
	Authority       string             `json:"authority"`
	Quality         string             `json:"quality"`
	LatencyMS       int                `json:"latency_ms"`
	PacketCounter   int                `json:"packet_counter"`
	Fields          map[string]float64 `json:"fields"`
	States          map[string]string  `json:"states"`
	Summary         string             `json:"summary"`
}

type BusVirtualizationTap struct {
	Envelope
	ConnectionID string      `json:"connection_id"`
	Description  string      `json:"description"`
	ReplayCursor string      `json:"replay_cursor"`
	Streams      []BusStream `json:"streams"`
	Events       []BusEvent  `json:"events"`
}

type Anomaly struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
	EvidenceRef string `json:"evidence_ref"`
	Disposition string `json:"disposition"`
}

type EvidenceReport struct {
	Envelope
	CampaignID        string        `json:"campaign_id"`
	Summary           string        `json:"summary"`
	Result            string        `json:"result"`
	Requirements      []Requirement `json:"requirements"`
	Sources           []Source      `json:"sources"`
	GraphEvidence     []string      `json:"graph_evidence"`
	Anomalies         []Anomaly     `json:"anomalies"`
	Reproducibility   []string      `json:"reproducibility"`
	SyntheticDataNote string        `json:"synthetic_data_note"`
}

type CommandAuthorityState struct {
	Envelope
	LeaseOwner      string   `json:"lease_owner"`
	LeaseState      string   `json:"lease_state"`
	AllowedCommands []string `json:"allowed_commands"`
	LastCommand     string   `json:"last_command"`
}

func ValidateEnvelope(e Envelope) error {
	if e.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version must be %d", SchemaVersion)
	}
	if strings.TrimSpace(e.GeneratedAt) == "" {
		return errors.New("generated_at is required")
	}
	if _, err := time.Parse(time.RFC3339, e.GeneratedAt); err != nil {
		return fmt.Errorf("generated_at must be RFC3339: %w", err)
	}
	return nil
}

func ValidateManifest(m Manifest) error {
	if err := ValidateEnvelope(m.Envelope); err != nil {
		return err
	}
	if empty(m.Name) || empty(m.TestArticle) {
		return errors.New("manifest name and test_article are required")
	}
	if len(m.Campaigns) == 0 {
		return errors.New("manifest requires campaigns")
	}
	return nil
}

func ValidateSourceCatalogue(c SourceCatalogue) error {
	if err := ValidateEnvelope(c.Envelope); err != nil {
		return err
	}
	for _, s := range c.Sources {
		if empty(s.ID) {
			return errors.New("source id is required")
		}
		if empty(s.Owner) {
			return fmt.Errorf("source %s owner is required", s.ID)
		}
		if !SourceQualityStates[s.Quality] {
			return fmt.Errorf("source %s has unknown quality %q", s.ID, s.Quality)
		}
	}
	return nil
}

func ValidateCampaign(c Campaign) error {
	if err := ValidateEnvelope(c.Envelope); err != nil {
		return err
	}
	if empty(c.ID) || empty(c.Name) {
		return errors.New("campaign id and name are required")
	}
	if !CampaignResultStates[c.Result] {
		return fmt.Errorf("campaign %s has unknown result %q", c.ID, c.Result)
	}
	for _, r := range c.Requirements {
		if empty(r.ID) {
			return errors.New("requirement id is required")
		}
		if !CampaignResultStates[r.Result] {
			return fmt.Errorf("requirement %s has unknown result %q", r.ID, r.Result)
		}
	}
	return nil
}

func ValidateGraphModel(g GraphModel) error {
	if err := ValidateEnvelope(g.Envelope); err != nil {
		return err
	}
	if empty(g.CampaignID) {
		return errors.New("graph campaign_id is required")
	}
	for _, lane := range g.Lanes {
		if empty(lane.ID) {
			return errors.New("graph lane id is required")
		}
		for _, series := range lane.Series {
			if empty(series.ID) || empty(series.Units) || empty(series.Role) {
				return fmt.Errorf("graph series %s requires id, units, and role", series.ID)
			}
		}
	}
	return nil
}

func ValidateSupervisorOverview(o SupervisorOverview) error {
	if err := ValidateEnvelope(o.Envelope); err != nil {
		return err
	}
	if len(o.Lanes) < 4 {
		return errors.New("supervisor overview requires at least four lanes")
	}
	hasTemperature := false
	for _, lane := range o.Lanes {
		if empty(lane.ID) || empty(lane.Facility) || empty(lane.Campaign) || empty(lane.State) {
			return fmt.Errorf("supervisor lane %s requires id, facility, campaign, and state", lane.ID)
		}
		if len(lane.HeroGraphs) == 0 {
			return fmt.Errorf("supervisor lane %s requires hero graphs", lane.ID)
		}
		for _, graph := range lane.HeroGraphs {
			if empty(graph.ID) || empty(graph.Signal) || empty(graph.Units) || empty(graph.Role) || empty(graph.Source) {
				return fmt.Errorf("supervisor graph %s requires id, signal, units, role, and source", graph.ID)
			}
			if len(graph.Values) == 0 {
				return fmt.Errorf("supervisor graph %s requires values", graph.ID)
			}
			if graph.Units == "degC" {
				hasTemperature = true
			}
		}
	}
	if !hasTemperature {
		return errors.New("supervisor overview requires at least one temperature hero graph")
	}
	return nil
}

func ValidateBusVirtualizationTap(tap BusVirtualizationTap) error {
	if err := ValidateEnvelope(tap.Envelope); err != nil {
		return err
	}
	if empty(tap.ConnectionID) {
		return errors.New("bus tap connection_id is required")
	}
	streams := map[string]BusStream{}
	for _, stream := range tap.Streams {
		if empty(stream.ID) || empty(stream.Direction) || empty(stream.SourceNode) || empty(stream.DestinationNode) || empty(stream.Bus) {
			return fmt.Errorf("bus stream %s requires id, direction, nodes, and bus", stream.ID)
		}
		if stream.Direction != "TM" && stream.Direction != "TC" {
			return fmt.Errorf("bus stream %s has invalid direction %q", stream.ID, stream.Direction)
		}
		streams[stream.ID] = stream
	}
	seenTM := false
	seenTC := false
	for _, event := range tap.Events {
		if empty(event.ID) || empty(event.StreamID) || empty(event.Direction) || empty(event.Timestamp) || empty(event.EventClass) {
			return fmt.Errorf("bus event %s requires id, stream, direction, timestamp, and class", event.ID)
		}
		if _, ok := streams[event.StreamID]; !ok {
			return fmt.Errorf("bus event %s references unknown stream %s", event.ID, event.StreamID)
		}
		switch event.Direction {
		case "TM":
			seenTM = true
		case "TC":
			seenTC = true
		default:
			return fmt.Errorf("bus event %s has invalid direction %q", event.ID, event.Direction)
		}
		if !SourceQualityStates[event.Quality] {
			return fmt.Errorf("bus event %s has unknown quality %q", event.ID, event.Quality)
		}
	}
	if !seenTM || !seenTC {
		return errors.New("bus tap requires both TM and TC events")
	}
	return nil
}

func empty(s string) bool {
	return strings.TrimSpace(s) == ""
}
