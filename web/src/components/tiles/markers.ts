import type { GraphMarker, GraphTile, TileSeries } from "../../types";
import { commandCenterProjectedSeries } from "./decimation";

export type TimeRange = {
  start: number;
  end: number;
};

export type MarkerLabelRect = {
  x: number;
  y: number;
  width: number;
  height: number;
};

const LABEL_COLLISION_PADDING = 8;

export function markerColor(marker: { role?: string; result?: string; kind?: string }) {
  if (marker.kind === "operator_breakdown") return "rgba(255,112,67,0.98)";
  if (marker.kind === "operator_reset") return "rgba(36,214,255,0.98)";
  if (marker.kind === "operator_reset_ready") return "rgba(146,255,111,0.98)";
  if (marker.role === "interlock" || marker.result === "fail") return "rgba(255,49,95,0.96)";
  if (marker.role === "evidence") return "rgba(176,121,255,0.96)";
  if (marker.kind === "functional_gate") return "rgba(255,176,0,0.98)";
  if (marker.kind === "stability" || marker.kind === "stability_achieved" || marker.result === "pass") return "rgba(0,214,163,0.96)";
  return "rgba(49,214,255,0.95)";
}

export function operatorMarkerLines(marker: GraphMarker, compact = false) {
  const kindLabel = marker.kind === "operator_breakdown"
    ? compact ? "BD" : "BREAKDOWN"
    : marker.kind === "operator_reset"
      ? compact ? "RST" : "RESET"
      : "READY";
  return [kindLabel, formatMarkerDateTime(marker.timestamp, compact)];
}

export function formatMarkerDateTime(value: string, compact = false) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(undefined, {
    weekday: "short",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: !compact
  });
}

export function placeMarkerLabel({
  x,
  y,
  labelWidth,
  labelHeight,
  left,
  top,
  width,
  height,
  placed,
  markerRadius
}: {
  x: number;
  y: number;
  labelWidth: number;
  labelHeight: number;
  left: number;
  top: number;
  width: number;
  height: number;
  placed: MarkerLabelRect[];
  markerRadius: number;
}): MarkerLabelRect | null {
  const preferLeft = x > left + width * 0.68;
  const directions = preferLeft ? [-1, 1] : [1, -1];
  const baseY = Math.max(top + 4, Math.min(top + height - labelHeight - 4, y - labelHeight / 2));
  const gap = labelHeight + LABEL_COLLISION_PADDING;
  const inPlotOffsets = [
    0,
    -1 * gap, 1 * gap,
    -2 * gap, 2 * gap,
    -3 * gap, 3 * gap,
    -4 * gap, 4 * gap,
    -5 * gap, 5 * gap,
    -6 * gap, 6 * gap,
    -7 * gap, 7 * gap,
    -8 * gap, 8 * gap,
    -9 * gap, 9 * gap,
    -10 * gap, 10 * gap
  ];
  const overflowOffsets = [-6 * gap, -7 * gap, -8 * gap, -9 * gap, -10 * gap, 6 * gap, 7 * gap, 8 * gap, 9 * gap, 10 * gap];
  for (const direction of directions) {
    const baseX = direction < 0 ? x - labelWidth - markerRadius - 8 : x + markerRadius + 8;
    const clampedX = Math.max(left + 4, Math.min(left + width - labelWidth - 4, baseX));
    for (const offset of [...inPlotOffsets, ...overflowOffsets]) {
      const rawY = baseY + offset;
      // Allow up to 2 label heights outside the plot box
      const candidate = {
        x: clampedX,
        y: Math.max(top - labelHeight * 2, Math.min(top + height + labelHeight, rawY)),
        width: labelWidth,
        height: labelHeight
      };
      if (!placed.some((other) => rectanglesOverlap(candidate, other))) {
        return candidate;
      }
    }
  }
  return null;
}

export function rectanglesOverlap(a: MarkerLabelRect, b: MarkerLabelRect) {
  return a.x < b.x + b.width + LABEL_COLLISION_PADDING
    && a.x + a.width + LABEL_COLLISION_PADDING > b.x
    && a.y < b.y + b.height + LABEL_COLLISION_PADDING
    && a.y + a.height + LABEL_COLLISION_PADDING > b.y;
}

export function fitCanvasText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number) {
  if (ctx.measureText(text).width <= maxWidth) return text;
  let out = text;
  while (out.length > 3 && ctx.measureText(`${out.slice(0, -1)}...`).width > maxWidth) {
    out = out.slice(0, -1);
  }
  return `${out.slice(0, -1)}...`;
}

export function shortGateLabel(label: string) {
  return label
    .replace(/\s+breakdown\s+start$/i, " BD")
    .replace(/\s+reset\s+start$/i, " RST")
    .replace(/\s+reset\s+ready$/i, " RDY")
    .replace(/^Stable\s+/i, "STBL ")
    .replace(/\s+confirmed$/i, "")
    .replace(/^Cycle\s+/i, "C")
    .replace(/\s+dwell\s+functional\s+test/i, " FT")
    .replace(/\s+functional\s+test/i, " FT")
    .slice(0, 8);
}

