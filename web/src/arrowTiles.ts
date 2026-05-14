import { tableFromIPC } from "apache-arrow";
import type { GraphModel, GraphTile, GraphTileCardRef, GraphTileManifest, GraphWallSignal, TileSeries } from "./types";

const arrowCache = new Map<string, Promise<ArrowTelemetry>>();
const tileCache = new Map<string, Promise<GraphTile>>();
const maxMaterializedPoints = 1400;

export function invalidateArrowCache() {
  arrowCache.clear();
  tileCache.clear();
}

type ArrowTelemetry = {
  bySensor: Map<string, ArrowRow[]>;
  t0: string;
  t1: string;
};

type ArrowRow = {
  t: number;
  value: number | null;
  state: string;
};

type NumericArrowRow = ArrowRow & { value: number };

type BuiltSeries = {
  series: TileSeries;
  rawCount: number;
  pointCount: number;
};

export async function arrowTile(campaignId: string, card: GraphTileCardRef, tileManifest: GraphTileManifest, graph: GraphModel, level = "arrow-native", requestedT0?: string, requestedT1?: string, dataVersion?: string): Promise<GraphTile> {
  const cacheKey = [
    dataVersion || "unversioned",
    campaignId,
    card.card_id,
    level,
    requestedT0 || graph.graph_wall?.time_range.start || "",
    requestedT1 || graph.graph_wall?.time_range.end || ""
  ].join("|");
  let cached = tileCache.get(cacheKey);
  if (!cached) {
    cached = buildArrowTile(campaignId, card, tileManifest, graph, level, requestedT0, requestedT1, dataVersion);
    tileCache.set(cacheKey, cached);
  }
  return cached;
}

async function buildArrowTile(campaignId: string, card: GraphTileCardRef, tileManifest: GraphTileManifest, graph: GraphModel, level = "arrow-native", requestedT0?: string, requestedT1?: string, dataVersion?: string): Promise<GraphTile> {
  const telemetry = await cachedArrowTelemetry(campaignId, dataVersion);
  const start = requestedT0 ? Date.parse(requestedT0) : Date.parse(graph.graph_wall?.time_range.start ?? telemetry.t0);
  const end = requestedT1 ? Date.parse(requestedT1) : Date.parse(graph.graph_wall?.time_range.end ?? telemetry.t1);
  const t0 = Number.isFinite(start) ? start : Date.parse(telemetry.t0);
  const t1 = Number.isFinite(end) ? end : Date.parse(telemetry.t1);
  const built = card.signals.map((signal) => buildSeries(signal, card, telemetry, t0, t1)).filter((item) => item.series.points?.length || item.series.spans?.length);
  const series = built.map((item) => item.series);
  const rawPointCount = built.reduce((sum, item) => sum + item.rawCount, 0);
  const pointCount = built.reduce((sum, item) => sum + item.pointCount, 0);
  const laneToken = commandCenterLaneToken(card);
  const markerEligible = card.include_markers || card.card_id === "thermal_program" || card.render_kind === "event_rail" || card.render_kind === "swimlane";
  const bands = filterLaneBands(intersectByWindow([...(graph.hero_graph?.phase_bands ?? []), ...(graph.hero_graph?.dwell_windows ?? [])], t0, t1), laneToken);
  const markers = markerEligible ? filterLaneMarkers(intersectMarkers(graph.hero_graph?.markers ?? [], t0, t1), laneToken) : [];
  const events = markers.map((marker) => ({
      id: marker.id,
      kind: marker.kind,
      label: marker.label,
      timestamp: marker.timestamp,
      result: marker.result,
      value: marker.value,
      evidence_ref: marker.evidence_ref
    }));
  return {
    schema_version: 1,
    generated_at: tileManifest.generated_at,
    id: `${campaignId}_${card.card_id}_${level}_arrow`,
    campaign_id: campaignId,
    card_id: card.card_id,
    level,
    t0: new Date(t0).toISOString(),
    t1: new Date(t1).toISOString(),
    series,
    bands,
    markers,
    events,
    diagnostics: {
      source: "arrow_telemetry",
      mode: "browser_native_arrow",
      raw_point_count: rawPointCount,
      point_count: pointCount,
      decimated: pointCount < rawPointCount,
      decimation: "min_max_envelope",
      time_span_ms: t1 - t0,
      freshness_ms: 0,
      source_quality: "arrow"
    },
    provenance: {
      source_node: "gossamer_arrow_fixture",
      source_family: card.signals[0]?.source_family,
      generation_mode: "arrow_stream_to_tile_view",
      fixture_version: tileManifest.source_fixture_version ?? "gossamer.telemetry.arrow.v2",
      synthetic: true
    }
  };
}

