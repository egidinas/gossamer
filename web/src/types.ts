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
  result: string;
  evidence: string[];
  rationale: string;
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

export type GraphModel = Envelope & {
  campaign_id: string;
  lanes: GraphLane[];
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
};

export type SupervisorOverview = Envelope & {
  test_article: string;
  summary: string;
  lanes: SupervisorLane[];
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
  reproducibility: string[];
  synthetic_data_note: string;
};
