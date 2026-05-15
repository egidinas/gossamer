import uPlot from "uplot";
import type { GraphMarker, GraphTile, HeroGraphModel, TileSeries } from "../../types";
import { colorForSignal, signalPriority } from "./visualPolicy";
import { timeTicks } from "./timeAxis";
import { viewportSeries, commandCenterGapBreaks, resampleSeries, decimationValue, commandCenterProjectedSeries, displayValue } from "./decimation";
import { markerColor, operatorMarkerLines, placeMarkerLabel, rectanglesOverlap, fitCanvasText, shortGateLabel, rawValueAt } from "./markers";

export type TimeRange = {
  start: number;
  end: number;
};

export type UPlotBuild = {
  data: uPlot.AlignedData;
  series: uPlot.Series[];
  scales: Record<string, uPlot.Scale>;
  axes: uPlot.Axis[];
};

const DAY_MS = 86_400_000;

export function uplotData(tile: GraphTile, currentTimeMs?: number, viewportWidth = 900): UPlotBuild {
  const tileSeries = tile.series.filter((series) => (series.points ?? []).length > 0).sort(seriesDrawOrder);
  const plottedSeries = tileSeries.map((series) => viewportSeries(tile, series, viewportWidth));
  const xValues = sharedTimeGrid(tile, plottedSeries);
  const data: uPlot.AlignedData = [xValues];
  const series: uPlot.Series[] = [{}];
  const scaleKeys = new Set<string>();
  plottedSeries.forEach((seriesTile, index) => {
    const scale = scaleForSeries(tile, seriesTile);
    scaleKeys.add(scale);
    data.push(resampleSeries(tile, seriesTile, xValues, currentTimeMs));
    series.push({
      label: seriesTile.label,
      scale,
      stroke: colorForSignal(seriesTile, index),
      width: lineWidthFor(seriesTile.role),
      dash: seriesTile.role === "ghost" ? [7, 4] : seriesTile.role === "acceptance_band" ? [2, 5] : undefined,
      points: { show: false }
    });
  });
  return { data, series, scales: buildScales(scaleKeys), axes: buildAxes(scaleKeys, tile) };
}

export function seriesDrawOrder(a: TileSeries, b: TileSeries) {
  const order: Record<string, number> = {
    ghost: 5,
    acceptance_band: 8,
    actual: 10,
    source_quality: 12,
    counter: 14,
    command: 45,
    event: 50,
    interlock: 55,
    evidence: 60,
  };
  const roleDelta = (order[a.role] ?? 15) - (order[b.role] ?? 15);
  if (roleDelta) return roleDelta;
  return signalPriority(a) - signalPriority(b);
}

export function lineWidthFor(role: string) {
  if (role === "command") return 1.55;
  if (role === "ghost") return 0.9;
  if (role === "acceptance_band") return 0.75;
  if (role === "counter" || role === "source_quality") return 1.05;
  return 0.85;
}

export function sharedTimeGrid(tile: GraphTile, tileSeries: TileSeries[]): number[] {
  const start = Date.parse(tile.t0);
  const end = Date.parse(tile.t1);
  const finiteTimes = tileSeries
    .flatMap((series) => (series.points ?? []).map((point) => Date.parse(point.timestamp)))
    .filter(Number.isFinite);
  const gapTimes = tileSeries.flatMap((series) => commandCenterGapBreaks(tile, series));
  const t0 = Number.isFinite(start) ? start : Math.min(...finiteTimes);
  const t1 = Number.isFinite(end) ? end : Math.max(...finiteTimes);
  if (!Number.isFinite(t0) || !Number.isFinite(t1) || t1 <= t0) {
    return Array.from(new Set([...finiteTimes, ...gapTimes])).sort((a, b) => a - b);
  }
  return Array.from(new Set([start, end, ...finiteTimes, ...gapTimes])).filter(Number.isFinite).sort((a, b) => a - b);
}

export function buildScales(scaleKeys: Set<string>): Record<string, uPlot.Scale> {
  const scales: Record<string, uPlot.Scale> = {};
  scaleKeys.forEach((key) => {
    if (key === "temperature_c") scales[key] = { range: paddedRange(12, [-92, 92]) };
    else if (key === "pressure_log") scales[key] = { distr: 3, log: 10, range: () => [1e-8, 1.2e3] };
    else if (key === "pressure_rate_log") scales[key] = { distr: 3, log: 10, range: () => [1e-8, 1e3] };
    else if (key === "pressure_bar") scales[key] = { range: paddedRange(0.08, [0, 12]) };
    else if (key === "percent") scales[key] = { range: (_u, _min, _max) => [0, 100] };
    else if (key === "heat_flux_w") scales[key] = { range: paddedRange(8, [-45, 45]) };
    else scales[key] = {};
  });
  return scales;
}

