import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type {
  Campaign,
  CampaignList,
  BusVirtualizationTap,
  CommandCenterFAT,
  CommandAuthorityState,
  EvidenceReport,
  FileViewModel,
  GraphTileManifest,
  GraphModel,
  Manifest,
  SourceCatalogue,
  SourceTreeConfig,
  SupervisorOverview,
  TileBundleManifest,
  Topology
} from "./types";
import { arrowTile, invalidateArrowCache } from "./arrowTiles";

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

async function getFirstJSON<T>(paths: string[]): Promise<T> {
  const errors: string[] = [];
  for (const path of paths) {
    try {
      return await getJSON<T>(path);
    } catch (err) {
      errors.push(err instanceof Error ? err.message : String(err));
    }
  }
  throw new Error(errors.join(" | "));
}

const tileManifestCache = new Map<string, Promise<GraphTileManifest>>();
const graphShellCache = new Map<string, Promise<GraphModel>>();
let currentDataVersion = "";

export function invalidateCaches(dataVersion: string) {
  if (dataVersion !== currentDataVersion) {
    currentDataVersion = dataVersion;
    tileManifestCache.clear();
    graphShellCache.clear();
    invalidateArrowCache();
  }
}

function cachedTileManifest(id: string) {
  const key = `${id}@${currentDataVersion}`;
  let cached = tileManifestCache.get(key);
  if (!cached) {
    cached = getJSON<GraphTileManifest>(`/data/current/campaigns/${id}/manifest.json`);
    tileManifestCache.set(key, cached);
  }
  return cached;
}

function cachedGraphShell(id: string) {
  const key = `${id}@${currentDataVersion}`;
  let cached = graphShellCache.get(key);
  if (!cached) {
    cached = getJSON<GraphModel>(`/data/current/campaigns/${id}/graph-shell.json`);
    graphShellCache.set(key, cached);
  }
  return cached;
}

export const api = {
  currentBundle: () => getJSON<TileBundleManifest>("/data/current/manifest.json"),
  manifest: () => getJSON<Manifest>("/api/manifest"),
  topology: () => getJSON<Topology>("/api/topology"),
  sources: () => getJSON<SourceCatalogue>("/api/sources"),
  sourceTreeConfig: () => getFirstJSON<SourceTreeConfig>(["/data/current/source_tree_config.json", "/api/source-tree-config"]),
  supervisor: () => getJSON<SupervisorOverview>("/api/supervisor"),
  commandCenterFAT: () => getFirstJSON<CommandCenterFAT>(["/data/current/command_center_fat.json", "/api/command-center/fat"]),
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
    return arrowTile(id, card, manifest, graph, level, t0, t1, currentDataVersion || undefined);
  },
  commandAuthority: () => getJSON<CommandAuthorityState>("/api/command-authority"),
  evidenceReport: (id: string) => getJSON<EvidenceReport>(`/api/campaigns/${id}/evidence-report`),
  fileViewer: (id: string) => getJSON<FileViewModel>(`/api/viewer/${id}`),
  requestLease: () => fetch("/api/command-authority/request-lease", { method: "POST", headers: { "X-Operator-ID": "operator-alpha-1" } }).then(() => api.commandAuthority()),
  releaseLease: () => fetch("/api/command-authority/release-lease", { method: "POST", headers: { "X-Operator-ID": "operator-alpha-1" } }).then(() => api.commandAuthority()),
  mockCommand: () => fetch("/api/command-authority/mock-command", { method: "POST", headers: { "X-Operator-ID": "operator-alpha-1" } }).then(() => api.commandAuthority())
};

// React Query Hooks
export function useBundleQuery() {
  return useQuery({
    queryKey: ["bundle"],
    queryFn: api.currentBundle,
    staleTime: 60000,
  });
}

export function useManifestQuery(enabled = true) {
  return useQuery({
    queryKey: ["manifest"],
    queryFn: api.manifest,
    enabled,
  });
}

export function useTopologyQuery(enabled = true) {
  return useQuery({
    queryKey: ["topology"],
    queryFn: api.topology,
    enabled,
  });
}

export function useSourcesQuery(enabled = true) {
  return useQuery({
    queryKey: ["sources"],
    queryFn: api.sources,
    enabled,
  });
}

export function useSourceTreeConfigQuery(enabled = true) {
  return useQuery({
    queryKey: ["source-tree-config"],
    queryFn: api.sourceTreeConfig,
    enabled,
  });
}

export function useSupervisorQuery(enabled = true) {
  return useQuery({
    queryKey: ["supervisor"],
    queryFn: api.supervisor,
    enabled,
  });
}

export function useCommandCenterFATQuery(enabled = true) {
  return useQuery({
    queryKey: ["command-center-fat"],
    queryFn: api.commandCenterFAT,
    enabled,
  });
}

export function useBusTapQuery(enabled = true) {
  return useQuery({
    queryKey: ["bus-tap"],
    queryFn: api.busTap,
    enabled,
  });
}

export function useCampaignQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: ["campaign", id],
    queryFn: () => api.campaign(id),
    enabled: enabled && !!id,
  });
}

export function useGraphShellQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: ["graph-shell", id],
    queryFn: () => api.graphShell(id),
    enabled: enabled && !!id,
  });
}

export function useCommandAuthorityQuery(enabled = true) {
  return useQuery({
    queryKey: ["command-authority"],
    queryFn: api.commandAuthority,
    enabled,
  });
}

export function useEvidenceReportQuery(id: string, enabled = true) {
  return useQuery({
    queryKey: ["evidence-report", id],
    queryFn: () => api.evidenceReport(id),
    enabled: enabled && !!id,
  });
}

export function useLeaseMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (action: "request" | "release" | "mock") => {
      switch (action) {
        case "request": return api.requestLease();
        case "release": return api.releaseLease();
        case "mock": return api.mockCommand();
      }
    },
    onSuccess: (data) => {
      queryClient.setQueryData(["command-authority"], data);
    },
  });
}
