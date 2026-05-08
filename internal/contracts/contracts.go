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
	NodeID              string   `json:"node_id"`   // node that originates this data
	ServedBy            string   `json:"served_by"` // node that aggregates and exposes this data to clients
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
	Expression  string   `json:"expression,omitempty"`
	Result      string   `json:"result"`
	Evidence    []string `json:"evidence"`
	Rationale   string   `json:"rationale"`
}

type RequirementProgress struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	Completed      int      `json:"completed"`
	Target         int      `json:"target"`
	Percent        float64  `json:"percent"`
	State          string   `json:"state"`
	Contributors   []string `json:"contributors"`
	NextMilestone  string   `json:"next_milestone,omitempty"`
	EvidenceSource string   `json:"evidence_source"`
}

type CyclePhase struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Kind       string  `json:"kind"`
	Start      string  `json:"start"`
	End        string  `json:"end"`
	TargetDegC float64 `json:"target_deg_c"`
}

type DwellWindow struct {
	ID             string  `json:"id"`
	Label          string  `json:"label"`
	CycleIndex     int     `json:"cycle_index"`
	Kind           string  `json:"kind"`
	Start          string  `json:"start"`
	End            string  `json:"end"`
	TargetDegC     float64 `json:"target_deg_c"`
	StabilityBandC float64 `json:"stability_band_c"`
	MinimumMinutes int     `json:"minimum_minutes"`
	EvidenceRef    string  `json:"evidence_ref"`
}

type FunctionalGate struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Gate        string `json:"gate"`
	CycleIndex  int    `json:"cycle_index"`
	PhaseID     string `json:"phase_id"`
	Timestamp   string `json:"timestamp"`
	Result      string `json:"result"`
	EvidenceRef string `json:"evidence_ref"`
}

type InterlockWindow struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Start       string `json:"start"`
	End         string `json:"end"`
	State       string `json:"state"`
	Severity    string `json:"severity"`
	EvidenceRef string `json:"evidence_ref"`
}

type EvidenceMarker struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Timestamp   string `json:"timestamp"`
	Kind        string `json:"kind"`
	Result      string `json:"result"`
	EvidenceRef string `json:"evidence_ref"`
}

type ThermalCycle struct {
	Index          int          `json:"index"`
	Label          string       `json:"label"`
	Start          string       `json:"start"`
	End            string       `json:"end"`
	ColdTargetDegC float64      `json:"cold_target_deg_c"`
	HotTargetDegC  float64      `json:"hot_target_deg_c"`
	Phases         []CyclePhase `json:"phases"`
}

type ThermalProgram struct {
	ID               string            `json:"id"`
	Kind             string            `json:"kind"`
	Label            string            `json:"label"`
	Facility         string            `json:"facility"`
	CycleCount       int               `json:"cycle_count"`
	ColdTargetDegC   float64           `json:"cold_target_deg_c"`
	HotTargetDegC    float64           `json:"hot_target_deg_c"`
	RampRateDegCMin  float64           `json:"ramp_rate_deg_c_min"`
	DwellMinutes     int               `json:"dwell_minutes"`
	Cycles           []ThermalCycle    `json:"cycles"`
	DwellWindows     []DwellWindow     `json:"dwell_windows"`
	FunctionalGates  []FunctionalGate  `json:"functional_gates"`
	InterlockWindows []InterlockWindow `json:"interlock_windows"`
	EvidenceMarkers  []EvidenceMarker  `json:"evidence_markers"`
}

type Campaign struct {
	Envelope
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Level          string          `json:"level"`
	State          string          `json:"state"`
	Result         string          `json:"result"`
	Article        string          `json:"article"`
	Facility       string          `json:"facility"`
	Start          string          `json:"start"`
	End            string          `json:"end"`
	Requirements   []Requirement   `json:"requirements"`
	Anomalies      []Anomaly       `json:"anomalies"`
	ThermalProgram *ThermalProgram `json:"thermal_program,omitempty"`
	SyntheticNote  string          `json:"synthetic_note"`
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
	NodeID string  `json:"node_id,omitempty"`
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

type GraphAnnotation struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Kind       string  `json:"kind"`
	Timestamp  string  `json:"timestamp,omitempty"`
	Start      string  `json:"start,omitempty"`
	End        string  `json:"end,omitempty"`
	CycleIndex int     `json:"cycle_index,omitempty"`
	TargetDegC float64 `json:"target_deg_c,omitempty"`
	Result     string  `json:"result,omitempty"`
}

