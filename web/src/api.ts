import type {
  Campaign,
  CampaignList,
  BusVirtualizationTap,
  CommandCenterFAT,
  CommandAuthorityState,
  EvidenceReport,
  GraphTileManifest,
  GraphModel,
  Manifest,
  SourceCatalogue,
  SupervisorOverview,
  TileBundleManifest,
  Topology
} from "./types";
import { arrowTile } from "./arrowTiles";

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path);
  if (!response.ok) {
    throw new Error(`${path} returned ${response.status}`);
  }
  const text = await response.text();
  const trimmed = text.trimStart();
  if (trimmed.startsWith("<!doctype") || trimmed.startsWith("<html") || trimmed.startsWith("<")) {
    throw new Error(`${path} returned HTML instead of JSON`);
  }
  try {
    return JSON.parse(text) as T;
  } catch (err) {
    throw new Error(`${path} returned invalid JSON: ${err instanceof Error ? err.message : String(err)}`);
  }
}

const tileManifestCache = new Map<string, Promise<GraphTileManifest>>();
const graphShellCache = new Map<string, Promise<GraphModel>>();

function cachedTileManifest(id: string) {
  const key = id;
  let cached = tileManifestCache.get(key);
  if (!cached) {
    cached = getJSON<GraphTileManifest>(`/data/current/campaigns/${id}/manifest.json`);
    tileManifestCache.set(key, cached);
  }
  return cached;
}

function cachedGraphShell(id: string) {
  let cached = graphShellCache.get(id);
  if (!cached) {
    cached = getJSON<GraphModel>(`/data/current/campaigns/${id}/graph-shell.json`);
    graphShellCache.set(id, cached);
  }
  return cached;
}

export const api = {
  currentBundle: () => getJSON<TileBundleManifest>("/data/current/manifest.json"),
  manifest: () => getJSON<Manifest>("/api/manifest"),
  topology: () => getJSON<Topology>("/api/topology"),
  sources: () => getJSON<SourceCatalogue>("/api/sources"),
  supervisor: () => getJSON<SupervisorOverview>("/api/supervisor"),
  commandCenterFAT: () => getJSON<CommandCenterFAT>("/data/current/command_center_fat.json"),
  busTap: () => getJSON<BusVirtualizationTap>("/api/bus-tap"),
  campaigns: () => getJSON<CampaignList>("/api/campaigns"),
  campaign: (id: string) => getJSON<Campaign>(`/api/campaigns/${id}`),
  graphModel: (id: string) => getJSON<GraphModel>(`/api/campaigns/${id}/graph-model`),
  graphShell: (id: string) => cachedGraphShell(id),
  tileManifest: (id: string) => cachedTileManifest(id),
  tile: async (id: string, cardID: string, level = "minute", t0?: string, t1?: string) => {
    const [manifest, graph] = await Promise.all([cachedTileManifest(id), cachedGraphShell(id)]);
    const card = manifest.cards.find((candidate) => candidate.card_id === cardID);
    if (!card) throw new Error(`unknown card ${cardID}`);
    return arrowTile(id, card, graph, level, t0, t1);
  },
  commandAuthority: () => getJSON<CommandAuthorityState>("/api/command-authority"),
  evidenceReport: (id: string) => getJSON<EvidenceReport>(`/api/campaigns/${id}/evidence-report`),
  requestLease: () => fetch("/api/command-authority/request-lease", { method: "POST" }).then(() => api.commandAuthority()),
  releaseLease: () => fetch("/api/command-authority/release-lease", { method: "POST" }).then(() => api.commandAuthority()),
  mockCommand: () => fetch("/api/command-authority/mock-command", { method: "POST" }).then(() => api.commandAuthority())
};
