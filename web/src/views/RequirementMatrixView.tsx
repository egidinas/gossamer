import type { Campaign } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { RequirementBadge } from "../components/RequirementBadge";

export function RequirementMatrixView({ campaign }: { campaign: Campaign }) {
  return (
    <OperatorPanel title="Requirement Matrix" meta={campaign.name}>
      <table>
        <thead>
          <tr><th>ID</th><th>Requirement</th><th>Result</th><th>Rationale</th></tr>
        </thead>
        <tbody>
          {campaign.requirements.map((req) => (
            <tr key={req.id}>
              <td>{req.id}</td>
              <td>{req.title}</td>
              <td><RequirementBadge result={req.result} /></td>
              <td>{req.rationale}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </OperatorPanel>
  );
}

