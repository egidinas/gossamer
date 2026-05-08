export type Envelope = {
  schema_version: number;
  generated_at: string;
};

export type Manifest = Envelope & {
  name: string;
  description: string;
  test_article: string;
  campaigns: string[];
  public_demo: boolean;
  synthetic_only: boolean;
};

export type Node = {
  id: string;
  label: string;
  kind: string;
  status: string;
  quality: string;
};

export type Link = {
  source: string;
  target: string;
  bus: string;
};

export type Topology = Envelope & {
  nodes: Node[];
  links: Link[];
};

export type Source = {
  id: string;
  label: string;
  node_id: string;
  served_by: string;
  owner: string;
  bus: string;
  quality: string;
  freshness_ms: number;
  provenance: string;
  evidence_suitability: string;
  signals: string[];
};

export type SourceCatalogue = Envelope & {
  sources: Source[];
};

export type Requirement = {
  id: string;
  title: string;
  description: string;
  expression?: string;
  result: string;
  evidence: string[];
  rationale: string;
};

export type NodeKind = "test_article" | "facility" | "data_system" | "supervisor" | string;

export type RequirementProgress = {
  id: string;
  label: string;
  completed: number;
  target: number;
  percent: number;
  state: string;
  contributors: string[];
  next_milestone?: string;
  evidence_source: string;
};

export type CyclePhase = {
  id: string;
  label: string;
  kind: string;
  start: string;
  end: string;
  target_deg_c: number;
};

export type DwellWindow = {
  id: string;
  label: string;
  cycle_index: number;
  kind: string;
  start: string;
  end: string;
  target_deg_c: number;
  stability_band_c: number;
  minimum_minutes: number;
  evidence_ref: string;
};

export type FunctionalGate = {
  id: string;
  label: string;
  gate: string;
  cycle_index: number;
  phase_id: string;
  timestamp: string;
  result: string;
  evidence_ref: string;
};

export type InterlockWindow = {
  id: string;
  label: string;
  start: string;
  end: string;
  state: string;
  severity: string;
  evidence_ref: string;
};

export type EvidenceMarker = {
  id: string;
  label: string;
  timestamp: string;
  kind: string;
  result: string;
  evidence_ref: string;
};

export type ThermalCycle = {
  index: number;
  label: string;
  start: string;
  end: string;
  cold_target_deg_c: number;
  hot_target_deg_c: number;
  phases: CyclePhase[];
};

export type ThermalProgram = {
  id: string;
  kind: string;
  label: string;
  facility: string;
  cycle_count: number;
  cold_target_deg_c: number;
  hot_target_deg_c: number;
  ramp_rate_deg_c_min: number;
  dwell_minutes: number;
  cycles: ThermalCycle[];
  dwell_windows: DwellWindow[];
  functional_gates: FunctionalGate[];
  interlock_windows: InterlockWindow[];
  evidence_markers: EvidenceMarker[];
};

export type Anomaly = {
  id: string;
  title: string;
  severity: string;
  status: string;
  evidence_ref: string;
  disposition: string;
};

export type Campaign = Envelope & {
  id: string;
  name: string;
  level: string;
  state: string;
  result: string;
  article: string;
  facility: string;
  requirements: Requirement[];
  anomalies: Anomaly[];
  thermal_program?: ThermalProgram;
  synthetic_note: string;
};

export type CampaignList = Envelope & {
  campaigns: Campaign[];
};

export type GraphSeries = {
  id: string;
  label: string;
  role: string;
  units: string;
  source: string;
  node_id?: string;
  min: number;
  max: number;
};

export type GraphPoint = {
  timestamp: string;
  value: number;
};

export type GraphLane = {
  id: string;
  label: string;
  series: GraphSeries[];
};

export type SimulationProvenance = {
  model: string;
  model_version: string;
  seed: number;
  step_seconds: number;
  source: string;
  parameters: Record<string, number>;
  deterministic: boolean;
};

export type GraphTimeAxis = {
  start: string;
  end: string;
  anchor: string;
  now?: string;
  default_window_start?: string;
  default_window_end?: string;
  range_seconds: number;
  clamp: boolean;
  latest_policy: string;
};

export type ExecutionState = {
  mode: string;
  now: string;
  percent_complete: number;
  acceleration: string;
  past_data_policy: string;
  future_data_policy: string;
  completed_cycles: number;
  target_cycles: number;
  current_cycle: number;
  current_phase: string;
  requirement_progress: RequirementProgress[];
};

export type GraphYAxis = {
  id: string;
  label: string;
  units: string;
  scale: string;
  min: number;
  max: number;
  side: string;
  format: string;
};

export type GraphTrace = {
  id: string;
  label: string;
  role: string;
  units: string;
  axis_id: string;
  source: string;
  values: GraphPoint[];
};

