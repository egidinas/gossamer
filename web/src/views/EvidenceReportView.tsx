import type { EvidenceReport } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";
import { RequirementBadge } from "../components/RequirementBadge";

export function EvidenceReportView({ report }: { report: EvidenceReport }) {
  const anomalies = report.anomalies ?? [];
  const reproducibility = report.reproducibility ?? [];
  const requirements = report.requirements ?? [];

  return (
    <div className="lane-stack">
      <OperatorPanel title="Evidence Report" meta={report.campaign_id}>
        <p className="summary">{report.summary}</p>
        <StatusBadge value={report.result} />
        {report.simulation_provenance && (
          <p className="disclaimer">
            Simulation {report.simulation_provenance.model} / seed {report.simulation_provenance.seed} / source {report.simulation_provenance.source}
          </p>
        )}
      </OperatorPanel>
      {requirements.length > 0 && (
        <OperatorPanel title="Requirement Outcomes" meta={`${requirements.length} requirements`}>
          <table>
            <thead>
              <tr><th>ID</th><th>Requirement</th><th>Result</th><th>Expression</th></tr>
            </thead>
            <tbody>
              {requirements.map((req) => (
                <tr key={req.id}>
                  <td>{req.id}</td>
                  <td>{req.title}</td>
                  <td><RequirementBadge result={req.result} /></td>
                  <td>{req.expression && <code className="requirement-expression">{req.expression}</code>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </OperatorPanel>
      )}
      <OperatorPanel title="Anomaly Disposition" meta={`${anomalies.length} anomalies`}>
        {anomalies.length === 0 ? <p>No open anomalies.</p> : anomalies.map((anomaly) => (
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
