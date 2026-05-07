import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
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
  "trace.command.chamber": "#ffe66d",
  "trace.ghost.profile": "#8fa2ad",
  "trace.acceptance.temperature": "#4ee28a",
  "trace.actual.chamber_air": "#39d5ff",
  "trace.context.chamber_air": "#39d5ff",
  "trace.table_loop": "#ff9f43",
  "trace.interface": "#ff9f43",
  "trace.shroud": "#c084fc",
  "trace.shroud_inlet": "#7dd3fc",
  "trace.shroud_outlet": "#f0abfc",
  "trace.dut_temp_a": "#ff5f7e",
  "trace.dut_temp_b": "#2dd4bf",
  "trace.tvac_pressure": "#3b82f6",
  "trace.actual.tvac_pressure": "#3b82f6",
  "trace.tvac_pressure_target": "#93c5fd",
  "trace.tvac_outgassing": "#38bdf8",
  "trace.tvac_virtual_leak": "#60a5fa",
  "trace.tvac_roughing_pump": "#2563eb",
  "trace.tvac_turbo_pump": "#0ea5e9",
  "trace.tvac_pump_removal": "#1d4ed8",
  "trace.tvac_volatile_inventory": "#bae6fd",
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
};

function colorForSignal(signal: Pick<TileSeries, "id" | "role" | "render_kind" | "kind"> | { id: string; role: string; kind?: string }, index = 0) {
  const kind = "kind" in signal ? signal.kind : ("render_kind" in signal ? signal.render_kind : undefined);
  return signalColors[signal.id] ?? roleColors[signal.role] ?? (kind ? roleColors[kind] : undefined) ?? palette(index);
}