export function buildAxes(scaleKeys: Set<string>, tile: GraphTile): uPlot.Axis[] {
  const leftAxisSize = 64;
  const rightAxisSize = 64;
  const axes: uPlot.Axis[] = [{ show: false }];
  const primary = scaleKeys.has("temperature_c")
    ? "temperature_c"
    : scaleKeys.has("power_w")
      ? "power_w"
      : scaleKeys.has("heat_flux_w")
        ? "heat_flux_w"
        : scaleKeys.has("bus_ms")
          ? "bus_ms"
          : scaleKeys.has("counter")
            ? "counter"
            : scaleKeys.has("pressure_log")
              ? "pressure_log"
              : scaleKeys.has("pressure_rate_log")
                ? "pressure_rate_log"
                : scaleKeys.has("pressure_bar")
                  ? "pressure_bar"
              : "percent";
  axes.push({
    show: true,
    scale: primary,
    stroke: "#7890a4",
    grid: { stroke: "rgba(83,112,140,0.26)", width: 1 },
    ticks: { stroke: "rgba(83,112,140,0.48)", width: 1, size: 4 },
    splits: (_u, _axisIdx, scaleMin, scaleMax) => logScale(primary) ? logSplits(scaleMin, scaleMax) : ySplits(scaleMin, scaleMax),
    size: leftAxisSize,
    gap: 0,
    label: axisLabel(primary, tile),
    labelSize: 12,
    labelGap: 0,
    values: logScale(primary) ? (_u, vals) => vals.map((v) => formatScientific(v)) : undefined,
  });
  const extra = Array.from(scaleKeys).filter((key) => key !== primary);
  extra.forEach((key) => {
    axes.push({
      show: true,
      scale: key,
      side: 1,
      stroke: key.includes("pressure") ? "#60a5fa" : "#8bd3a5",
      grid: { show: false },
      ticks: { show: false },
      size: rightAxisSize,
      gap: 0,
      label: axisLabel(key, tile),
      labelSize: 12,
      labelGap: 0,
      splits: logScale(key) ? (_u, _axisIdx, scaleMin, scaleMax) => logSplits(scaleMin, scaleMax) : undefined,
      values: logScale(key) ? (_u, vals) => vals.map((v) => formatScientific(v)) : undefined,
    });
  });
  if (!extra.length) {
    axes.push({
      show: true,
      side: 1,
      scale: primary,
      size: rightAxisSize,
      gap: 0,
      label: "",
      labelSize: 12,
      labelGap: 0,
      grid: { show: false },
      ticks: { show: false },
      values: () => [],
    });
  }
  return axes;
}

export function paddedRange(minPad: number, clamp?: [number, number]): uPlot.Range.Function {
  return (_u: uPlot, min: number, max: number) => {
    if (!Number.isFinite(min) || !Number.isFinite(max)) return clamp ?? [0, 1] as [number, number];
    if (max <= min) return [min - minPad, max + minPad] as [number, number];
    const pad = Math.max(minPad, (max - min) * 0.08);
    const low = min - pad;
    const high = max + pad;
    if (!clamp) return [low, high] as [number, number];
    return [Math.max(clamp[0], low), Math.min(clamp[1], high)] as [number, number];
  };
}

export function logScale(scale: string) {
  return scale === "pressure_log" || scale === "pressure_rate_log";
}

export function logSplits(min: number, max: number) {
  if (!Number.isFinite(min) || !Number.isFinite(max) || max <= 0 || max <= min) return [];
  const first = Math.ceil(Math.log10(Math.max(min, 1e-12)));
  const last = Math.floor(Math.log10(max));
  const values: number[] = [];
  for (let exp = first; exp <= last; exp += 1) values.push(Math.pow(10, exp));
  return values;
}