type SimulationProvenance struct {
	Model         string             `json:"model"`
	ModelVersion  string             `json:"model_version"`
	Seed          int64              `json:"seed"`
	StepSeconds   int                `json:"step_seconds"`
	Source        string             `json:"source"`
	Parameters    map[string]float64 `json:"parameters"`
	Deterministic bool               `json:"deterministic"`
}

type GraphTimeAxis struct {
	Start              string `json:"start"`
	End                string `json:"end"`
	Anchor             string `json:"anchor"`
	Now                string `json:"now,omitempty"`
	DefaultWindowStart string `json:"default_window_start,omitempty"`
	DefaultWindowEnd   string `json:"default_window_end,omitempty"`
	RangeSeconds       int    `json:"range_seconds"`
	Clamp              bool   `json:"clamp"`
	LatestPolicy       string `json:"latest_policy"`
}

type ExecutionState struct {
	Mode                string                `json:"mode"`
	Now                 string                `json:"now"`
	PercentComplete     float64               `json:"percent_complete"`
	Acceleration        string                `json:"acceleration"`
	PastDataPolicy      string                `json:"past_data_policy"`
	FutureDataPolicy    string                `json:"future_data_policy"`
	CompletedCycles     int                   `json:"completed_cycles"`
	TargetCycles        int                   `json:"target_cycles"`
	CurrentCycle        int                   `json:"current_cycle"`
	CurrentPhase        string                `json:"current_phase"`
	RequirementProgress []RequirementProgress `json:"requirement_progress"`
}

type GraphYAxis struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Units  string  `json:"units"`
	Scale  string  `json:"scale"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Side   string  `json:"side"`
	Format string  `json:"format"`
}

type GraphTrace struct {
	ID     string       `json:"id"`
	Label  string       `json:"label"`
	Role   string       `json:"role"`
	Units  string       `json:"units"`
	AxisID string       `json:"axis_id"`
	Source string       `json:"source"`
	Values []GraphPoint `json:"values"`
}

type GraphBand struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Kind       string  `json:"kind"`
	Start      string  `json:"start"`
	End        string  `json:"end"`
	CycleIndex int     `json:"cycle_index,omitempty"`
	TargetDegC float64 `json:"target_deg_c,omitempty"`
	Result     string  `json:"result,omitempty"`
}

type GraphMarker struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Kind        string  `json:"kind"`
	Role        string  `json:"role"`
	Timestamp   string  `json:"timestamp"`
	CycleIndex  int     `json:"cycle_index,omitempty"`
	AxisID      string  `json:"axis_id,omitempty"`
	Value       float64 `json:"value,omitempty"`
	Result      string  `json:"result,omitempty"`
	Severity    string  `json:"severity,omitempty"`
	EvidenceRef string  `json:"evidence_ref,omitempty"`
}

type CompanionGraphGroup struct {
	ID     string       `json:"id"`
	Label  string       `json:"label"`
	Axes   []GraphYAxis `json:"axes"`
	Traces []GraphTrace `json:"traces"`
}

type ThermalDiagramNode struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Kind   string  `json:"kind"`
	Role   string  `json:"role"`
	Signal string  `json:"signal,omitempty"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

type ThermalDiagramLink struct {
	ID       string  `json:"id"`
	Source   string  `json:"source"`
	Target   string  `json:"target"`
	Kind     string  `json:"kind"`
	Label    string  `json:"label"`
	Strength float64 `json:"strength"`
	Signal   string  `json:"signal,omitempty"`
}

type TestItemThermalDiagram struct {
	ID      string               `json:"id"`
	Label   string               `json:"label"`
	Context string               `json:"context"`
	Summary string               `json:"summary"`
	Nodes   []ThermalDiagramNode `json:"nodes"`
	Links   []ThermalDiagramLink `json:"links"`
	Notes   []string             `json:"notes,omitempty"`
}

type HeroGraphModel struct {
	ID              string                  `json:"id"`
	Title           string                  `json:"title"`
	Owner           string                  `json:"owner"`
	Provenance      string                  `json:"provenance"`
	TimeAxis        GraphTimeAxis           `json:"time_axis"`
	Execution       *ExecutionState         `json:"execution,omitempty"`
	Axes            []GraphYAxis            `json:"axes"`
	Traces          []GraphTrace            `json:"traces"`
	PhaseBands      []GraphBand             `json:"phase_bands"`
	DwellWindows    []GraphBand             `json:"dwell_windows"`
	Markers         []GraphMarker           `json:"markers"`
	CompanionGroups []CompanionGraphGroup   `json:"companion_groups"`
	ThermalDiagram  *TestItemThermalDiagram `json:"thermal_diagram,omitempty"`
}

