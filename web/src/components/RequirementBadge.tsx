import { StatusBadge } from "./StatusBadge";

export function RequirementBadge({ result }: { result: string }) {
  return <StatusBadge value={result} />;
}

