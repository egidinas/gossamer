import type { Campaign, GraphModel, SupervisorOverview } from "../types";
import { HeroTrace } from "../components/HeroTrace";
import { OperatorGraphWall } from "../components/OperatorGraphWall";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";
import { ThermalModelMap } from "../components/ThermalModelMap";

type Props = {
  overview: SupervisorOverview;
  campaign: Campaign;
  graph: GraphModel;
};

export function SupervisorView({ overview, campaign, graph }: Props) {
  if (graph.thermal_program) {
    const matchingLane = overview.lanes.find((lane) => lane.campaign === campaign.id);
    const campaignText = campaign.id === "tvac_qualification"
      ? "Platen, shroud, pressure, DUT temperatures, functional tests, and evidence share one time base."
      : "Chamber air, interface temperature, DUT sensors, infrastructure, functional tests, and evidence share one time base.";
    const modelMeta = [
      { label: "Facility", value: campaign.facility },
      { label: "Program", value: `${graph.thermal_program.cycle_count} cycles` },
      { label: "Result", value: campaign.result },
      { label: "Source quality", value: matchingLane?.source_quality ?? "fresh" },
    ];
    const campaignContext = (
      <div className="campaign-context-strip">
        {graph.hero_graph?.thermal_diagram && <ThermalModelMap diagram={graph.hero_graph.thermal_diagram} meta={modelMeta} />}
        <p>{campaignText}</p>
      </div>
    );
    return (
      <div className="supervisor-view single-chamber-supervisor">
        <OperatorPanel title={campaign.name} meta={`${graph.thermal_program.cycle_count} cycles · ${campaign.facility}`}>
          {graph.graph_wall && graph.hero_graph && (
            <OperatorGraphWall campaignId={campaign.id} wall={graph.graph_wall} heroGraph={graph.hero_graph} afterProgress={campaignContext} />
          )}
        </OperatorPanel>
      </div>
    );
  }

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
                <div><span className="label">Facility</span><strong>{lane.facility}</strong></div>
                <div><span className="label">Primary bus</span><strong>{lane.primary_bus}</strong></div>
                <div><span className="label">Source quality</span><StatusBadge value={lane.source_quality} /></div>
                {lane.thermal_program && (
                  <div><span className="label">Program</span><strong>{lane.thermal_program.cycle_count} cycles</strong></div>
                )}
              </div>
              {lane.thermal_program && (
                <div className="lane-program-summary">
                  <strong>{lane.thermal_program.label}</strong>
                  <span>{lane.thermal_program.cycle_count} cycles</span>
                </div>
              )}
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
              {(lane.functional_gates || lane.evidence_markers) && (
                <div className="gate-strip">
                  {(lane.functional_gates || []).map((gate) => (
                    <div key={gate.id}>
                      <span className="label">{gate.label}</span>
                      <StatusBadge value={gate.result} />
                    </div>
                  ))}
                </div>
              )}
              <p className="disclaimer">{lane.requirement_summary}</p>
            </section>
          ))}
        </div>
      </OperatorPanel>
    </div>
  );
}