export function ySplits(min: number, max: number) {
  if (!Number.isFinite(min) || !Number.isFinite(max) || max <= min) return [];
  const target = 8;
  const rough = (max - min) / target;
  const mag = Math.pow(10, Math.floor(Math.log10(rough)));
  const step = [1, 2, 2.5, 5, 10].map((m) => m * mag).find((candidate) => rough <= candidate) ?? mag * 10;
  const first = Math.ceil(min / step) * step;
  const values: number[] = [];
  for (let v = first; v <= max + step * 0.25; v += step) values.push(Number(v.toFixed(6)));
  return values;
}

export function axisLabel(scale: string, _tile: GraphTile) {
  if (scale === "temperature_c") return "degC";
  if (scale === "pressure_log") return "log10 mbar";
  if (scale === "pressure_rate_log") return "log10 mbar/min";
  if (scale === "pressure_bar") return "bar";
  if (scale === "heat_flux_w") return "W";
  if (scale === "power_w") return "W";
  if (scale === "bus_ms") return "ms";
  if (scale === "counter") return "count";
  if (scale === "percent") return "%";
  return scale;
}

export function scaleForSeries(_tile: GraphTile, series: TileSeries): string {
  if (series.axis_id === "pressure_mbar") return "pressure_log";
  if (series.axis_id === "pressure_rate") return "pressure_rate_log";
  if (series.axis_id === "pressure_bar") return "pressure_bar";
  if (series.axis_id === "power_w") return "power_w";
  if (series.axis_id === "heat_flux_w") return "heat_flux_w";
  if (series.axis_id === "counter") return "counter";
  if (series.axis_id === "bus_ms") return "bus_ms";
  if (series.axis_id === "percent") return "percent";
  return "temperature_c";
}

export function stateBlocks(series: TileSeries, start: number, span: number) {
  if (series.spans?.length) {
    return series.spans.flatMap((state, index) => {
      const stateStart = Date.parse(state.start);
      const stateEnd = Date.parse(state.end);
      if (!Number.isFinite(stateStart) || !Number.isFinite(stateEnd) || stateEnd < start || stateStart > start + span) return [];
      const left = Math.max(0, Math.min(100, ((stateStart - start) / span) * 100));
      const right = Math.max(left + 0.15, Math.min(100, ((stateEnd - start) / span) * 100));
      return [{
        key: `${series.id}-span-${index}`,
        left,
        width: right - left,
        value: state.value ?? Number(state.state ?? 0),
        label: state.label ?? state.state ?? "",
      }];
    });
  }
  const sorted = [...(series.points ?? [])].sort((a, b) => Date.parse(a.timestamp) - Date.parse(b.timestamp));
  return sorted.flatMap((point, index) => {
    const pointTime = Date.parse(point.timestamp);
    const nextTime = index + 1 < sorted.length ? Date.parse(sorted[index + 1].timestamp) : start + span;
    if (!Number.isFinite(pointTime) || !Number.isFinite(nextTime) || nextTime < start || pointTime > start + span) return [];
    const left = Math.max(0, Math.min(100, ((pointTime - start) / span) * 100));
    const right = Math.max(left + 0.15, Math.min(100, ((nextTime - start) / span) * 100));
    return [{ key: `${series.id}-${index}`, left, width: right - left, value: point.value, label: String(point.value) }];
  });
}

export function inTimeRange(timestamp: string, range: TimeRange) {
  const t = Date.parse(timestamp);
  return Number.isFinite(t) && t >= range.start && t <= range.end;
}

export function renderKindFor(kind: string) {
  if (kind === "state") return "swimlane";
  if (kind === "event") return "event_rail";
  if (kind === "counter") return "counter";
  return "line";
}

function formatScientific(value: number) {
  if (!Number.isFinite(value)) return "";
  if (value === 0) return "0";
  return value.toExponential(2).replace("e", "E");
}


function markerAnchor(plot: uPlot, tile: GraphTile, marker: GraphMarker, timeMs: number, top: number, height: number) {
  const anchorSeries = rankedMarkerAnchorSeries(tile, marker);
  for (const series of anchorSeries) {
    const raw = rawValueAt(series, timeMs);
    if (raw === undefined) continue;
    const scale = scaleForSeries(tile, series);
    const y = plot.valToPos(displayValue(tile, series, raw), scale);
    if (!Number.isFinite(y)) continue;
    return { y: Math.max(top + 12, Math.min(top + height - 10, y)) };
  }
  return null;
}

