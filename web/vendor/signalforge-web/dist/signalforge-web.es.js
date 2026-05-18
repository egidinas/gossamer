var me = Object.defineProperty;
var pe = (t, e, n) => e in t ? me(t, e, { enumerable: !0, configurable: !0, writable: !0, value: n }) : t[e] = n;
var at = (t, e, n) => pe(t, typeof e != "symbol" ? e + "" : e, n);
import { useState as O, useEffect as V, useRef as U, useMemo as Tt, useCallback as he } from "react";
import { jsx as _, jsxs as x } from "react/jsx-runtime";
import _e from "uplot";
function Jt(t) {
  return `${t}.assignments`;
}
function ct(t, e, n) {
  return `${t}@${e}/${n}`;
}
function ge(t) {
  const e = String(t || "").match(/^(\d+)@([^/]+)\/(\d+)$/);
  return e ? { param_id: parseInt(e[1], 10), device_id: e[2], instance: parseInt(e[3], 10) || 1 } : null;
}
function Xt(t, e) {
  const n = t || {}, r = ge(n.target_id ?? ""), a = n.options || {}, o = Number(n.param_id ?? a.param_id ?? (r == null ? void 0 : r.param_id) ?? NaN), i = String(n.device_id ?? a.device_id ?? (r == null ? void 0 : r.device_id) ?? ""), c = Number(n.instance ?? a.instance ?? (r == null ? void 0 : r.instance) ?? 1) || 1, s = String((n.wall_id ?? "") || "wall");
  if (!i || !Number.isFinite(o)) return null;
  const l = n.target_id ?? ct(o, i, c), u = n.tile_id ?? `${s}-${l}`;
  return {
    wall_id: s,
    tile_id: u,
    target_id: l,
    kind: n.kind ?? "trend",
    options: { ...a, param_id: o, device_id: i, instance: c },
    param_id: o,
    device_id: i,
    instance: c
  };
}
function ot(t) {
  try {
    const e = JSON.parse(localStorage.getItem(Jt(t.namespace)) || "[]");
    return Array.isArray(e) ? e.map((n) => Xt(n, t.namespace)).filter((n) => n !== null) : [];
  } catch {
    return [];
  }
}
function Dt(t, e) {
  const n = t.map((r) => Xt(r, e.namespace)).filter((r) => r !== null);
  localStorage.setItem(Jt(e.namespace), JSON.stringify(n)), typeof window < "u" && window.dispatchEvent(new CustomEvent(`${e.namespace}-assignments-changed`));
}
function be(t, e, n, r = 1) {
  const a = ct(e, n, r);
  return {
    wall_id: t,
    tile_id: `${t}-${a}`,
    target_id: a,
    kind: "trend",
    options: { param_id: e, device_id: n, instance: r },
    param_id: e,
    device_id: n,
    instance: r
  };
}
function we(t) {
  const [e, n] = O(() => ot(t));
  return V(() => {
    const r = `${t.namespace}-assignments-changed`, a = () => n(ot(t));
    return window.addEventListener(r, a), () => window.removeEventListener(r, a);
  }, [t.namespace]), {
    list: e,
    add(r, a, o, i = 1) {
      const c = ot(t), s = be(r, a, o, i);
      c.find((l) => l.wall_id === r && l.target_id === s.target_id) || Dt([...c, s], t);
    },
    remove(r, a, o, i = 1) {
      const c = ct(a, o, i);
      Dt(ot(t).filter((s) => !(s.wall_id === r && s.target_id === c)), t);
    },
    forWall(r) {
      return e.filter((a) => a.wall_id === r);
    },
    hasAssignment(r, a, o, i = 1) {
      const c = ct(a, o, i);
      return e.some((s) => s.wall_id === r && s.target_id === c);
    }
  };
}
function Kt(t) {
  return `${t}.walls`;
}
const xt = (t) => `${t}-walls-changed`;
function Z(t) {
  try {
    const e = JSON.parse(localStorage.getItem(Kt(t)) || "[]");
    return Array.isArray(e) ? e.filter((n) => n && typeof n.wall_id == "string" && typeof n.label == "string") : [];
  } catch {
    return [];
  }
}
function gt(t, e) {
  localStorage.setItem(Kt(e), JSON.stringify(t)), typeof window < "u" && window.dispatchEvent(new CustomEvent(xt(e)));
}
function Fn(t) {
  const [e, n] = O(() => Z(t));
  return V(() => {
    const r = () => n(Z(t));
    return window.addEventListener(xt(t), r), () => window.removeEventListener(xt(t), r);
  }, [t]), {
    walls: e,
    add(r) {
      const a = { wall_id: `${t}-wall-${Date.now()}`, label: r };
      return gt([...Z(t), a], t), a;
    },
    rename(r, a) {
      gt(Z(t).map((o) => o.wall_id === r ? { ...o, label: a } : o), t);
    },
    remove(r) {
      gt(Z(t).filter((a) => a.wall_id !== r), t);
    },
    wallForDevice(r) {
      return { wall_id: `device-${r}`, label: `Device · ${r}` };
    }
  };
}
function Zt(t) {
  return t <= 5 * 6e4 ? "live" : t <= 6 * 60 * 6e4 ? "minute" : "hour";
}
class ve {
  constructor(e, n = {}) {
    at(this, "cache", /* @__PURE__ */ new Map());
    at(this, "inflight", /* @__PURE__ */ new Map());
    at(this, "ttlMs");
    this.adapter = e, this.ttlMs = n.ttlMs ?? 3e4;
  }
  cacheKey(e, n, r) {
    return `${e}/${n}@${r}`;
  }
  async fetch(e, n, r) {
    const a = this.cacheKey(e, n, r), o = this.cache.get(a);
    if (o && Date.now() - o.fetchedAt < this.ttlMs) return o.tile;
    const i = this.inflight.get(a);
    if (i) return i;
    const c = this.adapter.fetchTile(e, n, r).then((s) => (this.cache.set(a, { tile: s, fetchedAt: Date.now() }), this.inflight.delete(a), s)).catch((s) => {
      throw this.inflight.delete(a), s;
    });
    return this.inflight.set(a, c), c;
  }
  fetchForViewport(e, n, r) {
    return this.fetch(e, n, Zt(r));
  }
  invalidate(e) {
    if (!e) {
      this.cache.clear();
      return;
    }
    for (const n of this.cache.keys())
      n.startsWith(`${e}/`) && this.cache.delete(n);
  }
}
function An(t, e, n, r, a = 5e3) {
  const [o, i] = O({ status: "loading", tile: null }), c = U(null);
  return c.current || (c.current = new ve(t)), V(() => {
    const s = c.current;
    let l = !1;
    const u = Zt(r);
    async function f() {
      try {
        const m = await s.fetch(e, n, u);
        l || i({ status: "ok", tile: m });
      } catch (m) {
        l || i({ status: "error", tile: null, error: String(m) });
      }
    }
    if (f(), u === "live") {
      const m = setInterval(f, a);
      return () => {
        l = !0, clearInterval(m);
      };
    }
    return () => {
      l = !0;
    };
  }, [e, n, r, a]), o;
}
const Lt = {
  command: "#ffd85f",
  ghost: "#8aa7c4",
  acceptance_band: "#3ddc84",
  actual: "#56d6df",
  dut: "#ff6b35",
  aux: "#9db4c8",
  source_quality: "#66b8ef",
  counter: "#b8a6ff",
  interlock: "#ff6374",
  evidence: "#b8a6ff",
  event: "#f2f7ff",
  state: "#8bd3a5"
}, Et = {
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
  "trace.tvac_exhaust_cold_recovery": "#b079ff"
}, lt = [
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
  "#ef476f"
];
function ye(t) {
  return lt[t % lt.length];
}
function xe(t, e) {
  let n = e + 17;
  for (let r = 0; r < t.length; r += 1) n = (n << 5) - n + t.charCodeAt(r) | 0;
  return lt[Math.abs(n) % lt.length];
}
function Se(t, e = 0) {
  const n = "kind" in t ? t.kind : "render_kind" in t ? t.render_kind : void 0;
  if (t.color && !t.color.includes("var(")) return t.color;
  if (Et[t.id]) return Et[t.id];
  const r = Me(t.id);
  if (r) return r;
  const a = Lt[t.role];
  if (a) return a;
  const o = n ? Lt[n] : void 0;
  return o || (xe(t.id, e) ?? ye(e));
}
function Me(t) {
  const e = t.toLowerCase();
  if (e.includes("dut_temp_a") || e.includes("dut.a") || e.includes("node_a")) return "#ff315f";
  if (e.includes("dut_temp_b") || e.includes("dut.b") || e.includes("node_b")) return "#00d6a3";
  if (e.includes("dut") && e.includes("temp")) return "#ff6b35";
  if (e.includes("command") || e.includes("target")) return "#ffd400";
  if (e.includes("ghost") || e.includes("profile")) return "#f8fafc";
  if (e.includes("pressure")) return "#1f6fff";
  if (e.includes("power")) return "#ff7a35";
  if (e.includes("packet") || e.includes("bus")) return "#b079ff";
  if (e.includes("ready") || e.includes("operative") || e.includes("stability")) return "#00d6a3";
  if (e.includes("fault") || e.includes("error") || e.includes("interlock")) return "#ff315f";
  if (e.includes("interface") || e.includes("table") || e.includes("platen")) return "#ff8a00";
  if (e.includes("shroud")) return "#b65cff";
  if (e.includes("chamber")) return "#00c8ff";
}
function ut(t) {
  const e = `${t.id} ${t.label ?? ""}`.toLowerCase();
  return t.role === "command" ? 0 : t.role === "ghost" ? 1 : t.role === "acceptance_band" ? 2 : e.includes("dut") ? 3 : e.includes("article") || e.includes("component") ? 4 : e.includes("interface") || e.includes("platen") || e.includes("table") ? 5 : e.includes("chamber") || e.includes("shroud") ? 6 : e.includes("pressure") ? 7 : e.includes("power") ? 8 : e.includes("bus") || e.includes("packet") ? 9 : t.kind === "state" || t.render_kind === "swimlane" ? 10 : 20;
}
function $n(t) {
  return [...t].sort((e, n) => ut(e) - ut(n));
}
function Cn(t, e) {
  return St(t) - St(e);
}
function Tn(t, e) {
  return Pt(t) - Pt(e);
}
function Pt(t) {
  return Math.min(...t.cards.map(St), 100);
}
function St(t) {
  const e = t.id.toLowerCase(), n = t.title.toLowerCase();
  return e === "thermal_program" ? 0 : e.includes("dut_temperature") || n.includes("dut temperature") ? 10 : e.includes("dut_power") || n.includes("dut power") ? 20 : e.includes("tmtc_health") ? 30 : e.includes("tmtc_counters") ? 40 : e.includes("state_change") || t.render_kind === "swimlane" ? 50 : e.includes("functional_events") || t.render_kind === "event_rail" ? 60 : e.includes("facility") || e.includes("building") || e.includes("source_quality") || n.includes("testbed") ? 80 : 70;
}
function Ot(t) {
  return {
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
    source_quality: 11
  }[t.card_id] ?? 40;
}
function Dn(t, e) {
  const n = Ot(t), r = Ot(e);
  return n !== r ? n - r : t.default_expanded !== e.default_expanded ? t.default_expanded ? -1 : 1 : t.card_id.localeCompare(e.card_id);
}
function Ln(t) {
  const e = (t ?? "").toLowerCase();
  return e.includes("functional") || e.includes("gate") ? "#ffb000" : e.includes("evidence") ? "#b079ff" : e.includes("interlock") || e.includes("fault") ? "#ff315f" : e.includes("stability") || e.includes("dwell") ? "#00d6a3" : e.includes("pressure") ? "#1f6fff" : "#31d6ff";
}
function En(t, e) {
  const n = String(t ?? "").trim();
  return n && n !== "0" && n !== "1" ? n : e > 0 ? "ACTIVE" : "idle";
}
function ke(t, e, n) {
  const r = Math.max(1, e.end - e.start), a = Math.min(r, Math.max(1, n)), o = Math.max(a, Math.min(r, t.end - t.start));
  let i = t.start, c = t.start + o;
  return i < e.start && (i = e.start, c = i + o), c > e.end && (c = e.end, i = c - o), { start: Math.round(i), end: Math.round(c) };
}
const Qt = 14;
function Mt(t, e, n) {
  const r = Date.parse(t), a = Date.parse(e), o = Math.max(1, a - r), i = Math.max(10, Math.min(20, n || Qt)), c = Ne(o, i), s = Math.ceil(r / c) * c, l = [];
  for (let f = s; f <= a && l.length < 24; f += c) {
    if (f < r) continue;
    const m = new Date(f);
    l.push({ iso: m.toISOString(), ratio: (f - r) / o, label: bt(m, c) });
  }
  (!l.length || l[0].ratio > 0.02) && l.unshift({ iso: new Date(r).toISOString(), ratio: 0, label: bt(new Date(r), c) });
  const u = l[l.length - 1];
  return u && u.ratio < 0.98 && l.push({ iso: new Date(a).toISOString(), ratio: 1, label: bt(new Date(a), c) }), l.filter((f, m, d) => m === 0 || f.iso !== d[m - 1].iso);
}
function Ne(t, e) {
  const n = t / Math.max(1, e - 1), r = [
    5 * 6e4,
    10 * 6e4,
    15 * 6e4,
    30 * 6e4,
    60 * 6e4,
    3 * 60 * 6e4,
    6 * 60 * 6e4,
    12 * 60 * 6e4,
    24 * 60 * 6e4,
    2 * 24 * 60 * 6e4,
    7 * 24 * 60 * 6e4,
    14 * 24 * 60 * 6e4,
    30 * 24 * 60 * 6e4
  ];
  return r.find((a) => a >= n) ?? r[r.length - 1];
}
function bt(t, e) {
  const n = t.toLocaleTimeString(void 0, { hour: "2-digit", minute: "2-digit" });
  return e < 24 * 60 * 6e4 ? n : `${t.toLocaleDateString(void 0, { month: "short", day: "2-digit" })} ${n}`;
}
function te({ ticks: t, start: e, end: n, nowRatio: r, hoverTimeMs: a, peekTimeMs: o, compact: i }) {
  return /* @__PURE__ */ x("div", { className: `time-axis-track ${i ? "time-axis-track-compact" : ""}`, children: [
    r !== void 0 && /* @__PURE__ */ _("i", { className: "time-axis-elapsed", style: { width: `${r * 100}%` } }),
    r !== void 0 && /* @__PURE__ */ _("b", { className: "time-axis-now", style: { left: `${r * 100}%` }, title: "Current replay time" }),
    o !== void 0 && /* @__PURE__ */ _("b", { className: "time-axis-peek", style: { left: `${Math.max(0, Math.min(100, (o - e) / Math.max(1, n - e) * 100))}%` }, title: "Drag peek time" }),
    a !== void 0 && /* @__PURE__ */ _("b", { className: "time-axis-hover", style: { left: `${Math.max(0, Math.min(100, (a - e) / Math.max(1, n - e) * 100))}%` } }),
    t.map((c) => /* @__PURE__ */ x("span", { className: "time-axis-tick", style: { left: `${c.ratio * 100}%` }, children: [
      /* @__PURE__ */ _("i", {}),
      /* @__PURE__ */ _("em", { children: c.label })
    ] }, c.iso))
  ] });
}
function Pn({ timeRange: t, currentTimeMs: e, hoverTimeMs: n, readoutTimeMs: r, tickCount: a }) {
  const o = t.start, i = t.end, c = typeof e == "number" && Number.isFinite(e) ? Math.max(0, Math.min(1, (e - o) / Math.max(1, i - o))) : void 0, s = Mt(new Date(o).toISOString(), new Date(i).toISOString(), a ?? Qt);
  return /* @__PURE__ */ _("div", { className: "hero-top-time-axis", "aria-label": "Hero graph top time axis", children: /* @__PURE__ */ _(te, { ticks: s, start: o, end: i, nowRatio: c, hoverTimeMs: n, peekTimeMs: r !== n ? r : void 0, compact: !0 }) });
}
function On({
  fullRange: t,
  timeRange: e,
  currentTimeMs: n,
  hoverTimeMs: r,
  peekTimeMs: a,
  plotBounds: o,
  onTimeRange: i,
  tickCount: c
}) {
  const s = Mt(new Date(e.start).toISOString(), new Date(e.end).toISOString(), c), l = e.start, u = e.end, f = n, m = typeof f == "number" && Number.isFinite(f) ? Math.max(0, Math.min(1, (f - l) / Math.max(1, u - l))) : void 0, d = Math.max(0, (u - l) / 36e5), b = Math.max(1, t.end - t.start), h = Math.max(1, e.end - e.start), p = h < b * 0.995, g = Math.max(6e4, b / 600), N = o ? {
    "--time-axis-grid-left": `${o.left}px`,
    "--time-axis-grid-right": `${o.right}px`,
    "--time-axis-left": `${o.left}px`,
    "--time-axis-right": `${o.right}px`
  } : void 0, F = (A) => {
    const W = Math.max(g, Math.min(b, h * A)), G = (e.start + e.end) / 2;
    i(ke({ start: Math.round(G - W / 2), end: Math.round(G + W / 2) }, t, g));
  }, w = (A) => {
    const W = Math.max(0, b - h), G = Number(A) / 1e3 * W;
    i({ start: Math.round(t.start + G), end: Math.round(t.start + G + h) });
  }, M = Math.round((e.start - t.start) / Math.max(1, b - h) * 1e3);
  return /* @__PURE__ */ x("div", { className: "operator-shared-time-axis", "aria-label": "Shared graph time axis", style: N, children: [
    /* @__PURE__ */ _("span", { className: "time-axis-label", children: "TIME" }),
    /* @__PURE__ */ _(te, { ticks: s, start: l, end: u, nowRatio: m, hoverTimeMs: r, peekTimeMs: a }),
    /* @__PURE__ */ x("div", { className: "time-axis-sub-row", children: [
      /* @__PURE__ */ x("div", { className: "time-axis-controls", children: [
        /* @__PURE__ */ x("span", { children: [
          d.toFixed(d >= 24 ? 0 : 1),
          " h"
        ] }),
        /* @__PURE__ */ _("small", { children: "zoom" }),
        /* @__PURE__ */ _("button", { type: "button", onClick: () => F(1.35), "aria-label": "Zoom out", children: "-" }),
        /* @__PURE__ */ _("button", { type: "button", onClick: () => F(0.72), "aria-label": "Zoom in", children: "+" }),
        /* @__PURE__ */ _("button", { type: "button", disabled: !p, onClick: () => i(t), children: "full" })
      ] }),
      /* @__PURE__ */ x("label", { className: "time-axis-scrollbar", children: [
        /* @__PURE__ */ _("small", { children: "scroll" }),
        /* @__PURE__ */ _("input", { type: "range", min: "0", max: "1000", step: "1", disabled: !p, value: Math.max(0, Math.min(1e3, M)), onChange: (A) => w(Number(A.currentTarget.value)) })
      ] })
    ] })
  ] });
}
function Fe(t, e, n) {
  const r = e.points ?? [];
  if (r.length < 4 || ht(e)) return e;
  const a = Math.max(180, Math.min(r.length, Math.round(n * 1.65)));
  if (r.length <= a) return e;
  const o = (i) => De(t, e, i);
  return tt(e.axis_id) ? { ...e, points: Ae(r, a, o) } : { ...e, points: dt(r, a, o) };
}
function dt(t, e, n) {
  if (!t || e >= t.length || e < 3) return t;
  const r = t.map((c) => ({ point: c, x: Date.parse(c.timestamp), y: n(c.value) })).filter((c) => Number.isFinite(c.x) && Number.isFinite(c.y));
  if (r.length <= e) return r.map((c) => c.point);
  const a = [r[0].point], o = (r.length - 2) / (e - 2);
  let i = 0;
  for (let c = 0; c < e - 2; c++) {
    const s = Math.floor((c + 0) * o) + 1, l = Math.floor((c + 1) * o) + 1, u = Math.floor((c + 1) * o) + 1, f = Math.floor((c + 2) * o) + 1, m = r.slice(s, Math.min(l, r.length - 1)), d = r.slice(u, Math.min(f, r.length)), b = d.reduce((w, M) => w + M.x, 0) / Math.max(1, d.length), h = d.reduce((w, M) => w + M.y, 0) / Math.max(1, d.length), p = r[i];
    let g = m[0] ?? r[Math.min(s, r.length - 2)], N = m.length ? s : Math.min(s, r.length - 2), F = -1;
    m.forEach((w, M) => {
      const A = Math.abs((p.x - b) * (w.y - p.y) - (p.x - w.x) * (h - p.y));
      A > F && (F = A, g = w, N = s + M);
    }), a.push(g.point), i = N;
  }
  return a.push(r[r.length - 1].point), a;
}
function Ae(t, e, n) {
  if (!t || e >= t.length || e < 3) return t;
  const r = [];
  let a = [];
  const o = () => {
    a.length && (r.push({ kind: "run", points: a }), a = []);
  };
  for (const h of t) {
    const p = Date.parse(h.timestamp), g = n(h.value);
    if (Number.isFinite(p) && Number.isFinite(g)) {
      a.push(h);
      continue;
    }
    o(), Number.isFinite(p) && r.push({ kind: "gap", point: h });
  }
  o();
  const i = r.filter((h) => h.kind === "run"), c = r.length - i.length;
  if (!c) return dt(t, e, n);
  const s = i.reduce((h, p) => h + p.points.length, 0), l = Math.max(0, e - c), u = i.map((h) => {
    if (h.points.length <= 2) return h.points.length;
    const p = s > 0 ? Math.round(h.points.length / s * l) : h.points.length;
    return Math.min(h.points.length, Math.max(3, p));
  }), f = (h) => h.points.length <= 2 ? h.points.length : 3;
  let m = c + u.reduce((h, p) => h + p, 0);
  for (; m > e; ) {
    let h = -1;
    for (let p = 0; p < u.length; p += 1)
      u[p] <= f(i[p]) || (h === -1 || u[p] > u[h]) && (h = p);
    if (h === -1) break;
    u[h] -= 1, m -= 1;
  }
  if (m > e)
    return $e(r, e, n);
  const d = [];
  let b = 0;
  for (const h of r) {
    if (h.kind === "gap") {
      d.push(h.point);
      continue;
    }
    const p = u[b] ?? h.points.length;
    d.push(...p >= h.points.length ? h.points : dt(h.points, p, n) ?? []), b += 1;
  }
  return d;
}
function $e(t, e, n) {
  const r = t.filter((d) => d.kind === "run"), a = t.map((d, b) => ({ segment: d, index: b })).filter((d) => d.segment.kind === "gap");
  if (!r.length) return a.slice(0, e).map((d) => d.segment.point);
  const o = r.reduce((d, b) => d + b.points.length, 0), i = Math.max(0, e - 1), c = Math.round(e * (a.length / (a.length + r.length))), s = Math.min(a.length, i, Math.max(1, c)), l = /* @__PURE__ */ new Set();
  if (s > 0)
    for (let d = 0; d < s; d += 1) {
      const b = a[Math.floor(d * a.length / s)];
      b && l.add(b.index);
    }
  const u = Ce(r, Math.max(0, e - l.size), o), f = [];
  let m = 0;
  for (let d = 0; d < t.length && f.length < e; d += 1) {
    const b = t[d];
    if (b.kind === "gap") {
      l.has(d) && f.push(b.point);
      continue;
    }
    const h = Math.min(u[m] ?? 0, e - f.length);
    f.push(...Te(b.points, h, n)), m += 1;
  }
  return f.slice(0, e);
}
function Ce(t, e, n) {
  const r = t.map(() => 0);
  if (e <= 0) return r;
  if (e < t.length) {
    for (let o = 0; o < e; o += 1) {
      const i = Math.floor(o * t.length / e);
      r[i] = 1;
    }
    return r;
  }
  t.forEach((o, i) => {
    r[i] = Math.min(o.points.length, 1);
  });
  let a = e - r.reduce((o, i) => o + i, 0);
  for (; a > 0; ) {
    let o = -1, i = 0;
    if (t.forEach((c, s) => {
      const l = n > 0 ? c.points.length / n * e : e / Math.max(1, t.length), u = Math.min(c.points.length, Math.max(1, Math.round(l))) - r[s];
      u > i && (i = u, o = s);
    }), o === -1) break;
    r[o] += 1, a -= 1;
  }
  return r;
}
function Te(t, e, n) {
  return e <= 0 ? [] : e >= t.length ? t : e === 1 ? [t[Math.floor((t.length - 1) / 2)]] : e === 2 ? [t[0], t[t.length - 1]] : dt(t, e, n) ?? [];
}
function De(t, e, n) {
  return tt(e.axis_id) ? n > 0 ? Math.log10(n) : Number.NaN : n;
}
function In(t, e, n, r) {
  return ee(t, e, pt(e), n, r);
}
function pt(t) {
  return [...t.points ?? []].map((e) => ({ t: Date.parse(e.timestamp), v: ie(t, e.value) })).filter((e) => Number.isFinite(e.t)).sort((e, n) => e.t - n.t);
}
function ee(t, e, n, r, a) {
  if (!n.length) return r.map(() => null);
  const o = ht(e) || e.render_kind === "swimlane", i = e.role === "ghost" || re(t, e), c = kt(t, e);
  let s = 0;
  return r.map((l) => {
    if (Number.isFinite(a) && l > a && !i) return null;
    for (; s + 1 < n.length && n[s + 1].t <= l; ) s += 1;
    const u = n[s], f = n[Math.min(s + 1, n.length - 1)];
    if (l < n[0].t || l > n[n.length - 1].t || c > 0 && f.t - u.t > c && l > u.t && l < f.t) return null;
    if (l === u.t) return Number.isFinite(u.v) ? R(e, u.v) : null;
    if (!Number.isFinite(u.v) || !Number.isFinite(f.v)) return null;
    if (o || f.t === u.t) return R(e, u.v);
    const m = (l - u.t) / (f.t - u.t), d = u.v + (f.v - u.v) * Math.max(0, Math.min(1, m));
    return R(e, d);
  });
}
function Rn(t, e) {
  return ne(t, e, pt(e));
}
function ne(t, e, n) {
  const r = kt(t, e);
  if (r <= 0) return [];
  const a = [];
  for (let o = 1; o < n.length; o += 1)
    n[o].t - n[o - 1].t > r && a.push(n[o - 1].t + 1, n[o].t - 1);
  return a;
}
function kt(t, e) {
  return t.campaign_id !== "command_center_fat" || e.render_kind === "swimlane" || ht(e) || e.role === "event" ? 0 : 2 * 60 * 60 * 1e3;
}
function re(t, e) {
  return t.campaign_id === "command_center_fat" && e.role === "command";
}
function Le(t, e, n) {
  return tt(e.axis_id) ? n > 0 ? n : Number.NaN : n;
}
function ht(t) {
  return !!t.step || t.render_kind === "counter" || t.kind === "counter" || t.role === "counter";
}
function ie(t, e) {
  return tt(t.axis_id) ? e > 0 ? Math.log10(e) : Number.NaN : e;
}
function R(t, e) {
  return Number.isFinite(e) ? tt(t.axis_id) ? 10 ** e : e : Number.NaN;
}
function tt(t) {
  return t === "pressure_log" || t === "pressure_rate_log";
}
const Q = 8;
function Ee(t) {
  return t.kind === "operator_breakdown" ? "rgba(255,112,67,0.98)" : t.kind === "operator_reset" ? "rgba(36,214,255,0.98)" : t.kind === "operator_reset_ready" ? "rgba(146,255,111,0.98)" : t.role === "interlock" || t.result === "fail" ? "rgba(255,49,95,0.96)" : t.role === "evidence" ? "rgba(176,121,255,0.96)" : t.kind === "functional_gate" ? "rgba(255,176,0,0.98)" : t.kind === "stability" || t.kind === "stability_achieved" || t.result === "pass" ? "rgba(0,214,163,0.96)" : "rgba(49,214,255,0.95)";
}
function Pe(t, e) {
  return t.kind === "operator_breakdown" ? e ? "BD" : "BREAKDOWN" : t.kind === "operator_reset" ? e ? "RST" : "RESET" : t.kind === "operator_reset_ready" ? e ? "RDY" : "READY" : ((t.kind ?? t.label ?? t.role ?? "operator").replace(/^operator[_\s-]*/i, "").replace(/[_-]+/g, " ").trim() || "operator").toUpperCase().slice(0, e ? 6 : 14);
}
function Oe(t, e = !1) {
  return [Pe(t, e), Ie(t.timestamp, e)];
}
function Ie(t, e = !1) {
  const n = new Date(t);
  return Number.isNaN(n.getTime()) ? t : n.toLocaleString(void 0, {
    weekday: "short",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: !e
  });
}
function It({
  x: t,
  y: e,
  labelWidth: n,
  labelHeight: r,
  left: a,
  top: o,
  width: i,
  height: c,
  placed: s,
  markerRadius: l
}) {
  const f = t > a + i * 0.68 ? [-1, 1] : [1, -1], m = Math.max(o + 4, Math.min(o + c - r - 4, e - r / 2)), d = r + Q, b = Array.from({ length: 21 }, (p, g) => Math.ceil(g / 2) * (g % 2 === 0 ? 1 : -1) * d), h = [-6 * d, -7 * d, -8 * d, -9 * d, -10 * d, 6 * d, 7 * d, 8 * d, 9 * d, 10 * d];
  for (const p of f) {
    const g = p < 0 ? t - n - l - 8 : t + l + 8, N = Math.max(a + 4, Math.min(a + i - n - 4, g));
    for (const F of [...b, ...h]) {
      const w = m + F, M = {
        x: N,
        y: Math.max(o - r * 2, Math.min(o + c + r, w)),
        width: n,
        height: r
      };
      if (!s.some((A) => Re(M, A)))
        return M;
    }
  }
  return null;
}
function Re(t, e) {
  return t.x < e.x + e.width + Q && t.x + t.width + Q > e.x && t.y < e.y + e.height + Q && t.y + t.height + Q > e.y;
}
function We(t, e, n) {
  if (t.measureText(e).width <= n) return e;
  let r = e;
  for (; r.length > 3 && t.measureText(`${r.slice(0, -1)}...`).width > n; )
    r = r.slice(0, -1);
  return `${r.slice(0, -1)}...`;
}
function ze(t) {
  return t.replace(/\s+breakdown\s+start$/i, " BD").replace(/\s+reset\s+start$/i, " RST").replace(/\s+reset\s+ready$/i, " RDY").replace(/^Stable\s+/i, "STBL ").replace(/\s+confirmed$/i, "").replace(/^Cycle\s+/i, "C").replace(/\s+dwell\s+functional\s+test/i, " FT").replace(/\s+functional\s+test/i, " FT").slice(0, 8);
}
function Wn(t, e, n, r) {
  const a = /* @__PURE__ */ new Map();
  if (n === void 0 || !Number.isFinite(n)) return a;
  const o = new Set(e.map((i) => i.id));
  return t.series.forEach((i) => {
    var s;
    if (!o.has(i.id) || Number.isFinite(n) && Number.isFinite(r) && n > r && i.role !== "ghost" && !re(t, i)) return;
    if ((s = i.spans) != null && s.length) {
      const l = je(i, n);
      l && a.set(i.id, l);
      return;
    }
    const c = ae(i, n, t);
    c !== void 0 && Be(i, c) && a.set(i.id, Ve(i, c));
  }), a;
}
function Be(t, e) {
  return Number.isFinite(e) && (!Ye(t.axis_id) || !qe(t.axis_id) || e > 0);
}
function zn(t, e) {
  if (!Number.isFinite(t) || !e.length) return t;
  const n = e[0], r = e[e.length - 1];
  return !Number.isFinite(n) || !Number.isFinite(r) ? t : Math.max(n, Math.min(r, t));
}
function ae(t, e, n) {
  const r = [...t.points ?? []].map((m) => ({ t: Date.parse(m.timestamp), v: ie(t, m.value) })).filter((m) => Number.isFinite(m.t)).sort((m, d) => m.t - d.t);
  if (!r.length) return;
  const a = r[0], o = r[r.length - 1], i = n ? kt(n, t) : 0;
  if (e < a.t) return;
  if (e > o.t)
    return i <= 0 || e - o.t > i || !Number.isFinite(o.v) ? void 0 : R(t, o.v);
  if (e === r[0].t) return R(t, r[0].v);
  if (e === r[r.length - 1].t) return R(t, r[r.length - 1].v);
  let c = 0;
  for (; c + 1 < r.length && r[c + 1].t <= e; ) c += 1;
  const s = r[c], l = r[Math.min(c + 1, r.length - 1)];
  if (i > 0 && l.t - s.t > i && e > s.t && e < l.t) return;
  if (e === s.t) return Number.isFinite(s.v) ? R(t, s.v) : void 0;
  if (!Number.isFinite(s.v) || !Number.isFinite(l.v)) return;
  if (ht(t) || l.t === s.t) return R(t, s.v);
  const u = (e - s.t) / (l.t - s.t), f = s.v + (l.v - s.v) * Math.max(0, Math.min(1, u));
  return R(t, f);
}
function je(t, e) {
  var r;
  const n = (r = t.spans) == null ? void 0 : r.find((a) => {
    const o = Date.parse(a.start), i = Date.parse(a.end);
    return Number.isFinite(o) && Number.isFinite(i) && e >= o && e <= i;
  });
  if (n)
    return oe(t, n.value, n.state, n.label);
}
function oe(t, e, n, r) {
  const a = t.value_table ?? {}, o = (i) => {
    if (i == null) return;
    const c = String(i).trim();
    if (c)
      return a[c] ?? c;
  };
  return o(r) ?? o(n) ?? o(e);
}
function Ve(t, e) {
  const n = Rt(t.unit ?? t.units) || Ge(t.axis_id);
  if (t.axis_id === "pressure_log") return `${zt(e)} mbar`;
  if (t.axis_id === "pressure_rate_log") return `${Wt(e)} mbar/min`;
  if (t.axis_id === "pressure_mbar") return `${zt(e)} mbar`;
  if (t.axis_id === "pressure_rate") return `${Wt(e)} mbar/min`;
  if (t.axis_id === "counter") {
    const r = Rt(t.unit ?? t.units);
    return r ? `${Math.round(e).toLocaleString()} ${r}` : Math.round(e).toLocaleString();
  }
  return t.axis_id === "temperature_c" ? `${e.toFixed(1)} degC` : t.axis_id === "percent" ? `${e.toFixed(0)}%` : n === "degC" ? `${e.toFixed(1)} degC` : n === "W" ? `${e.toFixed(1)} W` : n === "ms" ? `${e.toFixed(1)} ms` : n === "bar" ? `${e.toFixed(2)} bar` : `${Number.isInteger(e) ? e.toFixed(0) : e.toFixed(2)}${n ? ` ${n}` : ""}`;
}
function Rt(t) {
  const e = typeof t == "string" ? t.trim() : "";
  return e && e !== "_" ? e : "";
}
function Wt(t) {
  return Number.isFinite(t) ? t === 0 ? "0" : t.toExponential(2).replace("e", "E") : "";
}
function zt(t) {
  return t <= 0 ? "0" : t < 1e-3 || t >= 1e3 ? t.toExponential(2).replace("e", "E") : t < 1 ? t.toPrecision(3) : t.toFixed(t < 10 ? 2 : 1);
}
function Ge(t) {
  return t === "temperature_c" ? "degC" : t === "pressure_log" || t === "pressure_mbar" ? "mbar" : t === "power_w" || t === "heat_flux_w" ? "W" : t === "bus_ms" ? "ms" : t === "pressure_bar" ? "bar" : t === "pressure_rate_log" || t === "pressure_rate" ? "mbar/min" : t === "percent" ? "%" : t === "voltage_v" ? "V" : t === "current_a" ? "A" : t === "rf_db" || t === "link_db" || t === "signal_db" ? "dB" : t === "frequency_hz" ? "Hz" : t === "ohm" ? "Ω" : "";
}
function Ye(t) {
  return t === "pressure_mbar" || t === "pressure_rate" || t === "pressure_log" || t === "pressure_rate_log";
}
function qe(t) {
  return t === "pressure_log" || t === "pressure_rate_log";
}
function wt(t, e, n = 900) {
  const o = t.series.filter((u) => (u.points ?? []).length > 0).sort(Ue).map((u) => Fe(t, u, n)).map((u) => ({ series: u, points: pt(u) })), i = se(t, o), c = [i], s = [{}], l = /* @__PURE__ */ new Set();
  return o.forEach(({ series: u, points: f }, m) => {
    const d = le(t, u);
    l.add(d), c.push(ee(t, u, f, i, e)), s.push({
      label: u.label,
      scale: d,
      stroke: Se(u, m),
      width: He(u.role),
      dash: u.role === "ghost" ? [7, 4] : u.role === "acceptance_band" ? [2, 5] : void 0,
      points: { show: !1 }
    });
  }), { data: c, series: s, scales: Ke(l), axes: Ze(l) };
}
function Ue(t, e) {
  const n = {
    ghost: 5,
    acceptance_band: 8,
    actual: 10,
    source_quality: 12,
    counter: 14,
    dut: 40,
    command: 45,
    aux: 50,
    event: 50,
    interlock: 55,
    evidence: 60
  }, r = (n[t.role] ?? 15) - (n[e.role] ?? 15);
  return r || ut(t) - ut(e);
}
function He(t) {
  return t === "command" ? 1.55 : t === "ghost" ? 0.9 : t === "acceptance_band" ? 0.75 : t === "counter" || t === "source_quality" ? 1.05 : t === "dut" ? 1.1 : t === "aux" ? 0.95 : 0.85;
}
function Bn(t, e) {
  return se(t, e.map((n) => ({ series: n, points: pt(n) })));
}
function se(t, e) {
  const n = Date.parse(t.t0), r = Date.parse(t.t1), a = e.flatMap((s) => s.points.map((l) => l.t)).filter(Number.isFinite), o = e.flatMap(({ series: s, points: l }) => ne(t, s, l)), i = Number.isFinite(n) ? n : Math.min(...a), c = Number.isFinite(r) ? r : Math.max(...a);
  return !Number.isFinite(i) || !Number.isFinite(c) || c <= i ? Array.from(/* @__PURE__ */ new Set([...a, ...o])).sort((s, l) => s - l) : Array.from(/* @__PURE__ */ new Set([n, r, ...a, ...o])).filter(Number.isFinite).sort((s, l) => s - l);
}
function Je(t) {
  return [
    ...(t.series ?? []).flatMap((e) => [
      ...(e.points ?? []).map((n) => Date.parse(n.timestamp)),
      ...(e.spans ?? []).flatMap((n) => [Date.parse(n.start), Date.parse(n.end)])
    ]),
    ...(t.markers ?? []).map((e) => Date.parse(e.timestamp)),
    ...(t.bands ?? []).flatMap((e) => [Date.parse(e.start), Date.parse(e.end)])
  ].filter(Number.isFinite);
}
function Xe(t, e) {
  const n = (e == null ? void 0 : e.start) ?? Date.parse(t.t0), r = (e == null ? void 0 : e.end) ?? Date.parse(t.t1);
  let a = Number.isFinite(n) ? n : void 0, o = Number.isFinite(r) ? r : void 0;
  if (a === void 0 || o === void 0) {
    const i = Je(t);
    if (!i.length) return null;
    a === void 0 && (a = Math.min(...i)), o === void 0 && (o = Math.max(...i));
  }
  return !Number.isFinite(a) || !Number.isFinite(o) ? null : o <= a ? { start: a, end: a + 1 } : { start: a, end: o };
}
function Ke(t) {
  const e = {};
  return t.forEach((n) => {
    n === "temperature_c" ? e[n] = { range: I(12, [-92, 92]) } : n === "pressure_log" ? e[n] = { distr: 3, log: 10, range: () => [1e-8, 1200] } : n === "pressure_rate_log" ? e[n] = { distr: 3, log: 10, range: () => [1e-8, 1e3] } : n === "pressure_mbar" ? e[n] = { range: I(0.08, [0, 1200]) } : n === "pressure_rate" ? e[n] = { range: I(0.08) } : n === "pressure_bar" ? e[n] = { range: I(0.08, [0, 12]) } : n === "percent" ? e[n] = { range: (r, a, o) => [0, 100] } : n === "heat_flux_w" ? e[n] = { range: I(8, [-45, 45]) } : n === "current_a" ? e[n] = { range: I(0.1) } : n === "voltage_v" ? e[n] = { range: I(0.5) } : n === "seconds" ? e[n] = { range: I(1) } : n === "generic_numeric" ? e[n] = { range: I(1) } : e[n] = {};
  }), e;
}
function Ze(t, e) {
  const a = [{ show: !1 }], i = [
    "temperature_c",
    "power_w",
    "heat_flux_w",
    "current_a",
    "voltage_v",
    "seconds",
    "bus_ms",
    "counter",
    "pressure_log",
    "pressure_rate_log",
    "pressure_mbar",
    "pressure_rate",
    "pressure_bar",
    "percent",
    "generic_numeric"
  ].find((s) => t.has(s)) ?? "generic_numeric";
  a.push({
    show: !0,
    scale: i,
    stroke: "#7890a4",
    grid: { stroke: "rgba(83,112,140,0.26)", width: 1 },
    ticks: { stroke: "rgba(83,112,140,0.48)", width: 1, size: 4 },
    splits: (s, l, u, f) => st(i) ? jt(u, f) : Qe(u, f),
    size: 64,
    gap: 0,
    label: Vt(i),
    labelSize: 12,
    labelGap: 0,
    values: st(i) ? (s, l) => l.map((u) => Gt(u)) : void 0
  });
  const c = Array.from(t).filter((s) => s !== i);
  return c.forEach((s) => {
    a.push({
      show: !0,
      scale: s,
      side: 1,
      stroke: s.includes("pressure") ? "#60a5fa" : "#8bd3a5",
      grid: { show: !1 },
      ticks: { show: !1 },
      size: 64,
      gap: 0,
      label: Vt(s),
      labelSize: 12,
      labelGap: 0,
      splits: st(s) ? (l, u, f, m) => jt(f, m) : void 0,
      values: st(s) ? (l, u) => u.map((f) => Gt(f)) : void 0
    });
  }), c.length || a.push({
    show: !0,
    side: 1,
    scale: i,
    size: 64,
    gap: 0,
    label: "",
    labelSize: 12,
    labelGap: 0,
    grid: { show: !1 },
    ticks: { show: !1 },
    values: () => []
  }), a;
}
function I(t, e) {
  return (n, r, a) => {
    if (!Number.isFinite(r) || !Number.isFinite(a)) return ce(e) ?? [0, 1];
    if (a <= r) return Bt(r - t, a + t, e);
    const o = Math.max(t, (a - r) * 0.08), i = r - o, c = a + o;
    return Bt(i, c, e);
  };
}
function Bt(t, e, n) {
  if (!n) return [t, e];
  const r = ce(n);
  if (!r) return [t, e];
  const a = Math.max(r[0], t), o = Math.min(r[1], e);
  return a <= o ? [a, o] : r;
}
function ce(t) {
  if (!(!t || !Number.isFinite(t[0]) || !Number.isFinite(t[1])))
    return t[0] <= t[1] ? t : [t[1], t[0]];
}
function st(t) {
  return t === "pressure_log" || t === "pressure_rate_log";
}
function jt(t, e) {
  if (!Number.isFinite(t) || !Number.isFinite(e) || e <= 0 || e <= t) return [];
  const n = Math.ceil(Math.log10(Math.max(t, 1e-12))), r = Math.floor(Math.log10(e)), a = [];
  for (let o = n; o <= r; o += 1) a.push(Math.pow(10, o));
  return a;
}
function Qe(t, e) {
  if (!Number.isFinite(t) || !Number.isFinite(e) || e <= t) return [];
  const r = (e - t) / 8, a = Math.pow(10, Math.floor(Math.log10(r))), o = [1, 2, 2.5, 5, 10].map((s) => s * a).find((s) => r <= s) ?? a * 10, i = Math.ceil(t / o) * o, c = [];
  for (let s = i; s <= e + o * 0.25; s += o) c.push(Number(s.toFixed(6)));
  return c;
}
function Vt(t, e) {
  return t === "temperature_c" ? "degC" : t === "pressure_log" ? "log10 mbar" : t === "pressure_rate_log" ? "log10 mbar/min" : t === "pressure_mbar" ? "mbar" : t === "pressure_rate" ? "mbar/min" : t === "pressure_bar" ? "bar" : t === "heat_flux_w" || t === "power_w" ? "W" : t === "current_a" ? "A" : t === "voltage_v" ? "V" : t === "seconds" ? "s" : t === "bus_ms" ? "ms" : t === "counter" ? "count" : t === "percent" ? "%" : t === "generic_numeric" ? "value" : t;
}
function le(t, e) {
  if (e.axis_id === "pressure_log") return "pressure_log";
  if (e.axis_id === "pressure_rate_log") return "pressure_rate_log";
  if (e.axis_id === "pressure_mbar") return "pressure_mbar";
  if (e.axis_id === "pressure_rate") return "pressure_rate";
  if (e.axis_id === "pressure_bar") return "pressure_bar";
  if (e.axis_id === "power_w") return "power_w";
  if (e.axis_id === "heat_flux_w") return "heat_flux_w";
  if (e.axis_id === "current_a") return "current_a";
  if (e.axis_id === "voltage_v") return "voltage_v";
  if (e.axis_id === "seconds") return "seconds";
  if (e.axis_id === "counter") return "counter";
  if (e.axis_id === "bus_ms") return "bus_ms";
  if (e.axis_id === "percent") return "percent";
  if (e.axis_id === "generic_numeric") return "generic_numeric";
  const n = (e.unit ?? e.units ?? "").trim().toLowerCase();
  return n === "a" || n === "amp" || n === "amps" ? "current_a" : n === "v" || n === "volt" || n === "volts" ? "voltage_v" : n === "s" || n === "sec" || n === "secs" || n === "second" || n === "seconds" ? "seconds" : n === "ms" || n === "millisecond" || n === "milliseconds" ? "bus_ms" : n === "w" || n === "watt" || n === "watts" ? "power_w" : n === "%" || n === "percent" ? "percent" : n.includes("deg") || n === "c" || n === "°c" ? "temperature_c" : e.role === "counter" || e.kind === "counter" ? "counter" : "generic_numeric";
}
function jn(t, e, n) {
  var a;
  if ((a = t.spans) != null && a.length)
    return t.spans.flatMap((o, i) => {
      const c = Date.parse(o.start), s = Date.parse(o.end);
      if (!Number.isFinite(c) || !Number.isFinite(s) || s < e || c > e + n) return [];
      const l = Math.max(0, Math.min(100, (c - e) / n * 100)), u = Math.max(l + 0.15, Math.min(100, (s - e) / n * 100));
      return [{
        key: `${t.id}-span-${i}`,
        left: l,
        width: u - l,
        value: o.value ?? Number(o.state ?? 0),
        label: oe(t, o.value, o.state, o.label) ?? ""
      }];
    });
  const r = [...t.points ?? []].sort((o, i) => Date.parse(o.timestamp) - Date.parse(i.timestamp));
  return r.flatMap((o, i) => {
    const c = Date.parse(o.timestamp), s = i + 1 < r.length ? Date.parse(r[i + 1].timestamp) : e + n;
    if (!Number.isFinite(c) || !Number.isFinite(s) || s < e || c > e + n) return [];
    const l = Math.max(0, Math.min(100, (c - e) / n * 100)), u = Math.max(l + 0.15, Math.min(100, (s - e) / n * 100));
    return [{ key: `${t.id}-${i}`, left: l, width: u - l, value: o.value, label: String(o.value) }];
  });
}
function Vn(t, e) {
  const n = Date.parse(t);
  return Number.isFinite(n) && n >= e.start && n <= e.end;
}
function Gn(t) {
  return t === "state" ? "swimlane" : t === "event" ? "event_rail" : t === "counter" ? "counter" : "line";
}
function Gt(t) {
  return Number.isFinite(t) ? t === 0 ? "0" : t.toExponential(2).replace("e", "E") : "";
}
function tn(t, e, n, r, a, o) {
  const i = en(e, n);
  for (const c of i) {
    const s = ae(c, r, e);
    if (s === void 0) continue;
    const l = le(e, c), u = t.valToPos(Le(e, c, s), l);
    if (Number.isFinite(u))
      return { y: Math.max(a + 12, Math.min(a + o - 10, u)) };
  }
  return null;
}
function en(t, e) {
  return t.series.filter((n) => (n.points ?? []).length).map((n) => ({ series: n, score: nn(n, e) })).filter((n) => n.score > 0).sort((n, r) => r.score - n.score).map((n) => n.series);
}
function nn(t, e) {
  const n = `${t.id} ${t.label} ${t.axis_id ?? ""} ${t.source ?? ""} ${t.role}`.toLowerCase(), r = `${e.id} ${e.label} ${e.kind} ${e.role} ${e.axis_id ?? ""}`.toLowerCase(), a = an(e), o = ue(e);
  let i = 0;
  const c = (s, l, u) => {
    s.some((f) => r.includes(f)) && l.some((f) => n.includes(f)) && (i += u);
  };
  return c(["pressure", "vacuum", "tvac"], ["pressure", "vacuum", "tvac"], 80), c(["dut", "functional", "stability", "dwell"], ["dut", "component", "interface", "chamber"], 70), c(["shroud"], ["shroud"], 70), c(["interlock"], ["interlock", "facility"], 70), c(["operator", "command"], ["command", "chamber"], 55), c(["pump", "exhaust"], ["pump", "exhaust", "cryo", "scavenger"], 55), o && (n.includes("interlock") || n.includes("facility")) && (i += 180), e.axis_id && t.axis_id === e.axis_id && (i += 90), a && t.role === "command" && (i += 220), t.role === "actual" && (i += a ? 4 : 18), t.role === "command" && (i += a ? 34 : 8), t.role === "ghost" && (i += 4), i;
}
function ue(t) {
  return t.role === "interlock" || t.kind === "interlock" || t.result === "fail";
}
function rn(t) {
  return t.kind === "functional_gate" || t.kind === "stability" || t.kind === "stability_achieved" || t.kind === "pressure_gate" || ue(t);
}
function an(t) {
  var e;
  return t.role === "operator_interaction" || ((e = t.kind) == null ? void 0 : e.startsWith("operator_")) || t.kind === "functional_gate" || t.kind === "stability" || t.kind === "stability_achieved";
}
function on(t, e, n, r, a, o = 0.42) {
  t.save(), t.globalAlpha = o, t.strokeStyle = a, t.lineWidth = 1, t.setLineDash([2, 4]), t.beginPath(), t.moveTo(e, n), t.lineTo(e, n + r), t.stroke(), t.restore();
}
function Yt(t, e, n, r, a, o, i, c) {
  t.save(), t.globalAlpha = 0.8, t.strokeStyle = c, t.lineWidth = 1, t.setLineDash([]), t.beginPath(), t.moveTo(e, n), t.lineTo(r < e ? r + o : r, a + i / 2), t.stroke(), t.restore();
}
function sn(t, e, n) {
  const r = n < 760, a = r ? 0.018 : 0.075;
  return t.campaign_id === "tvac_qualification" && (e.includes("vacuum") || t.card_id.includes("pressure")) ? `rgba(59,130,246,${r ? 0.018 : 0.065})` : e.includes("breakdown") ? `rgba(255,112,67,${r ? 0.026 : 0.11})` : e.includes("reset") ? `rgba(36,214,255,${r ? 0.022 : 0.09})` : e.includes("cold") ? `rgba(61,133,198,${a})` : `rgba(198,119,61,${a})`;
}
function cn(t, e) {
  return t.campaign_id === "tvac_qualification" && (e.includes("vacuum") || t.card_id.includes("pressure")) ? "rgba(96,165,250,0.16)" : e.includes("breakdown") ? "rgba(255,112,67,0.22)" : e.includes("reset") ? "rgba(36,214,255,0.18)" : e.includes("cold") ? "rgba(96,165,250,0.16)" : "rgba(255,176,0,0.14)";
}
function ln(t, e, n, r, a, o) {
  var Ft, At, $t;
  const i = t.ctx, c = t.bbox, s = c.left, l = c.top, u = c.width, f = c.height, m = Xe(e, o);
  if (!m) return;
  const { start: d, end: b } = m, h = Math.max(1, b - d);
  i.save();
  const p = Mt(new Date(d).toISOString(), new Date(b).toISOString(), 14);
  i.strokeStyle = "rgba(83,112,140,0.16)", i.lineWidth = 1, i.setLineDash([]), p.forEach((v) => {
    const S = s + v.ratio * u;
    i.beginPath(), i.moveTo(S, l), i.lineTo(S, l + f), i.stroke();
  }), (e.bands ?? []).forEach((v) => {
    const S = s + (Date.parse(v.start) - d) / h * u, y = s + (Date.parse(v.end) - d) / h * u, C = (v.kind ?? "").toLowerCase(), D = Math.max(1, y - S), Y = u < 760;
    if (i.fillStyle = sn(e, C, u), Y) {
      const E = Math.max(2, Math.min(7, f * 0.04));
      i.fillRect(S, l, D, E), i.fillRect(S, l + f - E, D, E);
    } else
      i.fillRect(S, l, D, f);
    i.strokeStyle = cn(e, C), i.lineWidth = u < 520 ? 0.75 : 1, Y ? (i.beginPath(), i.moveTo(S + 0.5, l + 0.5), i.lineTo(S + 0.5, l + f - 0.5), i.moveTo(S + D - 0.5, l + 0.5), i.lineTo(S + D - 0.5, l + f - 0.5), i.stroke()) : i.strokeRect(S + 0.5, l + 0.5, Math.max(0, D - 1), Math.max(0, f - 1));
  });
  const g = [], N = un(e.markers ?? [], d, d + h, u, e.campaign_id);
  let F = 0, w = 0, M = 0;
  const A = (v, S, y) => {
    const C = l - (M + 1) * (S + 3), D = Math.max(s, Math.min(s + u - v, y));
    return M += 1, { x: D, y: C, width: v, height: S };
  };
  (e.markers ?? []).forEach((v) => {
    var Ct;
    const S = Date.parse(v.timestamp);
    if (!Number.isFinite(S)) return;
    const y = s + (S - d) / h * u;
    if (y < s || y > s + u) return;
    const C = Ee(v), D = v.role === "operator_interaction" || ((Ct = v.kind) == null ? void 0 : Ct.startsWith("operator_")), Y = rn(v), E = Y || D ? tn(t, e, v, S, l, f) : null, z = (E == null ? void 0 : E.y) ?? l + 10;
    if ((Y || D) && on(i, y, l, f, C, Y ? 0.48 : 0.36), D) {
      F += 1;
      const L = e.campaign_id === "command_center_fat" || u < 760, T = (E == null ? void 0 : E.y) ?? l + 18 + (v.kind === "operator_reset" ? 34 : v.kind === "operator_reset_ready" ? 68 : 0), $ = L ? 9 : 12;
      i.save(), i.shadowColor = "rgba(0,0,0,0.72)", i.shadowBlur = 6, i.fillStyle = "rgba(2,6,11,0.88)", i.strokeStyle = C, i.lineWidth = 2, i.beginPath(), i.arc(y, T, $ + 2, 0, Math.PI * 2), i.fill(), i.stroke(), i.beginPath(), v.kind === "operator_breakdown" ? (i.moveTo(y, T - $), i.lineTo(y + $, T), i.lineTo(y, T + $), i.lineTo(y - $, T), i.closePath()) : v.kind === "operator_reset" ? i.rect(y - $ + 1, T - $ + 1, ($ - 1) * 2, ($ - 1) * 2) : (i.moveTo(y, T - $), i.lineTo(y + $, T + $ - 2), i.lineTo(y - $, T + $ - 2), i.closePath()), i.fillStyle = C, i.fill(), i.lineWidth = 1.4, i.strokeStyle = "rgba(2,6,11,0.96)", i.stroke();
      const P = Oe(v, L), B = L ? Math.max(8.5, Math.min(10.5, u / 118)) : 12, j = L ? 11 : 14;
      i.font = `850 ${B}px system-ui, sans-serif`;
      const H = L ? Math.max(76, Math.min(118, u * 0.11)) : Math.max(110, Math.min(170, u * 0.16)), J = Math.max(...P.map((_t) => i.measureText(_t).width)) + 12, q = Math.min(H, J), K = P.length * j + 8, nt = It({ x: y, y: T, labelWidth: q, labelHeight: K, left: s, top: l, width: u, height: f, placed: g, markerRadius: $ }) ?? A(q, K, y - q / 2);
      if (!nt) {
        i.restore();
        return;
      }
      g.push(nt);
      const rt = nt.x, it = nt.y;
      Yt(i, y, T, rt, it, q, K, C), i.fillStyle = "rgba(2,6,11,0.94)", i.fillRect(rt, it, q, K), i.strokeStyle = C, i.lineWidth = 1.2, i.strokeRect(rt, it, q, K), i.fillStyle = C, P.forEach((_t, fe) => i.fillText(We(i, _t, q - 10), rt + 6, it + j + 1 + fe * j)), w += 1, i.restore();
    } else if (Y) {
      const L = e.campaign_id === "command_center_fat";
      if (i.save(), i.fillStyle = "rgba(2,6,11,0.86)", i.strokeStyle = C, i.lineWidth = L ? 2.2 : 1.8, i.beginPath(), i.arc(y, z, L ? 8 : v.kind === "functional_gate" ? 10 : 8, 0, Math.PI * 2), i.fill(), i.stroke(), i.restore(), i.fillStyle = C, i.beginPath(), v.kind === "functional_gate" ? (i.moveTo(y, z - 7), i.lineTo(y + 7, z), i.lineTo(y, z + 7), i.lineTo(y - 7, z), i.closePath()) : i.arc(y, z, 5.6, 0, Math.PI * 2), i.fill(), !N.has(v.id)) return;
      F += 1;
      const T = L ? "FT" : ze(v.label);
      i.save(), i.font = L ? "850 10px system-ui, sans-serif" : "850 12px system-ui, sans-serif";
      const $ = i.measureText(T), P = Math.max(L ? 22 : 36, $.width + 10), B = L ? 16 : 18, j = It({ x: y, y: z, labelWidth: P, labelHeight: B, left: s, top: l, width: u, height: f, placed: g, markerRadius: 8 }) ?? A(P, B, y - P / 2);
      if (!j) {
        i.restore();
        return;
      }
      g.push(j);
      const H = j.x, J = j.y;
      Yt(i, y, z, H, J, P, B, C), i.fillStyle = "rgba(2,6,11,0.92)", i.fillRect(H, J, P, B), i.strokeStyle = C, i.lineWidth = 1, i.strokeRect(H, J, P, B), i.fillStyle = v.kind === "functional_gate" ? "#fff0a8" : "#c9ffef", i.shadowColor = "rgba(0,0,0,0.88)", i.shadowBlur = 5, i.fillText(T, H + 5, J + Math.min(13, B - 5)), w += 1, i.restore();
    } else
      i.fillStyle = C, i.beginPath(), i.arc(y, l + 10, 3.2, 0, Math.PI * 2), i.fill();
  });
  const W = (Ft = t.root) == null ? void 0 : Ft.closest("[data-uplot-card]");
  W && (W.dataset.markerLabelsExpected = String(F), W.dataset.markerLabelsDrawn = String(w));
  const G = ((At = n == null ? void 0 : n.time_axis) == null ? void 0 : At.now) ?? (($t = n == null ? void 0 : n.execution) == null ? void 0 : $t.now) ?? "", Nt = r ?? Date.parse(G);
  if (Number.isFinite(Nt)) {
    const v = s + (Nt - d) / h * u, S = Math.max(s, Math.min(s + u, v));
    i.fillStyle = "rgba(3,7,12,0.58)", i.fillRect(S, l, Math.max(0, s + u - S), f), i.strokeStyle = "rgba(242,247,255,0.9)", i.setLineDash([3, 3]), i.beginPath(), i.moveTo(S, l), i.lineTo(S, l + f), i.stroke();
  }
  if (Number.isFinite(a)) {
    const v = s + (a - d) / h * u;
    v >= s && v <= s + u && (i.strokeStyle = "rgba(255,216,95,0.95)", i.setLineDash([]), i.lineWidth = 1, i.beginPath(), i.moveTo(v, l), i.lineTo(v, l + f), i.stroke());
  }
  i.restore();
}
function un(t, e, n, r, a) {
  return new Set(
    t.filter((o) => {
      const i = Date.parse(o.timestamp);
      return Number.isFinite(i) && i >= e && i <= n && vt(o) > 0;
    }).sort((o, i) => vt(i) - vt(o) || Date.parse(o.timestamp) - Date.parse(i.timestamp)).map((o) => o.id)
  );
}
function vt(t) {
  let e = 0;
  return (t.role === "interlock" || t.result === "fail" || t.kind === "interlock") && (e += 1e3), t.kind === "functional_gate" && (e += 760), t.kind === "pressure_gate" && (e += 640), (t.kind === "stability" || t.kind === "stability_achieved") && (e += 440), t.result === "pass" && (e += 120), e;
}
const ft = "signalforge.tile.uplot", qt = Object.freeze({
  cmd: { label: "target / command", rank: 10, className: "cmd", dash: "6,4", width: 2.2, opacity: 0.98 },
  command: { label: "target / command", rank: 10, className: "cmd", dash: "6,4", width: 2.2, opacity: 0.98 },
  actual: { label: "actual", rank: 20, className: "actual", dash: "", width: 2.2, opacity: 0.98 },
  ghost: { label: "reference / sink", rank: 30, className: "ghost", dash: "2,4", width: 1.8, opacity: 0.86 },
  dut: { label: "power / load", rank: 40, className: "dut", dash: "", width: 2, opacity: 0.94 },
  aux: { label: "auxiliary", rank: 50, className: "aux", dash: "8,4", width: 1.8, opacity: 0.86 }
});
function de(t) {
  return qt[t || "actual"] || qt.actual;
}
function dn(t, e = "var(--series-actual)") {
  const n = de(t), r = n.className ? `--series-${n.className}` : "";
  return !r || typeof document > "u" ? e : getComputedStyle(document.documentElement).getPropertyValue(r).trim() || e;
}
function Yn(t) {
  if (!t) return 0;
  const e = typeof t.getBoundingClientRect == "function" ? t.getBoundingClientRect() : null;
  return Math.floor(e && e.width || t.clientWidth || 0);
}
function fn(t = {}) {
  const e = k(t.tile_id ?? t.tileId, "empty"), n = Date.now(), r = mt(t.time_window_ms ?? t.timeWindowMs) ?? 9e4, a = new Date(n).toISOString();
  return {
    schema_version: "signalforge.graph_tile.v1",
    id: e,
    card_id: e,
    level: "live",
    t0: new Date(n - r).toISOString(),
    t1: a,
    generated_at: a,
    renderer: ft,
    kind: "timeseries",
    tile_id: e,
    title: k(t.title),
    time_window_ms: r,
    axes: Array.isArray(t.axes) ? t.axes : [],
    bands: [],
    markers: [],
    events: [],
    diagnostics: {
      status: "empty",
      point_count: 0,
      raw_point_count: 0,
      decimation: "none",
      freshness_ms: 0,
      renderer: ft,
      series_count: 0
    },
    provenance: { source: "empty-graph-tile", generated_at: a },
    series: []
  };
}
function qn(t) {
  if (!et(t) || !Array.isArray(t.series)) return [];
  const e = mn(t);
  return e.series.map((n) => {
    const r = n, a = k(r.seriesRole ?? (n.role === "command" ? "cmd" : n.role), "actual"), o = X(r.source_obj);
    return {
      key: k(r.series_id ?? n.id ?? r.key ?? r.target_id ?? r.targetId ?? n.label, "series"),
      tileId: e.tile_id || e.id,
      targetId: r.target_id ?? r.targetId,
      label: n.label,
      fullLabel: r.full_label ?? r.fullLabel ?? n.label,
      role: a,
      seriesRole: a,
      roleRank: mt(r.role_rank) ?? de(a).rank,
      color: n.color || dn(a),
      unit: k(n.unit ?? n.units, "_"),
      provenance: r.provenance || "",
      source: r.source_obj ?? n.source ?? null,
      paramId: r.param_id ?? o.param_id,
      deviceId: r.device_id ?? o.device_id,
      instance: r.instance ?? o.instance,
      signalId: r.signal_id ?? o.signal_id,
      history: bn(n)
    };
  }).sort((n, r) => n.roleRank !== r.roleRank ? n.roleRank - r.roleRank : String(n.tileId || n.key || n.label || "").localeCompare(String(r.tileId || r.key || r.label || "")));
}
function mn(t, e = {}) {
  const n = fn({
    tile_id: e.tile_id ?? e.tileId,
    timeWindowMs: e.timeWindowMs ?? e.time_window_ms
  }), r = X(t), a = X(r.diagnostics), o = mt(r.time_window_ms ?? e.timeWindowMs ?? e.time_window_ms) ?? n.time_window_ms ?? 9e4, i = (Array.isArray(r.series) ? r.series : []).map(pn).filter((w) => {
    var M, A;
    return (((M = w.points) == null ? void 0 : M.length) ?? 0) > 0 || (((A = w.spans) == null ? void 0 : A.length) ?? 0) > 0;
  }), c = Array.isArray(r.bands) ? r.bands : [], s = Array.isArray(r.markers) ? r.markers : [], l = Array.isArray(r.events) ? r.events : [], u = [
    ...i.flatMap((w) => [
      ...(w.points || []).map((M) => M.timestamp),
      ...(w.spans || []).flatMap((M) => [M.start, M.end])
    ]),
    ...c.flatMap((w) => [w.start, w.end]),
    ...s.map((w) => w.timestamp),
    ...l.map((w) => w.timestamp)
  ].map((w) => Date.parse(w)).filter(Number.isFinite), f = Date.now(), m = Ht(r.t0), d = Ht(r.t1), b = m ?? (u.length ? Math.min(...u) : f - o), h = d ?? (u.length ? Math.max(...u) : f), p = new Date(b).toISOString(), g = new Date(Math.max(h, b + 1)).toISOString(), N = i.reduce((w, M) => {
    var A;
    return w + (((A = M.points) == null ? void 0 : A.length) || 0);
  }, 0), F = k(a.status, i.length > 0 ? "ok" : "empty");
  return {
    ...n,
    ...r,
    schema_version: vn(r.schema_version, n.schema_version),
    id: k(r.id ?? r.tile_id, n.id),
    card_id: k(r.card_id ?? r.tile_id ?? r.id, n.card_id),
    level: k(r.level, "live"),
    t0: p,
    t1: g,
    generated_at: k(r.generated_at, new Date(f).toISOString()),
    renderer: ft,
    kind: k(r.kind, "timeseries"),
    tile_id: k(r.tile_id ?? r.id, n.tile_id),
    title: k(r.title, n.title),
    time_window_ms: o,
    axes: Array.isArray(r.axes) ? r.axes : n.axes,
    bands: c,
    markers: s,
    events: l,
    diagnostics: {
      ...n.diagnostics,
      ...a,
      status: F,
      point_count: N,
      raw_point_count: mt(a.raw_point_count) ?? N,
      decimation: k(a.decimation, "none"),
      renderer: ft,
      series_count: i.length
    },
    provenance: et(r.provenance) ? r.provenance : n.provenance,
    series: i
  };
}
function pn(t) {
  const e = X(t), n = yt(e.source_obj) ?? yt(e.source_ref) ?? yt(e.source), r = k(e.role ?? e.seriesRole, "actual"), a = hn(r), o = k(e.series_id ?? e.id ?? e.key ?? e.target_id ?? e.targetId ?? e.label, "series"), i = k(e.unit ?? e.units, "_"), c = gn(e);
  return {
    ...e,
    id: o,
    series_id: e.series_id || o,
    label: k(e.label, o),
    role: a,
    seriesRole: r,
    unit: i,
    units: i,
    axis_id: k(e.axis_id ?? _n({ ...e, id: o, role: a, seriesRole: r }, i)),
    source: wn(e.source_ref ?? e.source ?? n ?? o),
    source_obj: n,
    color: typeof e.color == "string" ? e.color : void 0,
    points: c,
    spans: Array.isArray(e.spans) ? e.spans : []
  };
}
function hn(t) {
  const e = k(t, "actual");
  return e === "cmd" ? "command" : e || "actual";
}
function _n(t, e) {
  const n = t.axis_id ?? t.axisId;
  if (n) return k(n);
  const r = k(e ?? t.unit ?? t.units).trim().toLowerCase(), a = [
    t.id,
    t.series_id,
    t.key,
    t.target_id,
    t.targetId,
    t.label,
    t.full_label,
    t.fullLabel
  ].filter(Boolean).join(" ").toLowerCase();
  return t.role === "counter" || t.seriesRole === "counter" || t.kind === "counter" ? "counter" : r === "a" || r === "amp" || r === "amps" ? "current_a" : r === "v" || r === "volt" || r === "volts" ? "voltage_v" : r === "w" || r === "watt" || r === "watts" ? "power_w" : r === "%" || r === "percent" ? "percent" : r === "ms" || r === "millisecond" || r === "milliseconds" ? "bus_ms" : r === "s" || r === "sec" || r === "secs" || r === "second" || r === "seconds" ? "seconds" : r === "mbar" || r === "millibar" || r === "millibars" ? "pressure_log" : r === "mbar/min" || r === "mbar/minute" || r === "millibar/min" || r === "millibars/minute" ? "pressure_rate_log" : r === "bar" ? "pressure_bar" : r.includes("deg") || r === "c" || r === "degc" || r === "deg c" || r === "°c" || r === "° c" ? "temperature_c" : a.includes("counter") ? "counter" : a.includes("pressure") || a.includes("vacuum") ? "pressure_log" : "generic_numeric";
}
function gn(t) {
  if (Array.isArray(t.points) && t.points.length)
    return t.points.flatMap((a) => Ut(a));
  const e = X(t.history), n = Array.isArray(e.ts) ? e.ts : [];
  return (Array.isArray(e.v) ? e.v : []).flatMap((a, o) => Ut({ t: n[o], v: a }));
}
function Ut(t) {
  const e = X(t), n = e.timestamp ?? e.t ?? e.time, r = e.value ?? e.v ?? e.y;
  if (n == null || n === "") return [];
  if (r == null || r === "") return [];
  const a = Number(r), o = typeof n == "number" ? n : Date.parse(String(n || ""));
  return !Number.isFinite(a) || !Number.isFinite(o) ? [] : [{ timestamp: new Date(o).toISOString(), value: a }];
}
function bn(t) {
  const e = t.points || [];
  return {
    ts: e.map((n) => Date.parse(n.timestamp)),
    v: e.map((n) => n.value),
    q: e.map(() => "ok")
  };
}
function wn(t) {
  if (!t) return "";
  if (typeof t == "string") return t;
  if (!et(t)) return String(t);
  const e = t.device_id || t.deviceId || "", n = t.param_id || t.paramId || "", r = t.instance || "", a = t.endpoint || "", o = t.signal_id || t.signalId || "";
  return `device=${e} param=${n} instance=${r} signal=${o} endpoint=${a}`.trim();
}
function Ht(t) {
  if (typeof t == "number" && Number.isFinite(t)) return t;
  if (typeof t != "string" || !t) return;
  const e = Date.parse(t);
  return Number.isFinite(e) ? e : void 0;
}
function k(t, e = "") {
  return t == null || t === "" ? e : String(t);
}
function mt(t) {
  if (t == null || t === "") return;
  const e = Number(t);
  return Number.isFinite(e) ? e : void 0;
}
function vn(t, e) {
  return typeof t == "string" || typeof t == "number" ? t : e;
}
function X(t) {
  return et(t) ? t : {};
}
function yt(t) {
  return et(t) ? t : void 0;
}
function et(t) {
  return !!t && typeof t == "object" && !Array.isArray(t);
}
function Un({ adapter: t, store: e, wallId: n, className: r }) {
  const [a, o] = O("monitor"), [i, c] = O(""), [s, l] = O(null), u = we(e), f = t.list();
  t.channels();
  const m = Tt(
    () => f.filter((p) => p.role === a),
    [f, a]
  ), d = Tt(() => {
    const p = {};
    return m.forEach((g) => {
      p[g.group] || (p[g.group] = {}), p[g.group][g.subgroup] || (p[g.group][g.subgroup] = []), p[g.group][g.subgroup].push(g);
    }), p;
  }, [m]);
  V(() => {
    var g;
    if (s && ((g = d[s.group]) != null && g[s.subgroup])) return;
    const p = Object.keys(d)[0];
    if (p) {
      const N = Object.keys(d[p])[0];
      l({ group: p, subgroup: N });
    } else
      l(null);
  }, [a, d]);
  const b = s && d[s.group] ? d[s.group][s.subgroup] || [] : [], h = b.filter(
    (p) => !i || p.name.toLowerCase().includes(i.toLowerCase()) || String(p.id).includes(i)
  );
  return /* @__PURE__ */ x("div", { className: "sf-dict" + (r ? " " + r : ""), children: [
    /* @__PURE__ */ x("div", { className: "sf-dict-rail", children: [
      /* @__PURE__ */ x("div", { className: "sf-dict-tabs", children: [
        /* @__PURE__ */ _("button", { className: a === "monitor" ? "active" : "", onClick: () => o("monitor"), children: "Telemetry" }),
        /* @__PURE__ */ _("button", { className: a === "control" ? "active" : "", onClick: () => o("control"), children: "Telecommands" })
      ] }),
      /* @__PURE__ */ _("input", { className: "sf-dict-search", placeholder: "search signals…", value: i, onChange: (p) => c(p.target.value) }),
      /* @__PURE__ */ _("div", { className: "sf-dict-groups", children: Object.entries(d).map(([p, g]) => /* @__PURE__ */ _(yn, { group: p, sgrps: g, selected: s, onSelect: l, query: i }, p)) })
    ] }),
    /* @__PURE__ */ x("div", { className: "sf-dict-main", children: [
      s && /* @__PURE__ */ x("div", { className: "sf-dict-breadcrumb", children: [
        /* @__PURE__ */ _("span", { children: s.group }),
        /* @__PURE__ */ _("span", { className: "sep", children: "›" }),
        /* @__PURE__ */ _("span", { children: s.subgroup }),
        /* @__PURE__ */ x("span", { className: "sf-count", children: [
          b.length,
          " signals"
        ] })
      ] }),
      h.map((p) => /* @__PURE__ */ _(
        xn,
        {
          signal: p,
          channels: t.channelsForSignal(p),
          adapter: t,
          wallId: n,
          assigns: u,
          tab: a
        },
        p.id
      )),
      b.length === 0 && /* @__PURE__ */ _("div", { className: "sf-dict-empty", children: "No signals here." })
    ] })
  ] });
}
function yn({
  group: t,
  sgrps: e,
  selected: n,
  onSelect: r,
  query: a
}) {
  const [o, i] = O(!0);
  let c = e;
  if (a) {
    const l = {};
    if (Object.entries(e).forEach(([u, f]) => {
      const m = f.filter((d) => d.name.toLowerCase().includes(a.toLowerCase()) || String(d.id).includes(a));
      m.length && (l[u] = m);
    }), !Object.keys(l).length) return null;
    c = l;
  }
  const s = Object.values(c).reduce((l, u) => l + u.length, 0);
  return /* @__PURE__ */ x("div", { className: "sf-dict-group", children: [
    /* @__PURE__ */ x("div", { className: "sf-dict-group-head", onClick: () => i((l) => !l), children: [
      /* @__PURE__ */ _("span", { children: o ? "▾" : "▸" }),
      /* @__PURE__ */ _("span", { children: t }),
      /* @__PURE__ */ _("span", { className: "sf-count", children: s })
    ] }),
    o && Object.entries(c).map(([l, u]) => {
      const f = (n == null ? void 0 : n.group) === t && (n == null ? void 0 : n.subgroup) === l;
      return /* @__PURE__ */ x(
        "div",
        {
          className: "sf-dict-sgrp" + (f ? " selected" : ""),
          onClick: () => r({ group: t, subgroup: l }),
          children: [
            /* @__PURE__ */ _("span", { children: l }),
            /* @__PURE__ */ _("span", { className: "sf-count", children: u.length })
          ]
        },
        l
      );
    })
  ] });
}
function xn({
  signal: t,
  channels: e,
  adapter: n,
  wallId: r,
  assigns: a,
  tab: o
}) {
  const i = e.every((s) => a.hasAssignment(r, t.id, s.device_id, s.instance));
  function c() {
    i ? e.forEach((s) => a.remove(r, t.id, s.device_id, s.instance)) : e.forEach((s) => a.add(r, t.id, s.device_id, s.instance));
  }
  return /* @__PURE__ */ x("div", { className: "sf-signal-row", children: [
    /* @__PURE__ */ x("div", { className: "sf-signal-head", children: [
      /* @__PURE__ */ _("span", { className: "sf-signal-name", children: t.name }),
      /* @__PURE__ */ x("span", { className: "sf-signal-meta", children: [
        "#",
        t.id,
        t.unit ? ` · ${t.unit}` : "",
        " · ",
        t.kind
      ] })
    ] }),
    e.length > 0 && o === "monitor" && /* @__PURE__ */ x("label", { className: "sf-signal-assign", children: [
      /* @__PURE__ */ _("input", { type: "checkbox", checked: i, onChange: c }),
      /* @__PURE__ */ _("span", { className: "sf-assign-swatch", style: { background: n.colorForRole(t.role) } }),
      "Show in wall · ",
      e.length,
      " ch"
    ] })
  ] });
}
function Hn({ walls: t, selectedWallId: e, onSelect: n }) {
  const [r, a] = O(""), [o, i] = O(null), [c, s] = O("");
  function l() {
    const m = r.trim();
    if (!m) return;
    const d = t.add(m);
    a(""), n(d.wall_id);
  }
  function u(m, d) {
    i(m), s(d);
  }
  function f() {
    o && c.trim() && t.rename(o, c.trim()), i(null), s("");
  }
  return /* @__PURE__ */ x("div", { className: "sf-wall-manager", children: [
    /* @__PURE__ */ x("div", { className: "sf-wall-list", children: [
      t.walls.map((m) => /* @__PURE__ */ x(
        "div",
        {
          className: "sf-wall-item" + (m.wall_id === e ? " selected" : "") + (m.preset ? " preset" : ""),
          onClick: () => n(m.wall_id),
          children: [
            o === m.wall_id ? /* @__PURE__ */ _(
              "input",
              {
                autoFocus: !0,
                value: c,
                onChange: (d) => s(d.target.value),
                onBlur: f,
                onKeyDown: (d) => {
                  d.key === "Enter" && f(), d.key === "Escape" && i(null);
                },
                onClick: (d) => d.stopPropagation()
              }
            ) : /* @__PURE__ */ _("span", { className: "sf-wall-label", children: m.label }),
            !m.preset && o !== m.wall_id && /* @__PURE__ */ x("span", { className: "sf-wall-actions", children: [
              /* @__PURE__ */ _("button", { onClick: (d) => {
                d.stopPropagation(), u(m.wall_id, m.label);
              }, children: "✎" }),
              /* @__PURE__ */ _("button", { onClick: (d) => {
                d.stopPropagation(), t.remove(m.wall_id);
              }, children: "✕" })
            ] })
          ]
        },
        m.wall_id
      )),
      t.walls.length === 0 && /* @__PURE__ */ _("div", { className: "sf-wall-empty", children: "No walls yet. Create one below." })
    ] }),
    /* @__PURE__ */ x("div", { className: "sf-wall-add", children: [
      /* @__PURE__ */ _(
        "input",
        {
          placeholder: "New wall label…",
          value: r,
          onChange: (m) => a(m.target.value),
          onKeyDown: (m) => {
            m.key === "Enter" && l();
          }
        }
      ),
      /* @__PURE__ */ _("button", { onClick: l, disabled: !r.trim(), children: "+ Add wall" })
    ] })
  ] });
}
function Jn({
  tile: t,
  heroGraph: e,
  height: n = 280,
  currentTimeMs: r,
  hoverTimeMs: a,
  className: o,
  dataGraphRenderer: i,
  syncKey: c = "sf-wall"
}) {
  const s = U(null), l = U(null), u = U(null), f = U(e), m = U(r), d = U(a);
  V(() => {
    var h;
    f.current = e, (h = l.current) == null || h.redraw();
  }, [e]), V(() => {
    var F;
    m.current = r;
    const h = l.current, p = u.current;
    if (!h || !p) return;
    const g = h.width || ((F = s.current) == null ? void 0 : F.offsetWidth) || 900, N = wt(p, r, g);
    h.setData(N.data, !1), h.redraw();
  }, [r]), V(() => {
    var h;
    d.current = a, (h = l.current) == null || h.redraw();
  }, [a]);
  const b = he(() => {
    if (!s.current) return;
    const h = s.current.offsetWidth || 900, p = wt(t, m.current, h), g = {
      draw: [
        (F) => {
          ln(
            F,
            t,
            f.current,
            m.current,
            d.current
          );
        }
      ]
    }, N = {
      width: h,
      height: n,
      series: p.series,
      scales: p.scales,
      axes: p.axes,
      hooks: g,
      cursor: { sync: { key: c } }
    };
    l.current && l.current.destroy(), l.current = new _e(N, p.data, s.current), u.current = t;
  }, [t, n, c]);
  return V(() => {
    b();
    const h = new ResizeObserver(() => {
      const p = l.current, g = u.current, N = s.current;
      if (!p || !g || !N) return;
      const F = N.offsetWidth || p.width || 900;
      p.setSize({ width: F, height: n });
      const w = wt(g, m.current, F);
      p.setData(w.data, !1), p.redraw();
    });
    return s.current && h.observe(s.current), () => {
      var p;
      h.disconnect(), (p = l.current) == null || p.destroy(), l.current = null, u.current = null;
    };
  }, [b, n]), /* @__PURE__ */ _(
    "div",
    {
      ref: s,
      className: o,
      "data-graph-renderer": i ?? t.renderer ?? "signalforge.tile.uplot",
      "data-graph-tile": t.tile_id ?? t.id,
      style: { width: "100%" }
    }
  );
}
export {
  ft as CANONICAL_TILE_RENDERER,
  Pn as HeroTopTimeAxis,
  qt as SERIES_ROLE_META,
  On as SharedTimeAxis,
  Un as SignalDictionary,
  ve as TileClient,
  te as TimeAxisTrack,
  Jn as UPlotTileRenderer,
  Hn as WallManager,
  Vt as axisLabel,
  En as blockLabel,
  Ze as buildAxes,
  Ke as buildScales,
  Ot as cardPriority,
  Ne as chooseTickStep,
  ke as clampRange,
  zn as clampTime,
  Se as colorForSignal,
  Rn as commandCenterGapBreaks,
  re as commandCenterProjectedSeries,
  kt as commandCenterTraceGapMs,
  De as decimationValue,
  Le as displayValue,
  lt as distinctivePalette,
  ln as drawTileOverlays,
  fn as emptyGraphTile,
  Ln as eventColor,
  We as fitCanvasText,
  Ve as formatLegendValue,
  Ie as formatMarkerDateTime,
  zt as formatPressure,
  Wt as formatScientific,
  Cn as graphCardPriority,
  St as graphCardRank,
  Tn as graphSectionPriority,
  Pt as graphSectionRank,
  Vn as inTimeRange,
  ie as interpolationValue,
  ht as isDiscreteSeries,
  Wn as legendReadouts,
  He as lineWidthFor,
  ot as loadAssignments,
  Z as loadWalls,
  st as logScale,
  jt as logSplits,
  dt as lttb,
  be as makeAssignment,
  Ee as markerColor,
  Yn as measuredElementWidth,
  mn as normalizeGraphTile,
  Oe as operatorMarkerLines,
  $n as orderLegendSignals,
  I as paddedRange,
  ye as palette,
  xe as paletteForID,
  Zt as pickTileLevel,
  It as placeMarkerLabel,
  ae as rawValueAt,
  Re as rectanglesOverlap,
  Gn as renderKindFor,
  qn as renderSeriesFromGraphTile,
  In as resampleSeries,
  Lt as roleColors,
  Dt as saveAssignments,
  gt as saveWalls,
  le as scaleForSeries,
  Me as semanticColor,
  Ue as seriesDrawOrder,
  dn as seriesRoleColor,
  de as seriesRoleMeta,
  Bn as sharedTimeGrid,
  ze as shortGateLabel,
  Et as signalColors,
  ut as signalPriority,
  je as stateAt,
  jn as stateBlocks,
  oe as stateLabel,
  bt as tickLabel,
  Dn as tileCardPriority,
  Mt as timeTicks,
  Ge as unitForAxis,
  wt as uplotData,
  we as useAssignments,
  An as useTileSeries,
  Fn as useWalls,
  R as valueFromInterpolation,
  Fe as viewportSeries,
  Qe as ySplits
};
//# sourceMappingURL=signalforge-web.es.js.map
