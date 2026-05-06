import { Activity, Database, FileCheck, GitBranch, Home, Network, RadioTower, ShieldCheck } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { api } from "./api";
import type { BusVirtualizationTap, Campaign, CommandAuthorityState, EvidenceReport, GraphModel, Manifest, SourceCatalogue, SupervisorOverview, TelemetrySample, Topology } from "./types";
import { LandingView } from "./views/LandingView";
import { MissionMapView } from "./views/MissionMapView";
import { SupervisorView } from "./views/SupervisorView";
import { GraphWallView } from "./views/GraphWallView";
import { SourceCatalogueView } from "./views/SourceCatalogueView";
import { RequirementMatrixView } from "./views/RequirementMatrixView";
import { CommandAuthorityView } from "./views/CommandAuthorityView";
import { EvidenceReportView } from "./views/EvidenceReportView";
import { BusTapView } from "./views/BusTapView";

type Route = "landing" | "mission-map" | "supervisor" | "graph-wall" | "sources" | "requirements" | "commands" | "bus-tap" | "report";

const routes: Array<{ id: Route; label: string; icon: typeof GitBranch }> = [
  { id: "landing", label: "Home", icon: Home },
  { id: "mission-map", label: "Mission", icon: GitBranch },
  { id: "supervisor", label: "Supervisor", icon: Activity },
  { id: "graph-wall", label: "Graphs", icon: Activity },
  { id: "sources", label: "Sources", icon: Database },
  { id: "requirements", label: "Requirements", icon: FileCheck },
  { id: "commands", label: "Commands", icon: RadioTower },
  { id: "bus-tap", label: "Bus Tap", icon: Network },
  { id: "report", label: "Report", icon: ShieldCheck }
];

export function App() {
  const [route, setRoute] = useState<Route>(hashRoute());
  const [manifest, setManifest] = useState<Manifest | null>(null);
  const [topology, setTopology] = useState<Topology | null>(null);
  const [sources, setSources] = useState<SourceCatalogue | null>(null);
  const [supervisor, setSupervisor] = useState<SupervisorOverview | null>(null);
  const [busTap, setBusTap] = useState<BusVirtualizationTap | null>(null);
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [activeCampaign, setActiveCampaign] = useState("thermal_acceptance_fat");
  const [graph, setGraph] = useState<GraphModel | null>(null);
  const [samples, setSamples] = useState<TelemetrySample[]>([]);
  const [commands, setCommands] = useState<CommandAuthorityState | null>(null);
  const [report, setReport] = useState<EvidenceReport | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    const onHash = () => setRoute(hashRoute());
    window.addEventListener("hashchange", onHash);
    if (!window.location.hash) window.location.hash = "#landing";
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  useEffect(() => {
    Promise.all([api.manifest(), api.topology(), api.sources(), api.supervisor(), api.busTap(), api.campaigns(), api.commandAuthority()])
      .then(([m, t, s, so, bt, c, ca]) => {
        setManifest(m);
        setTopology(t);
        setSources(s);
        setSupervisor(so);
        setBusTap(bt);
        setCampaigns(c.campaigns);
        setCommands(ca);
      })
      .catch((err: Error) => setError(err.message));
  }, []);

  useEffect(() => {
    Promise.all([api.campaign(activeCampaign), api.graphModel(activeCampaign), api.telemetry(activeCampaign), api.evidenceReport(activeCampaign)])
      .then(([campaign, graphModel, telemetry, evidence]) => {
        setCampaigns((existing) => existing.map((item) => item.id === campaign.id ? campaign : item));
        setGraph(graphModel);
        setSamples(telemetry);
        setReport(evidence);
      })
      .catch((err: Error) => setError(err.message));
  }, [activeCampaign]);

  const selectedCampaign = useMemo(() => campaigns.find((campaign) => campaign.id === activeCampaign) ?? campaigns[0], [campaigns, activeCampaign]);

  const refreshCommands = (action: () => Promise<CommandAuthorityState>) => {
    action().then(setCommands).catch((err: Error) => setError(err.message));
  };

  if (error) return <main className="shell"><div className="error">{error}</div></main>;
  if (!manifest || !topology || !sources || !supervisor || !busTap || !selectedCampaign || !graph || !commands || !report) {
    return <main className="shell"><div className="loading">Loading Gossamer demo contracts...</div></main>;
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <h1>Gossamer</h1>
          <p>{manifest.description}</p>
        </div>
        <select value={activeCampaign} onChange={(event) => setActiveCampaign(event.target.value)}>
          {campaigns.map((campaign) => <option key={campaign.id} value={campaign.id}>{campaign.name}</option>)}
        </select>
      </header>
      <nav className="nav">
        {routes.map((item) => {
          const Icon = item.icon;
          return (
            <a key={item.id} href={`#${item.id}`} className={route === item.id ? "active" : ""}>
              <Icon size={16} /> {item.label}
            </a>
          );
        })}
      </nav>
      {route === "landing" && <LandingView manifest={manifest} campaigns={campaigns} supervisor={supervisor} />}
      {route === "mission-map" && <MissionMapView manifest={manifest} topology={topology} campaigns={campaigns} />}
      {route === "supervisor" && <SupervisorView overview={supervisor} />}
      {route === "graph-wall" && <GraphWallView model={graph} samples={samples} />}
      {route === "sources" && <SourceCatalogueView catalogue={sources} />}
      {route === "requirements" && <RequirementMatrixView campaign={selectedCampaign} />}
      {route === "commands" && (
        <CommandAuthorityView
          state={commands}
          onRequest={() => refreshCommands(api.requestLease)}
          onRelease={() => refreshCommands(api.releaseLease)}
          onCommand={() => refreshCommands(api.mockCommand)}
        />
      )}
      {route === "bus-tap" && <BusTapView tap={busTap} />}
      {route === "report" && <EvidenceReportView report={report} />}
    </main>
  );
}

function hashRoute(): Route {
  const candidate = window.location.hash.replace("#", "") as Route;
  return routes.some((route) => route.id === candidate) ? candidate : "landing";
}
