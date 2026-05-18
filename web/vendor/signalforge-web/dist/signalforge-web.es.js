var le = Object.defineProperty;
var ue = (t, e, r) => e in t ? le(t, e, { enumerable: !0, configurable: !0, writable: !0, value: r }) : t[e] = r;
var at = (t, e, r) => ue(t, typeof e != "symbol" ? e + "" : e, r);
import { useState as P, useEffect as V, useRef as U, useMemo as Ct, useCallback as de } from "react";
import { jsx as _, jsxs as x } from "react/jsx-runtime";
import fe from "uplot";
function Ht(t) {
  return `${t}.assignments`;
}
function ct(t, e, r) {
  return `${t}@${e}/${r}`;
}
function me(t) {
  const e = String(t || "").match(/^(\d+)@([^/]+)\/(\d+)$/);
  return e ? { param_id: parseInt(e[1], 10), device_id: e[2], instance: parseInt(e[3], 10) || 1 } : null;
}
function Jt(t, e) {
  const r = t || {}, n = me(r.target_id ?? ""), a = r.options || {}, o = Number(r.param_id ?? a.param_id ?? (n == null ? void 0 : n.param_id) ?? NaN), i = String(r.device_id ?? a.device_id ?? (n == null ? void 0 : n.device_id) ?? ""), l = Number(r.instance ?? a.instance ?? (n == null ? void 0 : n.instance) ?? 1) || 1, s = String((r.wall_id ?? "") || "wall");
  if (!i || !Number.isFinite(o)) return null;
  const c = r.target_id ?? ct(o, i, l), u = r.tile_id ?? `${s}-${c}`;
  return {
    wall_id: s,
    tile_id: u,
    target_id: c,
    kind: r.kind ?? "trend",
    options: { ...a, param_id: o, device_id: i, instance: l },
    param_id: o,
    device_id: i,
    instance: l
  };
}
function ot(t) {
  try {
    const e = JSON.parse(localStorage.getItem(Ht(t.namespace)) || "[]");
    return Array.isArray(e) ? e.map((r) => Jt(r, t.namespace)).filter((r) => r !== null) : [];
  } catch {
    return [];
  }
}
function Tt(t, e) {
  const r = t.map((n) => Jt(n, e.namespace)).filter((n) => n !== null);
  localStorage.setItem(Ht(e.namespace), JSON.stringify(r)), typeof window < "u" && window.dispatchEvent(new CustomEvent(`${e.namespace}-assignments-changed`));
}
function pe(t, e, r, n = 1) {
  const a = ct(e, r, n);
  return {
    wall_id: t,
    tile_id: `${t}-${a}`,
    target_id: a,
    kind: "trend",
    options: { param_id: e, device_id: r, instance: n },
    param_id: e,
    device_id: r,
    instance: n
  };
}
function he(t) {
  const [e, r] = P(() => ot(t));
  return V(() => {
    const n = `${t.namespace}-assignments-changed`, a = () => r(ot(t));
    return window.addEventListener(n, a), () => window.removeEventListener(n, a);
  }, [t.namespace]), {
    list: e,
    add(n, a, o, i = 1) {
      const l = ot(t), s = pe(n, a, o, i);
      l.find((c) => c.wall_id === n && c.target_id === s.target_id) || Tt([...l, s], t);
    },
    remove(n, a, o, i = 1) {
      const l = ct(a, o, i);
      Tt(ot(t).filter((s) => !(s.wall_id === n && s.target_id === l)), t);
    },
    forWall(n) {
      return e.filter((a) => a.wall_id === n);
    },
    hasAssignment(n, a, o, i = 1) {
      const l = ct(a, o, i);
      return e.some((s) => s.wall_id === n && s.target_id === l);
    }
  };
}
function Xt(t) {
  return `${t}.walls`;
}
const yt = (t) => `${t}-walls-changed`;
function Z(t) {
  try {
    const e = JSON.parse(localStorage.getItem(Xt(t)) || "[]");
    return Array.isArray(e) ? e.filter((r) => r && typeof r.wall_id == "string" && typeof r.label == "string") : [];
  } catch {
    return [];
  }
}
function _t(t, e) {
  localStorage.setItem(Xt(e), JSON.stringify(t)), typeof window < "u" && window.dispatchEvent(new CustomEvent(yt(e)));
}
function Nn(t) {
  const [e, r] = P(() => Z(t));
  return V(() => {
    const n = () => r(Z(t));
    return window.addEventListener(yt(t), n), () => window.removeEventListener(yt(t), n);
  }, [t]), {
    walls: e,
    add(n) {
      const a = { wall_id: `${t}-wall-${Date.now()}`, label: n };
      return _t([...Z(t), a], t), a;
    },
    rename(n, a) {
      _t(Z(t).map((o) => o.wall_id === n ? { ...o, label: a } : o), t);
    },
    remove(n) {
      _t(Z(t).filter((a) => a.wall_id !== n), t);
    },
    wallForDevice(n) {
      return { wall_id: `device-${n}`, label: `Device · ${n}` };
    }
  };
}
function Kt(t) {
  return t <= 5 * 6e4 ? "live" : t <= 6 * 60 * 6e4 ? "minute" : "hour";
}
class _e {
  constructor(e, r = {}) {
    at(this, "cache", /* @__PURE__ */ new Map());
    at(this, "inflight", /* @__PURE__ */ new Map());
    at(this, "ttlMs");
    this.adapter = e, this.ttlMs = r.ttlMs ?? 3e4;
  }
  cacheKey(e, r, n) {
    return `${e}/${r}@${n}`;
  }
  async fetch(e, r, n) {
    const a = this.cacheKey(e, r, n), o = this.cache.get(a);
    if (o && Date.now() - o.fetchedAt < this.ttlMs) return o.tile;
    const i = this.inflight.get(a);
    if (i) return i;
    const l = this.adapter.fetchTile(e, r, n).then((s) => (this.cache.set(a, { tile: s, fetchedAt: Date.now() }), this.inflight.delete(a), s)).catch((s) => {
      throw this.inflight.delete(a), s;
    });
    return this.inflight.set(a, l), l;
  }
  fetchForViewport(e, r, n) {
    return this.fetch(e, r, Kt(n));
  }
  invalidate(e) {
    if (!e) {
      this.cache.clear();
      return;
    }
    for (const r of this.cache.keys())
      r.startsWith(`${e}/`) && this.cache.delete(r);
  }
}
function Fn(t, e, r, n, a = 5e3) {
  const [o, i] = P({ status: "loading", tile: null }), l = U(null);
  return l.current || (l.current = new _e(t)), V(() => {
    const s = l.current;
    let c = !1;
    const u = Kt(n);
    async function f() {
      try {
        const h = await s.fetch(e, r, u);
        c || i({ status: "ok", tile: h });
      } catch (h) {
        c || i({ status: "error", tile: null, error: String(h) });
      }
    }
    if (f(), u === "live") {
      const h = setInterval(f, a);
      return () => {
        c = !0, clearInterval(h);
      };
    }
    return () => {
      c = !0;
    };
  }, [e, r, n, a]), o;
}
const Dt = {
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
}, Lt = {
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
function ge(t) {
  return lt[t % lt.length];
}
function be(t, e) {
  let r = e + 17;
  for (let n = 0; n < t.length; n += 1) r = (r << 5) - r + t.charCodeAt(n) | 0;
  return lt[Math.abs(r) % lt.length];
}
function we(t, e = 0) {
  const r = "kind" in t ? t.kind : "render_kind" in t ? t.render_kind : void 0;
  if (t.color && !t.color.includes("var(")) return t.color;
  if (Lt[t.id]) return Lt[t.id];
  const n = ve(t.id);
  if (n) return n;
  const a = Dt[t.role];
  if (a) return a;
  const o = r ? Dt[r] : void 0;
  return o || (be(t.id, e) ?? ge(e));
}
function ve(t) {
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
function An(t) {
  return [...t].sort((e, r) => ut(e) - ut(r));
}
function $n(t, e) {
  return xt(t) - xt(e);
}
function Cn(t, e) {
  return Et(t) - Et(e);
}
function Et(t) {
  return Math.min(...t.cards.map(xt), 100);
}
function xt(t) {
  const e = t.id.toLowerCase(), r = t.title.toLowerCase();
  return e === "thermal_program" ? 0 : e.includes("dut_temperature") || r.includes("dut temperature") ? 10 : e.includes("dut_power") || r.includes("dut power") ? 20 : e.includes("tmtc_health") ? 30 : e.includes("tmtc_counters") ? 40 : e.includes("state_change") || t.render_kind === "swimlane" ? 50 : e.includes("functional_events") || t.render_kind === "event_rail" ? 60 : e.includes("facility") || e.includes("building") || e.includes("source_quality") || r.includes("testbed") ? 80 : 70;
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
function Tn(t, e) {
  const r = Ot(t), n = Ot(e);
  return r !== n ? r - n : t.default_expanded !== e.default_expanded ? t.default_expanded ? -1 : 1 : t.card_id.localeCompare(e.card_id);
}
function Dn(t) {
  const e = (t ?? "").toLowerCase();
  return e.includes("functional") || e.includes("gate") ? "#ffb000" : e.includes("evidence") ? "#b079ff" : e.includes("interlock") || e.includes("fault") ? "#ff315f" : e.includes("stability") || e.includes("dwell") ? "#00d6a3" : e.includes("pressure") ? "#1f6fff" : "#31d6ff";
}
function Ln(t, e) {
  const r = String(t ?? "").trim();
  return r && r !== "0" && r !== "1" ? r : e > 0 ? "ACTIVE" : "idle";
}
function ye(t, e, r) {
  const n = Math.max(1, e.end - e.start), a = Math.min(n, Math.max(1, r)), o = Math.max(a, Math.min(n, t.end - t.start));
  let i = t.start, l = t.start + o;
  return i < e.start && (i = e.start, l = i + o), l > e.end && (l = e.end, i = l - o), { start: Math.round(i), end: Math.round(l) };
}
const Zt = 14;
function Mt(t, e, r) {
  const n = Date.parse(t), a = Date.parse(e), o = Math.max(1, a - n), i = Math.max(10, Math.min(20, r || Zt)), l = xe(o, i), s = Math.ceil(n / l) * l, c = [];
  for (let f = s; f <= a && c.length < 24; f += l) {
    if (f < n) continue;
    const h = new Date(f);
    c.push({ iso: h.toISOString(), ratio: (f - n) / o, label: gt(h, l) });
  }
  (!c.length || c[0].ratio > 0.02) && c.unshift({ iso: new Date(n).toISOString(), ratio: 0, label: gt(new Date(n), l) });
  const u = c[c.length - 1];
  return u && u.ratio < 0.98 && c.push({ iso: new Date(a).toISOString(), ratio: 1, label: gt(new Date(a), l) }), c.filter((f, h, d) => h === 0 || f.iso !== d[h - 1].iso);
}
function xe(t, e) {
  const r = t / Math.max(1, e - 1), n = [
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
  return n.find((a) => a >= r) ?? n[n.length - 1];
}
function gt(t, e) {
  const r = t.toLocaleTimeString(void 0, { hour: "2-digit", minute: "2-digit" });
  return e < 24 * 60 * 6e4 ? r : `${t.toLocaleDateString(void 0, { month: "short", day: "2-digit" })} ${r}`;
}
function Qt({ ticks: t, start: e, end: r, nowRatio: n, hoverTimeMs: a, peekTimeMs: o, compact: i }) {
  return /* @__PURE__ */ x("div", { className: `time-axis-track ${i ? "time-axis-track-compact" : ""}`, children: [
    n !== void 0 && /* @__PURE__ */ _("i", { className: "time-axis-elapsed", style: { width: `${n * 100}%` } }),
    n !== void 0 && /* @__PURE__ */ _("b", { className: "time-axis-now", style: { left: `${n * 100}%` }, title: "Current replay time" }),
    o !== void 0 && /* @__PURE__ */ _("b", { className: "time-axis-peek", style: { left: `${Math.max(0, Math.min(100, (o - e) / Math.max(1, r - e) * 100))}%` }, title: "Drag peek time" }),
    a !== void 0 && /* @__PURE__ */ _("b", { className: "time-axis-hover", style: { left: `${Math.max(0, Math.min(100, (a - e) / Math.max(1, r - e) * 100))}%` } }),
    t.map((l) => /* @__PURE__ */ x("span", { className: "time-axis-tick", style: { left: `${l.ratio * 100}%` }, children: [
      /* @__PURE__ */ _("i", {}),
      /* @__PURE__ */ _("em", { children: l.label })
    ] }, l.iso))
  ] });
}
function En({ timeRange: t, currentTimeMs: e, hoverTimeMs: r, readoutTimeMs: n, tickCount: a }) {
  const o = t.start, i = t.end, l = typeof e == "number" && Number.isFinite(e) ? Math.max(0, Math.min(1, (e - o) / Math.max(1, i - o))) : void 0, s = Mt(new Date(o).toISOString(), new Date(i).toISOString(), a ?? Zt);
  return /* @__PURE__ */ _("div", { className: "hero-top-time-axis", "aria-label": "Hero graph top time axis", children: /* @__PURE__ */ _(Qt, { ticks: s, start: o, end: i, nowRatio: l, hoverTimeMs: r, peekTimeMs: n !== r ? n : void 0, compact: !0 }) });
}
function On({
  fullRange: t,
  timeRange: e,
  currentTimeMs: r,
  hoverTimeMs: n,
  peekTimeMs: a,
  plotBounds: o,
  onTimeRange: i,
  tickCount: l
}) {
  const s = Mt(new Date(e.start).toISOString(), new Date(e.end).toISOString(), l), c = e.start, u = e.end, f = r, h = typeof f == "number" && Number.isFinite(f) ? Math.max(0, Math.min(1, (f - c) / Math.max(1, u - c))) : void 0, d = Math.max(0, (u - c) / 36e5), b = Math.max(1, t.end - t.start), p = Math.max(1, e.end - e.start), m = p < b * 0.995, g = Math.max(6e4, b / 600), N = o ? {
    "--time-axis-grid-left": `${o.left}px`,
    "--time-axis-grid-right": `${o.right}px`,
    "--time-axis-left": `${o.left}px`,
    "--time-axis-right": `${o.right}px`
  } : void 0, F = (A) => {
    const R = Math.max(g, Math.min(b, p * A)), G = (e.start + e.end) / 2;
    i(ye({ start: Math.round(G - R / 2), end: Math.round(G + R / 2) }, t, g));
  }, w = (A) => {
    const R = Math.max(0, b - p), G = Number(A) / 1e3 * R;
    i({ start: Math.round(t.start + G), end: Math.round(t.start + G + p) });
  }, S = Math.round((e.start - t.start) / Math.max(1, b - p) * 1e3);
  return /* @__PURE__ */ x("div", { className: "operator-shared-time-axis", "aria-label": "Shared graph time axis", style: N, children: [
    /* @__PURE__ */ _("span", { className: "time-axis-label", children: "TIME" }),
    /* @__PURE__ */ _(Qt, { ticks: s, start: c, end: u, nowRatio: h, hoverTimeMs: n, peekTimeMs: a }),
    /* @__PURE__ */ x("div", { className: "time-axis-sub-row", children: [
      /* @__PURE__ */ x("div", { className: "time-axis-controls", children: [
        /* @__PURE__ */ x("span", { children: [
          d.toFixed(d >= 24 ? 0 : 1),
          " h"
        ] }),
        /* @__PURE__ */ _("small", { children: "zoom" }),
        /* @__PURE__ */ _("button", { type: "button", onClick: () => F(1.35), "aria-label": "Zoom out", children: "-" }),
        /* @__PURE__ */ _("button", { type: "button", onClick: () => F(0.72), "aria-label": "Zoom in", children: "+" }),
        /* @__PURE__ */ _("button", { type: "button", disabled: !m, onClick: () => i(t), children: "full" })
      ] }),
      /* @__PURE__ */ x("label", { className: "time-axis-scrollbar", children: [
        /* @__PURE__ */ _("small", { children: "scroll" }),
        /* @__PURE__ */ _("input", { type: "range", min: "0", max: "1000", step: "1", disabled: !m, value: Math.max(0, Math.min(1e3, S)), onChange: (A) => w(Number(A.currentTarget.value)) })
      ] })
    ] })
  ] });
}
function Me(t, e, r) {
  const n = e.points ?? [];
  if (n.length < 4 || pt(e)) return e;
  const a = Math.max(180, Math.min(n.length, Math.round(r * 1.65)));
  if (n.length <= a) return e;
  const o = (i) => Ae(t, e, i);
  return tt(e.axis_id) ? { ...e, points: Se(n, a, o) } : { ...e, points: dt(n, a, o) };
}
function dt(t, e, r) {
  if (!t || e >= t.length || e < 3) return t;
  const n = t.map((l) => ({ point: l, x: Date.parse(l.timestamp), y: r(l.value) })).filter((l) => Number.isFinite(l.x) && Number.isFinite(l.y));
  if (n.length <= e) return n.map((l) => l.point);
  const a = [n[0].point], o = (n.length - 2) / (e - 2);
  let i = 0;
  for (let l = 0; l < e - 2; l++) {
    const s = Math.floor((l + 0) * o) + 1, c = Math.floor((l + 1) * o) + 1, u = Math.floor((l + 1) * o) + 1, f = Math.floor((l + 2) * o) + 1, h = n.slice(s, Math.min(c, n.length - 1)), d = n.slice(u, Math.min(f, n.length)), b = d.reduce((w, S) => w + S.x, 0) / Math.max(1, d.length), p = d.reduce((w, S) => w + S.y, 0) / Math.max(1, d.length), m = n[i];
    let g = h[0] ?? n[Math.min(s, n.length - 2)], N = h.length ? s : Math.min(s, n.length - 2), F = -1;
    h.forEach((w, S) => {
      const A = Math.abs((m.x - b) * (w.y - m.y) - (m.x - w.x) * (p - m.y));
      A > F && (F = A, g = w, N = s + S);
    }), a.push(g.point), i = N;
  }
  return a.push(n[n.length - 1].point), a;
}
function Se(t, e, r) {
  if (!t || e >= t.length || e < 3) return t;
  const n = [];
  let a = [];
  const o = () => {
    a.length && (n.push({ kind: "run", points: a }), a = []);
  };
  for (const p of t) {
    const m = Date.parse(p.timestamp), g = r(p.value);
    if (Number.isFinite(m) && Number.isFinite(g)) {
      a.push(p);
      continue;
    }
    o(), Number.isFinite(m) && n.push({ kind: "gap", point: p });
  }
  o();
  const i = n.filter((p) => p.kind === "run"), l = n.length - i.length;
  if (!l) return dt(t, e, r);
  const s = i.reduce((p, m) => p + m.points.length, 0), c = Math.max(0, e - l), u = i.map((p) => {
    if (p.points.length <= 2) return p.points.length;
    const m = s > 0 ? Math.round(p.points.length / s * c) : p.points.length;
    return Math.min(p.points.length, Math.max(3, m));
  }), f = (p) => p.points.length <= 2 ? p.points.length : 3;
  let h = l + u.reduce((p, m) => p + m, 0);
  for (; h > e; ) {
    let p = -1;
    for (let m = 0; m < u.length; m += 1)
      u[m] <= f(i[m]) || (p === -1 || u[m] > u[p]) && (p = m);
    if (p === -1) break;
    u[p] -= 1, h -= 1;
  }
  if (h > e)
    return ke(n, e, r);
  const d = [];
  let b = 0;
  for (const p of n) {
    if (p.kind === "gap") {
      d.push(p.point);
      continue;
    }
    const m = u[b] ?? p.points.length;
    d.push(...m >= p.points.length ? p.points : dt(p.points, m, r) ?? []), b += 1;
  }
  return d;
}
function ke(t, e, r) {
  const n = t.filter((d) => d.kind === "run"), a = t.map((d, b) => ({ segment: d, index: b })).filter((d) => d.segment.kind === "gap");
  if (!n.length) return a.slice(0, e).map((d) => d.segment.point);
  const o = n.reduce((d, b) => d + b.points.length, 0), i = Math.max(0, e - 1), l = Math.round(e * (a.length / (a.length + n.length))), s = Math.min(a.length, i, Math.max(1, l)), c = /* @__PURE__ */ new Set();
  if (s > 0)
    for (let d = 0; d < s; d += 1) {
      const b = a[Math.floor(d * a.length / s)];
      b && c.add(b.index);
    }
  const u = Ne(n, Math.max(0, e - c.size), o), f = [];
  let h = 0;
  for (let d = 0; d < t.length && f.length < e; d += 1) {
    const b = t[d];
    if (b.kind === "gap") {
      c.has(d) && f.push(b.point);
      continue;
    }
    const p = Math.min(u[h] ?? 0, e - f.length);
    f.push(...Fe(b.points, p, r)), h += 1;
  }
  return f.slice(0, e);
}
function Ne(t, e, r) {
  const n = t.map(() => 0);
  if (e <= 0) return n;
  if (e < t.length) {
    for (let o = 0; o < e; o += 1) {
      const i = Math.floor(o * t.length / e);
      n[i] = 1;
    }
    return n;
  }
  t.forEach((o, i) => {
    n[i] = Math.min(o.points.length, 1);
  });
  let a = e - n.reduce((o, i) => o + i, 0);
  for (; a > 0; ) {
    let o = -1, i = 0;
    if (t.forEach((l, s) => {
      const c = r > 0 ? l.points.length / r * e : e / Math.max(1, t.length), u = Math.min(l.points.length, Math.max(1, Math.round(c))) - n[s];
      u > i && (i = u, o = s);
    }), o === -1) break;
    n[o] += 1, a -= 1;
  }
  return n;
}
function Fe(t, e, r) {
  return e <= 0 ? [] : e >= t.length ? t : e === 1 ? [t[Math.floor((t.length - 1) / 2)]] : e === 2 ? [t[0], t[t.length - 1]] : dt(t, e, r) ?? [];
}
function Ae(t, e, r) {
  return tt(e.axis_id) ? r > 0 ? Math.log10(r) : Number.NaN : r;
}
function $e(t, e, r, n) {
  const a = [...e.points ?? []].map((c) => ({ t: Date.parse(c.timestamp), v: ee(e, c.value) })).filter((c) => Number.isFinite(c.t)).sort((c, u) => c.t - u.t);
  if (!a.length) return r.map(() => null);
  const o = pt(e) || e.render_kind === "swimlane", i = e.role === "ghost" || te(t, e), l = St(t, e);
  let s = 0;
  return r.map((c) => {
    if (Number.isFinite(n) && c > n && !i) return null;
    for (; s + 1 < a.length && a[s + 1].t <= c; ) s += 1;
    const u = a[s], f = a[Math.min(s + 1, a.length - 1)];
    if (c < a[0].t || c > a[a.length - 1].t || l > 0 && f.t - u.t > l && c > u.t && c < f.t) return null;
    if (c === u.t) return Number.isFinite(u.v) ? j(e, u.v) : null;
    if (!Number.isFinite(u.v) || !Number.isFinite(f.v)) return null;
    if (o || f.t === u.t) return j(e, u.v);
    const h = (c - u.t) / (f.t - u.t), d = u.v + (f.v - u.v) * Math.max(0, Math.min(1, h));
    return j(e, d);
  });
}
function Ce(t, e) {
  const r = St(t, e);
  if (r <= 0) return [];
  const n = [...e.points ?? []].map((o) => Date.parse(o.timestamp)).filter(Number.isFinite).sort((o, i) => o - i), a = [];
  for (let o = 1; o < n.length; o += 1)
    n[o] - n[o - 1] > r && a.push(n[o - 1] + 1, n[o] - 1);
  return a;
}
function St(t, e) {
  return t.campaign_id !== "command_center_fat" || e.render_kind === "swimlane" || pt(e) || e.role === "event" ? 0 : 2 * 60 * 60 * 1e3;
}
function te(t, e) {
  return t.campaign_id === "command_center_fat" && e.role === "command";
}
function Te(t, e, r) {
  return tt(e.axis_id) ? r > 0 ? r : Number.NaN : r;
}
function pt(t) {
  return !!t.step || t.render_kind === "counter" || t.kind === "counter" || t.role === "counter";
}
function ee(t, e) {
  return tt(t.axis_id) ? e > 0 ? Math.log10(e) : Number.NaN : e;
}
function j(t, e) {
  return Number.isFinite(e) ? tt(t.axis_id) ? 10 ** e : e : Number.NaN;
}
function tt(t) {
  return t === "pressure_log" || t === "pressure_rate_log";
}
const Q = 8;
function De(t) {
  return t.kind === "operator_breakdown" ? "rgba(255,112,67,0.98)" : t.kind === "operator_reset" ? "rgba(36,214,255,0.98)" : t.kind === "operator_reset_ready" ? "rgba(146,255,111,0.98)" : t.role === "interlock" || t.result === "fail" ? "rgba(255,49,95,0.96)" : t.role === "evidence" ? "rgba(176,121,255,0.96)" : t.kind === "functional_gate" ? "rgba(255,176,0,0.98)" : t.kind === "stability" || t.kind === "stability_achieved" || t.result === "pass" ? "rgba(0,214,163,0.96)" : "rgba(49,214,255,0.95)";
}
function Le(t, e) {
  return t.kind === "operator_breakdown" ? e ? "BD" : "BREAKDOWN" : t.kind === "operator_reset" ? e ? "RST" : "RESET" : t.kind === "operator_reset_ready" ? e ? "RDY" : "READY" : ((t.kind ?? t.label ?? t.role ?? "operator").replace(/^operator[_\s-]*/i, "").replace(/[_-]+/g, " ").trim() || "operator").toUpperCase().slice(0, e ? 6 : 14);
}
function Ee(t, e = !1) {
  return [Le(t, e), Oe(t.timestamp, e)];
}
function Oe(t, e = !1) {
  const r = new Date(t);
  return Number.isNaN(r.getTime()) ? t : r.toLocaleString(void 0, {
    weekday: "short",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: !e
  });
}
function Pt({
  x: t,
  y: e,
  labelWidth: r,
  labelHeight: n,
  left: a,
  top: o,
  width: i,
  height: l,
  placed: s,
  markerRadius: c
}) {
  const f = t > a + i * 0.68 ? [-1, 1] : [1, -1], h = Math.max(o + 4, Math.min(o + l - n - 4, e - n / 2)), d = n + Q, b = Array.from({ length: 21 }, (m, g) => Math.ceil(g / 2) * (g % 2 === 0 ? 1 : -1) * d), p = [-6 * d, -7 * d, -8 * d, -9 * d, -10 * d, 6 * d, 7 * d, 8 * d, 9 * d, 10 * d];
  for (const m of f) {
    const g = m < 0 ? t - r - c - 8 : t + c + 8, N = Math.max(a + 4, Math.min(a + i - r - 4, g));
    for (const F of [...b, ...p]) {
      const w = h + F, S = {
        x: N,
        y: Math.max(o - n * 2, Math.min(o + l + n, w)),
        width: r,
        height: n
      };
      if (!s.some((A) => Pe(S, A)))
        return S;
    }
  }
  return null;
}
function Pe(t, e) {
  return t.x < e.x + e.width + Q && t.x + t.width + Q > e.x && t.y < e.y + e.height + Q && t.y + t.height + Q > e.y;
}
function Ie(t, e, r) {
  if (t.measureText(e).width <= r) return e;
  let n = e;
  for (; n.length > 3 && t.measureText(`${n.slice(0, -1)}...`).width > r; )
    n = n.slice(0, -1);
  return `${n.slice(0, -1)}...`;
}
function Re(t) {
  return t.replace(/\s+breakdown\s+start$/i, " BD").replace(/\s+reset\s+start$/i, " RST").replace(/\s+reset\s+ready$/i, " RDY").replace(/^Stable\s+/i, "STBL ").replace(/\s+confirmed$/i, "").replace(/^Cycle\s+/i, "C").replace(/\s+dwell\s+functional\s+test/i, " FT").replace(/\s+functional\s+test/i, " FT").slice(0, 8);
}
function Pn(t, e, r, n) {
  const a = /* @__PURE__ */ new Map();
  if (r === void 0 || !Number.isFinite(r)) return a;
  const o = new Set(e.map((i) => i.id));
  return t.series.forEach((i) => {
    var s;
    if (!o.has(i.id) || Number.isFinite(r) && Number.isFinite(n) && r > n && i.role !== "ghost" && !te(t, i)) return;
    if ((s = i.spans) != null && s.length) {
      const c = ze(i, r);
      c && a.set(i.id, c);
      return;
    }
    const l = ne(i, r, t);
    l !== void 0 && We(i, l) && a.set(i.id, Be(i, l));
  }), a;
}
function We(t, e) {
  return Number.isFinite(e) && (!Ve(t.axis_id) || !Ge(t.axis_id) || e > 0);
}
function In(t, e) {
  if (!Number.isFinite(t) || !e.length) return t;
  const r = e[0], n = e[e.length - 1];
  return !Number.isFinite(r) || !Number.isFinite(n) ? t : Math.max(r, Math.min(n, t));
}
function ne(t, e, r) {
  const n = [...t.points ?? []].map((u) => ({ t: Date.parse(u.timestamp), v: ee(t, u.value) })).filter((u) => Number.isFinite(u.t)).sort((u, f) => u.t - f.t);
  if (!n.length || e < n[0].t || e > n[n.length - 1].t) return;
  if (e === n[0].t) return j(t, n[0].v);
  if (e === n[n.length - 1].t) return j(t, n[n.length - 1].v);
  let a = 0;
  for (; a + 1 < n.length && n[a + 1].t <= e; ) a += 1;
  const o = n[a], i = n[Math.min(a + 1, n.length - 1)], l = r ? St(r, t) : 0;
  if (l > 0 && i.t - o.t > l && e > o.t && e < i.t) return;
  if (e === o.t) return Number.isFinite(o.v) ? j(t, o.v) : void 0;
  if (!Number.isFinite(o.v) || !Number.isFinite(i.v)) return;
  if (pt(t) || i.t === o.t) return j(t, o.v);
  const s = (e - o.t) / (i.t - o.t), c = o.v + (i.v - o.v) * Math.max(0, Math.min(1, s));
  return j(t, c);
}
function ze(t, e) {
  var n;
  const r = (n = t.spans) == null ? void 0 : n.find((a) => {
    const o = Date.parse(a.start), i = Date.parse(a.end);
    return Number.isFinite(o) && Number.isFinite(i) && e >= o && e <= i;
  });
  if (r)
    return re(t, r.value, r.state, r.label);
}
function re(t, e, r, n) {
  const a = t.value_table ?? {}, o = (i) => {
    if (i == null) return;
    const l = String(i).trim();
    if (l)
      return a[l] ?? l;
  };
  return o(n) ?? o(r) ?? o(e);
}
function Be(t, e) {
  const r = It(t.unit ?? t.units) || je(t.axis_id);
  if (t.axis_id === "pressure_log") return `${Wt(e)} mbar`;
  if (t.axis_id === "pressure_rate_log") return `${Rt(e)} mbar/min`;
  if (t.axis_id === "pressure_mbar") return `${Wt(e)} mbar`;
  if (t.axis_id === "pressure_rate") return `${Rt(e)} mbar/min`;
  if (t.axis_id === "counter") {
    const n = It(t.unit ?? t.units);
    return n ? `${Math.round(e).toLocaleString()} ${n}` : Math.round(e).toLocaleString();
  }
  return t.axis_id === "temperature_c" ? `${e.toFixed(1)} degC` : t.axis_id === "percent" ? `${e.toFixed(0)}%` : r === "degC" ? `${e.toFixed(1)} degC` : r === "W" ? `${e.toFixed(1)} W` : r === "ms" ? `${e.toFixed(1)} ms` : r === "bar" ? `${e.toFixed(2)} bar` : `${Number.isInteger(e) ? e.toFixed(0) : e.toFixed(2)}${r ? ` ${r}` : ""}`;
}
function It(t) {
  const e = typeof t == "string" ? t.trim() : "";
  return e && e !== "_" ? e : "";
}
function Rt(t) {
  return Number.isFinite(t) ? t === 0 ? "0" : t.toExponential(2).replace("e", "E") : "";
}
function Wt(t) {
  return t <= 0 ? "0" : t < 1e-3 || t >= 1e3 ? t.toExponential(2).replace("e", "E") : t < 1 ? t.toPrecision(3) : t.toFixed(t < 10 ? 2 : 1);
}
function je(t) {
  return t === "temperature_c" ? "degC" : t === "pressure_log" || t === "pressure_mbar" ? "mbar" : t === "power_w" || t === "heat_flux_w" ? "W" : t === "bus_ms" ? "ms" : t === "pressure_bar" ? "bar" : t === "pressure_rate_log" || t === "pressure_rate" ? "mbar/min" : t === "percent" ? "%" : t === "voltage_v" ? "V" : t === "current_a" ? "A" : t === "rf_db" || t === "link_db" || t === "signal_db" ? "dB" : t === "frequency_hz" ? "Hz" : t === "ohm" ? "Ω" : "";
}
function Ve(t) {
  return t === "pressure_mbar" || t === "pressure_rate" || t === "pressure_log" || t === "pressure_rate_log";
}
function Ge(t) {
  return t === "pressure_log" || t === "pressure_rate_log";
}
function bt(t, e, r = 900) {
  const a = t.series.filter((c) => (c.points ?? []).length > 0).sort(Ye).map((c) => Me(t, c, r)), o = Ue(t, a), i = [o], l = [{}], s = /* @__PURE__ */ new Set();
  return a.forEach((c, u) => {
    const f = ae(t, c);
    s.add(f), i.push($e(t, c, o, e)), l.push({
      label: c.label,
      scale: f,
      stroke: we(c, u),
      width: qe(c.role),
      dash: c.role === "ghost" ? [7, 4] : c.role === "acceptance_band" ? [2, 5] : void 0,
      points: { show: !1 }
    });
  }), { data: i, series: l, scales: Xe(s), axes: Ke(s) };
}
function Ye(t, e) {
  const r = {
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
  }, n = (r[t.role] ?? 15) - (r[e.role] ?? 15);
  return n || ut(t) - ut(e);
}
function qe(t) {
  return t === "command" ? 1.55 : t === "ghost" ? 0.9 : t === "acceptance_band" ? 0.75 : t === "counter" || t === "source_quality" ? 1.05 : t === "dut" ? 1.1 : t === "aux" ? 0.95 : 0.85;
}
function Ue(t, e) {
  const r = Date.parse(t.t0), n = Date.parse(t.t1), a = e.flatMap((s) => (s.points ?? []).map((c) => Date.parse(c.timestamp))).filter(Number.isFinite), o = e.flatMap((s) => Ce(t, s)), i = Number.isFinite(r) ? r : Math.min(...a), l = Number.isFinite(n) ? n : Math.max(...a);
  return !Number.isFinite(i) || !Number.isFinite(l) || l <= i ? Array.from(/* @__PURE__ */ new Set([...a, ...o])).sort((s, c) => s - c) : Array.from(/* @__PURE__ */ new Set([r, n, ...a, ...o])).filter(Number.isFinite).sort((s, c) => s - c);
}
function He(t) {
  return [
    ...(t.series ?? []).flatMap((e) => [
      ...(e.points ?? []).map((r) => Date.parse(r.timestamp)),
      ...(e.spans ?? []).flatMap((r) => [Date.parse(r.start), Date.parse(r.end)])
    ]),
    ...(t.markers ?? []).map((e) => Date.parse(e.timestamp)),
    ...(t.bands ?? []).flatMap((e) => [Date.parse(e.start), Date.parse(e.end)])
  ].filter(Number.isFinite);
}
function Je(t, e) {
  const r = (e == null ? void 0 : e.start) ?? Date.parse(t.t0), n = (e == null ? void 0 : e.end) ?? Date.parse(t.t1);
  let a = Number.isFinite(r) ? r : void 0, o = Number.isFinite(n) ? n : void 0;
  if (a === void 0 || o === void 0) {
    const i = He(t);
    if (!i.length) return null;
    a === void 0 && (a = Math.min(...i)), o === void 0 && (o = Math.max(...i));
  }
  return !Number.isFinite(a) || !Number.isFinite(o) ? null : o <= a ? { start: a, end: a + 1 } : { start: a, end: o };
}
function Xe(t) {
  const e = {};
  return t.forEach((r) => {
    r === "temperature_c" ? e[r] = { range: I(12, [-92, 92]) } : r === "pressure_log" ? e[r] = { distr: 3, log: 10, range: () => [1e-8, 1200] } : r === "pressure_rate_log" ? e[r] = { distr: 3, log: 10, range: () => [1e-8, 1e3] } : r === "pressure_mbar" ? e[r] = { range: I(0.08, [0, 1200]) } : r === "pressure_rate" ? e[r] = { range: I(0.08) } : r === "pressure_bar" ? e[r] = { range: I(0.08, [0, 12]) } : r === "percent" ? e[r] = { range: (n, a, o) => [0, 100] } : r === "heat_flux_w" ? e[r] = { range: I(8, [-45, 45]) } : r === "current_a" ? e[r] = { range: I(0.1) } : r === "voltage_v" ? e[r] = { range: I(0.5) } : r === "seconds" ? e[r] = { range: I(1) } : r === "generic_numeric" ? e[r] = { range: I(1) } : e[r] = {};
  }), e;
}
function Ke(t, e) {
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
    splits: (s, c, u, f) => st(i) ? Bt(u, f) : Ze(u, f),
    size: 64,
    gap: 0,
    label: jt(i),
    labelSize: 12,
    labelGap: 0,
    values: st(i) ? (s, c) => c.map((u) => Vt(u)) : void 0
  });
  const l = Array.from(t).filter((s) => s !== i);
  return l.forEach((s) => {
    a.push({
      show: !0,
      scale: s,
      side: 1,
      stroke: s.includes("pressure") ? "#60a5fa" : "#8bd3a5",
      grid: { show: !1 },
      ticks: { show: !1 },
      size: 64,
      gap: 0,
      label: jt(s),
      labelSize: 12,
      labelGap: 0,
      splits: st(s) ? (c, u, f, h) => Bt(f, h) : void 0,
      values: st(s) ? (c, u) => u.map((f) => Vt(f)) : void 0
    });
  }), l.length || a.push({
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
  return (r, n, a) => {
    if (!Number.isFinite(n) || !Number.isFinite(a)) return ie(e) ?? [0, 1];
    if (a <= n) return zt(n - t, a + t, e);
    const o = Math.max(t, (a - n) * 0.08), i = n - o, l = a + o;
    return zt(i, l, e);
  };
}
function zt(t, e, r) {
  if (!r) return [t, e];
  const n = ie(r);
  if (!n) return [t, e];
  const a = Math.max(n[0], t), o = Math.min(n[1], e);
  return a <= o ? [a, o] : n;
}
function ie(t) {
  if (!(!t || !Number.isFinite(t[0]) || !Number.isFinite(t[1])))
    return t[0] <= t[1] ? t : [t[1], t[0]];
}
function st(t) {
  return t === "pressure_log" || t === "pressure_rate_log";
}
function Bt(t, e) {
  if (!Number.isFinite(t) || !Number.isFinite(e) || e <= 0 || e <= t) return [];
  const r = Math.ceil(Math.log10(Math.max(t, 1e-12))), n = Math.floor(Math.log10(e)), a = [];
  for (let o = r; o <= n; o += 1) a.push(Math.pow(10, o));
  return a;
}
function Ze(t, e) {
  if (!Number.isFinite(t) || !Number.isFinite(e) || e <= t) return [];
  const n = (e - t) / 8, a = Math.pow(10, Math.floor(Math.log10(n))), o = [1, 2, 2.5, 5, 10].map((s) => s * a).find((s) => n <= s) ?? a * 10, i = Math.ceil(t / o) * o, l = [];
  for (let s = i; s <= e + o * 0.25; s += o) l.push(Number(s.toFixed(6)));
  return l;
}
function jt(t, e) {
  return t === "temperature_c" ? "degC" : t === "pressure_log" ? "log10 mbar" : t === "pressure_rate_log" ? "log10 mbar/min" : t === "pressure_mbar" ? "mbar" : t === "pressure_rate" ? "mbar/min" : t === "pressure_bar" ? "bar" : t === "heat_flux_w" || t === "power_w" ? "W" : t === "current_a" ? "A" : t === "voltage_v" ? "V" : t === "seconds" ? "s" : t === "bus_ms" ? "ms" : t === "counter" ? "count" : t === "percent" ? "%" : t === "generic_numeric" ? "value" : t;
}
function ae(t, e) {
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
  const r = (e.unit ?? e.units ?? "").trim().toLowerCase();
  return r === "a" || r === "amp" || r === "amps" ? "current_a" : r === "v" || r === "volt" || r === "volts" ? "voltage_v" : r === "s" || r === "sec" || r === "secs" || r === "second" || r === "seconds" ? "seconds" : r === "ms" || r === "millisecond" || r === "milliseconds" ? "bus_ms" : r === "w" || r === "watt" || r === "watts" ? "power_w" : r === "%" || r === "percent" ? "percent" : r.includes("deg") || r === "c" || r === "°c" ? "temperature_c" : e.role === "counter" || e.kind === "counter" ? "counter" : "generic_numeric";
}
function Rn(t, e, r) {
  var a;
  if ((a = t.spans) != null && a.length)
    return t.spans.flatMap((o, i) => {
      const l = Date.parse(o.start), s = Date.parse(o.end);
      if (!Number.isFinite(l) || !Number.isFinite(s) || s < e || l > e + r) return [];
      const c = Math.max(0, Math.min(100, (l - e) / r * 100)), u = Math.max(c + 0.15, Math.min(100, (s - e) / r * 100));
      return [{
        key: `${t.id}-span-${i}`,
        left: c,
        width: u - c,
        value: o.value ?? Number(o.state ?? 0),
        label: re(t, o.value, o.state, o.label) ?? ""
      }];
    });
  const n = [...t.points ?? []].sort((o, i) => Date.parse(o.timestamp) - Date.parse(i.timestamp));
  return n.flatMap((o, i) => {
    const l = Date.parse(o.timestamp), s = i + 1 < n.length ? Date.parse(n[i + 1].timestamp) : e + r;
    if (!Number.isFinite(l) || !Number.isFinite(s) || s < e || l > e + r) return [];
    const c = Math.max(0, Math.min(100, (l - e) / r * 100)), u = Math.max(c + 0.15, Math.min(100, (s - e) / r * 100));
    return [{ key: `${t.id}-${i}`, left: c, width: u - c, value: o.value, label: String(o.value) }];
  });
}
function Wn(t, e) {
  const r = Date.parse(t);
  return Number.isFinite(r) && r >= e.start && r <= e.end;
}
function zn(t) {
  return t === "state" ? "swimlane" : t === "event" ? "event_rail" : t === "counter" ? "counter" : "line";
}
function Vt(t) {
  return Number.isFinite(t) ? t === 0 ? "0" : t.toExponential(2).replace("e", "E") : "";
}
function Qe(t, e, r, n, a, o) {
  const i = tn(e, r);
  for (const l of i) {
    const s = ne(l, n, e);
    if (s === void 0) continue;
    const c = ae(e, l), u = t.valToPos(Te(e, l, s), c);
    if (Number.isFinite(u))
      return { y: Math.max(a + 12, Math.min(a + o - 10, u)) };
  }
  return null;
}
function tn(t, e) {
  return t.series.filter((r) => (r.points ?? []).length).map((r) => ({ series: r, score: en(r, e) })).filter((r) => r.score > 0).sort((r, n) => n.score - r.score).map((r) => r.series);
}
function en(t, e) {
  const r = `${t.id} ${t.label} ${t.axis_id ?? ""} ${t.source ?? ""} ${t.role}`.toLowerCase(), n = `${e.id} ${e.label} ${e.kind} ${e.role} ${e.axis_id ?? ""}`.toLowerCase(), a = rn(e), o = oe(e);
  let i = 0;
  const l = (s, c, u) => {
    s.some((f) => n.includes(f)) && c.some((f) => r.includes(f)) && (i += u);
  };
  return l(["pressure", "vacuum", "tvac"], ["pressure", "vacuum", "tvac"], 80), l(["dut", "functional", "stability", "dwell"], ["dut", "component", "interface", "chamber"], 70), l(["shroud"], ["shroud"], 70), l(["interlock"], ["interlock", "facility"], 70), l(["operator", "command"], ["command", "chamber"], 55), l(["pump", "exhaust"], ["pump", "exhaust", "cryo", "scavenger"], 55), o && (r.includes("interlock") || r.includes("facility")) && (i += 180), e.axis_id && t.axis_id === e.axis_id && (i += 90), a && t.role === "command" && (i += 220), t.role === "actual" && (i += a ? 4 : 18), t.role === "command" && (i += a ? 34 : 8), t.role === "ghost" && (i += 4), i;
}
function oe(t) {
  return t.role === "interlock" || t.kind === "interlock" || t.result === "fail";
}
function nn(t) {
  return t.kind === "functional_gate" || t.kind === "stability" || t.kind === "stability_achieved" || t.kind === "pressure_gate" || oe(t);
}
function rn(t) {
  var e;
  return t.role === "operator_interaction" || ((e = t.kind) == null ? void 0 : e.startsWith("operator_")) || t.kind === "functional_gate" || t.kind === "stability" || t.kind === "stability_achieved";
}
function an(t, e, r, n, a, o = 0.42) {
  t.save(), t.globalAlpha = o, t.strokeStyle = a, t.lineWidth = 1, t.setLineDash([2, 4]), t.beginPath(), t.moveTo(e, r), t.lineTo(e, r + n), t.stroke(), t.restore();
}
function Gt(t, e, r, n, a, o, i, l) {
  t.save(), t.globalAlpha = 0.8, t.strokeStyle = l, t.lineWidth = 1, t.setLineDash([]), t.beginPath(), t.moveTo(e, r), t.lineTo(n < e ? n + o : n, a + i / 2), t.stroke(), t.restore();
}
function on(t, e, r) {
  const n = r < 760, a = n ? 0.018 : 0.075;
  return t.campaign_id === "tvac_qualification" && (e.includes("vacuum") || t.card_id.includes("pressure")) ? `rgba(59,130,246,${n ? 0.018 : 0.065})` : e.includes("breakdown") ? `rgba(255,112,67,${n ? 0.026 : 0.11})` : e.includes("reset") ? `rgba(36,214,255,${n ? 0.022 : 0.09})` : e.includes("cold") ? `rgba(61,133,198,${a})` : `rgba(198,119,61,${a})`;
}
function sn(t, e) {
  return t.campaign_id === "tvac_qualification" && (e.includes("vacuum") || t.card_id.includes("pressure")) ? "rgba(96,165,250,0.16)" : e.includes("breakdown") ? "rgba(255,112,67,0.22)" : e.includes("reset") ? "rgba(36,214,255,0.18)" : e.includes("cold") ? "rgba(96,165,250,0.16)" : "rgba(255,176,0,0.14)";
}
function cn(t, e, r, n, a, o) {
  var Nt, Ft, At;
  const i = t.ctx, l = t.bbox, s = l.left, c = l.top, u = l.width, f = l.height, h = Je(e, o);
  if (!h) return;
  const { start: d, end: b } = h, p = Math.max(1, b - d);
  i.save();
  const m = Mt(new Date(d).toISOString(), new Date(b).toISOString(), 14);
  i.strokeStyle = "rgba(83,112,140,0.16)", i.lineWidth = 1, i.setLineDash([]), m.forEach((v) => {
    const M = s + v.ratio * u;
    i.beginPath(), i.moveTo(M, c), i.lineTo(M, c + f), i.stroke();
  }), (e.bands ?? []).forEach((v) => {
    const M = s + (Date.parse(v.start) - d) / p * u, y = s + (Date.parse(v.end) - d) / p * u, C = (v.kind ?? "").toLowerCase(), D = Math.max(1, y - M), Y = u < 760;
    if (i.fillStyle = on(e, C, u), Y) {
      const E = Math.max(2, Math.min(7, f * 0.04));
      i.fillRect(M, c, D, E), i.fillRect(M, c + f - E, D, E);
    } else
      i.fillRect(M, c, D, f);
    i.strokeStyle = sn(e, C), i.lineWidth = u < 520 ? 0.75 : 1, Y ? (i.beginPath(), i.moveTo(M + 0.5, c + 0.5), i.lineTo(M + 0.5, c + f - 0.5), i.moveTo(M + D - 0.5, c + 0.5), i.lineTo(M + D - 0.5, c + f - 0.5), i.stroke()) : i.strokeRect(M + 0.5, c + 0.5, Math.max(0, D - 1), Math.max(0, f - 1));
  });
  const g = [], N = ln(e.markers ?? [], d, d + p, u, e.campaign_id);
  let F = 0, w = 0, S = 0;
  const A = (v, M, y) => {
    const C = c - (S + 1) * (M + 3), D = Math.max(s, Math.min(s + u - v, y));
    return S += 1, { x: D, y: C, width: v, height: M };
  };
  (e.markers ?? []).forEach((v) => {
    var $t;
    const M = Date.parse(v.timestamp);
    if (!Number.isFinite(M)) return;
    const y = s + (M - d) / p * u;
    if (y < s || y > s + u) return;
    const C = De(v), D = v.role === "operator_interaction" || (($t = v.kind) == null ? void 0 : $t.startsWith("operator_")), Y = nn(v), E = Y || D ? Qe(t, e, v, M, c, f) : null, W = (E == null ? void 0 : E.y) ?? c + 10;
    if ((Y || D) && an(i, y, c, f, C, Y ? 0.48 : 0.36), D) {
      F += 1;
      const L = e.campaign_id === "command_center_fat" || u < 760, T = (E == null ? void 0 : E.y) ?? c + 18 + (v.kind === "operator_reset" ? 34 : v.kind === "operator_reset_ready" ? 68 : 0), $ = L ? 9 : 12;
      i.save(), i.shadowColor = "rgba(0,0,0,0.72)", i.shadowBlur = 6, i.fillStyle = "rgba(2,6,11,0.88)", i.strokeStyle = C, i.lineWidth = 2, i.beginPath(), i.arc(y, T, $ + 2, 0, Math.PI * 2), i.fill(), i.stroke(), i.beginPath(), v.kind === "operator_breakdown" ? (i.moveTo(y, T - $), i.lineTo(y + $, T), i.lineTo(y, T + $), i.lineTo(y - $, T), i.closePath()) : v.kind === "operator_reset" ? i.rect(y - $ + 1, T - $ + 1, ($ - 1) * 2, ($ - 1) * 2) : (i.moveTo(y, T - $), i.lineTo(y + $, T + $ - 2), i.lineTo(y - $, T + $ - 2), i.closePath()), i.fillStyle = C, i.fill(), i.lineWidth = 1.4, i.strokeStyle = "rgba(2,6,11,0.96)", i.stroke();
      const O = Ee(v, L), z = L ? Math.max(8.5, Math.min(10.5, u / 118)) : 12, B = L ? 11 : 14;
      i.font = `850 ${z}px system-ui, sans-serif`;
      const H = L ? Math.max(76, Math.min(118, u * 0.11)) : Math.max(110, Math.min(170, u * 0.16)), J = Math.max(...O.map((ht) => i.measureText(ht).width)) + 12, q = Math.min(H, J), K = O.length * B + 8, nt = Pt({ x: y, y: T, labelWidth: q, labelHeight: K, left: s, top: c, width: u, height: f, placed: g, markerRadius: $ }) ?? A(q, K, y - q / 2);
      if (!nt) {
        i.restore();
        return;
      }
      g.push(nt);
      const rt = nt.x, it = nt.y;
      Gt(i, y, T, rt, it, q, K, C), i.fillStyle = "rgba(2,6,11,0.94)", i.fillRect(rt, it, q, K), i.strokeStyle = C, i.lineWidth = 1.2, i.strokeRect(rt, it, q, K), i.fillStyle = C, O.forEach((ht, ce) => i.fillText(Ie(i, ht, q - 10), rt + 6, it + B + 1 + ce * B)), w += 1, i.restore();
    } else if (Y) {
      const L = e.campaign_id === "command_center_fat";
      if (i.save(), i.fillStyle = "rgba(2,6,11,0.86)", i.strokeStyle = C, i.lineWidth = L ? 2.2 : 1.8, i.beginPath(), i.arc(y, W, L ? 8 : v.kind === "functional_gate" ? 10 : 8, 0, Math.PI * 2), i.fill(), i.stroke(), i.restore(), i.fillStyle = C, i.beginPath(), v.kind === "functional_gate" ? (i.moveTo(y, W - 7), i.lineTo(y + 7, W), i.lineTo(y, W + 7), i.lineTo(y - 7, W), i.closePath()) : i.arc(y, W, 5.6, 0, Math.PI * 2), i.fill(), !N.has(v.id)) return;
      F += 1;
      const T = L ? "FT" : Re(v.label);
      i.save(), i.font = L ? "850 10px system-ui, sans-serif" : "850 12px system-ui, sans-serif";
      const $ = i.measureText(T), O = Math.max(L ? 22 : 36, $.width + 10), z = L ? 16 : 18, B = Pt({ x: y, y: W, labelWidth: O, labelHeight: z, left: s, top: c, width: u, height: f, placed: g, markerRadius: 8 }) ?? A(O, z, y - O / 2);
      if (!B) {
        i.restore();
        return;
      }
      g.push(B);
      const H = B.x, J = B.y;
      Gt(i, y, W, H, J, O, z, C), i.fillStyle = "rgba(2,6,11,0.92)", i.fillRect(H, J, O, z), i.strokeStyle = C, i.lineWidth = 1, i.strokeRect(H, J, O, z), i.fillStyle = v.kind === "functional_gate" ? "#fff0a8" : "#c9ffef", i.shadowColor = "rgba(0,0,0,0.88)", i.shadowBlur = 5, i.fillText(T, H + 5, J + Math.min(13, z - 5)), w += 1, i.restore();
    } else
      i.fillStyle = C, i.beginPath(), i.arc(y, c + 10, 3.2, 0, Math.PI * 2), i.fill();
  });
  const R = (Nt = t.root) == null ? void 0 : Nt.closest("[data-uplot-card]");
  R && (R.dataset.markerLabelsExpected = String(F), R.dataset.markerLabelsDrawn = String(w));
  const G = ((Ft = r == null ? void 0 : r.time_axis) == null ? void 0 : Ft.now) ?? ((At = r == null ? void 0 : r.execution) == null ? void 0 : At.now) ?? "", kt = n ?? Date.parse(G);
  if (Number.isFinite(kt)) {
    const v = s + (kt - d) / p * u, M = Math.max(s, Math.min(s + u, v));
    i.fillStyle = "rgba(3,7,12,0.58)", i.fillRect(M, c, Math.max(0, s + u - M), f), i.strokeStyle = "rgba(242,247,255,0.9)", i.setLineDash([3, 3]), i.beginPath(), i.moveTo(M, c), i.lineTo(M, c + f), i.stroke();
  }
  if (Number.isFinite(a)) {
    const v = s + (a - d) / p * u;
    v >= s && v <= s + u && (i.strokeStyle = "rgba(255,216,95,0.95)", i.setLineDash([]), i.lineWidth = 1, i.beginPath(), i.moveTo(v, c), i.lineTo(v, c + f), i.stroke());
  }
  i.restore();
}
function ln(t, e, r, n, a) {
  return new Set(
    t.filter((o) => {
      const i = Date.parse(o.timestamp);
      return Number.isFinite(i) && i >= e && i <= r && wt(o) > 0;
    }).sort((o, i) => wt(i) - wt(o) || Date.parse(o.timestamp) - Date.parse(i.timestamp)).map((o) => o.id)
  );
}
function wt(t) {
  let e = 0;
  return (t.role === "interlock" || t.result === "fail" || t.kind === "interlock") && (e += 1e3), t.kind === "functional_gate" && (e += 760), t.kind === "pressure_gate" && (e += 640), (t.kind === "stability" || t.kind === "stability_achieved") && (e += 440), t.result === "pass" && (e += 120), e;
}
const ft = "signalforge.tile.uplot", Yt = Object.freeze({
  cmd: { label: "target / command", rank: 10, className: "cmd", dash: "6,4", width: 2.2, opacity: 0.98 },
  command: { label: "target / command", rank: 10, className: "cmd", dash: "6,4", width: 2.2, opacity: 0.98 },
  actual: { label: "actual", rank: 20, className: "actual", dash: "", width: 2.2, opacity: 0.98 },
  ghost: { label: "reference / sink", rank: 30, className: "ghost", dash: "2,4", width: 1.8, opacity: 0.86 },
  dut: { label: "power / load", rank: 40, className: "dut", dash: "", width: 2, opacity: 0.94 },
  aux: { label: "auxiliary", rank: 50, className: "aux", dash: "8,4", width: 1.8, opacity: 0.86 }
});
function se(t) {
  return Yt[t || "actual"] || Yt.actual;
}
function un(t, e = "var(--series-actual)") {
  const r = se(t), n = r.className ? `--series-${r.className}` : "";
  return !n || typeof document > "u" ? e : getComputedStyle(document.documentElement).getPropertyValue(n).trim() || e;
}
function Bn(t) {
  if (!t) return 0;
  const e = typeof t.getBoundingClientRect == "function" ? t.getBoundingClientRect() : null;
  return Math.floor(e && e.width || t.clientWidth || 0);
}
function dn(t = {}) {
  const e = k(t.tile_id ?? t.tileId, "empty"), r = Date.now(), n = mt(t.time_window_ms ?? t.timeWindowMs) ?? 9e4, a = new Date(r).toISOString();
  return {
    schema_version: "signalforge.graph_tile.v1",
    id: e,
    card_id: e,
    level: "live",
    t0: new Date(r - n).toISOString(),
    t1: a,
    generated_at: a,
    renderer: ft,
    kind: "timeseries",
    tile_id: e,
    title: k(t.title),
    time_window_ms: n,
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
function jn(t) {
  if (!et(t) || !Array.isArray(t.series)) return [];
  const e = fn(t);
  return e.series.map((r) => {
    const n = r, a = k(n.seriesRole ?? (r.role === "command" ? "cmd" : r.role), "actual"), o = X(n.source_obj);
    return {
      key: k(n.series_id ?? r.id ?? n.key ?? n.target_id ?? n.targetId ?? r.label, "series"),
      tileId: e.tile_id || e.id,
      targetId: n.target_id ?? n.targetId,
      label: r.label,
      fullLabel: n.full_label ?? n.fullLabel ?? r.label,
      role: a,
      seriesRole: a,
      roleRank: mt(n.role_rank) ?? se(a).rank,
      color: r.color || un(a),
      unit: k(r.unit ?? r.units, "_"),
      provenance: n.provenance || "",
      source: n.source_obj ?? r.source ?? null,
      paramId: n.param_id ?? o.param_id,
      deviceId: n.device_id ?? o.device_id,
      instance: n.instance ?? o.instance,
      signalId: n.signal_id ?? o.signal_id,
      history: gn(r)
    };
  }).sort((r, n) => r.roleRank !== n.roleRank ? r.roleRank - n.roleRank : String(r.tileId || r.key || r.label || "").localeCompare(String(n.tileId || n.key || n.label || "")));
}
function fn(t, e = {}) {
  const r = dn({
    tile_id: e.tile_id ?? e.tileId,
    timeWindowMs: e.timeWindowMs ?? e.time_window_ms
  }), n = X(t), a = X(n.diagnostics), o = mt(n.time_window_ms ?? e.timeWindowMs ?? e.time_window_ms) ?? r.time_window_ms ?? 9e4, i = (Array.isArray(n.series) ? n.series : []).map(mn).filter((w) => {
    var S, A;
    return (((S = w.points) == null ? void 0 : S.length) ?? 0) > 0 || (((A = w.spans) == null ? void 0 : A.length) ?? 0) > 0;
  }), l = Array.isArray(n.bands) ? n.bands : [], s = Array.isArray(n.markers) ? n.markers : [], c = Array.isArray(n.events) ? n.events : [], u = [
    ...i.flatMap((w) => [
      ...(w.points || []).map((S) => S.timestamp),
      ...(w.spans || []).flatMap((S) => [S.start, S.end])
    ]),
    ...l.flatMap((w) => [w.start, w.end]),
    ...s.map((w) => w.timestamp),
    ...c.map((w) => w.timestamp)
  ].map((w) => Date.parse(w)).filter(Number.isFinite), f = Date.now(), h = Ut(n.t0), d = Ut(n.t1), b = h ?? (u.length ? Math.min(...u) : f - o), p = d ?? (u.length ? Math.max(...u) : f), m = new Date(b).toISOString(), g = new Date(Math.max(p, b + 1)).toISOString(), N = i.reduce((w, S) => {
    var A;
    return w + (((A = S.points) == null ? void 0 : A.length) || 0);
  }, 0), F = k(a.status, i.length > 0 ? "ok" : "empty");
  return {
    ...r,
    ...n,
    schema_version: wn(n.schema_version, r.schema_version),
    id: k(n.id ?? n.tile_id, r.id),
    card_id: k(n.card_id ?? n.tile_id ?? n.id, r.card_id),
    level: k(n.level, "live"),
    t0: m,
    t1: g,
    generated_at: k(n.generated_at, new Date(f).toISOString()),
    renderer: ft,
    kind: k(n.kind, "timeseries"),
    tile_id: k(n.tile_id ?? n.id, r.tile_id),
    title: k(n.title, r.title),
    time_window_ms: o,
    axes: Array.isArray(n.axes) ? n.axes : r.axes,
    bands: l,
    markers: s,
    events: c,
    diagnostics: {
      ...r.diagnostics,
      ...a,
      status: F,
      point_count: N,
      raw_point_count: mt(a.raw_point_count) ?? N,
      decimation: k(a.decimation, "none"),
      renderer: ft,
      series_count: i.length
    },
    provenance: et(n.provenance) ? n.provenance : r.provenance,
    series: i
  };
}
function mn(t) {
  const e = X(t), r = vt(e.source_obj) ?? vt(e.source_ref) ?? vt(e.source), n = k(e.role ?? e.seriesRole, "actual"), a = pn(n), o = k(e.series_id ?? e.id ?? e.key ?? e.target_id ?? e.targetId ?? e.label, "series"), i = k(e.unit ?? e.units, "_"), l = _n(e);
  return {
    ...e,
    id: o,
    series_id: e.series_id || o,
    label: k(e.label, o),
    role: a,
    seriesRole: n,
    unit: i,
    units: i,
    axis_id: k(e.axis_id ?? hn({ ...e, id: o, role: a, seriesRole: n }, i)),
    source: bn(e.source_ref ?? e.source ?? r ?? o),
    source_obj: r,
    color: typeof e.color == "string" ? e.color : void 0,
    points: l,
    spans: Array.isArray(e.spans) ? e.spans : []
  };
}
function pn(t) {
  const e = k(t, "actual");
  return e === "cmd" ? "command" : e || "actual";
}
function hn(t, e) {
  const r = t.axis_id ?? t.axisId;
  if (r) return k(r);
  const n = k(e ?? t.unit ?? t.units).trim().toLowerCase(), a = [
    t.id,
    t.series_id,
    t.key,
    t.target_id,
    t.targetId,
    t.label,
    t.full_label,
    t.fullLabel
  ].filter(Boolean).join(" ").toLowerCase();
  return t.role === "counter" || t.seriesRole === "counter" || t.kind === "counter" ? "counter" : n === "a" || n === "amp" || n === "amps" ? "current_a" : n === "v" || n === "volt" || n === "volts" ? "voltage_v" : n === "w" || n === "watt" || n === "watts" ? "power_w" : n === "%" || n === "percent" ? "percent" : n === "ms" || n === "millisecond" || n === "milliseconds" ? "bus_ms" : n === "s" || n === "sec" || n === "secs" || n === "second" || n === "seconds" ? "seconds" : n === "mbar" || n === "millibar" || n === "millibars" ? "pressure_log" : n === "mbar/min" || n === "mbar/minute" || n === "millibar/min" || n === "millibars/minute" ? "pressure_rate_log" : n === "bar" ? "pressure_bar" : n.includes("deg") || n === "c" || n === "degc" || n === "deg c" || n === "°c" || n === "° c" ? "temperature_c" : a.includes("counter") ? "counter" : a.includes("pressure") || a.includes("vacuum") ? "pressure_log" : "generic_numeric";
}
function _n(t) {
  if (Array.isArray(t.points) && t.points.length)
    return t.points.flatMap((a) => qt(a));
  const e = X(t.history), r = Array.isArray(e.ts) ? e.ts : [];
  return (Array.isArray(e.v) ? e.v : []).flatMap((a, o) => qt({ t: r[o], v: a }));
}
function qt(t) {
  const e = X(t), r = e.timestamp ?? e.t ?? e.time, n = e.value ?? e.v ?? e.y;
  if (r == null || r === "") return [];
  if (n == null || n === "") return [];
  const a = Number(n), o = typeof r == "number" ? r : Date.parse(String(r || ""));
  return !Number.isFinite(a) || !Number.isFinite(o) ? [] : [{ timestamp: new Date(o).toISOString(), value: a }];
}
function gn(t) {
  const e = t.points || [];
  return {
    ts: e.map((r) => Date.parse(r.timestamp)),
    v: e.map((r) => r.value),
    q: e.map(() => "ok")
  };
}
function bn(t) {
  if (!t) return "";
  if (typeof t == "string") return t;
  if (!et(t)) return String(t);
  const e = t.device_id || t.deviceId || "", r = t.param_id || t.paramId || "", n = t.instance || "", a = t.endpoint || "", o = t.signal_id || t.signalId || "";
  return `device=${e} param=${r} instance=${n} signal=${o} endpoint=${a}`.trim();
}
function Ut(t) {
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
function wn(t, e) {
  return typeof t == "string" || typeof t == "number" ? t : e;
}
function X(t) {
  return et(t) ? t : {};
}
function vt(t) {
  return et(t) ? t : void 0;
}
function et(t) {
  return !!t && typeof t == "object" && !Array.isArray(t);
}
function Vn({ adapter: t, store: e, wallId: r, className: n }) {
  const [a, o] = P("monitor"), [i, l] = P(""), [s, c] = P(null), u = he(e), f = t.list();
  t.channels();
  const h = Ct(
    () => f.filter((m) => m.role === a),
    [f, a]
  ), d = Ct(() => {
    const m = {};
    return h.forEach((g) => {
      m[g.group] || (m[g.group] = {}), m[g.group][g.subgroup] || (m[g.group][g.subgroup] = []), m[g.group][g.subgroup].push(g);
    }), m;
  }, [h]);
  V(() => {
    var g;
    if (s && ((g = d[s.group]) != null && g[s.subgroup])) return;
    const m = Object.keys(d)[0];
    if (m) {
      const N = Object.keys(d[m])[0];
      c({ group: m, subgroup: N });
    } else
      c(null);
  }, [a, d]);
  const b = s && d[s.group] ? d[s.group][s.subgroup] || [] : [], p = b.filter(
    (m) => !i || m.name.toLowerCase().includes(i.toLowerCase()) || String(m.id).includes(i)
  );
  return /* @__PURE__ */ x("div", { className: "sf-dict" + (n ? " " + n : ""), children: [
    /* @__PURE__ */ x("div", { className: "sf-dict-rail", children: [
      /* @__PURE__ */ x("div", { className: "sf-dict-tabs", children: [
        /* @__PURE__ */ _("button", { className: a === "monitor" ? "active" : "", onClick: () => o("monitor"), children: "Telemetry" }),
        /* @__PURE__ */ _("button", { className: a === "control" ? "active" : "", onClick: () => o("control"), children: "Telecommands" })
      ] }),
      /* @__PURE__ */ _("input", { className: "sf-dict-search", placeholder: "search signals…", value: i, onChange: (m) => l(m.target.value) }),
      /* @__PURE__ */ _("div", { className: "sf-dict-groups", children: Object.entries(d).map(([m, g]) => /* @__PURE__ */ _(vn, { group: m, sgrps: g, selected: s, onSelect: c, query: i }, m)) })
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
      p.map((m) => /* @__PURE__ */ _(
        yn,
        {
          signal: m,
          channels: t.channelsForSignal(m),
          adapter: t,
          wallId: r,
          assigns: u,
          tab: a
        },
        m.id
      )),
      b.length === 0 && /* @__PURE__ */ _("div", { className: "sf-dict-empty", children: "No signals here." })
    ] })
  ] });
}
function vn({
  group: t,
  sgrps: e,
  selected: r,
  onSelect: n,
  query: a
}) {
  const [o, i] = P(!0);
  let l = e;
  if (a) {
    const c = {};
    if (Object.entries(e).forEach(([u, f]) => {
      const h = f.filter((d) => d.name.toLowerCase().includes(a.toLowerCase()) || String(d.id).includes(a));
      h.length && (c[u] = h);
    }), !Object.keys(c).length) return null;
    l = c;
  }
  const s = Object.values(l).reduce((c, u) => c + u.length, 0);
  return /* @__PURE__ */ x("div", { className: "sf-dict-group", children: [
    /* @__PURE__ */ x("div", { className: "sf-dict-group-head", onClick: () => i((c) => !c), children: [
      /* @__PURE__ */ _("span", { children: o ? "▾" : "▸" }),
      /* @__PURE__ */ _("span", { children: t }),
      /* @__PURE__ */ _("span", { className: "sf-count", children: s })
    ] }),
    o && Object.entries(l).map(([c, u]) => {
      const f = (r == null ? void 0 : r.group) === t && (r == null ? void 0 : r.subgroup) === c;
      return /* @__PURE__ */ x(
        "div",
        {
          className: "sf-dict-sgrp" + (f ? " selected" : ""),
          onClick: () => n({ group: t, subgroup: c }),
          children: [
            /* @__PURE__ */ _("span", { children: c }),
            /* @__PURE__ */ _("span", { className: "sf-count", children: u.length })
          ]
        },
        c
      );
    })
  ] });
}
function yn({
  signal: t,
  channels: e,
  adapter: r,
  wallId: n,
  assigns: a,
  tab: o
}) {
  const i = e.every((s) => a.hasAssignment(n, t.id, s.device_id, s.instance));
  function l() {
    i ? e.forEach((s) => a.remove(n, t.id, s.device_id, s.instance)) : e.forEach((s) => a.add(n, t.id, s.device_id, s.instance));
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
      /* @__PURE__ */ _("input", { type: "checkbox", checked: i, onChange: l }),
      /* @__PURE__ */ _("span", { className: "sf-assign-swatch", style: { background: r.colorForRole(t.role) } }),
      "Show in wall · ",
      e.length,
      " ch"
    ] })
  ] });
}
function Gn({ walls: t, selectedWallId: e, onSelect: r }) {
  const [n, a] = P(""), [o, i] = P(null), [l, s] = P("");
  function c() {
    const h = n.trim();
    if (!h) return;
    const d = t.add(h);
    a(""), r(d.wall_id);
  }
  function u(h, d) {
    i(h), s(d);
  }
  function f() {
    o && l.trim() && t.rename(o, l.trim()), i(null), s("");
  }
  return /* @__PURE__ */ x("div", { className: "sf-wall-manager", children: [
    /* @__PURE__ */ x("div", { className: "sf-wall-list", children: [
      t.walls.map((h) => /* @__PURE__ */ x(
        "div",
        {
          className: "sf-wall-item" + (h.wall_id === e ? " selected" : "") + (h.preset ? " preset" : ""),
          onClick: () => r(h.wall_id),
          children: [
            o === h.wall_id ? /* @__PURE__ */ _(
              "input",
              {
                autoFocus: !0,
                value: l,
                onChange: (d) => s(d.target.value),
                onBlur: f,
                onKeyDown: (d) => {
                  d.key === "Enter" && f(), d.key === "Escape" && i(null);
                },
                onClick: (d) => d.stopPropagation()
              }
            ) : /* @__PURE__ */ _("span", { className: "sf-wall-label", children: h.label }),
            !h.preset && o !== h.wall_id && /* @__PURE__ */ x("span", { className: "sf-wall-actions", children: [
              /* @__PURE__ */ _("button", { onClick: (d) => {
                d.stopPropagation(), u(h.wall_id, h.label);
              }, children: "✎" }),
              /* @__PURE__ */ _("button", { onClick: (d) => {
                d.stopPropagation(), t.remove(h.wall_id);
              }, children: "✕" })
            ] })
          ]
        },
        h.wall_id
      )),
      t.walls.length === 0 && /* @__PURE__ */ _("div", { className: "sf-wall-empty", children: "No walls yet. Create one below." })
    ] }),
    /* @__PURE__ */ x("div", { className: "sf-wall-add", children: [
      /* @__PURE__ */ _(
        "input",
        {
          placeholder: "New wall label…",
          value: n,
          onChange: (h) => a(h.target.value),
          onKeyDown: (h) => {
            h.key === "Enter" && c();
          }
        }
      ),
      /* @__PURE__ */ _("button", { onClick: c, disabled: !n.trim(), children: "+ Add wall" })
    ] })
  ] });
}
function Yn({
  tile: t,
  heroGraph: e,
  height: r = 280,
  currentTimeMs: n,
  hoverTimeMs: a,
  className: o,
  dataGraphRenderer: i,
  syncKey: l = "sf-wall"
}) {
  const s = U(null), c = U(null), u = U(null), f = U(e), h = U(n), d = U(a);
  V(() => {
    var p;
    f.current = e, (p = c.current) == null || p.redraw();
  }, [e]), V(() => {
    var F;
    h.current = n;
    const p = c.current, m = u.current;
    if (!p || !m) return;
    const g = p.width || ((F = s.current) == null ? void 0 : F.offsetWidth) || 900, N = bt(m, n, g);
    p.setData(N.data, !1), p.redraw();
  }, [n]), V(() => {
    var p;
    d.current = a, (p = c.current) == null || p.redraw();
  }, [a]);
  const b = de(() => {
    if (!s.current) return;
    const p = s.current.offsetWidth || 900, m = bt(t, h.current, p), g = {
      draw: [
        (F) => {
          cn(
            F,
            t,
            f.current,
            h.current,
            d.current
          );
        }
      ]
    }, N = {
      width: p,
      height: r,
      series: m.series,
      scales: m.scales,
      axes: m.axes,
      hooks: g,
      cursor: { sync: { key: l } }
    };
    c.current && c.current.destroy(), c.current = new fe(N, m.data, s.current), u.current = t;
  }, [t, r, l]);
  return V(() => {
    b();
    const p = new ResizeObserver(() => {
      const m = c.current, g = u.current, N = s.current;
      if (!m || !g || !N) return;
      const F = N.offsetWidth || m.width || 900;
      m.setSize({ width: F, height: r });
      const w = bt(g, h.current, F);
      m.setData(w.data, !1), m.redraw();
    });
    return s.current && p.observe(s.current), () => {
      var m;
      p.disconnect(), (m = c.current) == null || m.destroy(), c.current = null, u.current = null;
    };
  }, [b, r]), /* @__PURE__ */ _(
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
  En as HeroTopTimeAxis,
  Yt as SERIES_ROLE_META,
  On as SharedTimeAxis,
  Vn as SignalDictionary,
  _e as TileClient,
  Qt as TimeAxisTrack,
  Yn as UPlotTileRenderer,
  Gn as WallManager,
  jt as axisLabel,
  Ln as blockLabel,
  Ke as buildAxes,
  Xe as buildScales,
  Ot as cardPriority,
  xe as chooseTickStep,
  ye as clampRange,
  In as clampTime,
  we as colorForSignal,
  Ce as commandCenterGapBreaks,
  te as commandCenterProjectedSeries,
  St as commandCenterTraceGapMs,
  Ae as decimationValue,
  Te as displayValue,
  lt as distinctivePalette,
  cn as drawTileOverlays,
  dn as emptyGraphTile,
  Dn as eventColor,
  Ie as fitCanvasText,
  Be as formatLegendValue,
  Oe as formatMarkerDateTime,
  Wt as formatPressure,
  Rt as formatScientific,
  $n as graphCardPriority,
  xt as graphCardRank,
  Cn as graphSectionPriority,
  Et as graphSectionRank,
  Wn as inTimeRange,
  ee as interpolationValue,
  pt as isDiscreteSeries,
  Pn as legendReadouts,
  qe as lineWidthFor,
  ot as loadAssignments,
  Z as loadWalls,
  st as logScale,
  Bt as logSplits,
  dt as lttb,
  pe as makeAssignment,
  De as markerColor,
  Bn as measuredElementWidth,
  fn as normalizeGraphTile,
  Ee as operatorMarkerLines,
  An as orderLegendSignals,
  I as paddedRange,
  ge as palette,
  be as paletteForID,
  Kt as pickTileLevel,
  Pt as placeMarkerLabel,
  ne as rawValueAt,
  Pe as rectanglesOverlap,
  zn as renderKindFor,
  jn as renderSeriesFromGraphTile,
  $e as resampleSeries,
  Dt as roleColors,
  Tt as saveAssignments,
  _t as saveWalls,
  ae as scaleForSeries,
  ve as semanticColor,
  Ye as seriesDrawOrder,
  un as seriesRoleColor,
  se as seriesRoleMeta,
  Ue as sharedTimeGrid,
  Re as shortGateLabel,
  Lt as signalColors,
  ut as signalPriority,
  ze as stateAt,
  Rn as stateBlocks,
  gt as tickLabel,
  Tn as tileCardPriority,
  Mt as timeTicks,
  je as unitForAxis,
  bt as uplotData,
  he as useAssignments,
  Fn as useTileSeries,
  Nn as useWalls,
  j as valueFromInterpolation,
  Me as viewportSeries,
  Ze as ySplits
};
//# sourceMappingURL=signalforge-web.es.js.map
