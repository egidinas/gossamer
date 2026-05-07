import { useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties, PointerEvent as ReactPointerEvent, ReactNode } from "react";
import uPlot from "uplot";
import "uplot/dist/uPlot.min.css";
import { api } from "../api";
import type { GraphTile, GraphTileCardRef, GraphTileManifest, GraphWallCard, GraphWallModel, HeroGraphModel, TileSeries } from "../types";

type Props = {
  campaignId: string;
  wall: GraphWallModel;
  heroGraph: HeroGraphModel;
  afterProgress?: ReactNode;
};

type TimeRange = {
  start: number;
  end: number;
};

const TIME_GRID_TICK_COUNT = 14;

const roleColors: Record<string, string> = {
  command: "#ffd85f",
  ghost: "#8aa7c4",
  acceptance_band: "#3ddc84",
  actual: "#56d6df",
  source_quality: "#66b8ef",
  counter: "#b8a6ff",
  interlock: "#ff6374",
  evidence: "#b8a6ff",
  event: "#f2f7ff",
  state: "#8bd3a5",
};

const signalColors: Record<string, string> = {
  "trace.command.chamber": "#ffd400",
  "trace.ghost.profile": "#f8fafc",
  "trace.acceptance.temperature": "#4ee28a",
  "trace.actual.chamber_air": "#00c8ff",
  "trace.context.chamber_air": "#00c8ff",
  "trace.table_loop": "#ff8a00",
  "trace.interface": "#ff8a00",
  "trace.shroud": "#b65cff",
  "trace.shroud_inlet": "#00a8ff",
  "trace.shroud_outlet": "#ff58c8",
  "trace.dut_temp_a": "#ff315f",
  "trace.dut_temp_b": "#00d6a3",
  "trace.tvac_pressure": "#1f6fff",
  "trace.actual.tvac_pressure": "#1f6fff",
  "trace.tvac_pressure_target": "#9cc7ff",
  "trace.tvac_outgassing": "#ff5c93",
  "trace.tvac_virtual_leak": "#f5d742",
  "trace.tvac_roughing_pump": "#ff7a35",
  "trace.tvac_turbo_pump": "#31d6ff",
  "trace.tvac_pump_removal": "#b079ff",
  "trace.tvac_volatile_inventory": "#9bff70",
  "trace.total_power": "#f97316",
  "trace.subsystem_power": "#60a5fa",
  "trace.bus_packets": "#a78bfa",
  "trace.bus_retries": "#fb7185",
  "trace.phase_enum": "#e5e7eb",
  "trace.functional_gate_active": "#fbbf24",
  "trace.stability_reached": "#34d399",
  "trace.dwell_active": "#38bdf8",
  "trace.dwell_complete": "#a78bfa",
  "trace.dut_ready": "#84cc16",
  "trace.dut_operative": "#22c55e",
  "trace.payload_active": "#f97316",
  "trace.rf_link_locked": "#06b6d4",
  "trace.fault_flag": "#fb7185",
  "trace.power_total": "#ff7a00",
  "trace.power_subsystem": "#00c8ff",
  "trace.power_payload": "#ff315f",
  "trace.power_avionics": "#00d6a3",
  "trace.power_link": "#b079ff",
  "trace.overall_packet_counter": "#f8fafc",
  "trace.tm_packet_counter": "#00c8ff",
  "trace.tc_packet_counter": "#ffd400",
  "trace.dropped_frame_count": "#ff315f",
  "trace.bus_latency": "#ffb000",
  "trace.source_freshness": "#00d6a3",
  "trace.cooling_water_temp": "#00a8ff",
  "trace.pressurized_air_supply": "#ffd400",
  "trace.air_dewpoint": "#b079ff",
  "trace.ln2_duty": "#00a8ff",
  "trace.freeze_margin": "#4ee28a",
  "trace.tvac_cryo_exhaust": "#1f6fff",
  "trace.tvac_scavenged_exhaust": "#00d6a3",
  "trace.tvac_scavenger_water_return": "#ff8a00",
  "trace.tvac_exhaust_cold_recovery": "#b079ff",
};

function colorForSignal(signal: Pick<TileSeries, "id" | "role" | "render_kind" | "kind"> | { id: string; role: string; kind?: string }, index = 0) {
  const kind = "kind" in signal ? signal.kind : ("render_kind" in signal ? signal.render_kind : undefined);
  if (signalColors[signal.id]) return signalColors[signal.id];
  const semantic = semanticColor(signal.id);
  if (semantic) return semantic;
  if (signal.role === "command" || signal.role === "ghost" || signal.role === "acceptance_band" || signal.role === "interlock" || signal.role === "evidence") return roleColors[signal.role];
  return paletteForID(signal.id, index) ?? roleColors[signal.role] ?? (kind ? roleColors[kind] : undefined) ?? palette(index);
}

function semanticColor(id: string) {
  const lower = id.toLowerCase();
  if (lower.includes("dut_temp_a") || lower.includes("dut.a") || lower.includes("node_a")) return "#ff315f";
  if (lower.includes("dut_temp_b") || lower.includes("dut.b") || lower.includes("node_b")) return "#00d6a3";
  if (lower.includes("dut") && lower.includes("temp")) return "#ff6b35";
  if (lower.includes("command") || lower.includes("target")) return "#ffd400";
  if (lower.includes("ghost") || lower.includes("profile")) return "#f8fafc";
  if (lower.includes("pressure")) return "#1f6fff";
  if (lower.includes("power")) return "#ff7a35";
  if (lower.includes("packet") || lower.includes("bus")) return "#b079ff";
  if (lower.includes("ready") || lower.includes("operative") || lower.includes("stability")) return "#00d6a3";
  if (lower.includes("fault") || lower.includes("error") || lower.includes("interlock")) return "#ff315f";
  if (lower.includes("interface") || lower.includes("table") || lower.includes("platen")) return "#ff8a00";
  if (lower.includes("shroud")) return "#b65cff";
  if (lower.includes("chamber")) return "#00c8ff";
  return undefined;
}

function orderLegendSignals<T extends { id: string; label?: string; role?: string; kind?: string; render_kind?: string }>(signals: T[]) {
  return [...signals].sort((a, b) => signalPriority(a) - signalPriority(b));
}

function signalPriority(signal: { id: string; label?: string; role?: string; kind?: string; render_kind?: string }) {
  const text = `${signal.id} ${signal.label ?? ""}`.toLowerCase();
  if (signal.role === "command") return 0;
  if (signal.role === "ghost") return 1;
  if (signal.role === "acceptance_band") return 2;
  if (text.includes("dut")) return 3;
  if (text.includes("article") || text.includes("component")) return 4;
  if (text.includes("interface") || text.includes("platen") || text.includes("table")) return 5;
  if (text.includes("chamber") || text.includes("shroud")) return 6;
  if (text.includes("pressure")) return 7;
  if (text.includes("power")) return 8;
  if (text.includes("bus") || text.includes("packet")) return 9;
  if (signal.kind === "state" || signal.render_kind === "swimlane") return 10;
  return 20;
}

function graphCardPriority(a: GraphWallCard, b: GraphWallCard) {
  return graphCardRank(a) - graphCardRank(b);
}

function graphSectionPriority(a: GraphWallModel["sections"][number], b: GraphWallModel["sections"][number]) {
  return graphSectionRank(a) - graphSectionRank(b);
}

function graphSectionRank(section: GraphWallModel["sections"][number]) {
  return Math.min(...section.cards.map(graphCardRank), 100);
}

function graphCardRank(card: GraphWallCard) {
  const id = card.id.toLowerCase();
  const title = card.title.toLowerCase();
  if (id === "thermal_program") return 0;
  if (id.includes("dut_temperature") || title.includes("dut temperature")) return 10;
  if (id.includes("dut_power") || title.includes("dut power")) return 20;
  if (id.includes("tmtc_health")) return 30;
  if (id.includes("tmtc_counters")) return 40;
  if (id.includes("state_change") || card.render_kind === "swimlane") return 50;
  if (id.includes("functional_events") || card.render_kind === "event_rail") return 60;
  if (id.includes("facility") || id.includes("building") || id.includes("source_quality") || title.includes("testbed")) return 80;
  return 70;
}

