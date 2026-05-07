import { useEffect, useState } from "react";
import { AlertCircle, CheckCircle2, Clock3, RotateCw } from "lucide-react";
import { api } from "../api";
import { OperatorGraphWall } from "../components/OperatorGraphWall";
import type { CommandCenterFAT, CommandCenterRun, GraphModel } from "../types";

type Props = {
  model: CommandCenterFAT;
};

export function CommandCenterFATView({ model }: Props) {
  const [graph, setGraph] = useState<GraphModel | null>(null);
  const runs = model.lanes.flatMap((lane) => lane.runs);
  const completed = runs.filter((run) => run.state === "complete").length;
  const running = runs.filter((run) => run.state === "running").length;
  const scheduled = runs.filter((run) => run.state === "scheduled").length;
  const graphCampaignID = model.graph_campaign_id ?? model.id;
  const wall = graph?.graph_wall ?? model.graph_wall;
  const heroGraph = graph?.hero_graph ?? model.hero_graph;

  useEffect(() => {
    let cancelled = false;
    api.graphShell(graphCampaignID)
      .then((nextGraph) => {
        if (!cancelled) setGraph(nextGraph);
      })
      .catch(() => {
        if (!cancelled) setGraph(null);
      });
    return () => {
      cancelled = true;
    };
  }, [graphCampaignID]);

  return (
    <section className="command-center-view">
      <div className="command-center-header">
        <div>
          <span className="eyebrow">standalone FAT command center</span>
          <h2>{model.title}</h2>
          <p>{model.summary}</p>
        </div>
        <div className="command-center-kpis" aria-label="FAT command center summary">
          <KPI label="complete" value={completed} />
          <KPI label="running" value={running} />
          <KPI label="scheduled" value={scheduled} />
          <KPI label="now" value={formatShort(model.now)} />
        </div>
      </div>

      {wall && heroGraph ? (
        <OperatorGraphWall
          campaignId={graphCampaignID}
          wall={wall}
          heroGraph={heroGraph}
          afterProgress={<CommandCenterManifestStrip model={model} />}
        />
      ) : (
        <>
          <div className="loading route-loading">Loading command center graph shell...</div>
          <CommandCenterManifestStrip model={model} />
        </>
      )}
    </section>
  );
}

function KPI({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function CommandCenterManifestStrip({ model }: { model: CommandCenterFAT }) {
  return (
    <div className="command-center-manifest-strip" aria-label="Command center FAT item manifests">
      {model.lanes.map((lane) => (
        <article className="command-center-manifest-lane" key={lane.id}>
          <div className="command-center-manifest-heading">
            <strong>{lane.chamber_name}</strong>
            <span>{lane.facility}</span>
          </div>
          <div className="command-center-run-list">
            {lane.runs.map((run) => (
              <RunManifest key={run.id} run={run} />
            ))}
          </div>
        </article>
      ))}
    </div>
  );
}

function RunManifest({ run }: { run: CommandCenterRun }) {
  return (
    <div className={`command-center-run-manifest ${run.state}`}>
      <div className="command-center-run-status">
        <StateBadge state={run.state} />
        <span>{formatShort(run.start)} to {formatShort(run.end)}</span>
      </div>
      <strong>{run.manifest.article}</strong>
      <span>{run.manifest.serial_number}</span>
      <em>{run.manifest.operator_next}</em>
      <span className="command-center-reset">
        <RotateCw size={12} />
        reset ready {formatShort(run.reset_end)}
      </span>
    </div>
  );
}

function StateBadge({ state }: { state: string }) {
  const Icon = state === "complete" ? CheckCircle2 : state === "running" ? AlertCircle : Clock3;
  return (
    <span className={`fat-state ${state}`}>
      <Icon size={13} />
      {state}
    </span>
  );
}

function formatShort(value: string) {
  return new Date(value).toLocaleString(undefined, { weekday: "short", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}
