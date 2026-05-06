import { Activity, GitBranch, Network, ShieldCheck } from "lucide-react";
import type { Campaign, Manifest, SupervisorOverview } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";

export function LandingView({ manifest, campaigns, supervisor }: { manifest: Manifest; campaigns: Campaign[]; supervisor: SupervisorOverview }) {
  const passing = campaigns.filter((campaign) => campaign.result === "pass").length;

  return (
    <div className="landing-grid">
      <section className="landing-hero">
        <div>
          <span className="eyebrow">clean-room public demonstrator</span>
          <h1>Gossamer</h1>
          <p>{manifest.description}</p>
        </div>
        <div className="hero-actions">
          <a href="#supervisor"><Activity size={17} /> Supervisor</a>
          <a href="#bus-tap"><Network size={17} /> Bus Tap</a>
        </div>
      </section>
      <OperatorPanel title="Demo Envelope" meta={manifest.test_article}>
        <div className="metric-grid">
          <div><span className="label">Campaigns</span><strong>{campaigns.length}</strong></div>
          <div><span className="label">Passing</span><strong>{passing}</strong></div>
          <div><span className="label">Supervisor lanes</span><strong>{supervisor.lanes.length}</strong></div>
          <div><span className="label">Synthetic only</span><StatusBadge value={manifest.synthetic_only ? "fresh" : "missing"} /></div>
        </div>
      </OperatorPanel>
      <OperatorPanel title="What It Demonstrates" meta="portfolio artifact">
        <div className="value-grid">
          <div><GitBranch size={18} /><strong>System model</strong><span>Facilities, buses, sources, requirements, and evidence share one contract vocabulary.</span></div>
          <div><Activity size={18} /><strong>Parallel test supervision</strong><span>FAT and qualification activities are visible as backend-defined swimlanes.</span></div>
          <div><Network size={18} /><strong>Virtual bus tap</strong><span>Fictional TM and TC replay shows how node-to-node observability can look without private protocol detail.</span></div>
          <div><ShieldCheck size={18} /><strong>Public safe</strong><span>All names, values, limits, events, and reports are synthetic and deterministic.</span></div>
        </div>
      </OperatorPanel>
    </div>
  );
}
