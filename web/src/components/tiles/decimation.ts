import type { GraphTile, TileSeries } from "../../types";

export function viewportSeries(tile: GraphTile, series: TileSeries, viewportWidth: number): TileSeries {
  const points = series.points ?? [];
  if (points.length < 4 || series.step || series.render_kind === "counter" || series.kind === "counter") return series;
  const budget = Math.max(180, Math.min(points.length, Math.round(viewportWidth * 1.65)));
  if (points.length <= budget) return series;
  return { ...series, points: lttb(points, budget, (value) => decimationValue(tile, series, value)) };
}

export function lttb(points: TileSeries["points"], threshold: number, yValue: (value: number) => number): TileSeries["points"] {
  if (!points || threshold >= points.length || threshold < 3) return points;
  const parsed = points
    .map((point) => ({ point, x: Date.parse(point.timestamp), y: yValue(point.value) }))
    .filter((point) => Number.isFinite(point.x) && Number.isFinite(point.y));
  if (parsed.length <= threshold) return points;
  const sampled = [parsed[0].point];
  const bucketSize = (parsed.length - 2) / (threshold - 2);
  let a = 0;
  for (let i = 0; i < threshold - 2; i++) {
    const rangeStart = Math.floor((i + 0) * bucketSize) + 1;
    const rangeEnd = Math.floor((i + 1) * bucketSize) + 1;
    const nextRangeStart = Math.floor((i + 1) * bucketSize) + 1;
    const nextRangeEnd = Math.floor((i + 2) * bucketSize) + 1;
    const range = parsed.slice(rangeStart, Math.min(rangeEnd, parsed.length - 1));
    const nextRange = parsed.slice(nextRangeStart, Math.min(nextRangeEnd, parsed.length));
    const avgX = nextRange.reduce((sum, point) => sum + point.x, 0) / Math.max(1, nextRange.length);
    const avgY = nextRange.reduce((sum, point) => sum + point.y, 0) / Math.max(1, nextRange.length);
    const anchor = parsed[a];
    let selected = range[0] ?? parsed[Math.min(rangeStart, parsed.length - 2)];
    let maxArea = -1;
    range.forEach((candidate) => {
      const area = Math.abs((anchor.x - avgX) * (candidate.y - anchor.y) - (anchor.x - candidate.x) * (avgY - anchor.y));
      if (area > maxArea) {
        maxArea = area;
        selected = candidate;
      }
    });
    sampled.push(selected.point);
    a = parsed.indexOf(selected);
  }
  sampled.push(parsed[parsed.length - 1].point);
  return sampled;
}

export function decimationValue(tile: GraphTile, series: TileSeries, value: number) {
  if (series.axis_id === "pressure_mbar" && tile.card_id === "thermal_program") return pressureHeroRailDegC(value);
  if (series.axis_id === "pressure_mbar" || series.axis_id === "pressure_rate") return value > 0 ? Math.log10(value) : Number.NaN;
  return value;
}

function pressureHeroRailDegC(mbar: number) {
  const minLog = Math.log10(0.00000001);
  const maxLog = Math.log10(1013.25);
  const ratio = (Math.log10(Math.max(0.00000001, Math.min(1013.25, mbar))) - minLog) / (maxLog - minLog);
  return -82 + ratio * 104;
}

export function resampleSeries(tile: GraphTile, series: TileSeries, xValues: number[], currentTimeMs?: number): Array<number | null> {
  const points = [...(series.points ?? [])]
    .map((point) => ({ t: Date.parse(point.timestamp), v: displayValue(tile, series, point.value) }))
    .filter((point) => Number.isFinite(point.t) && Number.isFinite(point.v))
    .sort((a, b) => a.t - b.t);
  if (!points.length) return xValues.map(() => null);

  const stepped = series.step || series.render_kind === "counter" || series.kind === "counter" || series.render_kind === "swimlane";
  const isFutureVisible = series.role === "ghost" || commandCenterProjectedSeries(tile, series);
  const gapThreshold = commandCenterTraceGapMs(tile, series);
  let cursor = 0;
  return xValues.map((x) => {
    if (Number.isFinite(currentTimeMs) && x > (currentTimeMs as number) && !isFutureVisible) return null;
    while (cursor + 1 < points.length && points[cursor + 1].t <= x) cursor += 1;
    const current = points[cursor];
    const next = points[Math.min(cursor + 1, points.length - 1)];
    if (x < points[0].t || x > points[points.length - 1].t) return null;
    if (gapThreshold > 0 && next.t - current.t > gapThreshold && x > current.t && x < next.t) return null;
    if (stepped || next.t === current.t) return current.v;
    const ratio = (x - current.t) / (next.t - current.t);
    return current.v + (next.v - current.v) * Math.max(0, Math.min(1, ratio));
  });
}

export function commandCenterGapBreaks(tile: GraphTile, series: TileSeries) {
  const gapThreshold = commandCenterTraceGapMs(tile, series);
  if (gapThreshold <= 0) return [];
  const points = [...(series.points ?? [])]
    .map((point) => Date.parse(point.timestamp))
    .filter(Number.isFinite)
    .sort((a, b) => a - b);
  const breaks: number[] = [];
  for (let i = 1; i < points.length; i += 1) {
    if (points[i] - points[i - 1] > gapThreshold) {
      breaks.push(points[i - 1] + 1, points[i] - 1);
    }
  }
  return breaks;
}

export function commandCenterTraceGapMs(tile: GraphTile, series: TileSeries) {
  if (tile.campaign_id !== "command_center_fat") return 0;
  if (series.render_kind === "swimlane" || series.kind === "counter" || series.role === "event") return 0;
  return 2 * 60 * 60 * 1000;
}

export function commandCenterProjectedSeries(tile: GraphTile, series: TileSeries) {
  return tile.campaign_id === "command_center_fat" && (series.role === "actual" || series.role === "command");
}

// displayValue is used internally by resampleSeries; also re-exported for uPlotAdapter
export function displayValue(tile: GraphTile, series: TileSeries, value: number) {
  if (series.axis_id === "pressure_mbar" && tile.card_id === "thermal_program") return pressureHeroRailDegC(value);
  if (series.axis_id === "pressure_mbar" || series.axis_id === "pressure_rate") return value > 0 ? value : Number.NaN;
  return value;
}
