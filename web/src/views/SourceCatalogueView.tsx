import type { SourceCatalogue } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";

function nodeLabel(nodeId: string): string {
  const labels: Record<string, string> = {
    reference_dut: "DUT",
    thermal_chamber_a: "Chamber A PLC",
    thermal_chamber_b: "Chamber B PLC",
    thermal_chamber_c: "Chamber C PLC",
    thermal_chamber_d: "Chamber D PLC",
    thermal_supervisor_pc: "Thermal Supervisor PC",
    tvac_chamber_q1: "TVac Chamber Q1",
    tvac_plc_q1: "TVac PLC Q1",
    tvac_computer_1: "TVac Computer 1",
    tvac_computer_2: "TVac Computer 2",
    flatsat_rack_a: "Flatsat A",
    house_plc: "House PLC",
    archive_node_a: "Archive",
    nas_a: "NAS A",
    librarian_a: "Librarian",
    gateway_a: "Gateway",
    supervisor_a: "Supervisor",
  };
  return labels[nodeId] ?? nodeId;
}

export function SourceCatalogueView({ catalogue }: { catalogue: SourceCatalogue }) {
  return (
    <OperatorPanel title="Source Catalogue" meta={`${catalogue.sources.length} sources`}>
      <table>
        <thead>
          <tr><th>Source</th><th>Node (origin)</th><th>Served by</th><th>Owner</th><th>Bus</th><th>Freshness</th><th>Quality</th><th>Evidence</th></tr>
        </thead>
        <tbody>
          {catalogue.sources.map((source) => (
            <tr key={source.id}>
              <td>{source.label}</td>
              <td><code className="requirement-expression">{nodeLabel(source.node_id)}</code></td>
              <td><code className="requirement-expression">{nodeLabel(source.served_by)}</code></td>
              <td>{source.owner}</td>
              <td>{source.bus}</td>
              <td>{source.freshness_ms} ms</td>
              <td><StatusBadge value={source.quality} /></td>
              <td>{source.evidence_suitability}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </OperatorPanel>
  );
}
