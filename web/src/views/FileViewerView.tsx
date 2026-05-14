import { useState, useEffect } from "react";
import type { FileViewModel, GraphLane } from "../types";
import { api } from "../api";
import { OperatorPanel } from "../components/OperatorPanel";

const CAMPAIGN_IDS = [
  { id: "flatsat_derisking", label: "Flatsat Derisking" },
  { id: "thermal_acceptance_fat", label: "Thermal Chamber FAT" },
  { id: "tvac_qualification", label: "TVac Qualification" },
  { id: "integrated_system_fat", label: "Integrated System FAT" },
];

function fileKindLabel(kind: string): string {
  return { thermal_fat: "Thermal FAT", tvac_qualification: "TVac Qualification", ambient_fat: "Ambient FAT" }[kind] ?? kind;
}

function LaneSummary({ lane }: { lane: GraphLane }) {
  const nodes = [...new Set(lane.series.map((s) => s.node_id).filter(Boolean))];
  const sources = [...new Set(lane.series.map((s) => s.source))];
  return (
    <div className="file-viewer-lane">
      <div className="file-viewer-lane-header">
        <strong>{lane.label}</strong>
        <span className="file-viewer-lane-meta">
          {lane.series.length} signals
          {nodes.length > 0 && <> · {nodes.map((n) => <code key={n} className="requirement-expression">{n}</code>)}</>}
        </span>
      </div>
      <div className="file-viewer-table-scroll">
        <table className="file-viewer-lane-table">
          <thead>
            <tr><th>Signal</th><th>Role</th><th>Units</th><th>Source</th><th>Node</th><th>Range</th></tr>
          </thead>
          <tbody>
            {lane.series.map((s) => (
              <tr key={s.id}>
                <td><code className="requirement-expression">{s.id}</code></td>
                <td>{s.role}</td>
                <td>{s.units}</td>
                <td>{s.source}</td>
                <td>{s.node_id && <code className="requirement-expression">{s.node_id}</code>}</td>
                <td>{s.min} – {s.max}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {sources.length > 0 && (
        <div className="file-viewer-lane-sources">
          {sources.map((src) => <span key={src} className="file-viewer-source-chip">{src}</span>)}
        </div>
      )}
    </div>
  );
}

export function FileViewerView() {
  const [selectedId, setSelectedId] = useState(CAMPAIGN_IDS[0].id);
  const [model, setModel] = useState<FileViewModel | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    setError(null);
    api.fileViewer(selectedId)
      .then((m) => { setModel(m); setLoading(false); })
      .catch((e) => { setError(String(e)); setLoading(false); });
  }, [selectedId]);

  return (
    <div className="lane-stack">
      <OperatorPanel title="File Viewer" meta="campaign telemetry file inspector">
        <div className="file-viewer-selector">
          <label htmlFor="file-viewer-select">Campaign file:</label>
          <select
            id="file-viewer-select"
            value={selectedId}
            onChange={(e) => setSelectedId(e.target.value)}
            className="file-viewer-select"
          >
            {CAMPAIGN_IDS.map(({ id, label }) => (
              <option key={id} value={id}>{label}</option>
            ))}
          </select>
        </div>
        {loading && <p className="disclaimer">Loading…</p>}
        {error && <p className="disclaimer">{error}</p>}
        {model && !loading && (
          <div className="file-viewer-meta-grid">
            <div><span className="file-viewer-meta-key">File</span><code className="requirement-expression">{model.file_ref}</code></div>
            <div><span className="file-viewer-meta-key">Kind</span><span>{fileKindLabel(model.file_kind)}</span></div>
            <div><span className="file-viewer-meta-key">From</span><span>{model.time_start}</span></div>
            <div><span className="file-viewer-meta-key">To</span><span>{model.time_end}</span></div>
            <div><span className="file-viewer-meta-key">Nodes</span>
              <span>{[...new Set(model.signal_groups.map((g) => g.node_label))].join(", ")}</span>
            </div>
            <div><span className="file-viewer-meta-key">Sources</span><span>{model.signal_groups.length}</span></div>
          </div>
        )}
      </OperatorPanel>

      {model && !loading && (
        <>
          <OperatorPanel title="Signal Groups by Node" meta={`${model.signal_groups.length} sources`}>
            <div className="file-viewer-table-scroll">
              <table>
                <thead>
                  <tr><th>Node</th><th>Source</th><th>Bus</th><th>Signals</th></tr>
                </thead>
                <tbody>
                  {model.signal_groups.map((g) => (
                    <tr key={g.source_id}>
                      <td><code className="requirement-expression">{g.node_label}</code></td>
                      <td>{g.source_label}</td>
                      <td>{g.bus}</td>
                      <td>{g.series.length}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </OperatorPanel>

          <OperatorPanel title="Graph Lanes" meta={`${model.lanes.length} lanes`}>
            <div className="file-viewer-lanes">
              {model.lanes.map((lane) => (
                <LaneSummary key={lane.id} lane={lane} />
              ))}
            </div>
          </OperatorPanel>
        </>
      )}
    </div>
  );
}