type GraphWallModel struct {
	ID           string             `json:"id"`
	Title        string             `json:"title"`
	GeneratedAt  string             `json:"generated_at"`
	SourceMode   string             `json:"source_mode"`
	GraphVersion string             `json:"graph_version"`
	Owner        string             `json:"owner"`
	Provenance   string             `json:"provenance"`
	TimeRange    GraphWallTimeRange `json:"time_range"`
	TilePolicy   GraphTilePolicy    `json:"tile_policy"`
	GraphGroups  []GraphGroup       `json:"graph_groups"`
	Sections     []GraphSection     `json:"sections"`
}

type GraphWallTimeRange struct {
	Start        string `json:"start"`
	End          string `json:"end"`
	Anchor       string `json:"anchor"`
	RangeSeconds int    `json:"range_seconds"`
	Mode         string `json:"mode"`
	Source       string `json:"source"`
}

type GraphTilePolicy struct {
	DefaultPoints               int      `json:"default_points"`
	MaxPoints                   int      `json:"max_points"`
	LiveTileMinRefreshMS        int      `json:"live_tile_min_refresh_ms"`
	HistoryTileMaxCount         int      `json:"history_tile_max_count"`
	ViewportPrefetchPX          int      `json:"viewport_prefetch_px"`
	TileBufferMaxEntries        int      `json:"tile_buffer_max_entries"`
	TileBufferTTLMS             int      `json:"tile_buffer_ttl_ms"`
	ResolutionLevels            []string `json:"resolution_levels"`
	SubscriberRole              string   `json:"subscriber_role"`
	SharedTimebaseRequired      bool     `json:"shared_timebase_required"`
	LegendMayAffectPlotWidth    bool     `json:"legend_may_affect_plot_width"`
	MalformedSVGPathHardFailure bool     `json:"malformed_svg_path_hard_failure"`
}

type GraphGroup struct {
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	Mode            string              `json:"mode"`
	BehaviorProfile string              `json:"behavior_profile"`
	Application     string              `json:"application"`
	SectionIDs      []string            `json:"section_ids"`
	Interaction     GraphInteraction    `json:"interaction"`
	Layout          GraphLayoutContract `json:"layout"`
}

type GraphInteraction struct {
	SharedTimeline   bool   `json:"shared_timeline"`
	SharedCrosshair  bool   `json:"shared_crosshair"`
	VerticalGrid     bool   `json:"vertical_grid"`
	SingleTimeAxis   bool   `json:"single_time_axis"`
	CursorMode       string `json:"cursor_mode"`
	CrosshairScope   string `json:"crosshair_scope"`
	TimelineGridMode string `json:"timeline_grid_mode"`
}

type GraphLayoutContract struct {
	PinnedCardsSeparate bool   `json:"pinned_cards_separate"`
	OverflowMode        string `json:"overflow_mode"`
	AxisRail            string `json:"axis_rail"`
	LegendRail          string `json:"legend_rail"`
	LabelRail           string `json:"label_rail"`
	PlotAreaPolicy      string `json:"plot_area_policy"`
}

type GraphSection struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	GroupID        string          `json:"group_id"`
	Transport      string          `json:"transport"`
	Direction      string          `json:"direction"`
	Status         string          `json:"status"`
	UnplottedCount int             `json:"unplotted_count"`
	Cards          []GraphWallCard `json:"cards"`
}

type GraphWallCard struct {
	ID               string             `json:"id"`
	Title            string             `json:"title"`
	Kind             string             `json:"kind"`
	Role             string             `json:"role"`
	Placement        GraphCardPlacement `json:"placement"`
	Transport        string             `json:"transport"`
	Direction        string             `json:"direction"`
	Unit             string             `json:"unit,omitempty"`
	AxisPolicy       string             `json:"axis_policy"`
	SourceFamily     string             `json:"source_family"`
	Overview         bool               `json:"overview,omitempty"`
	Bucket           string             `json:"bucket,omitempty"`
	Note             string             `json:"note,omitempty"`
	RenderKind       string             `json:"render_kind,omitempty"`
	IncludeMarkers   bool               `json:"include_markers,omitempty"`
	TileEndpoint     string             `json:"tile_endpoint,omitempty"`
	LatestEndpoint   string             `json:"latest_endpoint,omitempty"`
	Collapsible      bool               `json:"collapsible,omitempty"`
	DefaultExpanded  bool               `json:"default_expanded,omitempty"`
	SupportsTimeZoom bool               `json:"supports_time_zoom,omitempty"`
	SupportsYZoom    bool               `json:"supports_y_zoom,omitempty"`
	Signals          []GraphWallSignal  `json:"signals"`
}

