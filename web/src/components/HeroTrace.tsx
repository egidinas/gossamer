import type { GraphPoint } from "../types";

export function HeroTrace({ points, units }: { points: GraphPoint[]; units: string }) {
  const values = points.map((point) => point.value).filter((value) => Number.isFinite(value));
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const path = values
    .map((value, index) => {
      const x = (index / Math.max(1, values.length - 1)) * 100;
      const y = 42 - ((value - min) / range) * 38;
      return `${x},${y}`;
    })
    .join(" ");
  const latest = values.at(-1) ?? 0;

  return (
    <div className="hero-trace-wrap">
      <svg className="hero-trace" viewBox="0 0 100 46" preserveAspectRatio="none" role="img" aria-label={`hero trace ${units}`}>
        <polyline points={path} />
      </svg>
      <strong>{latest.toFixed(units === "count" ? 0 : 1)} {units}</strong>
    </div>
  );
}