function blockLabel(label: string, value: number) {
  const normalized = String(label ?? "").trim();
  if (normalized && normalized !== "0" && normalized !== "1") return normalized;
  return value > 0 ? "ACTIVE" : "idle";
}

function eventColor(kind?: string) {
  const lower = (kind ?? "").toLowerCase();
  if (lower.includes("functional") || lower.includes("gate")) return "#ffb000";
  if (lower.includes("evidence")) return "#b079ff";
  if (lower.includes("interlock") || lower.includes("fault")) return "#ff315f";
  if (lower.includes("stability") || lower.includes("dwell")) return "#00d6a3";
  if (lower.includes("pressure")) return "#1f6fff";
  return "#31d6ff";
}

export function OperatorGraphWall({ campaignId, wall, heroGraph, afterProgress }: Props) {
  const [manifest, setManifest] = useState<GraphTileManifest | null>(null);
  const [tiles, setTiles] = useState<Record<string, GraphTile>>({});
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [pinOverrides, setPinOverrides] = useState<Record<string, boolean>>({});
  const [cardHeights, setCardHeights] = useState<Record<string, number>>({});
  const [hoverTimeMs, setHoverTimeMs] = useState<number | undefined>(undefined);
  const [peekTimeMs, setPeekTimeMs] = useState<number | undefined>(undefined);
  const [timeAxisBounds, setTimeAxisBounds] = useState<{ left: number; right: number } | undefined>(undefined);
  const fullTimeRange = useMemo(() => graphTimeRange(heroGraph), [heroGraph]);
  const [viewRange, setViewRange] = useState<TimeRange>(fullTimeRange);
  const scrollFrameRef = useRef<HTMLDivElement | null>(null);
  const requestedTiles = useRef<Set<string>>(new Set());
  const loadGeneration = useRef(0);
  const execution = heroGraph.execution;
  const currentTimeMs = useAnimatedReplayTime(campaignId, heroGraph);
  const readoutTimeMs = peekTimeMs ?? hoverTimeMs ?? currentTimeMs;
  const readoutMode = peekTimeMs !== undefined ? "peek" : hoverTimeMs !== undefined ? "crosshair" : "live";

  useEffect(() => {
    let cancelled = false;
    loadGeneration.current += 1;
    setViewRange(fullTimeRange);
    setManifest(null);
    setTiles({});
    setPinOverrides({});
    setCardHeights({});
    requestedTiles.current.clear();
    api.tileManifest(campaignId).then((next) => {
      if (cancelled) return;
      setManifest(next);
      const initialCollapsed: Record<string, boolean> = {};
      next.cards.forEach((card) => {
        initialCollapsed[card.card_id] = card.collapsible && !card.default_expanded;
      });
      setCollapsed(initialCollapsed);
    }).catch((err) => console.error(err));
    return () => {
      cancelled = true;
    };
  }, [campaignId, fullTimeRange]);

  const manifestCards = useMemo(() => new Map((manifest?.cards ?? []).map((card) => [card.card_id, card])), [manifest]);
  const firstSectionID = wall.sections[0]?.id;
  const primaryCardID = wall.sections[0]?.cards[0]?.id;
  useEffect(() => {
    if (!manifest) return;
    const generation = loadGeneration.current;
    const cardsToFetch = manifest.cards
      .filter((card) => !collapsed[card.card_id] && !tiles[card.card_id] && !requestedTiles.current.has(card.card_id))
      .sort(tileCardPriority)
      .slice(0, 8)
      .map((card) => card.card_id);
    if (!cardsToFetch.length) return;
    const fetchCard = (cardID: string, index: number) => {
      requestedTiles.current.add(cardID);
      scheduleTileWork(() => {
        if (loadGeneration.current !== generation) return;
        api.tile(campaignId, cardID, "minute")
          .then((tile) => {
            if (loadGeneration.current !== generation) return;
            setTiles((existing) => ({ ...existing, [tile.card_id]: tile }));
          })
          .catch((err) => console.error(err));
      }, index < 3 ? index * 35 : 130 + index * 45);
    };
    cardsToFetch.forEach(fetchCard);
  }, [campaignId, collapsed, manifest, tiles]);

  useEffect(() => {
    const frame = scrollFrameRef.current;
    if (!frame) return;
    let raf = 0;
    const measure = () => {
      window.cancelAnimationFrame(raf);
      raf = window.requestAnimationFrame(() => {
        const plot = frame.querySelector(".u-over, .tile-time-plane") as HTMLElement | null;
        const rect = plot?.getBoundingClientRect();
        if (!rect || rect.width <= 0) return;
        const next = {
          left: Math.round(rect.left),
          right: Math.round(window.innerWidth - rect.right),
        };
        setTimeAxisBounds((existing) => {
          if (existing && Math.abs(existing.left - next.left) < 1 && Math.abs(existing.right - next.right) < 1) return existing;
          return next;
        });
      });
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(frame);
    window.addEventListener("resize", measure);
    frame.addEventListener("scroll", measure, { passive: true });
    return () => {
      window.cancelAnimationFrame(raf);
      observer.disconnect();
      window.removeEventListener("resize", measure);
      frame.removeEventListener("scroll", measure);
    };
  }, [collapsed, manifest, tiles, viewRange]);

  return (
    <div className="operator-graph-wall" data-campaign-id={campaignId} data-graph-wall-version={wall.graph_version} data-tile-backed="true">
      <SharedTimeAxis
        fullRange={fullTimeRange}
        timeRange={viewRange}
        currentTimeMs={currentTimeMs}
        hoverTimeMs={hoverTimeMs}
        peekTimeMs={peekTimeMs}
        plotBounds={timeAxisBounds}
        onTimeRange={setViewRange}
      />
      <div className="operator-wall-scrollframe" ref={scrollFrameRef}>
        {[...wall.sections].sort(graphSectionPriority).map((section) => (
          <section className="operator-wall-section" key={section.id} data-section-id={section.id}>
            {!(section.id === firstSectionID && primaryCardID) && <div className="operator-wall-section-title">
              <strong>{section.title}</strong>
              <span>{section.transport} / {section.direction}</span>
            </div>}
            <div className="operator-wall-cards">
              {[...section.cards].sort(graphCardPriority).map((card) => {
                const isPrimary = card.id === primaryCardID;
                const cardRef = manifestCards.get(card.id);
                const isCollapsed = collapsed[card.id] ?? false;
                const isPinned = pinOverrides[card.id] ?? ((isPrimary && campaignId !== "command_center_fat") || card.placement.pinned);
                return (
                  <GraphWallCardView
                    key={card.id}
                    card={card}
                    cardRef={cardRef}
                    collapsed={isCollapsed}
                    pinned={isPinned}
                    height={cardHeights[card.id]}
                    currentTimeMs={currentTimeMs}
                    hoverTimeMs={hoverTimeMs}
                    heroGraph={heroGraph}
                    onHoverTime={setHoverTimeMs}
                    onPeekTime={setPeekTimeMs}
                    readoutMode={readoutMode}
                    readoutTimeMs={readoutTimeMs}
                    timeRange={viewRange}
                    onTimeRange={setViewRange}
                    tile={tiles[card.id]}
                    onToggle={() => setCollapsed((existing) => ({ ...existing, [card.id]: !isCollapsed }))}
                    onPinToggle={() => setPinOverrides((existing) => ({ ...existing, [card.id]: !isPinned }))}
                    onHeightChange={(height) => setCardHeights((existing) => ({ ...existing, [card.id]: height }))}
                  />
                );
              })}
            </div>
          </section>
        ))}
      </div>
      {afterProgress}
      {execution && <ExecutionProgress execution={execution} heroGraph={heroGraph} currentTimeMs={currentTimeMs} />}
      <div className="operator-wall-meta">
        <span>{manifest ? "tile manifest ready" : "loading tile manifest"}</span>
        <span>{wall.graph_version}</span>
        <span>{wall.source_mode}</span>
        <span>{wall.time_range.mode}</span>
        <span>{wall.tile_policy.shared_timebase_required ? "shared timebase" : "local timebase"}</span>
        {execution && <span>{execution.acceleration}</span>}
      </div>
    </div>
  );
}

function graphTimeRange(heroGraph: HeroGraphModel): TimeRange {
  const start = Date.parse(heroGraph.time_axis.start);
  const end = Date.parse(heroGraph.time_axis.end);
  return {
    start: Number.isFinite(start) ? start : 0,
    end: Number.isFinite(end) && end > start ? end : start + 1,
  };
}

function tileCardPriority(a: GraphTileCardRef, b: GraphTileCardRef) {
  const aPriority = cardPriority(a);
  const bPriority = cardPriority(b);
  if (aPriority !== bPriority) return aPriority - bPriority;
  if (a.default_expanded !== b.default_expanded) return a.default_expanded ? -1 : 1;
  return a.card_id.localeCompare(b.card_id);
}

function cardPriority(card: GraphTileCardRef) {
  const order: Record<string, number> = {
    thermal_program: 0,
    dut_temperature: 1,
    tvac_pressure: 2,
    facility_actuation: 3,
    dut_power: 4,
    tmtc_counters: 5,
    state_change_swimlane: 6,
    functional_events: 7,
    tvac_exhaust_scavenger: 8,
    building_infrastructure: 8,
    facility_temperature_safety: 9,
    tmtc_health: 10,
    source_quality: 11,
  };
  return order[card.card_id] ?? 40;
}

function useAnimatedReplayTime(campaignId: string, heroGraph: HeroGraphModel) {
  const startMs = Date.parse(heroGraph.time_axis.start);
  const endMs = Date.parse(heroGraph.time_axis.end);
  const baseNow = Date.parse(heroGraph.time_axis.now ?? heroGraph.execution?.now ?? "");
  const acceleration = replayAcceleration(heroGraph.execution?.acceleration);
  const [wallStart, setWallStart] = useState(() => Date.now());
  const [now, setNow] = useState(baseNow);
  const [streaming, setStreaming] = useState(false);

  useEffect(() => {
    setWallStart(Date.now());
    setNow(baseNow);
    setStreaming(false);
  }, [baseNow, heroGraph.id]);

  useEffect(() => {
    if (typeof EventSource === "undefined") return;
    if (!Number.isFinite(baseNow) || !Number.isFinite(startMs) || !Number.isFinite(endMs)) return;
    const stream = new EventSource(api.liveCursorPath(campaignId));
    stream.addEventListener("cursor", (event) => {
      try {
        const payload = JSON.parse((event as MessageEvent).data) as { now?: string };
        const next = Date.parse(payload.now ?? "");
        if (!Number.isFinite(next)) return;
        setStreaming(true);
        setNow(Math.min(endMs, Math.max(startMs, next)));
      } catch (err) {
        console.error(err);
      }
    });
    stream.onerror = () => {
      setStreaming(false);
      stream.close();
    };
    return () => stream.close();
  }, [baseNow, campaignId, endMs, startMs]);

  useEffect(() => {
    if (streaming) return;
    if (!Number.isFinite(baseNow) || !Number.isFinite(startMs) || !Number.isFinite(endMs)) return;
    const timer = window.setInterval(() => {
      const elapsed = Date.now() - wallStart;
      const next = Math.min(endMs, Math.max(startMs, baseNow + elapsed * acceleration));
      setNow(next);
    }, 1000);
    return () => window.clearInterval(timer);
  }, [acceleration, baseNow, endMs, startMs, streaming, wallStart]);

  return Number.isFinite(now) ? now : undefined;
}

function replayAcceleration(value?: string) {
  if (!value) return 60;
  const match = value.match(/(\d+(?:\.\d+)?)\s+simulated\s+hour/i);
  if (!match) return 60;
  return Number(match[1]) * 60;
}

function ExecutionProgress({ execution, heroGraph, currentTimeMs }: { execution: NonNullable<HeroGraphModel["execution"]>; heroGraph: HeroGraphModel; currentTimeMs?: number }) {
  const livePercent = replayPercent(heroGraph, currentTimeMs) ?? execution.percent_complete;
  return (
    <div className="execution-progress-panel" aria-label="Live accelerated campaign execution">
      <div className="execution-now-strip">
        <span>LIVE REPLAY</span>
        <strong>{livePercent.toFixed(0)}%</strong>
        <em>{execution.current_phase.replaceAll("_", " ")} / cycle {execution.current_cycle || "-"}</em>
        {currentTimeMs && <small>{new Date(currentTimeMs).toISOString().slice(0, 16).replace("T", " ")}</small>}
      </div>
      <div className="execution-progress-track">
        <i style={{ width: `${Math.max(0, Math.min(100, livePercent))}%` }} />
      </div>
      <div className="requirement-progress-grid">
        {(execution.requirement_progress ?? []).map((req) => (
          <div className="requirement-progress-card" key={req.id}>
            <span>{req.id}</span>
            <strong>{req.completed}/{req.target}</strong>
            <em>{req.label}</em>
            <div><i style={{ width: `${Math.max(0, Math.min(100, req.percent))}%` }} /></div>
            <small title={req.contributors.join(", ")}>{req.evidence_source}</small>
          </div>
        ))}
      </div>
    </div>
  );
}

function replayPercent(heroGraph: HeroGraphModel, currentTimeMs?: number) {
  if (!currentTimeMs) return undefined;
  const start = Date.parse(heroGraph.time_axis.start);
  const end = Date.parse(heroGraph.time_axis.end);
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return undefined;
  return ((currentTimeMs - start) / (end - start)) * 100;
}

function GraphWallCardView({
  card,
  cardRef,
  collapsed,
  pinned,
  height,
  currentTimeMs,
  hoverTimeMs,
  heroGraph,
  onHoverTime,
  onPeekTime,
  readoutMode,
  readoutTimeMs,
  timeRange,
  onTimeRange,
  tile,
  onToggle,
  onPinToggle,
  onHeightChange
}: {
  card: GraphWallCard;
  cardRef?: GraphTileCardRef;
  collapsed: boolean;
  pinned: boolean;
  height?: number;
  currentTimeMs?: number;
  hoverTimeMs?: number;
  heroGraph: HeroGraphModel;
  onHoverTime: (timeMs: number | undefined) => void;
  onPeekTime: (timeMs: number | undefined) => void;
  readoutMode: "live" | "crosshair" | "peek";
  readoutTimeMs?: number;
  timeRange: TimeRange;
  onTimeRange: (range: TimeRange) => void;
  tile?: GraphTile;
  onToggle: () => void;
  onPinToggle: () => void;
  onHeightChange: (height: number) => void;
}) {
  const renderKind = cardRef?.render_kind ?? card.render_kind ?? renderKindFor(card.kind);
  const visibleSignals = orderLegendSignals(cardRef?.signals ?? card.signals).slice(0, renderKind === "swimlane" ? 10 : 7);
  const readouts = tile ? legendReadouts(tile, visibleSignals, readoutTimeMs, currentTimeMs) : new Map<string, string>();
  const cardRefEl = useRef<HTMLElement | null>(null);
  const minHeight = renderKind === "swimlane" ? 190 : renderKind === "event_rail" ? 170 : 190;
  const maxHeight = card.id === "thermal_program" ? 760 : 560;
  const style = height ? ({ "--plot-height": `${height}px` } as CSSProperties) : undefined;
  const startResize = (event: ReactPointerEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    const startY = event.clientY;
    const startHeight = cardRefEl.current?.getBoundingClientRect().height ?? height ?? (card.id === "thermal_program" ? 560 : 260);
    const pointerID = event.pointerId;
    event.currentTarget.setPointerCapture?.(pointerID);
    const move = (moveEvent: PointerEvent) => {
      const next = Math.round(Math.max(minHeight, Math.min(maxHeight, startHeight + moveEvent.clientY - startY)));
      onHeightChange(next);
    };
    const stop = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", stop);
      window.removeEventListener("pointercancel", stop);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", stop, { once: true });
    window.addEventListener("pointercancel", stop, { once: true });
  };

  return (
    <article
      ref={cardRefEl}
      className={`graph-wall-card graph-card-${card.kind} graph-render-${renderKind} ${pinned ? "graph-card-pinned" : ""} ${collapsed ? "graph-card-collapsed" : ""}`}
      data-card-id={card.id}
      data-card-kind={card.kind}
      data-render-kind={renderKind}
      style={style}
    >
      <div className="graph-card-label-rail">
        <div className="graph-card-actions">
          <button className="graph-card-toggle" type="button" onClick={onToggle} aria-label={collapsed ? `Expand ${card.title}` : `Collapse ${card.title}`}>
            <span aria-hidden="true">{collapsed ? "+" : "-"}</span>
          </button>
          <button className={`graph-card-pin ${pinned ? "active" : ""}`} type="button" onClick={onPinToggle} aria-label={pinned ? `Unpin ${card.title}` : `Pin ${card.title}`}>
            <span aria-hidden="true">{pinned ? "●" : "○"}</span>
          </button>
        </div>
        <strong>{card.title}</strong>
      </div>
      {!collapsed && (
        <>
          <div className="graph-card-plot-shell">
            <div className="graph-card-inline-title">
              <strong>{card.title}</strong>
            </div>
            {!tile && <div className="graph-card-loading">Loading decimated tile...</div>}
            {tile && renderKind === "swimlane" && <SwimlaneTile tile={tile} heroGraph={heroGraph} currentTimeMs={currentTimeMs} hoverTimeMs={hoverTimeMs} readoutTimeMs={readoutTimeMs} timeRange={timeRange} />}
            {tile && renderKind === "event_rail" && <EventRailTile tile={tile} heroGraph={heroGraph} currentTimeMs={currentTimeMs} hoverTimeMs={hoverTimeMs} readoutTimeMs={readoutTimeMs} timeRange={timeRange} />}
            {tile && renderKind !== "swimlane" && renderKind !== "event_rail" && (
              <>
                {card.id === "thermal_program" && <HeroTopTimeAxis timeRange={timeRange} currentTimeMs={currentTimeMs} hoverTimeMs={hoverTimeMs} readoutTimeMs={readoutTimeMs} />}
                <UPlotTile
                  tile={tile}
                  heroGraph={heroGraph}
                  renderKind={renderKind}
                  currentTimeMs={currentTimeMs}
                  hoverTimeMs={hoverTimeMs}
                  onHoverTime={onHoverTime}
                  onPeekTime={onPeekTime}
                  timeRange={timeRange}
                  onTimeRange={onTimeRange}
                />
                {card.id === "thermal_program" && <HeroStateFooter tile={tile} heroGraph={heroGraph} currentTimeMs={currentTimeMs} timeRange={timeRange} />}
              </>
            )}
          </div>
          <div className="graph-card-legend-rail">
            {visibleSignals.map((signal) => (
              <span key={signal.id} title={`${signal.label} / ${signal.source_family}`}>
                <i style={{ background: colorForSignal(signal) }} />
                <b>{signal.label}</b>
                <em>{readouts.get(signal.id) ?? "-"}</em>
              </span>
            ))}
            {readoutTimeMs && <small>{readoutMode} {new Date(readoutTimeMs).toISOString().slice(5, 16).replace("T", " ")}</small>}
            {cardRef?.supports_y_zoom && <small>time + y zoom</small>}
          </div>
          <button className="graph-card-resize" type="button" aria-label={`Resize ${card.title}`} onPointerDown={startResize}>
            <span aria-hidden="true" />
          </button>
        </>
      )}
    </article>
  );
}

function HeroStateFooter({ tile, heroGraph, currentTimeMs, timeRange }: { tile: GraphTile; heroGraph: HeroGraphModel; currentTimeMs?: number; timeRange: TimeRange }) {
  const start = timeRange.start;
  const end = timeRange.end;
  const now = currentTimeMs ?? Date.parse(heroGraph.time_axis.now ?? heroGraph.execution?.now ?? "");
  const span = Math.max(1, end - start);
  const stateIDs = new Set(["trace.phase_enum", "trace.stability_reached", "trace.dwell_active", "trace.functional_gate_active"]);
  const states = tile.series.filter((series) => stateIDs.has(series.id));
  if (!states.length) return null;
  return (
    <div className="hero-state-footer" aria-label="Integrated test stage status">
      {states.map((series) => (
        <div className="hero-state-row" key={series.id}>
          <span>{series.label}</span>
          <div>
            {stateBlocks(series, start, span).map((block) => {
              const blockStart = start + (block.left / 100) * span;
              const blockEnd = blockStart + (block.width / 100) * span;
              if (Number.isFinite(now) && blockStart > now) return null;
              const clippedWidth = Number.isFinite(now) && blockEnd > now ? ((now - blockStart) / span) * 100 : block.width;
              return <i key={block.key} style={{ left: `${block.left}%`, width: `${Math.max(0.1, clippedWidth)}%`, background: block.value > 0 ? colorForSignal(series) : "rgba(64,82,99,0.45)" }} />;
            })}
          </div>
        </div>
      ))}
    </div>
  );
}

function UPlotTile({
  tile,
  heroGraph,
  renderKind,
  currentTimeMs,
  hoverTimeMs,
  onHoverTime,
  onPeekTime,
  timeRange,
  onTimeRange
}: {
  tile: GraphTile;
  heroGraph: HeroGraphModel;
  renderKind: string;
  currentTimeMs?: number;
  hoverTimeMs?: number;
  onHoverTime: (timeMs: number | undefined) => void;
  onPeekTime: (timeMs: number | undefined) => void;
  timeRange: TimeRange;
  onTimeRange: (range: TimeRange) => void;
}) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const onHoverTimeRef = useRef(onHoverTime);
  const onPeekTimeRef = useRef(onPeekTime);
  const hoverTimeRef = useRef(hoverTimeMs);
  const pointerInsideRef = useRef(false);
  const draggingRef = useRef(false);
  const onTimeRangeRef = useRef(onTimeRange);

  useEffect(() => {
    onHoverTimeRef.current = onHoverTime;
  }, [onHoverTime]);

  useEffect(() => {
    onPeekTimeRef.current = onPeekTime;
  }, [onPeekTime]);

  useEffect(() => {
    hoverTimeRef.current = hoverTimeMs;
  }, [hoverTimeMs]);

  useEffect(() => {
    onTimeRangeRef.current = onTimeRange;
  }, [onTimeRange]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const build = () => {
      host.replaceChildren();
      const rect = host.getBoundingClientRect();
      const width = Math.max(240, Math.floor(rect.width));
      const height = Math.max(42, Math.floor(rect.height));
      const { data, series, scales, axes } = uplotData(tile, currentTimeMs, width);
      let u: uPlot;
      const timeFromPointer = (event: PointerEvent | MouseEvent) => {
        const over = host.querySelector(".u-over");
        const rect = (over ?? host).getBoundingClientRect();
        const left = event.clientX - rect.left;
        return clampTime(u.posToVal(left, "x"), data[0] as number[]);
      };
      const opts = {
        width,
        height,
        ms: 1,
        sync: { key: `${tile.campaign_id}-shared-timebase` },
        cursor: { drag: { x: true, y: true } },
        legend: { show: false },
        scales: { x: { time: true, range: () => [timeRange.start, timeRange.end] as [number, number] }, ...scales },
        axes,
        series,
        hooks: {
          setScale: [
            (plot, scaleKey) => {
              if (scaleKey !== "x") return;
              const min = plot.scales.x.min;
              const max = plot.scales.x.max;
              if (typeof min !== "number" || typeof max !== "number" || !Number.isFinite(min) || !Number.isFinite(max) || max <= min) return;
              if (Math.abs(min - timeRange.start) < 2 && Math.abs(max - timeRange.end) < 2) return;
              onTimeRangeRef.current({ start: Math.round(min), end: Math.round(max) });
            }
          ],
          setCursor: [
            (plot) => {
              if (!pointerInsideRef.current || draggingRef.current) return;
              const left = plot.cursor.left;
              if (left == null) return;
              const next = clampTime(plot.posToVal(left, "x"), data[0] as number[]);
              if (Number.isFinite(next)) onHoverTimeRef.current(Math.round(next));
            }
          ],
          drawClear: [
            (plot) => {
              drawTileOverlays(plot, tile, heroGraph, currentTimeMs, hoverTimeRef.current, timeRange);
            }
          ]
        }
      } as uPlot.Options & { sync: { key: string } };
      u = new uPlot(opts as uPlot.Options, data, host);
      const movePointer = (event: PointerEvent) => {
        pointerInsideRef.current = true;
        const next = timeFromPointer(event);
        if (draggingRef.current) {
          event.preventDefault();
          if (Number.isFinite(next)) onPeekTimeRef.current(Math.round(next));
          return;
        }
        if (Number.isFinite(next)) onHoverTimeRef.current(Math.round(next));
      };
      const startDrag = (event: PointerEvent) => {
        event.preventDefault();
        pointerInsideRef.current = true;
        draggingRef.current = true;
        host.setPointerCapture?.(event.pointerId);
        const next = timeFromPointer(event);
        if (Number.isFinite(next)) onPeekTimeRef.current(Math.round(next));
      };
      const stopDrag = (event: PointerEvent) => {
        draggingRef.current = false;
        host.releasePointerCapture?.(event.pointerId);
        pointerInsideRef.current = false;
        onPeekTimeRef.current(undefined);
        onHoverTimeRef.current(undefined);
      };
      const clearHover = () => {
        pointerInsideRef.current = false;
        draggingRef.current = false;
        onPeekTimeRef.current(undefined);
        onHoverTimeRef.current(undefined);
      };
      host.addEventListener("pointermove", movePointer);
      host.addEventListener("pointerdown", startDrag);
      host.addEventListener("pointerup", stopDrag);
      host.addEventListener("pointercancel", stopDrag);
      host.addEventListener("mouseleave", clearHover);
      const originalDestroy = u.destroy.bind(u);
      u.destroy = () => {
        host.removeEventListener("pointermove", movePointer);
        host.removeEventListener("pointerdown", startDrag);
        host.removeEventListener("pointerup", stopDrag);
        host.removeEventListener("pointercancel", stopDrag);
        host.removeEventListener("mouseleave", clearHover);
        originalDestroy();
      };
      return u;
    };
    let plot = build();
    const resize = new ResizeObserver(() => {
      plot?.destroy();
      plot = build();
    });
    resize.observe(host);
    return () => {
      resize.disconnect();
      plot?.destroy();
    };
  }, [currentTimeMs, heroGraph, renderKind, tile, timeRange]);

  return <div className="graph-card-uplot" ref={hostRef} data-uplot-card={tile.card_id} />;
}