type GraphCardPlacement struct {
	SectionID      string  `json:"section_id"`
	GroupID        string  `json:"group_id"`
	Order          int     `json:"order"`
	HeightWeight   float64 `json:"height_weight"`
	DefaultVisible bool    `json:"default_visible"`
	Pinned         bool    `json:"pinned"`
	ColocatedWith  string  `json:"colocated_with,omitempty"`
	ResizePolicy   string  `json:"resize_policy"`
}

type GraphWallSignal struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	Unit         string            `json:"unit,omitempty"`
	Source       string            `json:"source"`
	SourceFamily string            `json:"source_family"`
	Kind         string            `json:"kind"`
	Category     string            `json:"category"`
	Role         string            `json:"role"`
	Subsystem    string            `json:"subsystem"`
	AxisID       string            `json:"axis_id,omitempty"`
	SectionID    string            `json:"section_id"`
	ValueTable   map[string]string `json:"value_table,omitempty"`
}

type GraphModel struct {
	Envelope
	CampaignID           string                `json:"campaign_id"`
	Lanes                []GraphLane           `json:"lanes"`
	ThermalProgram       *ThermalProgram       `json:"thermal_program,omitempty"`
	Annotations          []GraphAnnotation     `json:"annotations,omitempty"`
	SimulationProvenance *SimulationProvenance `json:"simulation_provenance,omitempty"`
	HeroGraph            *HeroGraphModel       `json:"hero_graph,omitempty"`
	GraphWall            *GraphWallModel       `json:"graph_wall,omitempty"`
	TileManifest         *GraphTileManifest    `json:"tile_manifest,omitempty"`
}

type GraphTileManifest struct {
	Envelope
	ID                   string                `json:"id"`
	CampaignID           string                `json:"campaign_id"`
	GraphWallID          string                `json:"graph_wall_id"`
	GeneratedAt          string                `json:"generated_at"`
	SourceMode           string                `json:"source_mode"`
	SourceFixtureVersion string                `json:"source_fixture_version,omitempty"`
	TimeRange            GraphWallTimeRange    `json:"time_range"`
	TilePolicy           GraphTilePolicy       `json:"tile_policy"`
	Levels               []TileLevel           `json:"levels"`
	SourceNodes          []SourceNode          `json:"source_nodes"`
	DataLensTranslations []DataLensTranslation `json:"datalens_translations"`
	Cards                []GraphTileCardRef    `json:"cards"`
	EvidenceLinks        []EvidenceLink        `json:"evidence_links"`
}

type TileLevel struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Resolution     string `json:"resolution"`
	DurationMS     int64  `json:"duration_ms"`
	MaxPoints      int    `json:"max_points"`
	DecimationMode string `json:"decimation_mode"`
}

type GraphTileCardRef struct {
	CardID           string            `json:"card_id"`
	Title            string            `json:"title"`
	RenderKind       string            `json:"render_kind"`
	IncludeMarkers   bool              `json:"include_markers,omitempty"`
	Unit             string            `json:"unit,omitempty"`
	AxisPolicy       string            `json:"axis_policy"`
	TileEndpoint     string            `json:"tile_endpoint"`
	LatestEndpoint   string            `json:"latest_endpoint"`
	TileFiles        []TileFile        `json:"tile_files,omitempty"`
	Collapsible      bool              `json:"collapsible"`
	DefaultExpanded  bool              `json:"default_expanded"`
	SupportsTimeZoom bool              `json:"supports_time_zoom"`
	SupportsYZoom    bool              `json:"supports_y_zoom"`
	Signals          []GraphWallSignal `json:"signals"`
	EvidenceLinks    []EvidenceLink    `json:"evidence_links,omitempty"`
}

type SourceNode struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	Mode       string `json:"mode"`
	Confidence string `json:"confidence"`
	Provenance string `json:"provenance"`
}

