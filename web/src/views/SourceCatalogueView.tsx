import type { SourceCatalogue } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";

export function SourceCatalogueView({ catalogue }: { catalogue: SourceCatalogue }) {
  return (
    <OperatorPanel title="Source Catalogue" meta={`${catalogue.sources.length} sources`}>
      <table>
        <thead>
          <tr><th>Source</th><th>Owner</th><th>Bus</th><th>Freshness</th><th>Quality</th><th>Evidence</th></tr>
        </thead>
        <tbody>
          {catalogue.sources.map((source) => (
            <tr key={source.id}>
              <td>{source.label}</td>
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