function rankedMarkerAnchorSeries(tile: GraphTile, marker: GraphMarker) {
  return tile.series
    .filter((series) => (series.points ?? []).length)
    .map((series) => ({ series, score: markerAnchorScore(series, marker) }))
    .filter((candidate) => candidate.score > 0)
    .sort((a, b) => b.score - a.score)
    .map((candidate) => candidate.series);
}

function markerAnchorScore(series: TileSeries, marker: GraphMarker) {
  const haystack = `${series.id} ${series.label} ${series.axis_id ?? ""} ${series.source ?? ""} ${series.role}`.toLowerCase();
  const markerText = `${marker.id} ${marker.label} ${marker.kind} ${marker.role} ${marker.axis_id ?? ""}`.toLowerCase();
  const commandAnchored = commandAnchoredMarker(marker);
  let score = 0;
  const addIfMarkerAndSeries = (markerTokens: string[], seriesTokens: string[], points: number) => {
    if (!markerTokens.some((token) => markerText.includes(token))) return;
    if (seriesTokens.some((token) => haystack.includes(token))) score += points;
  };
  addIfMarkerAndSeries(["pressure", "vacuum", "tvac"], ["pressure", "vacuum", "tvac"], 80);
  addIfMarkerAndSeries(["dut", "functional", "stability", "dwell"], ["dut", "component", "interface", "chamber"], 70);
  addIfMarkerAndSeries(["shroud"], ["shroud"], 70);
  addIfMarkerAndSeries(["interlock"], ["interlock", "facility"], 70);
  addIfMarkerAndSeries(["operator", "command"], ["command", "chamber"], 55);
  addIfMarkerAndSeries(["pump", "exhaust"], ["pump", "exhaust", "cryo", "scavenger"], 55);
  if (marker.axis_id && series.axis_id === marker.axis_id) score += 90;
  if (commandAnchored && series.role === "command") score += 220;
  if (series.role === "actual") score += commandAnchored ? 4 : 18;
  if (series.role === "command") score += commandAnchored ? 34 : 8;
  if (series.role === "ghost") score += 4;
  return score;
}

function commandAnchoredMarker(marker: GraphMarker) {
  return marker.role === "operator_interaction"
    || marker.kind?.startsWith("operator_")
    || marker.kind === "functional_gate"
    || marker.kind === "stability"
    || marker.kind === "stability_achieved"
    || marker.kind === "interlock";
}

function drawExactMarkerAnchorLine(ctx: CanvasRenderingContext2D, x: number, top: number, height: number, color: string, alpha = 0.42) {
  ctx.save();
  ctx.globalAlpha = alpha;
  ctx.strokeStyle = color;
  ctx.lineWidth = 1;
  ctx.setLineDash([2, 4]);
  ctx.beginPath();
  ctx.moveTo(x, top);
  ctx.lineTo(x, top + height);
  ctx.stroke();
  ctx.restore();
}

function drawMarkerLeader(ctx: CanvasRenderingContext2D, x: number, y: number, labelX: number, labelY: number, labelWidth: number, labelHeight: number, color: string) {
  ctx.save();
  ctx.globalAlpha = 0.8;
  ctx.strokeStyle = color;
  ctx.lineWidth = 1;
  ctx.setLineDash([]);
  ctx.beginPath();
  ctx.moveTo(x, y);
  ctx.lineTo(labelX < x ? labelX + labelWidth : labelX, labelY + labelHeight / 2);
  ctx.stroke();
  ctx.restore();
}

function bandFillStyle(tile: GraphTile, bandKind: string, width: number) {
  const compact = width < 760;
  const opacity = compact ? 0.018 : 0.075;
  if (tile.campaign_id === "tvac_qualification" && (bandKind.includes("vacuum") || tile.card_id.includes("pressure"))) {
    return `rgba(59,130,246,${compact ? 0.018 : 0.065})`;
  }
  if (bandKind.includes("breakdown")) return `rgba(255,112,67,${compact ? 0.026 : 0.11})`;
  if (bandKind.includes("reset")) return `rgba(36,214,255,${compact ? 0.022 : 0.09})`;
  if (bandKind.includes("cold")) return `rgba(61,133,198,${opacity})`;
  return `rgba(198,119,61,${opacity})`;
}

function bandStrokeStyle(tile: GraphTile, bandKind: string) {
  if (tile.campaign_id === "tvac_qualification" && (bandKind.includes("vacuum") || tile.card_id.includes("pressure"))) return "rgba(96,165,250,0.16)";
  if (bandKind.includes("breakdown")) return "rgba(255,112,67,0.22)";
  if (bandKind.includes("reset")) return "rgba(36,214,255,0.18)";
  if (bandKind.includes("cold")) return "rgba(96,165,250,0.16)";
  return "rgba(255,176,0,0.14)";
}