type DataLensTranslation struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	SourceFormat string `json:"source_format"`
	TargetSchema string `json:"target_schema"`
	Mode         string `json:"mode"`
	Confidence   string `json:"confidence"`
	Provenance   string `json:"provenance"`
}

type EvidenceLink struct {
	ID            string `json:"id"`
	RequirementID string `json:"requirement_id"`
	CardID        string `json:"card_id"`
	SignalID      string `json:"signal_id,omitempty"`
	MarkerID      string `json:"marker_id,omitempty"`
	TileID        string `json:"tile_id,omitempty"`
	Timestamp     string `json:"timestamp"`
	CycleID       string `json:"cycle_id,omitempty"`
	PhaseID       string `json:"phase_id,omitempty"`
	Status        string `json:"status"`
	Label         string `json:"label"`
}

type TileBundleManifest struct {
	Envelope
	ID                     string                 `json:"id"`
	DataVersion            string                 `json:"data_version"`
	UIVersion              string                 `json:"ui_version,omitempty"`
	GeneratedAt            string                 `json:"generated_at"`
	SimulationModelVersion string                 `json:"simulation_model_version,omitempty"`
	SourceFixtureVersion   string                 `json:"source_fixture_version,omitempty"`
	TimeRange              GraphWallTimeRange     `json:"time_range"`
	ReplaySpeed            string                 `json:"replay_speed"`
	PresentCursorPolicy    string                 `json:"present_cursor_policy"`
	Campaigns              []TileCampaignManifest `json:"campaigns"`
	SourceNodes            []SourceNode           `json:"source_nodes,omitempty"`
	DataLensTranslations   []DataLensTranslation  `json:"datalens_translations,omitempty"`
	EvidenceLinks          []EvidenceLink         `json:"evidence_links,omitempty"`
	Provenance             TileBundleProvenance   `json:"provenance"`
}

type TileCampaignManifest struct {
	CampaignID           string             `json:"campaign_id"`
	Title                string             `json:"title"`
	GraphShellPath       string             `json:"graph_shell_path"`
	ManifestPath         string             `json:"manifest_path"`
	TimeRange            GraphWallTimeRange `json:"time_range"`
	ReplaySpeed          string             `json:"replay_speed"`
	Levels               []TileLevel        `json:"levels"`
	Cards                []GraphTileCardRef `json:"cards"`
	EvidenceLinks        []EvidenceLink     `json:"evidence_links,omitempty"`
	CompressedBytes      int64              `json:"compressed_bytes,omitempty"`
	UncompressedBytes    int64              `json:"uncompressed_bytes,omitempty"`
	SourceFixtureVersion string             `json:"source_fixture_version,omitempty"`
}

type TileFile struct {
	ID              string `json:"id"`
	Level           string `json:"level"`
	Path            string `json:"path"`
	CompressedPath  string `json:"compressed_path,omitempty"`
	T0              string `json:"t0"`
	T1              string `json:"t1"`
	RenderKind      string `json:"render_kind"`
	PointCount      int    `json:"point_count"`
	RawPointCount   int    `json:"raw_point_count"`
	CompressedBytes int64  `json:"compressed_bytes,omitempty"`
	Bytes           int64  `json:"bytes,omitempty"`
}

type TileBundleProvenance struct {
	Generator              string            `json:"generator"`
	GeneratorVersion       string            `json:"generator_version"`
	BuildHost              string            `json:"build_host,omitempty"`
	GeneratedFrom          []string          `json:"generated_from"`
	HeavyComputationPolicy string            `json:"heavy_computation_policy"`
	RuntimePolicy          string            `json:"runtime_policy"`
	Parameters             map[string]string `json:"parameters,omitempty"`
}

type GraphTile struct {
	Envelope
	ID          string          `json:"id"`
	ManifestID  string          `json:"manifest_id"`
	CampaignID  string          `json:"campaign_id"`
	CardID      string          `json:"card_id"`
	Level       string          `json:"level"`
	T0          string          `json:"t0"`
	T1          string          `json:"t1"`
	Diagnostics TileDiagnostics `json:"diagnostics"`
	Provenance  TileProvenance  `json:"provenance"`
	Series      []TileSeries    `json:"series"`
	Bands       []GraphBand     `json:"bands,omitempty"`
	Markers     []GraphMarker   `json:"markers,omitempty"`
	Events      []TileEvent     `json:"events,omitempty"`
}

