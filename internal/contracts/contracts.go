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
	CampaignID       string        `json:"campaign_id"`
	Summary          string        `json:"summary"`
	Result           string        `json:"result"`
	Requirements     []Requirement `json:"requirements"`
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
	LastCommand      string   `json:"last_command"`
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

func empty(s string) bool {
	return strings.TrimSpace(s) == ""
}