function SwimlaneTile({ tile, heroGraph, currentTimeMs, hoverTimeMs, readoutTimeMs, timeRange }: { tile: GraphTile; heroGraph: HeroGraphModel; currentTimeMs?: number; hoverTimeMs?: number; readoutTimeMs?: number; timeRange: TimeRange }) {
  const start = timeRange.start;
  const end = timeRange.end;
  const span = Math.max(1, end - start);
  const now = currentTimeMs ?? Date.parse(heroGraph.time_axis.now ?? heroGraph.execution?.now ?? "");
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), TIME_GRID_TICK_COUNT);
  return (
    <div className="tile-swimlane" data-swimlane-card={tile.card_id}>
      <div className="tile-time-plane">
        {ticks.map((tick) => <i className="tile-shared-gridline" key={tick.iso} style={{ left: `${tick.ratio * 100}%` }} />)}
        {tile.series.map((series) => (
          <div className="tile-swimlane-row" key={series.id}>
            <span>{series.label}</span>
            <div>
              {stateBlocks(series, start, span).map((block) => (
                <i key={block.key} style={{ left: `${block.left}%`, width: `${block.width}%`, background: block.value > 0 ? colorForSignal(series) : "rgba(64,82,99,0.35)" }}>
                  {block.width > 7 && <small>{blockLabel(block.label, block.value)}</small>}
                </i>
              ))}
              {Number.isFinite(now) && <b style={{ left: `${Math.max(0, Math.min(100, ((now - start) / span) * 100))}%` }} />}
              {Number.isFinite(hoverTimeMs) && <em style={{ left: `${Math.max(0, Math.min(100, (((hoverTimeMs as number) - start) / span) * 100))}%` }} />}
              {Number.isFinite(readoutTimeMs) && readoutTimeMs !== hoverTimeMs && <em className="peek" style={{ left: `${Math.max(0, Math.min(100, (((readoutTimeMs as number) - start) / span) * 100))}%` }} />}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function EventRailTile({ tile, heroGraph, currentTimeMs, hoverTimeMs, readoutTimeMs, timeRange }: { tile: GraphTile; heroGraph: HeroGraphModel; currentTimeMs?: number; hoverTimeMs?: number; readoutTimeMs?: number; timeRange: TimeRange }) {
  const start = timeRange.start;
  const end = timeRange.end;
  const span = Math.max(1, end - start);
  const now = currentTimeMs ?? Date.parse(heroGraph.time_axis.now ?? heroGraph.execution?.now ?? "");
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), TIME_GRID_TICK_COUNT);
  return (
    <div className="tile-event-rail" data-event-card={tile.card_id}>
      <div className="tile-time-plane">
        {ticks.map((tick) => <span className="tile-shared-gridline" key={tick.iso} style={{ left: `${tick.ratio * 100}%` }} />)}
        {(tile.markers ?? []).filter((marker) => inTimeRange(marker.timestamp, timeRange)).map((marker, index) => {
          const left = Math.max(0, Math.min(100, ((Date.parse(marker.timestamp) - start) / span) * 100));
          const color = markerColor(marker);
          return (
            <span
              className={`event-marker-wrap event-marker-${marker.kind ?? marker.role ?? "marker"}`}
              key={marker.id}
              style={{ left: `${left}%`, top: `${12 + (index % 5) * 21}px`, color }}
              title={`${marker.label} ${marker.timestamp}`}
            >
              <i className={`event-marker event-${marker.result ?? marker.kind}`} style={{ background: color }} />
              <strong>{shortGateLabel(marker.label)}</strong>
            </span>
          );
        })}
        {(tile.events ?? []).filter((event) => inTimeRange(event.timestamp, timeRange)).map((event, index) => {
          const left = Math.max(0, Math.min(100, ((Date.parse(event.timestamp) - start) / span) * 100));
          const color = eventColor(event.kind);
          return (
            <span
              className={`event-chip-wrap event-chip-${event.kind}`}
              key={event.id}
              style={{ left: `${left}%`, top: `${118 + (index % 3) * 19}px`, color }}
              title={`${event.label} ${event.timestamp}`}
            >
              <b className={`event-chip event-${event.kind}`} style={{ background: color }} />
              <strong>{shortGateLabel(event.label)}</strong>
            </span>
          );
        })}
        {Number.isFinite(now) && <em style={{ left: `${Math.max(0, Math.min(100, ((now - start) / span) * 100))}%` }} />}
        {Number.isFinite(hoverTimeMs) && <em className="hover" style={{ left: `${Math.max(0, Math.min(100, (((hoverTimeMs as number) - start) / span) * 100))}%` }} />}
        {Number.isFinite(readoutTimeMs) && readoutTimeMs !== hoverTimeMs && <em className="peek" style={{ left: `${Math.max(0, Math.min(100, (((readoutTimeMs as number) - start) / span) * 100))}%` }} />}
      </div>
    </div>
  );
}

function SharedTimeAxis({
  fullRange,
  timeRange,
  currentTimeMs,
  hoverTimeMs,
  peekTimeMs,
  plotBounds,
  onTimeRange
}: {
  fullRange: TimeRange;
  timeRange: TimeRange;
  currentTimeMs?: number;
  hoverTimeMs?: number;
  peekTimeMs?: number;
  plotBounds?: { left: number; right: number };
  onTimeRange: (range: TimeRange) => void;
}) {
  const ticks = timeTicks(new Date(timeRange.start).toISOString(), new Date(timeRange.end).toISOString(), TIME_GRID_TICK_COUNT);
  const start = timeRange.start;
  const end = timeRange.end;
  const now = currentTimeMs;
  const nowRatio = typeof now === "number" && Number.isFinite(now) ? Math.max(0, Math.min(1, (now - start) / Math.max(1, end - start))) : undefined;
  const spanHours = Math.max(0, (end - start) / 3_600_000);
  const fullSpan = Math.max(1, fullRange.end - fullRange.start);
  const viewSpan = Math.max(1, timeRange.end - timeRange.start);
  const isZoomed = viewSpan < fullSpan * 0.995;
  const minSpan = Math.max(60_000, fullSpan / 600);
  const axisStyle = plotBounds ? ({
    "--time-axis-left": `${plotBounds.left}px`,
    "--time-axis-right": `${plotBounds.right}px`,
  } as CSSProperties) : undefined;
  const zoomBy = (factor: number) => {
    const nextSpan = Math.max(minSpan, Math.min(fullSpan, viewSpan * factor));
    const center = (timeRange.start + timeRange.end) / 2;
    onTimeRange(clampRange({ start: Math.round(center - nextSpan / 2), end: Math.round(center + nextSpan / 2) }, fullRange, minSpan));
  };
  const setScroll = (value: number) => {
    const maxOffset = Math.max(0, fullSpan - viewSpan);
    const offset = (Number(value) / 1000) * maxOffset;
    onTimeRange({ start: Math.round(fullRange.start + offset), end: Math.round(fullRange.start + offset + viewSpan) });
  };
  const scrollValue = Math.round(((timeRange.start - fullRange.start) / Math.max(1, fullSpan - viewSpan)) * 1000);
  return (
    <div className="operator-shared-time-axis" aria-label="Shared graph time axis" style={axisStyle}>
      <span className="time-axis-label">TIME</span>
      <TimeAxisTrack ticks={ticks} start={start} end={end} nowRatio={nowRatio} hoverTimeMs={hoverTimeMs} peekTimeMs={peekTimeMs} />
      <div className="time-axis-controls">
        <span>{spanHours.toFixed(spanHours >= 24 ? 0 : 1)} h</span>
        <small>zoom</small>
        <button type="button" onClick={() => zoomBy(1.35)} aria-label="Zoom out">-</button>
        <button type="button" onClick={() => zoomBy(0.72)} aria-label="Zoom in">+</button>
        <button type="button" disabled={!isZoomed} onClick={() => onTimeRange(fullRange)}>full</button>
      </div>
      <label className="time-axis-scrollbar">
        <small>scroll</small>
        <input type="range" min="0" max="1000" step="1" disabled={!isZoomed} value={Math.max(0, Math.min(1000, scrollValue))} onChange={(event) => setScroll(Number(event.currentTarget.value))} />
      </label>
    </div>
  );
}

function HeroTopTimeAxis({ timeRange, currentTimeMs, hoverTimeMs, readoutTimeMs }: { timeRange: TimeRange; currentTimeMs?: number; hoverTimeMs?: number; readoutTimeMs?: number }) {
  const start = timeRange.start;
  const end = timeRange.end;
  const nowRatio = typeof currentTimeMs === "number" && Number.isFinite(currentTimeMs) ? Math.max(0, Math.min(1, (currentTimeMs - start) / Math.max(1, end - start))) : undefined;
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), TIME_GRID_TICK_COUNT);
  return (
    <div className="hero-top-time-axis" aria-label="Hero graph top time axis">
      <TimeAxisTrack ticks={ticks} start={start} end={end} nowRatio={nowRatio} hoverTimeMs={hoverTimeMs} peekTimeMs={readoutTimeMs !== hoverTimeMs ? readoutTimeMs : undefined} compact />
    </div>
  );
}