async function cachedArrowTelemetry(campaignId: string, dataVersion?: string) {
  const key = dataVersion ? `${campaignId}@${dataVersion}` : campaignId;
  let cached = arrowCache.get(key);
  if (!cached) {
    cached = fetchArrowTelemetry(campaignId, dataVersion);
    arrowCache.set(key, cached);
  }
  return cached;
}

async function fetchArrowTelemetry(campaignId: string, dataVersion?: string): Promise<ArrowTelemetry> {
  const buffer = await fetchArrowBuffer(campaignId, dataVersion);
  const table = tableFromIPC(new Uint8Array(await decodeArrowBuffer(buffer)));
  const timestamp = table.getChild("timestamp_ns");
  const sensor = table.getChild("sensor");
  const value = table.getChild("value");
  const state = table.getChild("state");
  if (!timestamp || !sensor || !value || !state) {
    throw new Error("Arrow telemetry is missing required columns");
  }
  const bySensor = new Map<string, ArrowRow[]>();
  let minT = Number.POSITIVE_INFINITY;
  let maxT = Number.NEGATIVE_INFINITY;
  for (let i = 0; i < table.numRows; i++) {
    const signalID = String(sensor.get(i) ?? "");
    if (!signalID) continue;
    const timestampNs = Number(timestamp.get(i) ?? 0);
    if (!Number.isFinite(timestampNs)) continue;
    const t = timestampNs / 1_000_000;
    minT = Math.min(minT, t);
    maxT = Math.max(maxT, t);
    const row: ArrowRow = { t, value: nullableNumber(value.get(i)), state: String(state.get(i) ?? "") };
    const rows = bySensor.get(signalID);
    if (rows) rows.push(row);
    else bySensor.set(signalID, [row]);
  }
  return {
    bySensor,
    t0: new Date(minT).toISOString(),
    t1: new Date(maxT).toISOString()
  };
}

async function fetchArrowBuffer(campaignId: string, dataVersion?: string) {
  const base = `/data/current/campaigns/${campaignId}/telemetry.arrow.gz`;
  const url = dataVersion ? `${base}?v=${encodeURIComponent(dataVersion)}` : base;
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`${url} returned ${response.status}`);
  }
  const buffer = await response.arrayBuffer();
  if (looksLikeHTML(buffer)) {
    throw new Error(`${url} returned HTML instead of Arrow telemetry`);
  }
  return buffer;
}

async function decodeArrowBuffer(buffer: ArrayBuffer) {
  const bytes = new Uint8Array(buffer);
  if (bytes.length >= 2 && bytes[0] === 0x1f && bytes[1] === 0x8b) {
    if (!("DecompressionStream" in globalThis)) {
      throw new Error("Gzip-compressed Arrow telemetry requires DecompressionStream support");
    }
    const stream = new Blob([bytes]).stream().pipeThrough(new DecompressionStream("gzip"));
    return new Response(stream).arrayBuffer();
  }
  return buffer;
}

function looksLikeHTML(buffer: ArrayBuffer) {
  const bytes = new Uint8Array(buffer);
  let offset = 0;
  while (offset < bytes.length && bytes[offset] <= 0x20) offset += 1;
  return bytes[offset] === 0x3c;
}

