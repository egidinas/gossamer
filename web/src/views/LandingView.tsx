import { Activity, Archive, Database, GitBranch, MonitorUp, ShieldCheck } from "lucide-react";
import type { Campaign, Manifest, SupervisorOverview } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";

export function LandingView({ manifest, campaigns, supervisor }: { manifest: Manifest; campaigns: Campaign[]; supervisor: SupervisorOverview }) {
  const previewLanes = supervisor.lanes.filter((lane) => lane.campaign === "thermal_acceptance_fat" || lane.campaign === "tvac_qualification");

  return (
    <div className="landing-grid">
      <section className="landing-hero">
        <div className="landing-hero-copy">
          <span className="eyebrow">environmental-test operating model</span>
          <h1>Gossamer</h1>
          <p>
            Gossamer explores an operating model for environmental testing where facility state, DUT telemetry, command
            counters, infrastructure context, requirements, and evidence move into one backend-owned telemetry pool instead
            of remaining scattered across instruments, logs, and reports.
          </p>
          <p>
            Source provenance, graph semantics, command authority, and evidence links are explicit so a TVac or factory
            acceptance campaign can be followed by engineers and stakeholders, then traced back to reportable proof.
          </p>
          <div className="landing-data-flow" aria-label="data flow">
            <span><Database size={15} /> Facility/DUT nodes</span>
            <span><Archive size={15} /> Legacy imports</span>
            <span><GitBranch size={15} /> Tile-backed pool</span>
            <span><MonitorUp size={15} /> Operator display</span>
          </div>
        </div>
        <div className="hero-actions">
          <a href="#acceptance"><Activity size={17} /> Acceptance FAT</a>
          <a href="#qualification"><Activity size={17} /> Qualification TVac</a>
        </div>
        <div className="landing-preview" aria-label="supervisor preview">
          <div className="landing-preview-head">
            <strong>Test Campaign Snapshot</strong>
            <span>{supervisor.test_article}</span>
          </div>
          {previewLanes.map((lane) => (
            <div className="preview-row" key={lane.id}>
              <span>{lane.label}</span>
              <StatusBadge value={lane.result || lane.state} />
            </div>
          ))}
        </div>
      </section>
      <OperatorPanel title="Shared Telemetry Pool" meta="live, historical, and legacy-translated">
        <div className="landing-architecture">
          <img src="/assets/gossamer/telemetry-architecture.webp" alt="Architecture diagram showing decentralized test nodes feeding a shared telemetry pool and common operator UI" loading="eager" />
          <div>
            <span className="eyebrow">data where it is produced</span>
            <p>
              The central idea is not that environmental test data becomes simple. It is that carefully declared sources,
              translation provenance, tile contracts, and evidence links can make current, historical, and legacy data visible
              through one shared interface.
            </p>
            <p>
              That is why the FAT and TVac pages are presented as campaign artifacts: the same pool can support live operation,
              stakeholder visibility, later exploration, and audit-oriented evidence review.
            </p>
          </div>
        </div>
      </OperatorPanel>
      <OperatorPanel title="Environmental-Test Execution Models" meta={manifest.test_article}>
        <div className="metric-grid">
          <div><span className="label">Acceptance FAT</span><strong>4 cycles</strong></div>
          <div><span className="label">TVac Qualification</span><strong>8 cycles</strong></div>
          <div><span className="label">Current campaigns</span><strong>{campaigns.length}</strong></div>
          <div><span className="label">Evidence model</span><StatusBadge value={manifest.synthetic_only ? "traceable" : "live"} /></div>
        </div>
      </OperatorPanel>
      <OperatorPanel title="What The Interface Makes Inspectable" meta="show the chain from source to evidence">
        <div className="value-grid">
          <div><Database size={18} /><strong>Source-owned data</strong><span>Facility sensors, DUT telemetry, commands, counters, and building infrastructure signals enter the same contract with provenance and quality flags.</span></div>
          <div><Archive size={18} /><strong>Legacy plus live-capable inputs</strong><span>CSV, binary, HDF5-like imports, and live-shaped fixture sources are translated into one tile schema instead of anonymous traces.</span></div>
          <div><GitBranch size={18} /><strong>Tile-backed operator grammar</strong><span>FAT and TVac use the same graph primitives for analog traces, counters, swimlanes, event rails, ghost data, and evidence markers.</span></div>
          <div><ShieldCheck size={18} /><strong>Requirements become evidence</strong><span>Stabilization windows, completed cycles, functional tests, and requirement progress link back to exact markers, signals, and report records.</span></div>
        </div>
      </OperatorPanel>
    </div>
  );
}
