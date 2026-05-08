import { Activity, Archive, CalendarDays, Database, GitBranch, MonitorUp, ShieldCheck } from "lucide-react";
import type { Campaign, Manifest } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";

const experiences = [
  {
    href: "#acceptance",
    title: "Acceptance FAT",
    meta: "thermal chamber as-run",
    kind: "acceptance",
    icon: Activity,
    stats: ["4 cycles", "ambient FT", "DUT/chamber"]
  },
  {
    href: "#command-center-fat",
    title: "Operator Center",
    meta: "four chamber ladder",
    kind: "command",
    icon: CalendarDays,
    stats: ["4 chambers", "4 weeks", "reset markers"]
  },
  {
    href: "#qualification",
    title: "Qualification TVac",
    meta: "thermal-vacuum profile",
    kind: "qualification",
    icon: Activity,
    stats: ["8 cycles", "pressure log", "outgassing"]
  }
];

export function LandingView({ manifest, campaigns }: { manifest: Manifest; campaigns: Campaign[] }) {
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
          <div className="hero-actions">
            <a href="#acceptance">Acceptance FAT →</a>
            <a href="#command-center-fat">Operator Center →</a>
            <a href="#qualification">Qualification TVac →</a>
          </div>
          <div className="landing-data-flow" aria-label="data flow">
            <span><Database size={15} /> Facility/DUT nodes</span>
            <span><CalendarDays size={15} /> Chamber schedule</span>
            <span><GitBranch size={15} /> Tile-backed pool</span>
            <span><MonitorUp size={15} /> Operator display</span>
          </div>
        </div>
      </section>
      <OperatorPanel title="Complete Operator Surfaces" meta="static current bundle">
        <div className="landing-experience-grid">
          {experiences.map((experience) => {
            const Icon = experience.icon;
            return (
              <a className="landing-experience-card" href={experience.href} key={experience.href} data-preview-kind={experience.kind}>
                <div className="landing-preview-screen" aria-hidden="true">
                  <div className="preview-topline">
                    <span />
                    <span />
                    <span />
                  </div>
                  <div className="preview-plot">
                    <i className="preview-gridline" style={{ left: "22%" }} />
                    <i className="preview-gridline" style={{ left: "49%" }} />
                    <i className="preview-gridline" style={{ left: "76%" }} />
                    <b className="preview-trace preview-command" />
                    <b className="preview-trace preview-actual" />
                    <b className="preview-trace preview-dut" />
                    <em className="preview-marker" style={{ left: "68%" }} />
                  </div>
                  <div className="preview-footer">
                    <span />
                    <span />
                    <span />
                  </div>
                </div>
                <div className="landing-experience-copy">
                  <span><Icon size={16} /> {experience.meta}</span>
                  <strong>{experience.title}</strong>
                  <small>{experience.stats.join(" / ")}</small>
                </div>
              </a>
            );
          })}
        </div>
      </OperatorPanel>
      <OperatorPanel title="Environmental-Test Execution Models" meta={manifest.test_article}>
        <div className="metric-grid">
          {campaigns.map((c) => {
            const reqs = c.requirements ?? [];
            const pass = reqs.filter((r) => r.result === "pass").length;
            const anomalies = (c.anomalies ?? []).length;
            const openAnomalies = (c.anomalies ?? []).filter((a) => a.status !== "closed").length;
            return (
              <div key={c.id}>
                <span className="label">{c.name ?? c.id}</span>
                <strong><span style={{ color: "var(--color-pass)" }}>{c.result ?? "—"}</span></strong>
                {reqs.length > 0 && <small>{pass}/{reqs.length} reqs</small>}
                {anomalies > 0 && <small>{openAnomalies > 0 ? `${openAnomalies} open anomaly` : `${anomalies} anomaly closed`}</small>}
              </div>
            );
          })}
          <div><span className="label">Evidence model</span><strong>{manifest.synthetic_only ? "traceable" : "live"}</strong></div>
        </div>
      </OperatorPanel>
      <OperatorPanel title="Underlying System" meta="single static bundle, browser-native graphing">
        <div className="landing-system-grid">
          <div>
            <strong>Build-time simulation</strong>
            <span>Physics and scheduling run before deployment, so expensive generation can produce dense as-run traces, ghost projections, markers, manifests, and campaign shells.</span>
          </div>
          <div>
            <strong>Arrow telemetry payloads</strong>
            <span>The deployed graph pages hydrate from compressed Apache Arrow files and materialize only the requested card/window into browser-native uPlot series.</span>
          </div>
          <div>
            <strong>Shared time contract</strong>
            <span>Acceptance FAT, Operator Center, and Qualification TVac use the same graph shell, tile manifest, marker, band, and source-provenance contracts.</span>
          </div>
          <div>
            <strong>Operator semantics</strong>
            <span>Functional-test gates, ambient verification, reset/breakdown windows, source quality, pressure/outgassing ranges, and requirement progress are explicit data, not UI guesses.</span>
          </div>
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