function buildSeries(signal: GraphWallSignal, card: GraphTileCardRef, telemetry: ArrowTelemetry, t0: number, t1: number): BuiltSeries {
  const rows = (telemetry.bySensor.get(signal.id) ?? []).filter((row) => row.t >= t0 && row.t <= t1);
  const numeric = rows.filter((row) => row.value !== null);
  const tileSeries: TileSeries = {
    id: signal.id,
    label: signal.label,
    unit: signal.unit,
    role: signal.role,
    kind: signal.kind,
    axis_id: signal.axis_id,
    source: signal.source,
    source_family: signal.source_family,
    step: card.render_kind === "counter" || card.render_kind === "swimlane",
    value_table: signal.value_table
  };
  if (card.render_kind === "swimlane") {
    tileSeries.spans = rowsToSpans(rows, signal.value_table, t1);
    return { series: tileSeries, rawCount: rows.length, pointCount: tileSeries.spans.length };
  }
  const numericRows = numeric as NumericArrowRow[];
  const materialized = decimateRows(numericRows, maxMaterializedPoints);
  tileSeries.points = materialized.map((row) => ({ timestamp: new Date(row.t).toISOString(), value: row.value }));
  return { series: tileSeries, rawCount: numericRows.length, pointCount: materialized.length };
}

function decimateRows(rows: NumericArrowRow[], budget: number): NumericArrowRow[] {
  if (rows.length <= budget || budget < 4) return rows;
  const out = [rows[0]];
  const bucketSize = (rows.length - 2) / (budget - 2);
  for (let bucket = 0; bucket < budget - 2; bucket++) {
    const start = Math.floor(bucket * bucketSize) + 1;
    const end = Math.min(rows.length - 1, Math.floor((bucket + 1) * bucketSize) + 1);
    let min = rows[start];
    let max = rows[start];
    for (let i = start + 1; i < end; i++) {
      const row = rows[i];
      if (row.value < min.value) min = row;
      if (row.value > max.value) max = row;
    }
    if (min.t <= max.t) out.push(min, max);
    else out.push(max, min);
  }
  out.push(rows[rows.length - 1]);
  return out.filter((row, index) => index === 0 || row.t !== out[index - 1].t);
}

function rowsToSpans(rows: ArrowRow[], valueTable: Record<string, string> | undefined, end: number) {
  const spans = [];
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    const next = rows[i + 1];
    const state = row.state || String(row.value ?? "");
    spans.push({
      start: new Date(row.t).toISOString(),
      end: new Date(next?.t ?? end).toISOString(),
      state,
      label: valueTable?.[state] ?? state,
      value: row.value ?? undefined
    });
  }
  return spans;
}

function nullableNumber(value: unknown) {
  if (value === null || value === undefined) return null;
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
}

function intersectByWindow<T extends { start: string; end: string }>(items: T[], t0: number, t1: number): T[] {
  return items.filter((item) => Date.parse(item.end) >= t0 && Date.parse(item.start) <= t1);
}

function intersectMarkers<T extends { timestamp: string }>(items: T[], t0: number, t1: number): T[] {
  return items.filter((item) => {
    const t = Date.parse(item.timestamp);
    return Number.isFinite(t) && t >= t0 && t <= t1;
  });
}

function commandCenterLaneToken(card: GraphTileCardRef) {
  for (const signal of card.signals ?? []) {
    const match = signal.id.match(/^cc\.([a-z0-9_-]+)\./i);
    if (match) return match[1].toLowerCase();
  }
  return "";
}

function filterLaneBands<T extends { id?: string; label?: string; kind?: string }>(items: T[], laneToken: string): T[] {
  if (!laneToken) return items;
  return items.filter((item) => item.kind === "weekend" || containsLaneToken(laneToken, item.id, item.label));
}

function filterLaneMarkers<T extends { id?: string; label?: string; evidence_ref?: string }>(items: T[], laneToken: string): T[] {
  if (!laneToken) return items;
  return items.filter((item) => containsLaneToken(laneToken, item.id, item.label, item.evidence_ref));
}

function containsLaneToken(laneToken: string, ...values: Array<string | undefined>) {
  return values.some((value) => (value ?? "").toLowerCase().includes(laneToken));
}
