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

export type GraphLane = {
  id: string;
  label: string;
  series: GraphSeries[];
};

export type GraphModel = Envelope & {
  campaign_id: string;
  lanes: GraphLane[];
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