export type GraphBand = {
  id: string;
  label: string;
  kind: string;
  start: string;
  end: string;
  cycle_index?: number;
  target_deg_c?: number;
  result?: string;
};

export type GraphMarker = {
  id: string;
  label: string;
  kind: string;
  role: string;
  timestamp: string;
  cycle_index?: number;
  axis_id?: string;
  value?: number;
  result?: string;
  severity?: string;
  evidence_ref?: string;
};

export type CompanionGraphGroup = {
  id: string;
  label: string;
  axes: GraphYAxis[];
  traces: GraphTrace[];
};

export type ThermalDiagramNode = {
  id: string;
  label: string;
  kind: string;
  role: string;
  signal?: string;
  x: number;
  y: number;
};

export type ThermalDiagramLink = {
  id: string;
  source: string;
  target: string;
  kind: string;
  label: string;
  strength: number;
  signal?: string;
};

export type TestItemThermalDiagram = {
  id: string;
  label: string;
  context: string;
  summary: string;
  nodes: ThermalDiagramNode[];
  links: ThermalDiagramLink[];
  notes?: string[];
};

export type HeroGraphModel = {
  id: string;
  title: string;
  owner: string;
  provenance: string;
  time_axis: GraphTimeAxis;
  execution?: ExecutionState;
  axes: GraphYAxis[];
  traces: GraphTrace[];
  phase_bands: GraphBand[];
  dwell_windows: GraphBand[];
  markers: GraphMarker[];
  companion_groups: CompanionGraphGroup[];
  thermal_diagram?: TestItemThermalDiagram;
};

export type GraphWallTimeRange = {
  start: string;
  end: string;
  anchor: string;
  range_seconds: number;
  mode: string;
  source: string;
};

export type GraphTilePolicy = {
  default_points: number;
  max_points: number;
  live_tile_min_refresh_ms: number;
  history_tile_max_count: number;
  viewport_prefetch_px: number;
  tile_buffer_max_entries: number;
  tile_buffer_ttl_ms: number;
  resolution_levels: string[];
  subscriber_role: string;
  shared_timebase_required: boolean;
  legend_may_affect_plot_width: boolean;
  malformed_svg_path_hard_failure: boolean;
};

export type GraphInteraction = {
  shared_timeline: boolean;
  shared_crosshair: boolean;
  vertical_grid: boolean;
  single_time_axis: boolean;
  cursor_mode: string;
  crosshair_scope: string;
  timeline_grid_mode: string;
};

export type GraphLayoutContract = {
  pinned_cards_separate: boolean;
  overflow_mode: string;
  axis_rail: string;
  legend_rail: string;
  label_rail: string;
  plot_area_policy: string;
};

export type GraphGroup = {
  id: string;
  title: string;
  mode: string;
  behavior_profile: string;
  application: string;
  section_ids: string[];
  interaction: GraphInteraction;
  layout: GraphLayoutContract;
};

export type GraphCardPlacement = {
  section_id: string;
  group_id: string;
  order: number;
  height_weight: number;
  default_visible: boolean;
  pinned: boolean;
  colocated_with?: string;
  resize_policy: string;
};

export type GraphWallSignal = {
  id: string;
  label: string;
  unit?: string;
  source: string;
  source_family: string;
  kind: string;
  category: string;
  role: string;
  subsystem: string;
  axis_id?: string;
  section_id: string;
  value_table?: Record<string, string>;
};

export type GraphWallCard = {
  id: string;
  title: string;
  kind: "line" | "counter" | "state" | "event" | string;
  role: string;
  placement: GraphCardPlacement;
  transport: string;
  direction: string;
  unit?: string;
  axis_policy: string;
  source_family: string;
  overview?: boolean;
  bucket?: string;
  note?: string;
  render_kind?: string;
  tile_endpoint?: string;
  latest_endpoint?: string;
  collapsible?: boolean;
  default_expanded?: boolean;
  supports_time_zoom?: boolean;
  supports_y_zoom?: boolean;
  include_markers?: boolean;
  signals: GraphWallSignal[];
};

export type GraphSection = {
  id: string;
  title: string;
  group_id: string;
  transport: string;
  direction: string;
  status: string;
  unplotted_count: number;
  cards: GraphWallCard[];
};

export type GraphWallModel = {
  id: string;
  title: string;
  generated_at: string;
  source_mode: string;
  graph_version: string;
  owner: string;
  provenance: string;
  time_range: GraphWallTimeRange;
  tile_policy: GraphTilePolicy;
  graph_groups: GraphGroup[];
  sections: GraphSection[];
};

export type GraphModel = Envelope & {
  campaign_id: string;
  lanes: GraphLane[];
  thermal_program?: ThermalProgram;
  simulation_provenance?: SimulationProvenance;
  hero_graph?: HeroGraphModel;
  graph_wall?: GraphWallModel;
  tile_manifest?: GraphTileManifest;
};

