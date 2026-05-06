import { Lock, Unlock, Send } from "lucide-react";
import type { CommandAuthorityState } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";

export function CommandAuthorityView({
  state,
  onRequest,
  onRelease,
  onCommand
}: {
  state: CommandAuthorityState;
  onRequest: () => void;
  onRelease: () => void;
  onCommand: () => void;
}) {
  return (
    <OperatorPanel title="Command Authority" meta="in-memory demo state">
      <div className="command-grid">
        <div>
          <span className="label">Lease state</span>
          <StatusBadge value={state.lease_state} />
        </div>
        <div>
          <span className="label">Lease owner</span>
          <strong>{state.lease_owner || "none"}</strong>
        </div>
        <div>
          <span className="label">Last command</span>
          <strong>{state.last_command || "none"}</strong>
        </div>
      </div>
      <div className="toolbar">
        <button onClick={onRequest} title="Request lease"><Lock size={16} /> Request</button>
        <button onClick={onRelease} title="Release lease"><Unlock size={16} /> Release</button>
        <button onClick={onCommand} title="Send mock command"><Send size={16} /> Mock command</button>
      </div>
    </OperatorPanel>
  );
}