export function OperatorGraphWall({ campaignId, wall, heroGraph, afterProgress }: Props) {
  const [manifest, setManifest] = useState<GraphTileManifest | null>(null);
  const [tiles, setTiles] = useState<Record<string, GraphTile>>({});
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [hoverTimeMs, setHoverTimeMs] = useState<number | undefined>(undefined);
  const [peekTimeMs, setPeekTimeMs] = useState<number | undefined>(undefined);
  const fullTimeRange = useMemo(() => graphTimeRange(heroGraph), [heroGraph]);
  const [viewRange, setViewRange] = useState<TimeRange>(fullTimeRange);
  const requestedTiles = useRef<Set<string>>(new Set());
  const loadGeneration = useRef(0);
  const execution = heroGraph.execution;
  const currentTimeMs = useAnimatedReplayTime(heroGraph);
  const readoutTimeMs = peekTimeMs ?? hoverTimeMs ?? currentTimeMs;
  const readoutMode = peekTimeMs !== undefined ? "peek" : hoverTimeMs !== undefined ? "crosshair" : "live";

  useEffect(() => {
    let cancelled = false;
    loadGeneration.current += 1;
    setViewRange(fullTimeRange);
    setManifest(null);
    setTiles({});
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

  return (
    <div className="operator-graph-wall" data-graph-wall-version={wall.graph_version} data-tile-backed="true">
      {wall.sections.map((section) => (
        <section className="operator-wall-section" key={section.id} data-section-id={section.id}>
          {!(section.id === firstSectionID && primaryCardID) && <div className="operator-wall-section-title">
            <strong>{section.title}</strong>
            <span>{section.transport} / {section.direction}</span>
          </div>}
          <div className="operator-wall-cards">
            {section.cards.map((card) => {
              const isPrimary = card.id === primaryCardID;
              const cardRef = manifestCards.get(card.id);
              const isCollapsed = collapsed[card.id] ?? false;
              return (
                <Fragment key={card.id}>
                  <GraphWallCardView
                    card={card}
                    cardRef={cardRef}
                    collapsed={isCollapsed}
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
                  />
                  {isPrimary && execution && <ExecutionProgress execution={execution} heroGraph={heroGraph} currentTimeMs={currentTimeMs} />}
                </Fragment>
              );
            })}
          </div>
        </section>
      ))}
      {afterProgress}
      <div className="operator-wall-meta">
        <span>{manifest ? "tile manifest ready" : "loading tile manifest"}</span>
        <span>{wall.graph_version}</span>
        <span>{wall.source_mode}</span>
        <span>{wall.time_range.mode}</span>
        <span>{wall.tile_policy.shared_timebase_required ? "shared timebase" : "local timebase"}</span>
        {execution && <span>{execution.acceleration}</span>}
      </div>
      <SharedTimeAxis
        fullRange={fullTimeRange}
        timeRange={viewRange}
        currentTimeMs={currentTimeMs}
        hoverTimeMs={hoverTimeMs}
        peekTimeMs={peekTimeMs}
        onTimeRange={setViewRange}
      />
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

function useAnimatedReplayTime(heroGraph: HeroGraphModel) {
  const startMs = Date.parse(heroGraph.time_axis.start);
  const endMs = Date.parse(heroGraph.time_axis.end);
  const baseNow = Date.parse(heroGraph.time_axis.now ?? heroGraph.execution?.now ?? "");
  const acceleration = replayAcceleration(heroGraph.execution?.acceleration);
  const [wallStart, setWallStart] = useState(() => Date.now());
  const [now, setNow] = useState(baseNow);

  useEffect(() => {
    setWallStart(Date.now());
    setNow(baseNow);
  }, [baseNow, heroGraph.id]);

  useEffect(() => {
    if (!Number.isFinite(baseNow) || !Number.isFinite(startMs) || !Number.isFinite(endMs)) return;
    const timer = window.setInterval(() => {
      const elapsed = Date.now() - wallStart;
      const next = Math.min(endMs, Math.max(startMs, baseNow + elapsed * acceleration));
      setNow(next);
    }, 1000);
    return () => window.clearInterval(timer);
  }, [acceleration, baseNow, endMs, startMs, wallStart]);

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
        {execution.requirement_progress.map((req) => (
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
  onToggle
}: {
  card: GraphWallCard;
  cardRef?: GraphTileCardRef;
  collapsed: boolean;
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
}) {
  const renderKind = cardRef?.render_kind ?? card.render_kind ?? renderKindFor(card.kind);
  const visibleSignals = (cardRef?.signals ?? card.signals).slice(0, renderKind === "swimlane" ? 10 : 7);
  const pointCount = tile?.diagnostics.point_count ?? 0;
  const readouts = tile ? legendReadouts(tile, visibleSignals, readoutTimeMs, currentTimeMs) : new Map<string, string>();

  return (
    <article
      className={`graph-wall-card graph-card-${card.kind} graph-render-${renderKind} ${card.placement.pinned ? "graph-card-pinned" : ""} ${collapsed ? "graph-card-collapsed" : ""}`}
      data-card-id={card.id}
      data-card-kind={card.kind}
      data-render-kind={renderKind}
    >
      <div className="graph-card-label-rail">
        <button className="graph-card-toggle" type="button" onClick={onToggle} aria-label={collapsed ? `Expand ${card.title}` : `Collapse ${card.title}`}>
          <span aria-hidden="true">{collapsed ? "+" : "-"}</span>
        </button>
        <strong>{card.title}</strong>
        <span>{renderKind} / {card.unit ?? card.axis_policy}</span>
        <small>{pointCount ? `${pointCount} tile points` : "backend tile pending"}</small>
      </div>
      {!collapsed && (
        <>
          <div className="graph-card-plot-shell">
            <div className="graph-card-inline-title">
              <strong>{card.title}</strong>
              <span>{renderKind} / {card.unit ?? card.axis_policy}</span>
            </div>
            {!tile && <div className="graph-card-loading">Loading decimated tile...</div>}
            {tile && renderKind === "swimlane" && <SwimlaneTile tile={tile} heroGraph={heroGraph} currentTimeMs={currentTimeMs} hoverTimeMs={hoverTimeMs} readoutTimeMs={readoutTimeMs} timeRange={timeRange} />}
            {tile && renderKind === "event_rail" && <EventRailTile tile={tile} heroGraph={heroGraph} currentTimeMs={currentTimeMs} hoverTimeMs={hoverTimeMs} readoutTimeMs={readoutTimeMs} timeRange={timeRange} />}
            {tile && renderKind !== "swimlane" && renderKind !== "event_rail" && (
              <>
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
  const hoverTimeRef = useRef(hoverTimeMs);
  const pointerInsideRef = useRef(false);
  const draggingRef = useRef(false);
  const onTimeRangeRef = useRef(onTimeRange);

  useEffect(() => {
    onHoverTimeRef.current = onHoverTime;
  }, [onHoverTime]);

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
              if (!pointerInsideRef.current) return;
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
      const markHover = (event: MouseEvent) => {
        pointerInsideRef.current = true;
        const next = timeFromPointer(event);
        if (draggingRef.current) {
          if (Number.isFinite(next)) onPeekTime(Math.round(next));
          return;
        }
        if (Number.isFinite(next)) onHoverTimeRef.current(Math.round(next));
      };
      const startDrag = (event: PointerEvent) => {
        pointerInsideRef.current = true;
        draggingRef.current = true;
        host.setPointerCapture?.(event.pointerId);
        const next = timeFromPointer(event);
        if (Number.isFinite(next)) onPeekTime(Math.round(next));
      };
      const stopDrag = (event: PointerEvent) => {
        draggingRef.current = false;
        host.releasePointerCapture?.(event.pointerId);
        pointerInsideRef.current = false;
        onPeekTime(undefined);
        onHoverTimeRef.current(undefined);
      };
      const clearHover = () => {
        pointerInsideRef.current = false;
        draggingRef.current = false;
        onPeekTime(undefined);
        onHoverTimeRef.current(undefined);
      };
      host.addEventListener("mousemove", markHover);
      host.addEventListener("pointerdown", startDrag);
      host.addEventListener("pointerup", stopDrag);
      host.addEventListener("mouseleave", clearHover);
      const originalDestroy = u.destroy.bind(u);
      u.destroy = () => {
        host.removeEventListener("mousemove", markHover);
        host.removeEventListener("pointerdown", startDrag);
        host.removeEventListener("pointerup", stopDrag);
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
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), 9);
  return (
    <div className="tile-swimlane" data-swimlane-card={tile.card_id}>
      {ticks.map((tick) => <i className="tile-shared-gridline" key={tick.iso} style={{ left: `${tick.ratio * 100}%` }} />)}
      {tile.series.map((series) => (
        <div className="tile-swimlane-row" key={series.id}>
          <span>{series.label}</span>
          <div>
            {stateBlocks(series, start, span).map((block) => (
              <i key={block.key} style={{ left: `${block.left}%`, width: `${block.width}%`, background: block.value > 0 ? colorForSignal(series) : "rgba(64,82,99,0.35)" }} />
            ))}
            {Number.isFinite(now) && <b style={{ left: `${Math.max(0, Math.min(100, ((now - start) / span) * 100))}%` }} />}
            {Number.isFinite(hoverTimeMs) && <em style={{ left: `${Math.max(0, Math.min(100, (((hoverTimeMs as number) - start) / span) * 100))}%` }} />}
            {Number.isFinite(readoutTimeMs) && readoutTimeMs !== hoverTimeMs && <em className="peek" style={{ left: `${Math.max(0, Math.min(100, (((readoutTimeMs as number) - start) / span) * 100))}%` }} />}
          </div>
        </div>
      ))}
    </div>
  );
}

function EventRailTile({ tile, heroGraph, currentTimeMs, hoverTimeMs, readoutTimeMs, timeRange }: { tile: GraphTile; heroGraph: HeroGraphModel; currentTimeMs?: number; hoverTimeMs?: number; readoutTimeMs?: number; timeRange: TimeRange }) {
  const start = timeRange.start;
  const end = timeRange.end;
  const span = Math.max(1, end - start);
  const now = currentTimeMs ?? Date.parse(heroGraph.time_axis.now ?? heroGraph.execution?.now ?? "");
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), 9);
  return (
    <div className="tile-event-rail" data-event-card={tile.card_id}>
      {ticks.map((tick) => <span className="tile-shared-gridline" key={tick.iso} style={{ left: `${tick.ratio * 100}%` }} />)}
      {(tile.markers ?? []).filter((marker) => inTimeRange(marker.timestamp, timeRange)).map((marker) => (
        <i
          className={`event-marker event-${marker.result ?? marker.kind}`}
          key={marker.id}
          style={{ left: `${Math.max(0, Math.min(100, ((Date.parse(marker.timestamp) - start) / span) * 100))}%` }}
          title={`${marker.label} ${marker.timestamp}`}
        />
      ))}
      {(tile.events ?? []).filter((event) => inTimeRange(event.timestamp, timeRange)).map((event) => (
        <b
          className={`event-chip event-${event.kind}`}
          key={event.id}
          style={{ left: `${Math.max(0, Math.min(100, ((Date.parse(event.timestamp) - start) / span) * 100))}%` }}
          title={`${event.label} ${event.timestamp}`}
        />
      ))}
      {Number.isFinite(now) && <em style={{ left: `${Math.max(0, Math.min(100, ((now - start) / span) * 100))}%` }} />}
      {Number.isFinite(hoverTimeMs) && <em className="hover" style={{ left: `${Math.max(0, Math.min(100, (((hoverTimeMs as number) - start) / span) * 100))}%` }} />}
      {Number.isFinite(readoutTimeMs) && readoutTimeMs !== hoverTimeMs && <em className="peek" style={{ left: `${Math.max(0, Math.min(100, (((readoutTimeMs as number) - start) / span) * 100))}%` }} />}
    </div>
  );
}

function SharedTimeAxis({
  fullRange,
  timeRange,
  currentTimeMs,
  hoverTimeMs,
  peekTimeMs,
  onTimeRange
}: {
  fullRange: TimeRange;
  timeRange: TimeRange;
  currentTimeMs?: number;
  hoverTimeMs?: number;
  peekTimeMs?: number;
  onTimeRange: (range: TimeRange) => void;
}) {
  const ticks = timeTicks(new Date(timeRange.start).toISOString(), new Date(timeRange.end).toISOString(), 7);
  const start = timeRange.start;
  const end = timeRange.end;
  const now = currentTimeMs;
  const nowRatio = typeof now === "number" && Number.isFinite(now) ? Math.max(0, Math.min(1, (now - start) / Math.max(1, end - start))) : undefined;
  const spanHours = Math.max(0, (end - start) / 3_600_000);
  const fullSpan = Math.max(1, fullRange.end - fullRange.start);
  const viewSpan = Math.max(1, timeRange.end - timeRange.start);
  const isZoomed = viewSpan < fullSpan * 0.995;
  const zoomPercent = Math.round((fullSpan / viewSpan) * 100);
  const minSpan = Math.max(60_000, fullSpan / 600);
  const setZoom = (value: number) => {
    const ratio = Number(value) / 100;
    const nextSpan = Math.max(minSpan, Math.min(fullSpan, fullSpan / Math.max(1, ratio)));
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
    <div className="operator-shared-time-axis" aria-label="Shared graph time axis">
      <span className="time-axis-label">TIME</span>
      <div className="time-axis-track">
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
      <div className="time-axis-controls">
        <span>{spanHours.toFixed(spanHours >= 24 ? 0 : 1)} h</span>
        <label>
          <small>zoom</small>
          <input type="range" min="100" max="60000" step="25" value={Math.max(100, Math.min(60000, zoomPercent))} onChange={(event) => setZoom(Number(event.currentTarget.value))} />
        </label>
        {isZoomed && (
          <label className="time-axis-scroll">
            <small>scroll</small>
            <input type="range" min="0" max="1000" step="1" value={Math.max(0, Math.min(1000, scrollValue))} onChange={(event) => setScroll(Number(event.currentTarget.value))} />
          </label>
        )}
        {isZoomed && <button type="button" onClick={() => onTimeRange(fullRange)}>full</button>}
      </div>
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
  const plottedSeries = tileSeries.map((series) => viewportSeries(series, viewportWidth));
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
    actual: 10,
    source_quality: 12,
    counter: 14,
    acceptance_band: 20,
    ghost: 30,
    command: 40,
    event: 50,
    interlock: 55,
    evidence: 60,
  };
  return (order[a.role] ?? 15) - (order[b.role] ?? 15);
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

function viewportSeries(series: TileSeries, viewportWidth: number): TileSeries {
  const points = series.points ?? [];
  if (points.length < 4 || series.step || series.render_kind === "counter" || series.kind === "counter") return series;
  const budget = Math.max(180, Math.min(points.length, Math.round(viewportWidth * 1.65)));
  if (points.length <= budget) return series;
  return { ...series, points: lttb(points, budget) };
}

function lttb(points: TileSeries["points"], threshold: number): TileSeries["points"] {
  if (!points || threshold >= points.length || threshold < 3) return points;
  const parsed = points
    .map((point) => ({ point, x: Date.parse(point.timestamp), y: point.value }))
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
    else if (key === "pressure_log") scales[key] = { range: (_u, _min, _max) => [-8.2, 3.2] };
    else if (key === "pressure_rate_log") scales[key] = { range: (_u, _min, _max) => [-8, 3] };
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
            : "percent";
  axes.push({
    show: true,
    scale: primary,
    stroke: "#7890a4",
    grid: { stroke: "rgba(83,112,140,0.26)", width: 1 },
    ticks: { stroke: "rgba(83,112,140,0.48)", width: 1, size: 4 },
    splits: (_u, _axisIdx, scaleMin, scaleMax) => ySplits(scaleMin, scaleMax),
    size: 46,
    label: axisLabel(primary, tile),
    labelSize: 12,
    labelGap: 0,
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
      size: 54,
      label: axisLabel(key, tile),
      labelSize: 12,
      labelGap: 0,
      values: key === "pressure_log" || key === "pressure_rate_log" ? (_u, vals) => vals.map((v) => `1e${Math.round(v)}`) : undefined,
    });
  });
  if (!extra.length) {
    axes.push({ show: false, side: 1, scale: primary, size: 54 });
  }
  return axes;
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
  if (series.axis_id === "pressure_mbar") return Math.log10(Math.max(0.00000001, value));
  if (series.axis_id === "pressure_rate") return Math.log10(Math.max(0.00000001, value));
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
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), 9);
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
    const color = marker.role === "interlock" ? "rgba(255,99,116,0.95)" : marker.role === "evidence" ? "rgba(184,166,255,0.95)" : "rgba(240,200,90,0.95)";
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    ctx.lineWidth = marker.kind === "functional_gate" ? 1.6 : 1.1;
    ctx.setLineDash(marker.role === "interlock" ? [5, 4] : []);
    ctx.beginPath();
    ctx.moveTo(x, top + 2);
    ctx.lineTo(x, top + height - 2);
    ctx.stroke();
    ctx.setLineDash([]);
    if (marker.kind === "functional_gate") {
      ctx.beginPath();
      ctx.moveTo(x, top + 4);
      ctx.lineTo(x - 4, top + 12);
      ctx.lineTo(x + 4, top + 12);
      ctx.closePath();
      ctx.fill();
      ctx.save();
      ctx.translate(x + 5, top + 30);
      ctx.rotate(-Math.PI / 4.5);
      ctx.font = "9px system-ui, sans-serif";
      ctx.fillText(shortGateLabel(marker.label), 0, 0);
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

function shortGateLabel(label: string) {
  return label
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
  if (series.axis_id === "counter") return `${Math.round(value).toLocaleString()}`;
  if (series.axis_id === "percent") return `${value.toFixed(0)}%`;
  if (unit === "degC") return `${value.toFixed(1)} degC`;
  if (unit === "W") return `${value.toFixed(1)} W`;
  if (unit === "ms") return `${value.toFixed(1)} ms`;
  if (unit === "bar") return `${value.toFixed(2)} bar`;
  return `${Number.isInteger(value) ? value.toFixed(0) : value.toFixed(2)}${unit ? ` ${unit}` : ""}`;
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
  return ["#56d6df", "#f0c85a", "#b8a6ff", "#8bd3a5", "#ff6374"][index % 5];
}

function timeTicks(startISO: string, endISO: string, count: number) {
  const start = Date.parse(startISO);
  const end = Date.parse(endISO);
  const span = Math.max(1, end - start);
  return Array.from({ length: count }, (_, index) => {
    const ratio = count === 1 ? 0 : index / (count - 1);
    const d = new Date(start + span * ratio);
    return { iso: d.toISOString(), ratio, label: `${d.toLocaleDateString(undefined, { month: "short", day: "2-digit" })} ${d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })}` };
  });
}

function scheduleTileWork(work: () => void, delayMs: number) {
  window.setTimeout(() => {
    work();
  }, delayMs);
}