export function legendReadouts(tile: GraphTile, visibleSignals: Array<{ id: string; label: string }>, timeMs?: number, currentTimeMs?: number) {
  const readouts = new Map<string, string>();
  if (!timeMs) return readouts;
  const visible = new Set(visibleSignals.map((signal) => signal.id));
  tile.series.forEach((series) => {
    if (!visible.has(series.id)) return;
    if (Number.isFinite(timeMs) && Number.isFinite(currentTimeMs) && (timeMs as number) > (currentTimeMs as number) && series.role !== "ghost" && !commandCenterProjectedSeries(tile, series)) return;
    if (series.spans?.length) {
      const state = stateAt(series, timeMs);
      if (state) readouts.set(series.id, state);
      return;
    }
    const value = rawValueAt(series, timeMs);
    if (value === undefined) return;
    readouts.set(series.id, formatLegendValue(series, value));
  });
  return readouts;
}

export function clampTime(timeMs: number, domain: number[]) {
  if (!Number.isFinite(timeMs) || !domain.length) return timeMs;
  const first = domain[0];
  const last = domain[domain.length - 1];
  if (!Number.isFinite(first) || !Number.isFinite(last)) return timeMs;
  return Math.max(first, Math.min(last, timeMs));
}

export function rawValueAt(series: TileSeries, timeMs: number) {
  const points = [...(series.points ?? [])]
    .map((point) => ({ t: Date.parse(point.timestamp), v: point.value }))
    .filter((point) => Number.isFinite(point.t) && Number.isFinite(point.v))
    .sort((a, b) => a.t - b.t);
  if (!points.length) return undefined;
  if (timeMs <= points[0].t) return points[0].v;
  if (timeMs >= points[points.length - 1].t) return points[points.length - 1].v;
  let cursor = 0;
  while (cursor + 1 < points.length && points[cursor + 1].t <= timeMs) cursor += 1;
  const current = points[cursor];
  const next = points[Math.min(cursor + 1, points.length - 1)];
  if (series.step || series.render_kind === "counter" || series.kind === "counter" || next.t === current.t) return current.v;
  const ratio = (timeMs - current.t) / (next.t - current.t);
  return current.v + (next.v - current.v) * Math.max(0, Math.min(1, ratio));
}

export function stateAt(series: TileSeries, timeMs: number) {
  const span = series.spans?.find((candidate) => {
    const start = Date.parse(candidate.start);
    const end = Date.parse(candidate.end);
    return Number.isFinite(start) && Number.isFinite(end) && timeMs >= start && timeMs <= end;
  });
  return span?.label ?? span?.state ?? (span?.value !== undefined ? String(span.value) : undefined);
}

export function formatLegendValue(series: TileSeries, value: number) {
  const unit = series.unit || unitForAxis(series.axis_id);
  if (series.axis_id === "pressure_mbar") return `${formatPressure(value)} mbar`;
  if (series.axis_id === "pressure_rate") return `${formatScientific(value)} mbar/min`;
  if (series.axis_id === "counter") {
    const u = series.unit || series.units || "";
    return u ? `${Math.round(value).toLocaleString()} ${u}` : Math.round(value).toLocaleString();
  }
  if (series.axis_id === "percent") return `${value.toFixed(0)}%`;
  if (unit === "degC") return `${value.toFixed(1)} degC`;
  if (unit === "W") return `${value.toFixed(1)} W`;
  if (unit === "ms") return `${value.toFixed(1)} ms`;
  if (unit === "bar") return `${value.toFixed(2)} bar`;
  return `${Number.isInteger(value) ? value.toFixed(0) : value.toFixed(2)}${unit ? ` ${unit}` : ""}`;
}

export function formatScientific(value: number) {
  if (!Number.isFinite(value)) return "";
  if (value === 0) return "0";
  return value.toExponential(2).replace("e", "E");
}

export function formatPressure(value: number) {
  if (value <= 0) return "0";
  if (value < 0.001 || value >= 1000) return value.toExponential(2).replace("e", "E");
  if (value < 1) return value.toPrecision(3);
  return value.toFixed(value < 10 ? 2 : 1);
}

export function unitForAxis(axisID?: string) {
  if (axisID === "temperature_c") return "degC";
  if (axisID === "pressure_mbar") return "mbar";
  if (axisID === "power_w" || axisID === "heat_flux_w") return "W";
  if (axisID === "bus_ms") return "ms";
  if (axisID === "pressure_bar") return "bar";
  if (axisID === "pressure_rate") return "mbar/min";
  if (axisID === "percent") return "%";
  if (axisID === "voltage_v") return "V";
  if (axisID === "current_a") return "A";
  if (axisID === "rf_db" || axisID === "link_db" || axisID === "signal_db") return "dB";
  if (axisID === "frequency_hz") return "Hz";
  if (axisID === "ohm") return "Ω";
  return "";
}