type TileSeries struct {
	ID         string            `json:"id"`
	Label      string            `json:"label"`
	Unit       string            `json:"unit,omitempty"`
	Role       string            `json:"role"`
	Kind       string            `json:"kind"`
	AxisID     string            `json:"axis_id,omitempty"`
	Source     string            `json:"source"`
	Step       bool              `json:"step,omitempty"`
	ValueTable map[string]string `json:"value_table,omitempty"`
	Points     []GraphPoint      `json:"points"`
	Spans      []TileSpan        `json:"spans,omitempty"`
}

type TileSpan struct {
	Start    string  `json:"start"`
	End      string  `json:"end"`
	Value    float64 `json:"value,omitempty"`
	State    string  `json:"state,omitempty"`
	Label    string  `json:"label,omitempty"`
	Severity string  `json:"severity,omitempty"`
}

type TileDiagnostics struct {
	Source        string `json:"source"`
	Mode          string `json:"mode"`
	PointCount    int    `json:"point_count"`
	RawPointCount int    `json:"raw_point_count"`
	Decimated     bool   `json:"decimated"`
	Decimation    string `json:"decimation"`
	TimeSpanMS    int64  `json:"time_span_ms"`
	FreshnessMS   int    `json:"freshness_ms"`
}

type TileProvenance struct {
	SourceNode     string `json:"source_node"`
	SourceFamily   string `json:"source_family"`
	FixtureVersion string `json:"fixture_version"`
	GenerationMode string `json:"generation_mode"`
	Synthetic      bool   `json:"synthetic"`
}

type TileEvent struct {
	ID            string  `json:"id"`
	Kind          string  `json:"kind"`
	Label         string  `json:"label"`
	Timestamp     string  `json:"timestamp"`
	RequirementID string  `json:"requirement_id,omitempty"`
	EvidenceRef   string  `json:"evidence_ref,omitempty"`
	Result        string  `json:"result,omitempty"`
	Value         float64 `json:"value,omitempty"`
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
	ThermalProgram     *ThermalProgram       `json:"thermal_program,omitempty"`
	HeroGraph          *HeroGraphModel       `json:"hero_graph,omitempty"`
	FunctionalGates    []FunctionalGate      `json:"functional_gates,omitempty"`
	InterlockWindows   []InterlockWindow     `json:"interlock_windows,omitempty"`
	EvidenceMarkers    []EvidenceMarker      `json:"evidence_markers,omitempty"`
}

type SupervisorOverview struct {
	Envelope
	TestArticle string           `json:"test_article"`
	Summary     string           `json:"summary"`
	Lanes       []SupervisorLane `json:"lanes"`
}

type CommandCenterBand struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type CommandCenterEvent struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Kind      string `json:"kind"`
	Timestamp string `json:"timestamp"`
	State     string `json:"state"`
}

type CommandCenterTrace struct {
	ID     string       `json:"id"`
	Label  string       `json:"label"`
	Role   string       `json:"role"`
	Units  string       `json:"units"`
	Min    float64      `json:"min"`
	Max    float64      `json:"max"`
	Values []GraphPoint `json:"values"`
}

type CommandCenterTestItemManifest struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Article        string `json:"article"`
	SerialNumber   string `json:"serial_number"`
	Facility       string `json:"facility"`
	ChamberName    string `json:"chamber_name"`
	CampaignID     string `json:"campaign_id"`
	OperatorNext   string `json:"operator_next"`
	State          string `json:"state"`
	Result         string `json:"result"`
	Start          string `json:"start"`
	End            string `json:"end"`
	BreakdownStart string `json:"breakdown_start"`
	BreakdownEnd   string `json:"breakdown_end"`
	ResetStart     string `json:"reset_start"`
	ResetEnd       string `json:"reset_end"`
}

type CommandCenterRun struct {
	ID                 string                        `json:"id"`
	CampaignID         string                        `json:"campaign_id"`
	Title              string                        `json:"title"`
	State              string                        `json:"state"`
	Result             string                        `json:"result"`
	Start              string                        `json:"start"`
	End                string                        `json:"end"`
	BreakdownStart     string                        `json:"breakdown_start"`
	BreakdownEnd       string                        `json:"breakdown_end"`
	ResetStart         string                        `json:"reset_start"`
	ResetEnd           string                        `json:"reset_end"`
	Manifest           CommandCenterTestItemManifest `json:"manifest"`
	Traces             []CommandCenterTrace          `json:"traces,omitempty"`
	InteractionWindows []CommandCenterBand           `json:"interaction_windows"`
	Events             []CommandCenterEvent          `json:"events"`
}

