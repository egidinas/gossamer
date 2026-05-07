import { Activity, FileCheck, Home, ShieldCheck, User } from "lucide-react";
import { lazy, Suspense, useEffect, useMemo, useState } from "react";
import { api } from "./api";
import type { BusVirtualizationTap, Campaign, CommandAuthorityState, EvidenceReport, GraphModel, Manifest, SourceCatalogue, SupervisorOverview, Topology } from "./types";
import { LandingView } from "./views/LandingView";

const MissionMapView = lazy(() => import("./views/MissionMapView").then((module) => ({ default: module.MissionMapView })));
const SupervisorView = lazy(() => import("./views/SupervisorView").then((module) => ({ default: module.SupervisorView })));
const GraphWallView = lazy(() => import("./views/GraphWallView").then((module) => ({ default: module.GraphWallView })));
const SourceCatalogueView = lazy(() => import("./views/SourceCatalogueView").then((module) => ({ default: module.SourceCatalogueView })));
const RequirementMatrixView = lazy(() => import("./views/RequirementMatrixView").then((module) => ({ default: module.RequirementMatrixView })));
const CommandAuthorityView = lazy(() => import("./views/CommandAuthorityView").then((module) => ({ default: module.CommandAuthorityView })));
const EvidenceReportView = lazy(() => import("./views/EvidenceReportView").then((module) => ({ default: module.EvidenceReportView })));
const BusTapView = lazy(() => import("./views/BusTapView").then((module) => ({ default: module.BusTapView })));
const ProfileView = lazy(() => import("./views/ProfileView").then((module) => ({ default: module.ProfileView })));

type Route = "landing" | "profile" | "acceptance" | "qualification" | "mission-map" | "supervisor" | "graph-wall" | "sources" | "requirements" | "commands" | "bus-tap" | "report";

const campaignRouteMap: Partial<Record<Route, string>> = {
  acceptance: "thermal_acceptance_fat",
  qualification: "tvac_qualification"
};

const routes: Array<{ id: Route; label: string; icon: typeof Activity; published: boolean }> = [
  { id: "landing", label: "Home", icon: Home, published: true },
  { id: "acceptance", label: "Acceptance FAT", icon: Activity, published: true },
  { id: "qualification", label: "Qualification TVac", icon: Activity, published: true },
  { id: "report", label: "Evidence", icon: ShieldCheck, published: true },
  { id: "profile", label: "Profile", icon: User, published: true },
  { id: "mission-map", label: "Mission", icon: FileCheck, published: false },
  { id: "supervisor", label: "Test Campaign", icon: Activity, published: false },
  { id: "graph-wall", label: "Graphs", icon: Activity, published: false },
  { id: "sources", label: "Sources", icon: FileCheck, published: false },
  { id: "requirements", label: "Requirements", icon: FileCheck, published: false },
  { id: "commands", label: "Commands", icon: FileCheck, published: false },
  { id: "bus-tap", label: "Bus Tap", icon: FileCheck, published: false }
];

const publishedRoutes = routes.filter((route) => route.published);
const bootGeneratedAt = "2026-01-15T00:00:00Z";
const bootManifest: Manifest = {
  schema_version: 1,
  generated_at: bootGeneratedAt,
  name: "Gossamer",
  description: "Tile-backed environmental-test evidence and telemetry exploration.",
  test_article: "Reference DUT",
  campaigns: ["thermal_acceptance_fat", "tvac_qualification"],
  public_demo: true,
  synthetic_only: true
};
const bootCampaigns: Campaign[] = [
  {
    schema_version: 1,
    generated_at: bootGeneratedAt,
    id: "thermal_acceptance_fat",
    name: "Thermal Chamber FAT",
    level: "acceptance",
    state: "running",
    result: "pass",
    article: "Reference DUT",
    facility: "thermal_chamber_a",
    requirements: [],
    anomalies: [],
    synthetic_note: "Reference campaign model with backend-owned telemetry, evidence, and graph contracts."
  },
  {
    schema_version: 1,
    generated_at: bootGeneratedAt,
    id: "tvac_qualification",
    name: "TVac Qualification",
    level: "qualification",
    state: "running",
    result: "inconclusive",
    article: "Reference DUT",
    facility: "tvac_chamber_q1",
    requirements: [],
    anomalies: [],
    synthetic_note: "Reference campaign model with backend-owned telemetry, evidence, and graph contracts."
  }
];
const bootSupervisor: SupervisorOverview = {
  schema_version: 1,
  generated_at: bootGeneratedAt,
  test_article: "Reference DUT",
  summary: "Tile-backed campaign shell. Backend contracts hydrate after first paint.",
  lanes: [
    {
      id: "boot-thermal_acceptance_fat",
      label: "Thermal Chamber FAT",
      facility: "thermal_chamber_a",
      campaign: "thermal_acceptance_fat",
      activity: "4-cycle acceptance profile",
      state: "running",
      result: "pass",
      primary_bus: "CAN/TMTC",
      requirement_summary: "4-cycle acceptance campaign shell.",
      source_quality: "fresh",
      hero_graphs: [],
      notes: []
    },
    {
      id: "boot-tvac_qualification",
      label: "TVac Qualification",
      facility: "tvac_chamber_q1",
      campaign: "tvac_qualification",
      activity: "8-cycle qualification profile",
      state: "running",
      result: "inconclusive",
      primary_bus: "CAN/TMTC",
      requirement_summary: "8-cycle qualification campaign shell.",
      source_quality: "degraded",
      hero_graphs: [],
      notes: []
    }
  ]
};

