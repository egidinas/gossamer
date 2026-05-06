import type { Campaign, Manifest, Topology } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";

export function MissionMapView({ manifest, topology, campaigns }: { manifest: Manifest; topology: Topology; campaigns: Campaign[] }) {
  return (
    <div className="view-grid">
      <OperatorPanel title="Mission Map" meta={manifest.test_article}>
        <div className="node-grid">
          {topology.nodes.map((node) => (
            <div className="node-card" key={node.id}>
              <strong>{node.label}</strong>
              <span>{node.kind}</span>
              <StatusBadge value={node.quality} />
            </div>
          ))}
        </div>
      </OperatorPanel>
      <OperatorPanel title="Campaign State" meta={`${campaigns.length} campaigns`}>
        <table>
          <tbody>
            {campaigns.map((campaign) => (
              <tr key={campaign.id}>
                <td>{campaign.name}</td>
                <td>{campaign.facility}</td>
                <td><StatusBadge value={campaign.result} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </OperatorPanel>
    </div>
  );
}