export type TileLevel = {
  id: string;
  label: string;
  resolution?: string;
  duration_ms?: number;
  decimation_mode?: string;
  resolution_ms?: number;
  max_points: number;
  span_seconds?: number;
  decimation?: string;
};

export type SourceNode = {
  id: string;
  label: string;
  kind: string;
  mode: string;
  confidence: string;
  provenance: string;
};

export type DataLensTranslation = {
  id: string;
  label: string;
  source_format: string;
  target_schema: string;
  mode: string;
  confidence: string;
  provenance: string;
};

export type EvidenceLink = {
  id: string;
  requirement_id: string;
  card_id: string;
  signal_id?: string;
  marker_id?: string;
  tile_id?: string;
  timestamp: string;
  status: string;
  label?: string;
  evidence_ref?: string;
};

export type GraphTileCardRef = {
  card_id: string;
  section_id?: string;
  title: string;
  render_kind: string;
  unit?: string;
  axis_policy: string;
  tile_endpoint: string;
  latest_endpoint: string;
  tile_files?: TileFile[];
  default_expanded: boolean;
  collapsible: boolean;
  supports_time_zoom: boolean;
  supports_y_zoom: boolean;
  include_markers?: boolean;
  signals: GraphWallSignal[];
  evidence_links?: EvidenceLink[];
};

export type GraphTileManifest = Envelope & {
  id: string;
  campaign_id: string;
  graph_wall_id: string;
  generated_at: string;
  source_mode: string;
  source_fixture_version?: string;
  time_range: GraphWallTimeRange;
  tile_policy: GraphTilePolicy;
  levels: TileLevel[];
  cards: GraphTileCardRef[];
  source_nodes: SourceNode[];
  datalens_translations: DataLensTranslation[];
  evidence_links: EvidenceLink[];
};

export type TileBundleManifest = Envelope & {
  id: string;
  data_version: string;
  ui_version?: string;
  generated_at: string;
  simulation_model_version?: string;
  source_fixture_version?: string;
  time_range: GraphWallTimeRange;
  replay_speed: string;
  present_cursor_policy: string;
  campaigns: TileCampaignManifest[];
  source_nodes?: SourceNode[];
  datalens_translations?: DataLensTranslation[];
  evidence_links?: EvidenceLink[];
  provenance: TileBundleProvenance;
};

export type TileCampaignManifest = {
  campaign_id: string;
  title: string;
  graph_shell_path: string;
  manifest_path: string;
  time_range: GraphWallTimeRange;
  replay_speed: string;
  levels: TileLevel[];
  cards: GraphTileCardRef[];
  evidence_links?: EvidenceLink[];
  compressed_bytes?: number;
  uncompressed_bytes?: number;
};

export type TileFile = {
  id: string;
  level: string;
  path: string;
  compressed_path?: string;
  t0: string;
  t1: string;
  render_kind: string;
  point_count: number;
  raw_point_count: number;
  compressed_bytes?: number;
  bytes?: number;
};

export type TileBundleProvenance = {
  generator: string;
  generator_version: string;
  build_host?: string;
  generated_from: string[];
  heavy_computation_policy: string;
  runtime_policy: string;
  parameters?: Record<string, string>;
};

export type TilePoint = {
  timestamp: string;
  value: number;
};

export type TileSpan = {
  start: string;
  end: string;
  value?: number;
  state?: string;
  label?: string;
  severity?: string;
};

export type TileSeries = {
  id: string;
  label: string;
  role: string;
  unit?: string;
  units?: string;
  kind?: string;
  axis_id?: string;
  source: string;
  source_family?: string;
  render_kind?: string;
  step?: boolean;
  value_table?: Record<string, string>;
  points?: TilePoint[];
  spans?: TileSpan[];
};

export type TileDiagnostics = {
  source?: string;
  mode?: string;
  level?: string;
  requested_t0?: string;
  requested_t1?: string;
  raw_point_count: number;
  point_count: number;
  decimated?: boolean;
  decimation: string;
  time_span_ms?: number;
  freshness_ms: number;
  source_quality?: string;
};

export type TileProvenance = {
  source?: string;
  source_node?: string;
  source_family?: string;
  mode?: string;
  generation_mode?: string;
  fixture_version?: string;
  point_count?: number;
  raw_point_count?: number;
  generated_at?: string;
  synthetic?: boolean;
};

export type TileEvent = {
  id: string;
  label: string;
  timestamp: string;
  kind: string;
  result?: string;
  requirement_id?: string;
  value?: number;
  severity?: string;
  evidence_ref?: string;
};