export function drawTileOverlays(plot: uPlot, tile: GraphTile, heroGraph: HeroGraphModel, currentTimeMs?: number, hoverTimeMs?: number, timeRange?: TimeRange) {
  const ctx = plot.ctx;
  const bbox = plot.bbox;
  const left = bbox.left;
  const top = bbox.top;
  const width = bbox.width;
  const height = bbox.height;
  const start = timeRange?.start ?? Date.parse(tile.t0);
  const end = timeRange?.end ?? Date.parse(tile.t1);
  const span = Math.max(1, end - start);
  ctx.save();
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), 14);
  ctx.strokeStyle = "rgba(83,112,140,0.16)";
  ctx.lineWidth = 1;
  ctx.setLineDash([]);
  ticks.forEach((tick) => {
    const x = left + tick.ratio * width;
    ctx.beginPath();
    ctx.moveTo(x, top);
    ctx.lineTo(x, top + height);
    ctx.stroke();
  });
  (tile.bands ?? []).forEach((band) => {
    const x = left + ((Date.parse(band.start) - start) / span) * width;
    const x2 = left + ((Date.parse(band.end) - start) / span) * width;
    const bandKind = band.kind.toLowerCase();
    const bandWidth = Math.max(1, x2 - x);
    const compact = width < 760;
    ctx.fillStyle = bandFillStyle(tile, bandKind, width);
    if (compact) {
      const railHeight = Math.max(2, Math.min(7, height * 0.04));
      ctx.fillRect(x, top, bandWidth, railHeight);
      ctx.fillRect(x, top + height - railHeight, bandWidth, railHeight);
    } else {
      ctx.fillRect(x, top, bandWidth, height);
    }
    ctx.strokeStyle = bandStrokeStyle(tile, bandKind);
    ctx.lineWidth = width < 520 ? 0.75 : 1;
    if (compact) {
      ctx.beginPath();
      ctx.moveTo(x + 0.5, top + 0.5);
      ctx.lineTo(x + 0.5, top + height - 0.5);
      ctx.moveTo(x + bandWidth - 0.5, top + 0.5);
      ctx.lineTo(x + bandWidth - 0.5, top + height - 0.5);
      ctx.stroke();
    } else {
      ctx.strokeRect(x + 0.5, top + 0.5, Math.max(0, bandWidth - 1), Math.max(0, height - 1));
    }
  });
  const placedMarkerLabels: Array<{ x: number; y: number; width: number; height: number }> = [];
  const labeledMarkers = markerLabelIDs(tile.markers ?? [], start, start + span, width, tile.campaign_id);
  let expectedMarkerLabels = 0;
  let drawnMarkerLabels = 0;
  // Overflow stacking: when placeMarkerLabel fails, stack labels in a rail above the inner plot.
  // Canvas drawing is unclipped so this area (uPlot header space) is always available.
  let overflowSlot = 0;
  const forcePlace = (labelWidth: number, labelHeight: number, preferX: number): { x: number; y: number; width: number; height: number } => {
    const y = top - (overflowSlot + 1) * (labelHeight + 3);
    const x = Math.max(left, Math.min(left + width - labelWidth, preferX));
    overflowSlot += 1;
    return { x, y, width: labelWidth, height: labelHeight };
  };
  (tile.markers ?? []).forEach((marker) => {
    const markerTime = Date.parse(marker.timestamp);
    if (!Number.isFinite(markerTime)) return;
    const x = left + ((markerTime - start) / span) * width;
    if (x < left || x > left + width) return;
    const color = markerColor(marker);
    const operatorMarker = marker.role === "operator_interaction" || marker.kind?.startsWith("operator_");
    const attachedMarker = marker.kind === "functional_gate" || marker.kind === "stability" || marker.kind === "stability_achieved" || marker.kind === "interlock" || marker.kind === "pressure_gate";
    const anchor = attachedMarker || operatorMarker ? markerAnchor(plot, tile, marker, markerTime, top, height) : null;
    const anchorY = anchor?.y ?? top + 10;
    if (attachedMarker || operatorMarker) {
      drawExactMarkerAnchorLine(ctx, x, top, height, color, attachedMarker ? 0.48 : 0.36);
    }
    if (operatorMarker) {
      expectedMarkerLabels += 1;
      const compact = tile.campaign_id === "command_center_fat" || width < 760;
      const y = anchor?.y ?? top + 18 + (marker.kind === "operator_reset" ? 34 : marker.kind === "operator_reset_ready" ? 68 : 0);
      const markerRadius = compact ? 9 : 12;
      ctx.save();
      ctx.shadowColor = "rgba(0,0,0,0.72)";
      ctx.shadowBlur = 6;
      ctx.fillStyle = "rgba(2,6,11,0.88)";
      ctx.strokeStyle = color;
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.arc(x, y, markerRadius + 2, 0, Math.PI * 2);
      ctx.fill();
      ctx.stroke();
      ctx.beginPath();
      if (marker.kind === "operator_breakdown") {
        ctx.moveTo(x, y - markerRadius);
        ctx.lineTo(x + markerRadius, y);
        ctx.lineTo(x, y + markerRadius);
        ctx.lineTo(x - markerRadius, y);
        ctx.closePath();
      } else if (marker.kind === "operator_reset") {
        ctx.rect(x - markerRadius + 1, y - markerRadius + 1, (markerRadius - 1) * 2, (markerRadius - 1) * 2);
      } else {
        ctx.moveTo(x, y - markerRadius);
        ctx.lineTo(x + markerRadius, y + markerRadius - 2);
        ctx.lineTo(x - markerRadius, y + markerRadius - 2);
        ctx.closePath();
      }
      ctx.fillStyle = color;
      ctx.fill();
      ctx.lineWidth = 1.4;
      ctx.strokeStyle = "rgba(2,6,11,0.96)";
      ctx.stroke();
      const lines = operatorMarkerLines(marker, compact);
      const fontSize = compact ? Math.max(8.5, Math.min(10.5, width / 118)) : 12;
      const lineHeight = compact ? 11 : 14;
      ctx.font = `850 ${fontSize}px system-ui, sans-serif`;
      const maxLabelWidth = compact ? Math.max(76, Math.min(118, width * 0.11)) : Math.max(110, Math.min(170, width * 0.16));
      const measuredWidth = Math.max(...lines.map((line) => ctx.measureText(line).width)) + 12;
      const labelWidth = Math.min(maxLabelWidth, measuredWidth);
      const labelHeight = lines.length * lineHeight + 8;
      const placed = placeMarkerLabel({ x, y, labelWidth, labelHeight, left, top, width, height, placed: placedMarkerLabels, markerRadius })
        ?? forcePlace(labelWidth, labelHeight, x - labelWidth / 2);
      if (!placed) {
        ctx.restore();
        return;
      }
      placedMarkerLabels.push(placed);
      const labelX = placed.x;
      const labelY = placed.y;
      drawMarkerLeader(ctx, x, y, labelX, labelY, labelWidth, labelHeight, color);
      ctx.fillStyle = "rgba(2,6,11,0.94)";
      ctx.fillRect(labelX, labelY, labelWidth, labelHeight);
      ctx.strokeStyle = color;
      ctx.lineWidth = 1.2;
      ctx.strokeRect(labelX, labelY, labelWidth, labelHeight);
      ctx.fillStyle = color;
      lines.forEach((line, lineIndex) => ctx.fillText(fitCanvasText(ctx, line, labelWidth - 10), labelX + 6, labelY + lineHeight + 1 + lineIndex * lineHeight));
      drawnMarkerLabels += 1;
      ctx.restore();
    } else if (attachedMarker) {
      const commandCenterMarker = tile.campaign_id === "command_center_fat";
      ctx.save();
      ctx.fillStyle = "rgba(2,6,11,0.86)";
      ctx.strokeStyle = color;
      ctx.lineWidth = commandCenterMarker ? 2.2 : 1.8;
      ctx.beginPath();
      ctx.arc(x, anchorY, commandCenterMarker ? 8 : marker.kind === "functional_gate" ? 10 : 8, 0, Math.PI * 2);
      ctx.fill();
      ctx.stroke();
      ctx.restore();
      ctx.fillStyle = color;
      ctx.beginPath();
      if (marker.kind === "functional_gate") {
        ctx.moveTo(x, anchorY - 7);
        ctx.lineTo(x + 7, anchorY);
        ctx.lineTo(x, anchorY + 7);
        ctx.lineTo(x - 7, anchorY);
        ctx.closePath();
      } else {
        ctx.arc(x, anchorY, 5.6, 0, Math.PI * 2);
      }
      ctx.fill();
      if (!labeledMarkers.has(marker.id)) return;
      expectedMarkerLabels += 1;
      const label = commandCenterMarker ? "FT" : shortGateLabel(marker.label);
      ctx.save();
      ctx.font = commandCenterMarker ? "850 10px system-ui, sans-serif" : "850 12px system-ui, sans-serif";
      const metrics = ctx.measureText(label);
      const labelWidth = Math.max(commandCenterMarker ? 22 : 36, metrics.width + 10);
      const labelHeight = commandCenterMarker ? 16 : 18;
      const placed = placeMarkerLabel({ x, y: anchorY, labelWidth, labelHeight, left, top, width, height, placed: placedMarkerLabels, markerRadius: 8 })
        ?? forcePlace(labelWidth, labelHeight, x - labelWidth / 2);
      if (!placed) {
        ctx.restore();
        return;
      }
      placedMarkerLabels.push(placed);
      const labelX = placed.x;
      const labelY = placed.y;
      drawMarkerLeader(ctx, x, anchorY, labelX, labelY, labelWidth, labelHeight, color);
      ctx.fillStyle = "rgba(2,6,11,0.92)";
      ctx.fillRect(labelX, labelY, labelWidth, labelHeight);
      ctx.strokeStyle = color;
      ctx.lineWidth = 1;
      ctx.strokeRect(labelX, labelY, labelWidth, labelHeight);
      ctx.fillStyle = marker.kind === "functional_gate" ? "#fff0a8" : "#c9ffef";
      ctx.shadowColor = "rgba(0,0,0,0.88)";
      ctx.shadowBlur = 5;
      ctx.fillText(label, labelX + 5, labelY + Math.min(13, labelHeight - 5));
      drawnMarkerLabels += 1;
      ctx.restore();
    } else {
      ctx.beginPath();
      ctx.arc(x, top + 10, 3.2, 0, Math.PI * 2);
      ctx.fill();
    }
  });
  const host = plot.root?.closest("[data-uplot-card]") as HTMLElement | null;
  if (host) {
    host.dataset.markerLabelsExpected = String(expectedMarkerLabels);
    host.dataset.markerLabelsDrawn = String(drawnMarkerLabels);
  }
  const now = currentTimeMs ?? Date.parse(heroGraph.time_axis.now ?? heroGraph.execution?.now ?? "");
  if (Number.isFinite(now)) {
    const x = left + ((now - start) / span) * width;
    ctx.fillStyle = "rgba(3,7,12,0.58)";
    ctx.fillRect(Math.max(left, x), top, Math.max(0, left + width - x), height);
    ctx.strokeStyle = "rgba(242,247,255,0.9)";
    ctx.setLineDash([3, 3]);
    ctx.beginPath();
    ctx.moveTo(x, top);
    ctx.lineTo(x, top + height);
    ctx.stroke();
  }
  if (Number.isFinite(hoverTimeMs)) {
    const x = left + (((hoverTimeMs as number) - start) / span) * width;
    if (x >= left && x <= left + width) {
      ctx.strokeStyle = "rgba(255,216,95,0.95)";
      ctx.setLineDash([]);
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.moveTo(x, top);
      ctx.lineTo(x, top + height);
      ctx.stroke();
    }
  }
  ctx.restore();
}

function markerLabelIDs(markers: GraphMarker[], start: number, end: number, _width: number, _campaignID?: string) {
  return new Set(
    markers
      .filter((marker) => {
        const markerTime = Date.parse(marker.timestamp);
        return Number.isFinite(markerTime) && markerTime >= start && markerTime <= end && markerLabelScore(marker) > 0;
      })
      .sort((a, b) => markerLabelScore(b) - markerLabelScore(a) || Date.parse(a.timestamp) - Date.parse(b.timestamp))
      .map((marker) => marker.id)
  );
}

function markerLabelScore(marker: GraphMarker) {
  let score = 0;
  if (marker.role === "interlock" || marker.result === "fail" || marker.kind === "interlock") score += 1000;
  if (marker.kind === "functional_gate") score += 760;
  if (marker.kind === "pressure_gate") score += 640;
  if (marker.kind === "stability" || marker.kind === "stability_achieved") score += 440;
  if (marker.result === "pass") score += 120;
  return score;
}