export function App() {
  const [route, setRoute] = useState<Route>(hashRoute());
  const [manifest, setManifest] = useState<Manifest>(bootManifest);
  const [topology, setTopology] = useState<Topology | null>(null);
  const [sources, setSources] = useState<SourceCatalogue | null>(null);
  const [supervisor, setSupervisor] = useState<SupervisorOverview>(bootSupervisor);
  const [busTap, setBusTap] = useState<BusVirtualizationTap | null>(null);
  const [campaigns, setCampaigns] = useState<Campaign[]>(bootCampaigns);
  const [activeCampaign, setActiveCampaign] = useState("thermal_acceptance_fat");
  const [graph, setGraph] = useState<GraphModel | null>(null);
  const [commands, setCommands] = useState<CommandAuthorityState | null>(null);
  const [report, setReport] = useState<EvidenceReport | null>(null);
  const [error, setError] = useState("");
  const routeCampaign = campaignRouteMap[route];
  const requestedCampaign = routeCampaign ?? activeCampaign;

  useEffect(() => {
    const onHash = () => setRoute(hashRoute());
    window.addEventListener("hashchange", onHash);
    if (!window.location.hash) window.location.hash = "#landing";
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  useEffect(() => {
    const nextCampaign = campaignRouteMap[route];
    if (nextCampaign && nextCampaign !== activeCampaign) {
      setActiveCampaign(nextCampaign);
    }
  }, [route, activeCampaign]);

  useEffect(() => {
    scheduleIdle(() => {
      Promise.all([api.manifest(), api.supervisor(), api.campaigns()])
      .then(([m, so, c]) => {
        setManifest(m);
        setSupervisor(so);
        setCampaigns(c.campaigns);
      })
      .catch((err: Error) => setError(err.message));
    });
  }, []);

  useEffect(() => {
    if (route === "landing") return;
    if (route === "report") {
      Promise.all([api.campaign(requestedCampaign), api.evidenceReport(requestedCampaign)])
        .then(([campaign, evidence]) => {
          setCampaigns((existing) => existing.map((item) => item.id === campaign.id ? campaign : item));
          setReport(evidence);
        })
        .catch((err: Error) => setError(err.message));
      return;
    }
    if (route === "mission-map") {
      api.topology().then(setTopology).catch((err: Error) => setError(err.message));
      return;
    }
    if (route === "sources") {
      api.sources().then(setSources).catch((err: Error) => setError(err.message));
      return;
    }
    if (route === "commands") {
      api.commandAuthority().then(setCommands).catch((err: Error) => setError(err.message));
      return;
    }
    if (route === "bus-tap") {
      api.busTap().then(setBusTap).catch((err: Error) => setError(err.message));
      return;
    }
    Promise.all([api.campaign(requestedCampaign), api.graphShell(requestedCampaign)])
      .then(([campaign, graphModel]) => {
        setCampaigns((existing) => existing.map((item) => item.id === campaign.id ? campaign : item));
        setGraph(graphModel);
      })
      .catch((err: Error) => setError(err.message));
  }, [requestedCampaign, route]);

  const selectedCampaign = useMemo(() => campaigns.find((campaign) => campaign.id === requestedCampaign), [campaigns, requestedCampaign]);
  const graphReady = graph?.campaign_id === requestedCampaign;
  const reportReady = report?.campaign_id === requestedCampaign;
  const landingReady = campaigns.length > 0;
  const campaignRouteReady = landingReady && !!selectedCampaign && !!graph && graphReady;
  const routeReady =
    route === "landing" ? landingReady :
    route === "profile" ? true :
    route === "mission-map" ? landingReady && !!topology :
    route === "sources" ? landingReady && !!sources :
    route === "commands" ? landingReady && !!commands :
    route === "bus-tap" ? landingReady && !!busTap :
    route === "report" ? landingReady && !!report && reportReady :
    route === "requirements" ? landingReady && !!selectedCampaign :
    route === "graph-wall" ? campaignRouteReady :
    route === "acceptance" || route === "qualification" || route === "supervisor" ? campaignRouteReady :
    landingReady;

  const refreshCommands = (action: () => Promise<CommandAuthorityState>) => {
    action().then(setCommands).catch((err: Error) => setError(err.message));
  };

  if (error) return <main className="shell"><div className="error">{error}</div></main>;
  return (
    <main className="shell">
      <header className="topbar">
        <div className="brand-lockup">
          <img className="brand-mark" src="/assets/gossamer/gossamer-mark.webp" alt="" />
          <div>
            <h1>Gossamer</h1>
            <p>{manifest.description}</p>
          </div>
        </div>
        <address className="topbar-owner" aria-label="Gossamer author and contact">
          <a className="owner-name" href="#profile">Dr. Jonathan Meyer</a>
          <a href="mailto:jonathan@jmeyer.space">jonathan@jmeyer.space</a>
        </address>
        <nav className="nav">
          {publishedRoutes.map((item) => {
            const Icon = item.icon;
            return (
              <a key={item.id} href={`#${item.id}`} className={route === item.id ? "active" : ""}>
                <Icon size={16} /> {item.label}
              </a>
            );
          })}
        </nav>
      </header>
      {route === "landing" && <LandingView manifest={manifest} campaigns={campaigns} />}
      {(route === "acceptance" || route === "qualification" || route === "supervisor") && !routeReady && selectedCampaign && (
        <section className="instant-route-shell" aria-label={`${selectedCampaign.name} loading shell`}>
          <span className="eyebrow">test campaign</span>
          <h2>{selectedCampaign.name}</h2>
          <p>Loading backend graph contracts, tile manifests, evidence links, and campaign state.</p>
          <div className="instant-route-grid">
            <div><span className="label">Facility</span><strong>{selectedCampaign.facility}</strong></div>
            <div><span className="label">Program</span><strong>{selectedCampaign.id === "tvac_qualification" ? "8 cycles" : "4 cycles"}</strong></div>
            <div><span className="label">Data path</span><strong>backend tile hydration</strong></div>
          </div>
          <div className="loading route-loading">Loading graph contracts in background...</div>
        </section>
      )}
      {route !== "landing" && route !== "profile" && !(route === "acceptance" || route === "qualification" || route === "supervisor") && !routeReady && <div className="loading route-loading">Loading {routeLabel(route)} contracts...</div>}
      <Suspense fallback={<div className="loading route-loading">Loading view...</div>}>
        {route === "profile" && routeReady && <ProfileView />}
        {route === "mission-map" && routeReady && topology && <MissionMapView manifest={manifest} topology={topology} campaigns={campaigns} />}
        {(route === "acceptance" || route === "qualification" || route === "supervisor") && routeReady && selectedCampaign && graph && <SupervisorView overview={supervisor} campaign={selectedCampaign} graph={graph} />}
        {route === "graph-wall" && routeReady && graph && <GraphWallView model={graph} samples={[]} />}
        {route === "sources" && routeReady && sources && <SourceCatalogueView catalogue={sources} />}
        {route === "requirements" && routeReady && selectedCampaign && <RequirementMatrixView campaign={selectedCampaign} />}
        {route === "commands" && routeReady && commands && (
          <CommandAuthorityView
            state={commands}
            onRequest={() => refreshCommands(api.requestLease)}
            onRelease={() => refreshCommands(api.releaseLease)}
            onCommand={() => refreshCommands(api.mockCommand)}
          />
        )}
        {route === "bus-tap" && routeReady && busTap && <BusTapView tap={busTap} />}
        {route === "report" && routeReady && report && <EvidenceReportView report={report} />}
      </Suspense>
      {/* source easter egg: every confident plot should be able to point back to the sample that made it move. */}
      <footer className="site-footer">
        <span>Dr. Jonathan Meyer · jonathan@jmeyer.space</span>
        <a href="#acceptance">Physical model and components: FAT</a>
        <a href="#qualification">Physical model and components: TVac</a>
        <a href="https://github.com/egidinas/gossamer" target="_blank" rel="noopener noreferrer">View source on GitHub</a>
      </footer>
    </main>
  );
}

function hashRoute(): Route {
  const hash = window.location.hash.replace("#", "");
  const aliases: Record<string, Route> = {
    "": "landing",
    home: "landing",
    evidence: "report",
    "acceptance-fat": "acceptance",
    "thermal-acceptance-fat": "acceptance",
    "qualification-tvac": "qualification",
    "tvac-qualification": "qualification",
    tvac: "qualification"
  };
  const candidate = aliases[hash] ?? (hash as Route);
  return routes.some((route) => route.id === candidate) ? candidate : "landing";
}

function routeLabel(route: Route): string {
  return routes.find((item) => item.id === route)?.label ?? route;
}

function scheduleIdle(work: () => void) {
  if ("requestIdleCallback" in window) {
    window.requestIdleCallback(work, { timeout: 800 });
    return;
  }
  globalThis.setTimeout(work, 0);
}
