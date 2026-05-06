import type { EvidenceReport } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";

export function EvidenceReportView({ report }: { report: EvidenceReport }) {
  const anomalies = report.anomalies ?? [];
  const reproducibility = report.reproducibility ?? [];

  return (
    <div className="lane-stack">
      <OperatorPanel title="Evidence Report" meta={report.campaign_id}>
        <p className="summary">{report.summary}</p>
        <StatusBadge value={report.result} />
        <p className="disclaimer">{report.synthetic_data_note}</p>
      </OperatorPanel>
      <OperatorPanel title="Anomaly Disposition" meta={`${anomalies.length} anomalies`}>
        {anomalies.length === 0 ? <p>No open synthetic anomalies.</p> : anomalies.map((anomaly) => (
          <div className="anomaly" key={anomaly.id}>
            <strong>{anomaly.id}: {anomaly.title}</strong>
            <span>{anomaly.disposition}</span>
            <StatusBadge value={anomaly.status} />
          </div>
        ))}
      </OperatorPanel>
      <OperatorPanel title="Reproducibility" meta="commands">
        <ul>
          {reproducibility.map((item) => <li key={item}><code>{item}</code></li>)}
        </ul>
      </OperatorPanel>
    </div>
  );
}
