import type { SupervisorOverview } from "../types";
import { HeroTrace } from "../components/HeroTrace";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";

export function SupervisorView({ overview }: { overview: SupervisorOverview }) {
  return (
    <div className="supervisor-view">
      <OperatorPanel title="Supervisor Overview" meta={overview.test_article}>
        <p className="summary">{overview.summary}</p>
        <div className="swimlane-board">
          {overview.lanes.map((lane) => (
            <section className="swimlane" key={lane.id}>
              <div className="swimlane-head">
                <div>
                  <h2>{lane.label}</h2>
                  <span>{lane.activity}</span>
                </div>
                <StatusBadge value={lane.result || lane.state} />
              </div>
              <div className="lane-meta-grid">
                <span>{lane.facility}</span>
                <span>{lane.primary_bus}</span>
                <StatusBadge value={lane.source_quality} />
              </div>
              <div className="hero-graph-grid">
                {lane.hero_graphs.map((graph) => (
                  <article className="hero-graph-card" key={`${lane.id}-${graph.id}`}>
                    <div className="series-meta">
                      <strong>{graph.label}</strong>
                      <span>{graph.role} / {graph.source}</span>
                    </div>
                    <HeroTrace points={graph.values} units={graph.units} />
                  </article>
                ))}
              </div>
              <p className="disclaimer">{lane.requirement_summary}</p>
            </section>
          ))}
        </div>
      </OperatorPanel>
    </div>
  );
}
