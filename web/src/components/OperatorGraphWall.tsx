import { useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties, PointerEvent as ReactPointerEvent, ReactNode } from "react";
import uPlot from "uplot";
import "uplot/dist/uPlot.min.css";
import { api } from "../api";
import type { GraphTile, GraphTileCardRef, GraphTileManifest, GraphWallCard, GraphWallModel, HeroGraphModel } from "../types";

// Sub-module imports
import { graphCardPriority, graphSectionPriority, tileCardPriority, orderLegendSignals, colorForSignal, blockLabel, eventColor } from "./tiles/visualPolicy";
import { clampRange, timeTicks, SharedTimeAxis, HeroTopTimeAxis, TimeAxisTrack } from "./tiles/timeAxis";
import type { TimeRange } from "./tiles/timeAxis";
import { legendReadouts, clampTime, shortGateLabel, markerColor } from "./tiles/markers";
import { uplotData, drawTileOverlays, stateBlocks, inTimeRange, renderKindFor, scaleForSeries } from "./tiles/uPlotAdapter";

export type { TimeRange };

type Props = {
  campaignId: string;
  wall: GraphWallModel;
  heroGraph: HeroGraphModel;
  afterProgress?: ReactNode;
};

function useViewportTickCount() {
  const [w, setW] = useState(() => typeof window !== "undefined" ? window.innerWidth : 1440);
  useEffect(() => {
    const onResize = () => setW(window.innerWidth);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);
  return w < 480 ? 6 : w < 820 ? 9 : 14;
}
const DAY_MS = 86_400_000;

export function OperatorGraphWall({ campaignId, wall, heroGraph, afterProgress }: Props) {
  const [manifest, setManifest] = useState<GraphTileManifest | null>(null);
  const [tiles, setTiles] = useState<Record<string, GraphTile>>({});
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [pinOverrides, setPinOverrides] = useState<Record<string, boolean>>({});
  const [cardHeights, setCardHeights] = useState<Record<string, number>>({});
  const [hoverTimeMs, setHoverTimeMs] = useState<number | undefined>(undefined);
  const [peekTimeMs, setPeekTimeMs] = useState<number | undefined>(undefined);
  const [timeAxisBounds, setTimeAxisBounds] = useState<{ left: number; right: number } | undefined>(undefined);
  const viewportWidth = useViewportWidth();
  const tickCount = useViewportTickCount();
  const fullTimeRange = useMemo(() => graphTimeRange(heroGraph), [heroGraph]);
  const defaultTimeRange = useMemo(() => defaultGraphTimeRange(campaignId, heroGraph, fullTimeRange, viewportWidth), [campaignId, heroGraph, fullTimeRange, viewportWidth]);
  const [viewRange, setViewRange] = useState<TimeRange>(defaultTimeRange);
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
    setViewRange(defaultTimeRange);
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
  }, [campaignId, defaultTimeRange]);

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
        const plot = frame.querySelector(".u-plot .u-over") as HTMLElement | null;
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
        tickCount={tickCount}
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
      <div className="operator-wall-meta" aria-label="Graph wall provenance">
        <span><b>Tiles</b><strong>{manifest ? "manifest ready" : "loading manifest"}</strong></span>
        <span><b>Contract</b><strong>{wall.graph_version}</strong></span>
        <span><b>Source</b><strong>{wall.source_mode}</strong></span>
        <span><b>Timebase</b><strong>{wall.time_range.mode}</strong></span>
        <span><b>Sync</b><strong>{wall.tile_policy.shared_timebase_required ? "shared" : "local"}</strong></span>
        {execution && <span><b>Replay</b><strong>{execution.acceleration}</strong></span>}
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

function defaultGraphTimeRange(campaignId: string, heroGraph: HeroGraphModel, fullRange: TimeRange, viewportWidth: number): TimeRange {
  const start = Date.parse(heroGraph.time_axis.default_window_start ?? "");
  const end = Date.parse(heroGraph.time_axis.default_window_end ?? "");
  let range = fullRange;
  if (Number.isFinite(start) && Number.isFinite(end) && end > start) {
    range = clampRange({ start, end }, fullRange, 60_000);
  }
  if (campaignId !== "command_center_fat") return range;
  const maxWindowDays = viewportWidth < 560 ? 5 : viewportWidth < 760 ? 7 : viewportWidth < 1180 ? 14 : 28;
  const maxWindowMs = maxWindowDays * DAY_MS;
  if (range.end - range.start <= maxWindowMs) return range;
  const now = Date.parse(heroGraph.time_axis.now ?? heroGraph.execution?.now ?? "");
  const center = Number.isFinite(now) ? now : range.start + (range.end - range.start) / 2;
  return clampRange({ start: center - maxWindowMs / 2, end: center + maxWindowMs / 2 }, fullRange, 60_000);
}

