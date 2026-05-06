import type { TelemetrySample } from "../types";

export function MiniTrace({ samples, signal }: { samples: TelemetrySample[]; signal: string }) {
  const values = samples.map((sample) => sample.signals[signal]).filter((value) => Number.isFinite(value));
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const points = values
    .map((value, index) => {
      const x = (index / Math.max(1, values.length - 1)) * 100;
      const y = 34 - ((value - min) / range) * 30;
      return `${x},${y}`;
    })
    .join(" ");
  return (
    <svg className="mini-trace" viewBox="0 0 100 36" preserveAspectRatio="none" role="img" aria-label={signal}>
      <polyline points={points} />
    </svg>
  );
}