function TimeAxisTrack({ ticks, start, end, nowRatio, hoverTimeMs, peekTimeMs, compact }: { ticks: ReturnType<typeof timeTicks>; start: number; end: number; nowRatio?: number; hoverTimeMs?: number; peekTimeMs?: number; compact?: boolean }) {
  return (
    <div className={`time-axis-track ${compact ? "time-axis-track-compact" : ""}`}>
      {nowRatio !== undefined && <i className="time-axis-elapsed" style={{ width: `${nowRatio * 100}%` }} />}
      {nowRatio !== undefined && <b className="time-axis-now" style={{ left: `${nowRatio * 100}%` }} title="Current replay time" />}
      {peekTimeMs !== undefined && <b className="time-axis-peek" style={{ left: `${Math.max(0, Math.min(100, ((peekTimeMs - start) / Math.max(1, end - start)) * 100))}%` }} title="Drag peek time" />}
      {hoverTimeMs !== undefined && <b className="time-axis-hover" style={{ left: `${Math.max(0, Math.min(100, ((hoverTimeMs - start) / Math.max(1, end - start)) * 100))}%` }} />}
      {ticks.map((tick) => (
        <span className="time-axis-tick" style={{ left: `${tick.ratio * 100}%` }} key={tick.iso}>
          <i />
          <em>{tick.label}</em>
        </span>
      ))}
    </div>
  );
}

