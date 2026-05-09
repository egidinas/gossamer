import { useState } from "react";
import type { Source, SourceCatalogue, SourceDiscoveryNode, SourceTreeConfig } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";

function nodeLabel(nodeId: string): string {
  const labels: Record<string, string> = {
    reference_dut: "Reference DUT",
    thermal_chamber_a: "Chamber Alpha PLC",
    thermal_chamber_b: "Chamber Bravo PLC",
    thermal_chamber_c: "Chamber Charlie PLC",
    thermal_chamber_d: "Chamber Delta PLC",
    thermal_supervisor_pc: "Thermal Supervisor PC",
    tvac_chamber_q1: "TVac Chamber Q1",
    tvac_plc_q1: "TVac PLC Q1",
    tvac_computer_1: "TVac Computer 1",
    tvac_computer_2: "TVac Computer 2",
    flatsat_rack_a: "Flatsat Rack A",
    house_plc: "House Control PLC",
    archive_node_a: "Archive Node A",
    nas_a: "NAS A",
    librarian_a: "Librarian",
    gateway_a: "Gateway A",
    supervisor_a: "Supervisor A",
  };
  return labels[nodeId] ?? nodeId;
}

function compactLabel(value: string): string {
  return value.replaceAll("_", " ");
}

function SourceBadges({ source }: { source: Source }) {
  return (
    <>
      <StatusBadge value={source.owner_mode} />
      <StatusBadge value={source.format_preference} />
      <StatusBadge value={source.evidence_suitability} />
    </>
  );
}

function DiscoveryNode({
  node,
  sources,
  activeSourceIDs,
}: {
  node: SourceDiscoveryNode;
  sources: Map<string, Source>;
  activeSourceIDs: Set<string> | null;
}) {
  const source = node.source_id ? sources.get(node.source_id) : undefined;
  const children = node.children ?? [];

  const isActive = activeSourceIDs === null || (node.source_id ? activeSourceIDs.has(node.source_id) : childrenHaveActive(children, activeSourceIDs));

  if (activeSourceIDs !== null && !isActive) return null;

  if (node.kind === "stream" && source) {
    return (
      <li>
        <div className="source-tree-row">
          <span>{node.label}</span>
          <code className="requirement-expression">{source.id}</code>
          <SourceBadges source={source} />
        </div>
      </li>
    );
  }
  return (
    <li>
      <details open={node.kind === "node" || activeSourceIDs !== null}>
        <summary>
          <span>{node.label}</span>
          <code className="requirement-expression">{compactLabel(node.kind)}</code>
        </summary>
        <ul>
          {children.map((child) => (
            <DiscoveryNode key={`${node.id}:${child.id}`} node={child} sources={sources} activeSourceIDs={activeSourceIDs} />
          ))}
        </ul>
      </details>
    </li>
  );
}

function childrenHaveActive(nodes: SourceDiscoveryNode[], active: Set<string>): boolean {
  for (const n of nodes) {
    if (n.source_id && active.has(n.source_id)) return true;
    if (n.children && childrenHaveActive(n.children, active)) return true;
  }
  return false;
}

export function SourceCatalogueView({
  catalogue,
  treeConfig,
}: {
  catalogue: SourceCatalogue;
  treeConfig?: SourceTreeConfig;
}) {
  const [activeView, setActiveView] = useState<string | null>(null);
  const sources = new Map(catalogue.sources.map((source) => [source.id, source]));

  const activeView$ = treeConfig?.views.find((v) => v.id === activeView) ?? null;
  const activeSourceIDs: Set<string> | null = activeView$ ? new Set(activeView$.source_ids) : null;
  // ordered source list: follow config order when a view is active (so infra listed last stays last)
  const orderedSources = activeView$
    ? activeView$.source_ids.flatMap((id) => { const s = sources.get(id); return s ? [s] : []; })
    : catalogue.sources;

  const activeCount = orderedSources.length;

  return (
    <OperatorPanel title="Source Catalogue" meta={`${activeCount} / ${catalogue.sources.length} sources`}>
      {treeConfig && (
        <div className="source-tree-view-selector">
          <span className="source-tree-view-label">View:</span>
          <button
            className={`source-tree-view-btn${activeView === null ? " active" : ""}`}
            onClick={() => setActiveView(null)}
          >
            All
          </button>
          {treeConfig.views.map((view) => (
            <button
              key={view.id}
              className={`source-tree-view-btn${activeView === view.id ? " active" : ""}`}
              onClick={() => setActiveView(activeView === view.id ? null : view.id)}
            >
              {view.label}
            </button>
          ))}
        </div>
      )}
      <div className="source-tree">
        <ul>
          {catalogue.tree.map((node) => (
            <DiscoveryNode key={node.id} node={node} sources={sources} activeSourceIDs={activeSourceIDs} />
          ))}
        </ul>
      </div>
      <table>
        <thead>
          <tr>
            <th>Source</th><th>Node</th><th>Served by</th><th>Owner</th>
            <th>Owner mode</th><th>Use</th><th>Format</th><th>Bus</th>
            <th>Freshness</th><th>Quality</th><th>Evidence</th>
            <th>Sensor type</th><th>Uncertainty</th><th>Last cal.</th><th>Cal. ref.</th>
          </tr>
        </thead>
        <tbody>
          {orderedSources.map((source) => (
              <tr key={source.id}>
                <td>{source.label}</td>
                <td><code className="requirement-expression">{nodeLabel(source.node_id)}</code></td>
                <td><code className="requirement-expression">{nodeLabel(source.served_by)}</code></td>
                <td>{source.owner}</td>
                <td><StatusBadge value={source.owner_mode} /></td>
                <td><StatusBadge value={source.use} /></td>
                <td><StatusBadge value={source.format_preference} /></td>
                <td>{source.bus}</td>
                <td>{source.freshness_ms} ms</td>
                <td><StatusBadge value={source.quality} /></td>
                <td>{source.evidence_suitability}</td>
                <td>{source.sensor_type ?? "—"}</td>
                <td>{source.uncertainty_pct != null ? `±${source.uncertainty_pct} %` : "—"}</td>
                <td>{source.last_calibration ?? "—"}</td>
                <td>{source.calibration_reference ? <code className="requirement-expression">{source.calibration_reference}</code> : "—"}</td>
              </tr>
            ))}
        </tbody>
      </table>
    </OperatorPanel>
  );
}
