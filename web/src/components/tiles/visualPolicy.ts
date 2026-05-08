import type { GraphTileCardRef, GraphWallCard, GraphWallModel, TileSeries } from "../../types";

export const roleColors: Record<string, string> = {
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

export const signalColors: Record<string, string> = {
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

export const distinctivePalette = [
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

export function palette(index: number) {
  return distinctivePalette[index % distinctivePalette.length];
}

export function paletteForID(id: string, fallbackIndex: number) {
  let hash = fallbackIndex + 17;
  for (let i = 0; i < id.length; i += 1) hash = ((hash << 5) - hash + id.charCodeAt(i)) | 0;
  return distinctivePalette[Math.abs(hash) % distinctivePalette.length];
}

export function colorForSignal(signal: Pick<TileSeries, "id" | "role" | "render_kind" | "kind"> | { id: string; role: string; kind?: string }, index = 0) {
  const kind = "kind" in signal ? signal.kind : ("render_kind" in signal ? signal.render_kind : undefined);
  if (signalColors[signal.id]) return signalColors[signal.id];
  const semantic = semanticColor(signal.id);
  if (semantic) return semantic;
  if (signal.role === "command" || signal.role === "ghost" || signal.role === "acceptance_band" || signal.role === "interlock" || signal.role === "evidence") return roleColors[signal.role];
  return paletteForID(signal.id, index) ?? roleColors[signal.role] ?? (kind ? roleColors[kind] : undefined) ?? palette(index);
}

export function semanticColor(id: string) {
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

export function signalPriority(signal: { id: string; label?: string; role?: string; kind?: string; render_kind?: string }) {
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

export function orderLegendSignals<T extends { id: string; label?: string; role?: string; kind?: string; render_kind?: string }>(signals: T[]) {
  return [...signals].sort((a, b) => signalPriority(a) - signalPriority(b));
}

export function graphCardPriority(a: GraphWallCard, b: GraphWallCard) {
  return graphCardRank(a) - graphCardRank(b);
}

export function graphSectionPriority(a: GraphWallModel["sections"][number], b: GraphWallModel["sections"][number]) {
  return graphSectionRank(a) - graphSectionRank(b);
}

export function graphSectionRank(section: GraphWallModel["sections"][number]) {
  return Math.min(...section.cards.map(graphCardRank), 100);
}

export function graphCardRank(card: GraphWallCard) {
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

export function cardPriority(card: GraphTileCardRef) {
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

export function tileCardPriority(a: GraphTileCardRef, b: GraphTileCardRef) {
  const aPriority = cardPriority(a);
  const bPriority = cardPriority(b);
  if (aPriority !== bPriority) return aPriority - bPriority;
  if (a.default_expanded !== b.default_expanded) return a.default_expanded ? -1 : 1;
  return a.card_id.localeCompare(b.card_id);
}

export function eventColor(kind?: string) {
  const lower = (kind ?? "").toLowerCase();
  if (lower.includes("functional") || lower.includes("gate")) return "#ffb000";
  if (lower.includes("evidence")) return "#b079ff";
  if (lower.includes("interlock") || lower.includes("fault")) return "#ff315f";
  if (lower.includes("stability") || lower.includes("dwell")) return "#00d6a3";
  if (lower.includes("pressure")) return "#1f6fff";
  return "#31d6ff";
}

export function blockLabel(label: string, value: number) {
  const normalized = String(label ?? "").trim();
  if (normalized && normalized !== "0" && normalized !== "1") return normalized;
  return value > 0 ? "ACTIVE" : "idle";
}
