import type { CSSProperties } from "react";

export type TimeRange = {
  start: number;
  end: number;
};

export function clampRange(range: TimeRange, fullRange: TimeRange, minSpan: number): TimeRange {
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

const TIME_GRID_TICK_COUNT_DEFAULT = 14;

export function timeTicks(startISO: string, endISO: string, count: number) {
  const start = Date.parse(startISO);
  const end = Date.parse(endISO);
  const span = Math.max(1, end - start);
  const target = Math.max(10, Math.min(20, count || TIME_GRID_TICK_COUNT_DEFAULT));
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

export function chooseTickStep(spanMs: number, targetCount: number) {
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

export function tickLabel(date: Date, stepMs: number) {
  const time = date.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
  if (stepMs < 24 * 60 * 60_000) return time;
  return `${date.toLocaleDateString(undefined, { month: "short", day: "2-digit" })} ${time}`;
}

export function TimeAxisTrack({ ticks, start, end, nowRatio, hoverTimeMs, peekTimeMs, compact }: { ticks: ReturnType<typeof timeTicks>; start: number; end: number; nowRatio?: number; hoverTimeMs?: number; peekTimeMs?: number; compact?: boolean }) {
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

export function HeroTopTimeAxis({ timeRange, currentTimeMs, hoverTimeMs, readoutTimeMs, tickCount }: { timeRange: TimeRange; currentTimeMs?: number; hoverTimeMs?: number; readoutTimeMs?: number; tickCount?: number }) {
  const start = timeRange.start;
  const end = timeRange.end;
  const nowRatio = typeof currentTimeMs === "number" && Number.isFinite(currentTimeMs) ? Math.max(0, Math.min(1, (currentTimeMs - start) / Math.max(1, end - start))) : undefined;
  const ticks = timeTicks(new Date(start).toISOString(), new Date(end).toISOString(), tickCount ?? TIME_GRID_TICK_COUNT_DEFAULT);
  return (
    <div className="hero-top-time-axis" aria-label="Hero graph top time axis">
      <TimeAxisTrack ticks={ticks} start={start} end={end} nowRatio={nowRatio} hoverTimeMs={hoverTimeMs} peekTimeMs={readoutTimeMs !== hoverTimeMs ? readoutTimeMs : undefined} compact />
    </div>
  );
}

export function SharedTimeAxis({
  fullRange,
  timeRange,
  currentTimeMs,
  hoverTimeMs,
  peekTimeMs,
  plotBounds,
  onTimeRange,
  tickCount,
}: {
  fullRange: TimeRange;
  timeRange: TimeRange;
  currentTimeMs?: number;
  hoverTimeMs?: number;
  peekTimeMs?: number;
  plotBounds?: { left: number; right: number };
  onTimeRange: (range: TimeRange) => void;
  tickCount: number;
}) {
  const ticks = timeTicks(new Date(timeRange.start).toISOString(), new Date(timeRange.end).toISOString(), tickCount);
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