export type GraphTile = Envelope & {
  id: string;
  campaign_id: string;
  card_id: string;
  level: string;
  t0: string;
  t1: string;
  series: TileSeries[];
  bands: GraphBand[];
  markers: GraphMarker[];
  events: TileEvent[];
  diagnostics: TileDiagnostics;
  provenance: TileProvenance;
};

export type SupervisorHeroGraph = {
  id: string;
  label: string;
  signal: string;
  units: string;
  role: string;
  source: string;
  min: number;
  max: number;
  values: GraphPoint[];
};

export type SupervisorLane = {
  id: string;
  label: string;
  facility: string;
  campaign: string;
  activity: string;
  state: string;
  result: string;
  primary_bus: string;
  requirement_summary: string;
  source_quality: string;
  hero_graphs: SupervisorHeroGraph[];
  notes: string[];
  thermal_program?: ThermalProgram;
  hero_graph?: HeroGraphModel;
  functional_gates?: FunctionalGate[];
  interlock_windows?: InterlockWindow[];
  evidence_markers?: EvidenceMarker[];
};

export type SupervisorOverview = Envelope & {
  test_article: string;
  summary: string;
  lanes: SupervisorLane[];
};

export type CommandCenterBand = {
  id: string;
  label: string;
  kind: string;
  start: string;
  end: string;
};

export type CommandCenterEvent = {
  id: string;
  label: string;
  kind: string;
  timestamp: string;
  state: string;
};

export type CommandCenterTrace = {
  id: string;
  label: string;
  role: string;
  units: string;
  min: number;
  max: number;
  values: GraphPoint[];
};

export type CommandCenterTestItemManifest = {
  id: string;
  label: string;
  article: string;
  serial_number: string;
  facility: string;
  chamber_name: string;
  campaign_id: string;
  operator_next: string;
  state: string;
  result: string;
  start: string;
  end: string;
  breakdown_start: string;
  breakdown_end: string;
  reset_start: string;
  reset_end: string;
};

export type CommandCenterRun = {
  id: string;
  campaign_id: string;
  title: string;
  state: string;
  result: string;
  start: string;
  end: string;
  breakdown_start: string;
  breakdown_end: string;
  reset_start: string;
  reset_end: string;
  manifest: CommandCenterTestItemManifest;
  traces?: CommandCenterTrace[];
  interaction_windows: CommandCenterBand[];
  events: CommandCenterEvent[];
};

export type CommandCenterLane = {
  id: string;
  chamber_name: string;
  facility: string;
  summary: string;
  graph_card_id?: string;
  runs: CommandCenterRun[];
};

export type CommandCenterFAT = Envelope & {
  id: string;
  title: string;
  summary: string;
  now: string;
  window_start: string;
  window_end: string;
  data_start?: string;
  data_end?: string;
  schedule_policy?: string;
  workday_start_hour: number;
  workday_end_hour: number;
  graph_campaign_id?: string;
  hero_graph?: HeroGraphModel;
  graph_wall?: GraphWallModel;
  weekend_bands: CommandCenterBand[];
  lanes: CommandCenterLane[];
};

export type BusStream = {
  id: string;
  label: string;
  direction: "TM" | "TC";
  source_node: string;
  destination_node: string;
  bus: string;
  quality: string;
  latency_ms: number;
  packet_counter: number;
  dropped_frames: number;
};

export type BusEvent = {
  id: string;
  stream_id: string;
  direction: "TM" | "TC";
  timestamp: string;
  source_node: string;
  destination_node: string;
  event_class: string;
  authority: string;
  quality: string;
  latency_ms: number;
  packet_counter: number;
  fields: Record<string, number>;
  states: Record<string, string>;
  summary: string;
};

export type BusVirtualizationTap = Envelope & {
  connection_id: string;
  description: string;
  replay_cursor: string;
  streams: BusStream[];
  events: BusEvent[];
};

export type TelemetrySample = {
  timestamp: string;
  signals: Record<string, number>;
  states: Record<string, string>;
  quality: string;
};

export type CommandAuthorityState = Envelope & {
  lease_owner: string;
  lease_state: string;
  allowed_commands: string[];
  last_command: string;
};

export type EvidenceReport = Envelope & {
  campaign_id: string;
  summary: string;
  result: string;
  requirements: Requirement[];
  sources: Source[];
  graph_evidence: string[];
  anomalies: Anomaly[];
  thermal_program?: ThermalProgram;
  simulation_provenance?: SimulationProvenance;
  reproducibility: string[];
  synthetic_data_note: string;
};

export type FileSignalGroup = {
  node_id: string;
  node_label: string;
  source_id: string;
  source_label: string;
  bus: string;
  series: GraphSeries[];
};

export type FileViewModel = Envelope & {
  campaign_id: string;
  campaign_name: string;
  file_ref: string;
  file_kind: string;
  time_start: string;
  time_end: string;
  signal_groups: FileSignalGroup[];
  lanes: GraphLane[];
};
