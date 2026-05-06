import type { GraphModel, TelemetrySample } from "../types";
import { MiniTrace } from "../components/MiniTrace";
import { OperatorPanel } from "../components/OperatorPanel";

export function GraphWallView({ model, samples }: { model: GraphModel; samples: TelemetrySample[] }) {
  return (
    <div className="lane-stack">
      {model.lanes.map((lane) => (
        <OperatorPanel key={lane.id} title={lane.label} meta={model.campaign_id}>
          <div className="series-grid">
            {lane.series.map((series) => (
              <div className="series-card" key={series.id}>
                <div className="series-meta">
                  <strong>{series.label}</strong>
                  <span>{series.role} / {series.units}</span>
                </div>
                <MiniTrace samples={samples} signal={series.id} />
              </div>
            ))}
          </div>
        </OperatorPanel>
      ))}
    </div>
  );
}