function clampRange(range: TimeRange, fullRange: TimeRange, minSpan: number): TimeRange {
  const fullSpan = Math.max(1, fullRange.end - fullRange.start);
  const span = Math.max(minSpan, Math.min(fullSpan, range.end - range.start));
  let start = range.start;
  let end = range.start + span;
  if (start < fullRange.start) {
    start = fullRange.start;
    end = start + span;
  }
  if (end > fullRange.end) {
    end = fullRange.end;
    start = end - span;
  }
  return { start: Math.round(start), end: Math.round(end) };
}

type UPlotBuild = {
  data: uPlot.AlignedData;
  series: uPlot.Series[];
  scales: Record<string, uPlot.Scale>;
  axes: uPlot.Axis[];
};

function uplotData(tile: GraphTile, currentTimeMs?: number, viewportWidth = 900): UPlotBuild {
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

function seriesDrawOrder(a: TileSeries, b: TileSeries) {
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

function lineWidthFor(role: string) {
  if (role === "command") return 1.55;
  if (role === "ghost") return 0.9;
  if (role === "acceptance_band") return 0.75;
  if (role === "counter" || role === "source_quality") return 1.05;
  return 0.85;
}

function sharedTimeGrid(tile: GraphTile, tileSeries: TileSeries[]): number[] {
  const start = Date.parse(tile.t0);
  const end = Date.parse(tile.t1);
  const finiteTimes = tileSeries
    .flatMap((series) => (series.points ?? []).map((point) => Date.parse(point.timestamp)))
    .filter(Number.isFinite);
  const t0 = Number.isFinite(start) ? start : Math.min(...finiteTimes);
  const t1 = Number.isFinite(end) ? end : Math.max(...finiteTimes);
  if (!Number.isFinite(t0) || !Number.isFinite(t1) || t1 <= t0) {
    return Array.from(new Set(finiteTimes)).sort((a, b) => a - b);
  }
  return Array.from(new Set([start, end, ...finiteTimes])).filter(Number.isFinite).sort((a, b) => a - b);
}

function viewportSeries(tile: GraphTile, series: TileSeries, viewportWidth: number): TileSeries {
  const points = series.points ?? [];
  if (points.length < 4 || series.step || series.render_kind === "counter" || series.kind === "counter") return series;
  const budget = Math.max(180, Math.min(points.length, Math.round(viewportWidth * 1.65)));
  if (points.length <= budget) return series;
  return { ...series, points: lttb(points, budget, (value) => decimationValue(tile, series, value)) };
}

function lttb(points: TileSeries["points"], threshold: number, yValue: (value: number) => number): TileSeries["points"] {
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

function decimationValue(tile: GraphTile, series: TileSeries, value: number) {
  if (series.axis_id === "pressure_mbar" && tile.card_id === "thermal_program") return pressureHeroRailDegC(value);
  if (series.axis_id === "pressure_mbar" || series.axis_id === "pressure_rate") return value > 0 ? Math.log10(value) : Number.NaN;
  return value;
}

function scaleForSeries(tile: GraphTile, series: TileSeries): string {
  if (series.axis_id === "pressure_mbar" && tile.card_id === "thermal_program") return "temperature_c";
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

function buildScales(scaleKeys: Set<string>): Record<string, uPlot.Scale> {
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

function paddedRange(minPad: number, clamp?: [number, number]): uPlot.Range.Function {
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

function buildAxes(scaleKeys: Set<string>, tile: GraphTile): uPlot.Axis[] {
  const leftAxisSize = 54;
  const rightAxisSize = 58;
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

function logScale(scale: string) {
  return scale === "pressure_log" || scale === "pressure_rate_log";
}

function logSplits(min: number, max: number) {
  if (!Number.isFinite(min) || !Number.isFinite(max) || max <= 0 || max <= min) return [];
  const first = Math.ceil(Math.log10(Math.max(min, 1e-12)));
  const last = Math.floor(Math.log10(max));
  const values: number[] = [];
  for (let exp = first; exp <= last; exp += 1) values.push(Math.pow(10, exp));
  return values;
}

function ySplits(min: number, max: number) {
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

function axisLabel(scale: string, tile: GraphTile) {
  if (scale === "temperature_c") return tile.card_id === "thermal_program" && hasPressure(tile) ? "degC + pressure rail" : "degC";
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

function hasPressure(tile: GraphTile) {
  return tile.series.some((series) => series.axis_id === "pressure_mbar");
}

function displayValue(tile: GraphTile, series: TileSeries, value: number) {
  if (series.axis_id === "pressure_mbar" && tile.card_id === "thermal_program") return pressureHeroRailDegC(value);
  if (series.axis_id === "pressure_mbar" || series.axis_id === "pressure_rate") return value > 0 ? value : Number.NaN;
  return value;
}

function pressureHeroRailDegC(mbar: number) {
  const minLog = Math.log10(0.00000001);
  const maxLog = Math.log10(1013.25);
  const ratio = (Math.log10(Math.max(0.00000001, Math.min(1013.25, mbar))) - minLog) / (maxLog - minLog);
  return -82 + ratio * 104;
}

function resampleSeries(tile: GraphTile, series: TileSeries, xValues: number[], currentTimeMs?: number): Array<number | null> {
  const points = [...(series.points ?? [])]
    .map((point) => ({ t: Date.parse(point.timestamp), v: displayValue(tile, series, point.value) }))
    .filter((point) => Number.isFinite(point.t) && Number.isFinite(point.v))
    .sort((a, b) => a.t - b.t);
  if (!points.length) return xValues.map(() => null);

  const stepped = series.step || series.render_kind === "counter" || series.kind === "counter" || series.render_kind === "swimlane";
  const isFutureVisible = series.role === "ghost";
  let cursor = 0;
  return xValues.map((x) => {
    if (Number.isFinite(currentTimeMs) && x > (currentTimeMs as number) && !isFutureVisible) return null;
    while (cursor + 1 < points.length && points[cursor + 1].t <= x) cursor += 1;
    const current = points[cursor];
    const next = points[Math.min(cursor + 1, points.length - 1)];
    if (x < points[0].t || x > points[points.length - 1].t) return null;
    if (stepped || next.t === current.t) return current.v;
    const ratio = (x - current.t) / (next.t - current.t);
    return current.v + (next.v - current.v) * Math.max(0, Math.min(1, ratio));
  });
}

function drawTileOverlays(plot: uPlot, tile: GraphTile, heroGraph: HeroGraphModel, currentTimeMs?: number, hoverTimeMs?: number, timeRange?: TimeRange) {
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
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), TIME_GRID_TICK_COUNT);
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
    const vacuumBand = tile.campaign_id === "tvac_qualification" && (bandKind.includes("vacuum") || tile.card_id.includes("pressure"));
    ctx.fillStyle = vacuumBand ? "rgba(59,130,246,0.09)" : bandKind.includes("cold") ? "rgba(61,133,198,0.12)" : "rgba(198,119,61,0.11)";
    ctx.fillRect(x, top, Math.max(1, x2 - x), height);
  });
  (tile.markers ?? []).forEach((marker) => {
    const markerTime = Date.parse(marker.timestamp);
    if (!Number.isFinite(markerTime)) return;
    const x = left + ((markerTime - start) / span) * width;
    if (x < left || x > left + width) return;
    const color = markerColor(marker);
    const attachedMarker = marker.kind === "functional_gate" || marker.kind === "stability" || marker.kind === "stability_achieved";
    const anchor = attachedMarker ? markerAnchor(plot, tile, markerTime, top, height) : null;
    const anchorY = anchor?.y ?? top + 10;
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    ctx.lineWidth = attachedMarker ? 1.6 : 1.1;
    ctx.setLineDash(marker.role === "interlock" ? [5, 4] : []);
    ctx.beginPath();
    ctx.moveTo(x, attachedMarker ? Math.max(top + 2, anchorY - 42) : top + 2);
    ctx.lineTo(x, top + height - 2);
    ctx.stroke();
    ctx.setLineDash([]);
    if (attachedMarker) {
      ctx.beginPath();
      if (marker.kind === "functional_gate") {
        ctx.moveTo(x, anchorY);
        ctx.lineTo(x - 5, anchorY - 9);
        ctx.lineTo(x + 5, anchorY - 9);
        ctx.closePath();
      } else {
        ctx.arc(x, anchorY, 4.2, 0, Math.PI * 2);
      }
      ctx.fill();
      const label = shortGateLabel(marker.label);
      ctx.save();
      ctx.font = "850 14px system-ui, sans-serif";
      const metrics = ctx.measureText(label);
      const labelWidth = Math.max(42, metrics.width + 11);
      const labelX = Math.max(left + 8, Math.min(left + width - labelWidth - 8, x + 8));
      const labelY = Math.max(top + 24, Math.min(top + height - 16, anchorY - 14));
      ctx.translate(labelX, labelY);
      const nearRightEdge = x > left + width - 72;
      ctx.rotate(nearRightEdge ? -Math.PI / 12 : -Math.PI / 6);
      ctx.fillStyle = "rgba(2,6,11,0.92)";
      ctx.fillRect(-5, -16, labelWidth, 20);
      ctx.strokeStyle = color;
      ctx.lineWidth = 1;
      ctx.strokeRect(-5, -16, labelWidth, 20);
      ctx.fillStyle = marker.kind === "functional_gate" ? "#fff0a8" : "#c9ffef";
      ctx.shadowColor = "rgba(0,0,0,0.88)";
      ctx.shadowBlur = 5;
      ctx.fillText(label, 0, 0);
      ctx.restore();
    } else {
      ctx.beginPath();
      ctx.arc(x, top + 10, 3.2, 0, Math.PI * 2);
      ctx.fill();
    }
  });
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

function markerAnchor(plot: uPlot, tile: GraphTile, timeMs: number, top: number, height: number) {
  const anchorSeries = tile.series.find((series) => series.id === "trace.command.chamber")
    ?? tile.series.find((series) => series.role === "command" && (series.points ?? []).length)
    ?? tile.series.find((series) => series.role === "ghost" && (series.points ?? []).length);
  if (!anchorSeries) return null;
  const raw = rawValueAt(anchorSeries, timeMs);
  if (raw === undefined) return null;
  const scale = scaleForSeries(tile, anchorSeries);
  const y = plot.valToPos(displayValue(tile, anchorSeries, raw), scale);
  if (!Number.isFinite(y)) return null;
  return { y: Math.max(top + 12, Math.min(top + height - 10, y)) };
}

function markerColor(marker: { role?: string; result?: string; kind?: string }) {
  if (marker.role === "interlock" || marker.result === "fail") return "rgba(255,49,95,0.96)";
  if (marker.role === "evidence") return "rgba(176,121,255,0.96)";
  if (marker.kind === "functional_gate") return "rgba(255,176,0,0.98)";
  if (marker.kind === "stability" || marker.kind === "stability_achieved" || marker.result === "pass") return "rgba(0,214,163,0.96)";
  return "rgba(49,214,255,0.95)";
}

function shortGateLabel(label: string) {
  return label
    .replace(/^Stable\s+/i, "STABLE ")
    .replace(/\s+confirmed$/i, "")
    .replace(/^Cycle\s+/i, "C")
    .replace(/\s+dwell\s+functional\s+test/i, " FT")
    .replace(/\s+functional\s+test/i, " FT")
    .slice(0, 18);
}

function legendReadouts(tile: GraphTile, visibleSignals: Array<{ id: string; label: string }>, timeMs?: number, currentTimeMs?: number) {
  const readouts = new Map<string, string>();
  if (!timeMs) return readouts;
  const visible = new Set(visibleSignals.map((signal) => signal.id));
  tile.series.forEach((series) => {
    if (!visible.has(series.id)) return;
    if (Number.isFinite(timeMs) && Number.isFinite(currentTimeMs) && (timeMs as number) > (currentTimeMs as number) && series.role !== "ghost") return;
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

function clampTime(timeMs: number, domain: number[]) {
  if (!Number.isFinite(timeMs) || !domain.length) return timeMs;
  const first = domain[0];
  const last = domain[domain.length - 1];
  if (!Number.isFinite(first) || !Number.isFinite(last)) return timeMs;
  return Math.max(first, Math.min(last, timeMs));
}

function rawValueAt(series: TileSeries, timeMs: number) {
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

function stateAt(series: TileSeries, timeMs: number) {
  const span = series.spans?.find((candidate) => {
    const start = Date.parse(candidate.start);
    const end = Date.parse(candidate.end);
    return Number.isFinite(start) && Number.isFinite(end) && timeMs >= start && timeMs <= end;
  });
  return span?.label ?? span?.state ?? (span?.value !== undefined ? String(span.value) : undefined);
}

function formatLegendValue(series: TileSeries, value: number) {
  const unit = series.unit || unitForAxis(series.axis_id);
  if (series.axis_id === "pressure_mbar") return `${formatPressure(value)} mbar`;
  if (series.axis_id === "pressure_rate") return `${formatScientific(value)} mbar/min`;
  if (series.axis_id === "counter") return `${Math.round(value).toLocaleString()}`;
  if (series.axis_id === "percent") return `${value.toFixed(0)}%`;
  if (unit === "degC") return `${value.toFixed(1)} degC`;
  if (unit === "W") return `${value.toFixed(1)} W`;
  if (unit === "ms") return `${value.toFixed(1)} ms`;
  if (unit === "bar") return `${value.toFixed(2)} bar`;
  return `${Number.isInteger(value) ? value.toFixed(0) : value.toFixed(2)}${unit ? ` ${unit}` : ""}`;
}

function formatScientific(value: number) {
  if (!Number.isFinite(value)) return "";
  if (value === 0) return "0";
  return value.toExponential(2).replace("e", "E");
}

function formatPressure(value: number) {
  if (value <= 0) return "0";
  if (value < 0.001 || value >= 1000) return value.toExponential(2);
  if (value < 1) return value.toPrecision(3);
  return value.toFixed(value < 10 ? 2 : 1);
}

function unitForAxis(axisID?: string) {
  if (axisID === "temperature_c") return "degC";
  if (axisID === "pressure_mbar") return "mbar";
  if (axisID === "power_w" || axisID === "heat_flux_w") return "W";
  if (axisID === "bus_ms") return "ms";
  if (axisID === "pressure_bar") return "bar";
  if (axisID === "pressure_rate") return "mbar/min";
  if (axisID === "percent") return "%";
  return "";
}

function stateBlocks(series: TileSeries, start: number, span: number) {
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

function inTimeRange(timestamp: string, range: TimeRange) {
  const t = Date.parse(timestamp);
  return Number.isFinite(t) && t >= range.start && t <= range.end;
}

function renderKindFor(kind: string) {
  if (kind === "state") return "swimlane";
  if (kind === "event") return "event_rail";
  if (kind === "counter") return "counter";
  return "line";
}

function palette(index: number) {
  return distinctivePalette[index % distinctivePalette.length];
}

const distinctivePalette = [
  "#31d6ff",
  "#ffb000",
  "#ff5c93",
  "#00d084",
  "#b079ff",
  "#ff7a35",
  "#44e0b7",
  "#7aa2ff",
  "#f5d742",
  "#f15bb5",
  "#2ec4b6",
  "#e76f51",
  "#9bff70",
  "#00a6fb",
  "#ffd166",
  "#ef476f",
];

function paletteForID(id: string, fallbackIndex: number) {
  let hash = fallbackIndex + 17;
  for (let i = 0; i < id.length; i += 1) hash = ((hash << 5) - hash + id.charCodeAt(i)) | 0;
  return distinctivePalette[Math.abs(hash) % distinctivePalette.length];
}

function timeTicks(startISO: string, endISO: string, count: number) {
  const start = Date.parse(startISO);
  const end = Date.parse(endISO);
  const span = Math.max(1, end - start);
  const target = Math.max(10, Math.min(20, count || TIME_GRID_TICK_COUNT));
  const step = chooseTickStep(span, target);
  const first = Math.ceil(start / step) * step;
  const ticks: Array<{ iso: string; ratio: number; label: string }> = [];
  for (let t = first; t <= end && ticks.length < 24; t += step) {
    if (t < start) continue;
    const d = new Date(t);
    ticks.push({ iso: d.toISOString(), ratio: (t - start) / span, label: tickLabel(d, step) });
  }
  if (!ticks.length || ticks[0].ratio > 0.02) ticks.unshift({ iso: new Date(start).toISOString(), ratio: 0, label: tickLabel(new Date(start), step) });
  const last = ticks[ticks.length - 1];
  if (last && last.ratio < 0.98) ticks.push({ iso: new Date(end).toISOString(), ratio: 1, label: tickLabel(new Date(end), step) });
  return ticks.filter((tick, index, all) => index === 0 || tick.iso !== all[index - 1].iso);
}

function chooseTickStep(spanMs: number, targetCount: number) {
  const targetStep = spanMs / Math.max(1, targetCount - 1);
  const steps = [
    5 * 60_000,
    10 * 60_000,
    15 * 60_000,
    30 * 60_000,
    60 * 60_000,
    3 * 60 * 60_000,
    6 * 60 * 60_000,
    12 * 60 * 60_000,
    24 * 60 * 60_000,
    2 * 24 * 60 * 60_000,
    7 * 24 * 60 * 60_000,
    14 * 24 * 60 * 60_000,
    30 * 24 * 60 * 60_000,
  ];
  return steps.find((step) => step >= targetStep) ?? steps[steps.length - 1];
}

function tickLabel(date: Date, stepMs: number) {
  const time = date.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  if (stepMs < 24 * 60 * 60_000) return time;
  return `${date.toLocaleDateString(undefined, { month: "short", day: "2-digit" })} ${time}`;
}

function scheduleTileWork(work: () => void, delayMs: number) {
  window.setTimeout(() => {
    work();
  }, delayMs);
}
