import type {
  Campaign,
  CampaignList,
  BusVirtualizationTap,
  CommandAuthorityState,
  EvidenceReport,
  GraphTile,
  GraphTileManifest,
  GraphModel,
  Manifest,
  SourceCatalogue,
  SupervisorOverview,
  TelemetrySample,
  Topology
} from "./types";

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path);
  if (!response.ok) {
    throw new Error(`${path} returned ${response.status}`);
  }
  return response.json() as Promise<T>;
}

async function getJSONL<T>(path: string): Promise<T[]> {
  const response = await fetch(path);
  if (!response.ok) {
    throw new Error(`${path} returned ${response.status}`);
  }
  const text = await response.text();
  return text.trim().split("\n").filter(Boolean).map((line) => JSON.parse(line) as T);
}

export const api = {
  manifest: () => getJSON<Manifest>("/api/manifest"),
  topology: () => getJSON<Topology>("/api/topology"),
  sources: () => getJSON<SourceCatalogue>("/api/sources"),
  supervisor: () => getJSON<SupervisorOverview>("/api/supervisor"),
  busTap: () => getJSON<BusVirtualizationTap>("/api/bus-tap"),
  campaigns: () => getJSON<CampaignList>("/api/campaigns"),
  campaign: (id: string) => getJSON<Campaign>(`/api/campaigns/${id}`),
  telemetry: (id: string) => getJSONL<TelemetrySample>(`/api/campaigns/${id}/telemetry`),
  graphModel: (id: string) => getJSON<GraphModel>(`/api/campaigns/${id}/graph-model`),
  graphShell: (id: string) => getJSON<GraphModel>(`/api/campaigns/${id}/graph-shell`),
  tileManifest: (id: string) => getJSON<GraphTileManifest>(`/api/campaigns/${id}/tile-manifest`),
  tile: (id: string, cardID: string, level = "minute", t0?: string, t1?: string) => {
    const params = new URLSearchParams({ card_id: cardID, level });
    if (t0) params.set("t0", t0);
    if (t1) params.set("t1", t1);
    return getJSON<GraphTile>(`/api/campaigns/${id}/tiles?${params.toString()}`);
  },
  commandAuthority: () => getJSON<CommandAuthorityState>("/api/command-authority"),
  evidenceReport: (id: string) => getJSON<EvidenceReport>(`/api/campaigns/${id}/evidence-report`),
  requestLease: () => fetch("/api/command-authority/request-lease", { method: "POST" }).then(() => api.commandAuthority()),
  releaseLease: () => fetch("/api/command-authority/release-lease", { method: "POST" }).then(() => api.commandAuthority()),
  mockCommand: () => fetch("/api/command-authority/mock-command", { method: "POST" }).then(() => api.commandAuthority())
};
