import { AlertCircle, CheckCircle2, Clock3, RotateCw } from "lucide-react";
import type { CommandCenterBand, CommandCenterFAT, CommandCenterRun, CommandCenterTrace, GraphPoint } from "../types";

type Props = {
  model: CommandCenterFAT;
};

export function CommandCenterFATView({ model }: Props) {
  const start = Date.parse(model.window_start);
  const end = Date.parse(model.window_end);
  const now = Date.parse(model.now);
  const completed = model.lanes.flatMap((lane) => lane.runs).filter((run) => run.state === "complete").length;
  const running = model.lanes.flatMap((lane) => lane.runs).filter((run) => run.state === "running").length;
  const scheduled = model.lanes.flatMap((lane) => lane.runs).filter((run) => run.state === "scheduled").length;

  return (
    <section className="command-center-view">
      <div className="command-center-header">
        <div>
          <span className="eyebrow">standalone FAT command center</span>
          <h2>{model.title}</h2>
          <p>{model.summary}</p>
        </div>
        <div className="command-center-kpis" aria-label="FAT command center summary">
          <KPI label="complete" value={completed} />
          <KPI label="running" value={running} />
          <KPI label="scheduled" value={scheduled} />
          <KPI label="now" value={formatShort(model.now)} />
        </div>
      </div>

      <div className="fat-timeline-shell">
        <TimelineHeader start={start} end={end} />
        <div className="fat-ladder">
          <div className="fat-weekend-layer" aria-hidden="true">
            {model.weekend_bands.map((band) => (
              <span key={band.id} className="fat-weekend-band" style={spanStyle(band.start, band.end, start, end)} />
            ))}
            <span className="fat-now-marker" style={{ left: `${percent(now, start, end)}%` }} />
          </div>
          {model.lanes.map((lane) => (
            <article className="fat-chamber-lane" key={lane.id}>
              <div className="fat-lane-label">
                <strong>{lane.chamber_name}</strong>
                <span>{lane.facility}</span>
              </div>
              <div className="fat-lane-track">
                {lane.runs.map((run) => (
                  <RunBlock key={run.id} run={run} windowStart={start} windowEnd={end} now={now} />
                ))}
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}

function KPI({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function TimelineHeader({ start, end }: { start: number; end: number }) {
  const days = [];
  for (let cursor = start; cursor < end; cursor += 24 * 60 * 60 * 1000) {
    const date = new Date(cursor);
    days.push(
      <span key={date.toISOString()} className={isWeekend(date) ? "weekend" : ""} style={{ left: `${percent(cursor, start, end)}%`, width: `${100 / 21}%` }}>
        {date.toLocaleDateString(undefined, { weekday: "short", day: "2-digit" })}
      </span>
    );
  }
  return <div className="fat-timeline-header">{days}</div>;
}

function RunBlock({ run, windowStart, windowEnd, now }: { run: CommandCenterRun; windowStart: number; windowEnd: number; now: number }) {
  const actual = run.traces.find((trace) => trace.role === "actual");
  const ghost = run.traces.find((trace) => trace.role === "ghost");
  return (
    <>
      <div className={`fat-run-block ${run.state}`} style={spanStyle(run.start, run.end, windowStart, windowEnd)}>
        <div className="fat-run-topline">
          <span>{run.title}</span>
          <StateBadge state={run.state} />
        </div>
        <TraceSVG actual={actual} ghost={ghost} start={Date.parse(run.start)} end={Date.parse(run.end)} />
        <div className="fat-run-manifest">
          <strong>{run.manifest.article}</strong>
          <span>{run.manifest.serial_number}</span>
          <em>{run.manifest.operator_next}</em>
        </div>
        {run.interaction_windows.map((window) => (
          <InteractionMarker key={window.id} window={window} run={run} now={now} />
        ))}
      </div>
      <div className="fat-reset-block" style={spanStyle(run.reset_start, run.reset_end, windowStart, windowEnd)}>
        <RotateCw size={12} />
        <span>reset</span>
      </div>
    </>
  );
}

function StateBadge({ state }: { state: string }) {
  const Icon = state === "complete" ? CheckCircle2 : state === "running" ? AlertCircle : Clock3;
  return (
    <span className={`fat-state ${state}`}>
      <Icon size={13} />
      {state}
    </span>
  );
}

function InteractionMarker({ window, run, now }: { window: CommandCenterBand; run: CommandCenterRun; now: number }) {
  const left = percent(Date.parse(window.start), Date.parse(run.start), Date.parse(run.end));
  const active = now >= Date.parse(window.start) && now <= Date.parse(window.end);
  return <span className={`fat-gate-marker ${active ? "active" : ""}`} title={window.label} style={{ left: `${left}%` }} />;
}

function TraceSVG({ actual, ghost, start, end }: { actual?: CommandCenterTrace; ghost?: CommandCenterTrace; start: number; end: number }) {
  return (
    <svg className="fat-hero-trace" viewBox="0 0 240 62" preserveAspectRatio="none" aria-hidden="true">
      {ghost && <path className="ghost" d={tracePath(ghost.values, start, end, ghost.min, ghost.max)} />}
      {actual && <path className="actual" d={tracePath(actual.values, start, end, actual.min, actual.max)} />}
    </svg>
  );
}

function tracePath(points: GraphPoint[], start: number, end: number, min: number, max: number) {
  return points.map((point, index) => {
    const x = percent(Date.parse(point.timestamp), start, end) * 2.4;
    const y = 58 - ((point.value - min) / (max - min)) * 54;
    return `${index === 0 ? "M" : "L"} ${x.toFixed(2)} ${y.toFixed(2)}`;
  }).join(" ");
}

function spanStyle(itemStart: string, itemEnd: string, windowStart: number, windowEnd: number) {
  const left = percent(Date.parse(itemStart), windowStart, windowEnd);
  const right = percent(Date.parse(itemEnd), windowStart, windowEnd);
  return { left: `${left}%`, width: `${Math.max(0.6, right - left)}%` };
}

function percent(value: number, start: number, end: number) {
  return Math.max(0, Math.min(100, ((value - start) / (end - start)) * 100));
}

function isWeekend(date: Date) {
  const day = date.getUTCDay();
  return day === 0 || day === 6;
}

function formatShort(value: string) {
  return new Date(value).toLocaleString(undefined, { weekday: "short", hour: "2-digit", minute: "2-digit" });
}