type CommandCenterLane struct {
	ID          string             `json:"id"`
	ChamberName string             `json:"chamber_name"`
	Facility    string             `json:"facility"`
	Summary     string             `json:"summary"`
	GraphCardID string             `json:"graph_card_id,omitempty"`
	Runs        []CommandCenterRun `json:"runs"`
}

type CommandCenterFAT struct {
	Envelope
	ID               string              `json:"id"`
	Title            string              `json:"title"`
	Summary          string              `json:"summary"`
	Now              string              `json:"now"`
	WindowStart      string              `json:"window_start"`
	WindowEnd        string              `json:"window_end"`
	DataStart        string              `json:"data_start,omitempty"`
	DataEnd          string              `json:"data_end,omitempty"`
	SchedulePolicy   string              `json:"schedule_policy,omitempty"`
	WorkdayStartHour int                 `json:"workday_start_hour"`
	WorkdayEndHour   int                 `json:"workday_end_hour"`
	WeekendBands     []CommandCenterBand `json:"weekend_bands"`
	Lanes            []CommandCenterLane `json:"lanes"`
	GraphCampaignID  string              `json:"graph_campaign_id,omitempty"`
	HeroGraph        *HeroGraphModel     `json:"hero_graph,omitempty"`
	GraphWall        *GraphWallModel     `json:"graph_wall,omitempty"`
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
	CampaignID           string                `json:"campaign_id"`
	Summary              string                `json:"summary"`
	Result               string                `json:"result"`
	Requirements         []Requirement         `json:"requirements"`
	Sources              []Source              `json:"sources"`
	GraphEvidence        []string              `json:"graph_evidence"`
	Anomalies            []Anomaly             `json:"anomalies"`
	ThermalProgram       *ThermalProgram       `json:"thermal_program,omitempty"`
	SimulationProvenance *SimulationProvenance `json:"simulation_provenance,omitempty"`
	Reproducibility      []string              `json:"reproducibility"`
	SyntheticDataNote    string                `json:"synthetic_data_note"`
}

type OperatorLogEntry struct {
	T        string `json:"t"`
	Operator string `json:"operator"`
	Action   string `json:"action"`
	Detail   string `json:"detail"`
}

type CommandAuthorityState struct {
	Envelope
	LeaseOwner      string              `json:"lease_owner"`
	LeaseState      string              `json:"lease_state"`
	AllowedCommands []string            `json:"allowed_commands"`
	LastCommand     string              `json:"last_command"`
	OperatorLog     []OperatorLogEntry  `json:"operator_log,omitempty"`
}

// FileSignalGroup groups signals from a single node/source for the file viewer.
type FileSignalGroup struct {
	NodeID   string        `json:"node_id"`
	NodeLabel string       `json:"node_label"`
	SourceID string        `json:"source_id"`
	SourceLabel string     `json:"source_label"`
	Bus      string        `json:"bus"`
	Series   []GraphSeries `json:"series"`
}

