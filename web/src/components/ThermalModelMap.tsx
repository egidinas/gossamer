import type { CSSProperties } from "react";
import type { TestItemThermalDiagram, ThermalDiagramLink, ThermalDiagramNode } from "../types";

type Props = {
  diagram: TestItemThermalDiagram;
  meta?: Array<{ label: string; value: string }>;
};

export function ThermalModelMap({ diagram, meta = [] }: Props) {
  const nodes = new Map(diagram.nodes.map((node) => [node.id, node]));

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
        <div className="thermal-diagram-links">
          {diagram.links.map((link) => {
            const source = nodes.get(link.source);
            const target = nodes.get(link.target);
            if (!source || !target) return null;
            return <ThermalLink key={link.id} link={link} source={source} target={target} />;
          })}
        </div>
        {diagram.nodes.map((node) => (
          <div
            className={`thermal-diagram-node thermal-node-${node.kind}`}
            key={node.id}
            style={{ left: `${node.x}%`, top: `${node.y}%` }}
            title={node.signal ? `${node.label}: ${node.signal}` : node.label}
          >
            <strong>{node.label}</strong>
            <span>{node.role.replaceAll("_", " ")}</span>
            {node.signal && <small>{node.signal}</small>}
          </div>
        ))}
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

function ThermalLink({ link, source, target }: { link: ThermalDiagramLink; source: ThermalDiagramNode; target: ThermalDiagramNode }) {
  const dx = target.x - source.x;
  const dy = target.y - source.y;
  const length = Math.sqrt(dx * dx + dy * dy);
  const angle = Math.atan2(dy, dx) * (180 / Math.PI);
  const style = {
    left: `${source.x}%`,
    top: `${source.y}%`,
    width: `${length}%`,
    transform: `rotate(${angle}deg)`,
    "--link-strength": `${Math.max(1, Math.min(5, link.strength * 5))}px`,
  } as CSSProperties;

  return (
    <i
      className={`thermal-diagram-link thermal-link-${link.kind}`}
      style={style}
      title={link.signal ? `${link.label}: ${link.signal}` : link.label}
    />
  );
}
