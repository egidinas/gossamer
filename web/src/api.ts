import type {
  Campaign,
  CampaignList,
  BusVirtualizationTap,
  CommandAuthorityState,
  EvidenceReport,
  GraphTile,
  GraphTileCardRef,
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

async function getJSONWithFallback<T>(primary: string, fallback: string): Promise<T> {
  try {
    return await getJSON<T>(primary);
  } catch (err) {
    if (!isMissingStaticData(err)) throw err;
    return getJSON<T>(fallback);
  }
}

function isMissingStaticData(err: unknown) {
  return err instanceof Error && (
    err.message.includes(" returned 404") ||
    err.message.includes(" returned 405") ||
    err.message.includes(" returned 500") ||
    err.message.includes(" returned HTML instead of JSON")
  );
}

function firstTileFile(card: GraphTileCardRef | undefined, level: string, t0?: string, t1?: string) {
  const files = card?.tile_files ?? [];
  if (!files.length) return undefined;
  const start = t0 ? Date.parse(t0) : Number.NEGATIVE_INFINITY;
  const end = t1 ? Date.parse(t1) : Number.POSITIVE_INFINITY;
  return files.find((file) => file.level === level && overlaps(file.t0, file.t1, start, end))
    ?? files.find((file) => file.level === level)
    ?? files[0];
}

function overlaps(fileT0: string, fileT1: string, start: number, end: number) {
  const a = Date.parse(fileT0);
  const b = Date.parse(fileT1);
  if (!Number.isFinite(start) || !Number.isFinite(end)) return true;
  if (!Number.isFinite(a) || !Number.isFinite(b)) return true;
  return b >= start && a <= end;
}

async function getJSONL<T>(path: string): Promise<T[]> {
  const response = await fetch(path);
  if (!response.ok) {
    throw new Error(`${path} returned ${response.status}`);
  }
  const text = await response.text();
  return text.trim().split("\n").filter(Boolean).map((line) => JSON.parse(line) as T);
}

const tileManifestCache = new Map<string, Promise<GraphTileManifest>>();
const tileCache = new Map<string, Promise<GraphTile>>();

function cachedTileManifest(id: string) {
  const key = id;
  let cached = tileManifestCache.get(key);
  if (!cached) {
    cached = getJSONWithFallback<GraphTileManifest>(
      `/data/current/campaigns/${id}/manifest.json`,
      `/api/campaigns/${id}/tile-manifest`
    );
    tileManifestCache.set(key, cached);
  }
  return cached;
}

function cachedTile(path: string) {
  let cached = tileCache.get(path);
  if (!cached) {
    cached = getJSON<GraphTile>(path);
    tileCache.set(path, cached);
  }
  return cached;
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
  graphShell: (id: string) => getJSONWithFallback<GraphModel>(
    `/data/current/campaigns/${id}/graph-shell.json`,
    `/api/campaigns/${id}/graph-shell`
  ),
  tileManifest: (id: string) => cachedTileManifest(id),
  tileByPath: (path: string) => cachedTile(path),
  tile: async (id: string, cardID: string, level = "minute", t0?: string, t1?: string) => {
    try {
      const manifest = await cachedTileManifest(id);
      const card = manifest.cards.find((candidate) => candidate.card_id === cardID);
      const tileFile = firstTileFile(card, level, t0, t1);
      if (tileFile) return cachedTile(tileFile.path);
    } catch (err) {
      if (!isMissingStaticData(err)) throw err;
    }
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
