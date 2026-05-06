import type {
  Campaign,
  CampaignList,
  CommandAuthorityState,
  EvidenceReport,
  GraphModel,
  Manifest,
  SourceCatalogue,
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
  campaigns: () => getJSON<CampaignList>("/api/campaigns"),
  campaign: (id: string) => getJSON<Campaign>(`/api/campaigns/${id}`),
  telemetry: (id: string) => getJSONL<TelemetrySample>(`/api/campaigns/${id}/telemetry`),
  graphModel: (id: string) => getJSON<GraphModel>(`/api/campaigns/${id}/graph-model`),
  commandAuthority: () => getJSON<CommandAuthorityState>("/api/command-authority"),
  evidenceReport: (id: string) => getJSON<EvidenceReport>(`/api/campaigns/${id}/evidence-report`),
  requestLease: () => fetch("/api/command-authority/request-lease", { method: "POST" }).then(() => api.commandAuthority()),
  releaseLease: () => fetch("/api/command-authority/release-lease", { method: "POST" }).then(() => api.commandAuthority()),
  mockCommand: () => fetch("/api/command-authority/mock-command", { method: "POST" }).then(() => api.commandAuthority())
};