// FileViewModel is the contract returned by /api/viewer/{campaign}.
type FileViewModel struct {
	Envelope
	CampaignID   string            `json:"campaign_id"`
	CampaignName string            `json:"campaign_name"`
	FileRef      string            `json:"file_ref"`
	FileKind     string            `json:"file_kind"`
	TimeStart    string            `json:"time_start"`
	TimeEnd      string            `json:"time_end"`
	SignalGroups []FileSignalGroup `json:"signal_groups"`
	Lanes        []GraphLane       `json:"lanes"`
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
	if g.HeroGraph != nil {
		if empty(g.HeroGraph.ID) || empty(g.HeroGraph.Owner) {
			return errors.New("hero graph requires id and owner")
		}
		if empty(g.HeroGraph.TimeAxis.Start) || empty(g.HeroGraph.TimeAxis.End) {
			return fmt.Errorf("hero graph %s requires time axis start and end", g.HeroGraph.ID)
		}
		if len(g.HeroGraph.Axes) == 0 {
			return fmt.Errorf("hero graph %s requires axes", g.HeroGraph.ID)
		}
		axes := map[string]bool{}
		for _, axis := range g.HeroGraph.Axes {
			if empty(axis.ID) || empty(axis.Units) || empty(axis.Scale) {
				return fmt.Errorf("hero graph axis %s requires id, units, and scale", axis.ID)
			}
			axes[axis.ID] = true
		}
		for _, trace := range g.HeroGraph.Traces {
			if empty(trace.ID) || empty(trace.Role) || empty(trace.AxisID) {
				return fmt.Errorf("hero trace %s requires id, role, and axis", trace.ID)
			}
			if !axes[trace.AxisID] {
				return fmt.Errorf("hero trace %s references unknown axis %s", trace.ID, trace.AxisID)
			}
			if len(trace.Values) == 0 {
				return fmt.Errorf("hero trace %s requires values", trace.ID)
			}
		}
	}
	if g.GraphWall != nil {
		if empty(g.GraphWall.ID) || empty(g.GraphWall.Owner) || empty(g.GraphWall.GraphVersion) || empty(g.GraphWall.SourceMode) {
			return errors.New("graph wall requires id, owner, graph_version, and source_mode")
		}
		if empty(g.GraphWall.TimeRange.Start) || empty(g.GraphWall.TimeRange.End) {
			return fmt.Errorf("graph wall %s requires time range start and end", g.GraphWall.ID)
		}
		if g.GraphWall.TilePolicy.DefaultPoints <= 0 || g.GraphWall.TilePolicy.MaxPoints < g.GraphWall.TilePolicy.DefaultPoints {
			return fmt.Errorf("graph wall %s has invalid tile policy", g.GraphWall.ID)
		}
		if len(g.GraphWall.GraphGroups) == 0 {
			return fmt.Errorf("graph wall %s requires graph groups", g.GraphWall.ID)
		}
		if len(g.GraphWall.Sections) == 0 {
			return fmt.Errorf("graph wall %s requires sections", g.GraphWall.ID)
		}
		for _, section := range g.GraphWall.Sections {
			if empty(section.ID) || empty(section.GroupID) {
				return fmt.Errorf("graph wall section %s requires id and group_id", section.ID)
			}
			for _, card := range section.Cards {
				if empty(card.ID) || empty(card.Kind) || empty(card.AxisPolicy) {
					return fmt.Errorf("graph wall card %s requires id, kind, and axis_policy", card.ID)
				}
				if empty(card.Placement.SectionID) || empty(card.Placement.GroupID) {
					return fmt.Errorf("graph wall card %s requires backend placement", card.ID)
				}
				if len(card.Signals) == 0 {
					return fmt.Errorf("graph wall card %s requires signals", card.ID)
				}
				for _, signal := range card.Signals {
					if empty(signal.ID) || empty(signal.Kind) || empty(signal.SourceFamily) {
						return fmt.Errorf("graph wall card %s signal %s requires id, kind, and source family", card.ID, signal.ID)
					}
				}
			}
		}
	}
	if g.TileManifest != nil {
		if empty(g.TileManifest.ID) || empty(g.TileManifest.CampaignID) || empty(g.TileManifest.GraphWallID) {
			return errors.New("tile manifest requires id, campaign_id, and graph_wall_id")
		}
		if g.TileManifest.CampaignID != g.CampaignID {
			return fmt.Errorf("tile manifest campaign_id %s does not match graph campaign_id %s", g.TileManifest.CampaignID, g.CampaignID)
		}
		if len(g.TileManifest.Levels) == 0 {
			return fmt.Errorf("tile manifest %s requires levels", g.TileManifest.ID)
		}
		if len(g.TileManifest.Cards) == 0 {
			return fmt.Errorf("tile manifest %s requires card refs", g.TileManifest.ID)
		}
		validRenderKinds := map[string]bool{"line": true, "stepped": true, "counter": true, "swimlane": true, "event_rail": true, "band": true, "annotation": true}
		for _, card := range g.TileManifest.Cards {
			if empty(card.CardID) || empty(card.RenderKind) || empty(card.TileEndpoint) || empty(card.LatestEndpoint) {
				return fmt.Errorf("tile manifest card %s requires id, render kind, tile endpoint, and latest endpoint", card.CardID)
			}
			if !validRenderKinds[card.RenderKind] {
				return fmt.Errorf("tile manifest card %s has unsupported render kind %q", card.CardID, card.RenderKind)
			}
			if len(card.Signals) == 0 && card.RenderKind != "annotation" {
				return fmt.Errorf("tile manifest card %s requires signals", card.CardID)
			}
		}
		for _, link := range g.TileManifest.EvidenceLinks {
			if empty(link.ID) || empty(link.RequirementID) || empty(link.CardID) || empty(link.Timestamp) || empty(link.Status) {
				return fmt.Errorf("tile manifest evidence link %s requires requirement, card, timestamp, and status", link.ID)
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
