import type { TestItemThermalDiagram, ThermalDiagramNode } from "../types";

type Props = {
  diagram: TestItemThermalDiagram;
  meta?: Array<{ label: string; value: string }>;
};

export function ThermalModelMap({ diagram, meta = [] }: Props) {
  const nodes = new Map(diagram.nodes.map((node) => [node.id, node]));
  const boundaryNodes = diagram.nodes.filter((node) => node.kind === "environment" || node.kind === "interface");
  const componentNodes = diagram.nodes.filter((node) => node.kind === "component");
  const itemNode = diagram.nodes.find((node) => node.kind === "test_item");
  const modifierNodes = diagram.nodes.filter((node) => node.kind !== "environment" && node.kind !== "interface" && node.kind !== "component" && node.kind !== "test_item");

  return (
    <section className="thermal-diagram-panel" aria-label={`${diagram.label} model map`}>
      <div className="thermal-diagram-meta">
        <span>{diagram.context.replaceAll("_", " ")}</span>
        <strong>{diagram.label}</strong>
        <p>{diagram.summary}</p>
        {meta.map((item) => (
          <div className="thermal-diagram-meta-row" key={`${item.label}-${item.value}`}>
            <span>{item.label}</span>
            <strong>{item.value}</strong>
          </div>
        ))}
      </div>
      <div className="thermal-diagram-canvas">
        <div className="thermal-map-boundaries">
          {boundaryNodes.map((node) => <ThermalNode key={node.id} node={node} />)}
          {modifierNodes.map((node) => <ThermalNode key={node.id} node={node} />)}
        </div>
        <div className="thermal-map-item">
          <span>{itemNode?.label ?? "Test item"}</span>
          <div>
            {componentNodes.map((node) => <ThermalNode key={node.id} node={node} />)}
          </div>
        </div>
        <div className="thermal-map-couplings">
          {diagram.links.map((link) => {
            const source = nodes.get(link.source);
            const target = nodes.get(link.target);
            if (!source || !target) return null;
            return (
              <div className={`thermal-coupling-row thermal-link-${link.kind}`} key={link.id}>
                <i style={{ opacity: 0.36 + Math.max(0, Math.min(1, link.strength)) * 0.6 }} />
                <span>{source.label}</span>
                <b>{link.label}</b>
                <span>{target.label}</span>
              </div>
            );
          })}
        </div>
      </div>
      <div className="thermal-diagram-notes">
        {diagram.links.map((link) => (
          <span key={link.id}>
            <i className={`thermal-link-key thermal-link-${link.kind}`} />{link.label}{link.signal ? ` · ${link.signal}` : ""}
          </span>
        ))}
      </div>
    </section>
  );
}

function ThermalNode({ node }: { node: ThermalDiagramNode }) {
  return (
    <div className={`thermal-map-node thermal-node-${node.kind}`} title={node.signal ? `${node.label}: ${node.signal}` : node.label}>
      <strong>{node.label}</strong>
      <span>{node.role.replaceAll("_", " ")}</span>
      {node.signal && <small>{node.signal}</small>}
    </div>
  );
}
