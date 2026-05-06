import { ArrowDownLeft, ArrowUpRight } from "lucide-react";
import type { BusVirtualizationTap } from "../types";
import { OperatorPanel } from "../components/OperatorPanel";
import { StatusBadge } from "../components/StatusBadge";

export function BusTapView({ tap }: { tap: BusVirtualizationTap }) {
  const tm = tap.events.filter((event) => event.direction === "TM").slice(-6).reverse();
  const tc = tap.events.filter((event) => event.direction === "TC").slice(-6).reverse();

  return (
    <div className="view-grid">
      <OperatorPanel title="Virtual Bus Connection" meta={tap.connection_id}>
        <p className="summary">{tap.description}</p>
        <div className="stream-grid">
          {tap.streams.map((stream) => (
            <div className="stream-card" key={stream.id}>
              <div className="stream-title">
                {stream.direction === "TM" ? <ArrowDownLeft size={18} /> : <ArrowUpRight size={18} />}
                <strong>{stream.label}</strong>
              </div>
              <span>{stream.source_node} {"->"} {stream.destination_node}</span>
              <div className="metric-grid compact">
                <div><span className="label">Bus</span><strong>{stream.bus}</strong></div>
                <div><span className="label">Latency</span><strong>{stream.latency_ms} ms</strong></div>
                <div><span className="label">Counter</span><strong>{stream.packet_counter}</strong></div>
                <div><span className="label">Dropped</span><strong>{stream.dropped_frames}</strong></div>
              </div>
              <StatusBadge value={stream.quality} />
            </div>
          ))}
        </div>
      </OperatorPanel>
      <OperatorPanel title="Live Tap Replay" meta={tap.replay_cursor}>
        <div className="tap-columns">
          <EventColumn title="TM" events={tm} />
          <EventColumn title="TC" events={tc} />
        </div>
      </OperatorPanel>
    </div>
  );
}

function EventColumn({ title, events }: { title: string; events: BusVirtualizationTap["events"] }) {
  return (
    <div className="event-column">
      <h2>{title}</h2>
      {events.map((event) => (
        <article className="event-card" key={event.id}>
          <div className="event-head">
            <strong>{event.id}</strong>
            <StatusBadge value={event.quality} />
          </div>
          <span>{event.source_node} {"->"} {event.destination_node}</span>
          <span>{event.event_class} / {event.latency_ms} ms / #{event.packet_counter}</span>
          <p>{event.summary}</p>
        </article>
      ))}
    </div>
  );
}