function useViewportWidth() {
  const [width, setWidth] = useState(() => (typeof window === "undefined" ? 1440 : window.innerWidth));
  useEffect(() => {
    const update = () => setWidth(window.innerWidth);
    update();
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
  }, []);
  return width;
}

export function useAnimatedReplayTime(campaignId: string, heroGraph: HeroGraphModel) {
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
    setStreaming(false);
  }, [campaignId]);

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

export function replayAcceleration(value?: string) {
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
  const heroTickCount = useViewportTickCount();
  const visibleSignals = orderLegendSignals(cardRef?.signals ?? card.signals).slice(0, renderKind === "swimlane" ? 10 : 7);
  const readouts = tile ? legendReadouts(tile, visibleSignals, readoutTimeMs, currentTimeMs) : new Map<string, string>();
  const cardRefEl = useRef<HTMLElement | null>(null);
  const isPrimary = card.role === "primary" || card.placement.pinned;
  const defaultPlotHeight = isPrimary ? 440 : renderKind === "swimlane" ? 180 : renderKind === "event_rail" ? 150 : 220;
  const minHeight = renderKind === "swimlane" ? 150 : renderKind === "event_rail" ? 120 : isPrimary ? 240 : 180;
  const maxHeight = renderKind === "event_rail" ? 360 : card.id === "thermal_program" ? 760 : 560;
  const style = height ? ({ "--plot-height": `${height}px` } as CSSProperties) : undefined;
  const startResize = (event: ReactPointerEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    const startY = event.clientY;
    const startHeight = cardRefEl.current?.getBoundingClientRect().height ?? height ?? defaultPlotHeight;
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
      data-card-priority={card.role === "primary" || card.placement.pinned ? "primary" : "secondary"}
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
                {card.id === "thermal_program" && <HeroTopTimeAxis timeRange={timeRange} currentTimeMs={currentTimeMs} hoverTimeMs={hoverTimeMs} readoutTimeMs={readoutTimeMs} tickCount={heroTickCount} />}
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
            {visibleSignals.map((signal) => {
              const readout = readouts.get(signal.id) ?? "-";
              return (
                <span
                  className="graph-card-readout-chip"
                  data-readout-value={readout}
                  data-signal-id={signal.id}
                  data-signal-source-family={signal.source_family}
                  key={signal.id}
                  title={`${signal.label} / ${signal.source_family}`}
                >
                  <i className="graph-card-readout-swatch" style={{ background: colorForSignal(signal) }} />
                  <b>{signal.label}</b>
                  <em>{readout}</em>
                </span>
              );
            })}
            {readoutTimeMs && <small className="graph-card-readout-context">{readoutMode} {new Date(readoutTimeMs).toISOString().slice(5, 16).replace("T", " ")}</small>}
            {cardRef?.supports_y_zoom && <small className="graph-card-readout-context">time + y zoom</small>}
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
            (plot: uPlot, scaleKey: string) => {
              if (scaleKey !== "x") return;
              const min = plot.scales.x.min;
              const max = plot.scales.x.max;
              if (typeof min !== "number" || typeof max !== "number" || !Number.isFinite(min) || !Number.isFinite(max) || max <= min) return;
              if (Math.abs(min - timeRange.start) < 2 && Math.abs(max - timeRange.end) < 2) return;
              onTimeRangeRef.current({ start: Math.round(min), end: Math.round(max) });
            }
          ],
          setCursor: [
            (plot: uPlot) => {
              if (!pointerInsideRef.current || draggingRef.current) return;
              const left = plot.cursor.left;
              if (left == null) return;
              const next = clampTime(plot.posToVal(left, "x"), data[0] as number[]);
              if (Number.isFinite(next)) onHoverTimeRef.current(Math.round(next));
            }
          ],
          drawClear: [
            (plot: uPlot) => {
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
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), 14);
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
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), 14);
  const eventRailMarkers = (tile.markers ?? []).filter((marker) => inTimeRange(marker.timestamp, timeRange));
  const markerIDs = new Set(eventRailMarkers.map((marker) => marker.id));
  const eventRailEvents = (tile.events ?? []).filter((event) => inTimeRange(event.timestamp, timeRange) && !markerIDs.has(event.id));
  const markerPlacements = railLabelPlacements(eventRailMarkers, timeRange, 3, 2.8, 1.4);
  const eventPlacements = railLabelPlacements(eventRailEvents, timeRange, 2, 3.0, 1.5);
  return (
    <div className="tile-event-rail" data-event-card={tile.card_id}>
      <div className="tile-time-plane">
        {ticks.map((tick) => <span className="tile-shared-gridline" key={tick.iso} style={{ left: `${tick.ratio * 100}%` }} />)}
        {eventRailMarkers.map((marker, index) => {
          const placement = markerPlacements.get(marker.id);
          const left = placement?.left ?? Math.max(0, Math.min(100, ((Date.parse(marker.timestamp) - start) / span) * 100));
          const color = markerColor(marker);
          return (
            <span
              className={`event-marker-wrap event-marker-${marker.kind ?? marker.role ?? "marker"}`}
              key={marker.id}
              style={{ left: `${left}%`, top: `${12 + (placement?.row ?? index % 3) * 44}px`, color }}
              title={`${marker.label} ${marker.timestamp}`}
            >
              <i className={`event-marker event-${marker.result ?? marker.kind}`} style={{ background: color }} />
              {placement?.showLabel && <strong>{shortGateLabel(marker.label)}</strong>}
            </span>
          );
        })}
        {eventRailEvents.map((event, index) => {
          const placement = eventPlacements.get(event.id);
          const left = placement?.left ?? Math.max(0, Math.min(100, ((Date.parse(event.timestamp) - start) / span) * 100));
          const color = eventColor(event.kind);
          return (
            <span
              className={`event-chip-wrap event-chip-${event.kind}`}
              key={event.id}
              style={{ left: `${left}%`, top: `${164 + (placement?.row ?? index % 2) * 42}px`, color }}
              title={`${event.label} ${event.timestamp}`}
            >
              <b className={`event-chip event-${event.kind}`} style={{ background: color }} />
              {placement?.showLabel && <strong>{shortGateLabel(event.label)}</strong>}
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

type RailLabelPlacement = {
  left: number;
  row: number;
  showLabel: boolean;
};

function railLabelPlacements<T extends { id: string; label: string; timestamp: string }>(items: T[], range: TimeRange, rows: number, minGapPct: number, widthScale: number) {
  const span = Math.max(1, range.end - range.start);
  const occupied = Array.from({ length: rows }, () => [] as Array<{ left: number; right: number }>);
  const placements = new Map<string, RailLabelPlacement>();
  items.forEach((item, index) => {
    const t = Date.parse(item.timestamp);
    const left = Number.isFinite(t) ? Math.max(0, Math.min(100, ((t - range.start) / span) * 100)) : 0;
    const labelWidth = Math.min(18, Math.max(5.5, shortGateLabel(item.label).length * widthScale));
    let row = index % rows;
    let showLabel = false;
    for (let candidate = 0; candidate < rows; candidate++) {
      const labelLeft = Math.max(0, left - labelWidth / 2);
      const labelRight = Math.min(100, left + labelWidth / 2);
      const blocked = occupied[candidate].some((used) => labelLeft < used.right + minGapPct && labelRight + minGapPct > used.left);
      if (!blocked) {
        row = candidate;
        showLabel = true;
        occupied[candidate].push({ left: labelLeft, right: labelRight });
        break;
      }
    }
    placements.set(item.id, { left, row, showLabel });
  });
  return placements;
}

export function scheduleTileWork(work: () => void, delayMs: number) {
  window.setTimeout(() => {
    work();
  }, delayMs);
}

// Re-export sub-module items that may be imported by other consumers
export { TimeAxisTrack, timeTicks, clampRange } from "./tiles/timeAxis";
export { colorForSignal, roleColors, signalColors, orderLegendSignals, graphCardPriority, graphSectionPriority, blockLabel, eventColor } from "./tiles/visualPolicy";
export { legendReadouts, clampTime, markerColor, shortGateLabel, rawValueAt, stateAt, formatLegendValue, formatScientific, formatPressure, unitForAxis } from "./tiles/markers";
export { viewportSeries, lttb, decimationValue, resampleSeries, commandCenterGapBreaks, commandCenterTraceGapMs, commandCenterProjectedSeries, displayValue } from "./tiles/decimation";
export { uplotData, seriesDrawOrder, lineWidthFor, sharedTimeGrid, buildScales, buildAxes, paddedRange, logScale, logSplits, ySplits, axisLabel, scaleForSeries, stateBlocks, inTimeRange, renderKindFor, drawTileOverlays } from "./tiles/uPlotAdapter";
