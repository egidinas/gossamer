export function StatusBadge({ value }: { value: string }) {
  return <span className={`badge badge-${value}`}>{value.replaceAll("_", " ")}</span>;
}

