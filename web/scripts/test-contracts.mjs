import { readFile } from "node:fs/promises";
import { join } from "node:path";

const root = new URL("../..", import.meta.url).pathname;
const publicDir = join(root, "fixtures", "public");
const appSource = await readFile(join(root, "web", "src", "App.tsx"), "utf8");
const architectureSource = await readFile(join(root, "docs", "architecture.md"), "utf8");
const generatorSource = await readFile(join(root, "internal", "synthetic", "generator.go"), "utf8");
const backlogSource = await readFile(join(root, "docs", "backlog", "BACKLOG.md"), "utf8");
const apiSource = await readFile(join(root, "web", "src", "api.ts"), "utf8");
const arrowTilesSource = await readFile(join(root, "web", "src", "arrowTiles.ts"), "utf8");
const graphWallSource = await readFile(join(root, "web", "src", "components", "OperatorGraphWall.tsx"), "utf8");
const graphCardCSS = await readFile(join(root, "web", "src", "styles", "graph-card.css"), "utf8");
const markerSource = await readFile(join(root, "web", "src", "components", "tiles", "markers.ts"), "utf8");
const visualPolicySource = await readFile(join(root, "web", "src", "components", "tiles", "visualPolicy.ts"), "utf8");
const timeAxisSource = await readFile(join(root, "web", "src", "components", "tiles", "timeAxis.tsx"), "utf8");
const viteConfigSource = await readFile(join(root, "web", "vite.config.ts"), "utf8");
const webTsConfig = JSON.parse(await readFile(join(root, "web", "tsconfig.json"), "utf8"));
const signalForgeSourceMap = JSON.parse(await readFile(join(root, "web", "vendor", "signalforge-web", "dist", "signalforge-web.es.js.map"), "utf8"));
const signalForgeUPlotAdapterSource = signalForgeSourceMap.sourcesContent?.[signalForgeSourceMap.sources?.indexOf("../src/render/uPlotAdapter.ts")] ?? "";
const viewsCSS = await readFile(join(root, "web", "src", "styles", "views.css"), "utf8");
const semanticsChecklist = await readFile(join(root, "docs", "backend_semantics_checklist.md"), "utf8");
const webPackage = JSON.parse(await readFile(join(root, "web", "package.json"), "utf8"));

function requireEnvelope(name, value) {
  if (value.schema_version !== 1) throw new Error(`${name} has invalid schema_version`);
  if (!value.generated_at) throw new Error(`${name} missing generated_at`);
}

const forbiddenVocabulary = ["CAN/TMTC", "TMTC", "TM/TC"];

const forbiddenVocabularyChecks = [
  { name: "public UI copy in App.tsx", path: "web/src/App.tsx", source: appSource },
  { name: "architecture documentation", path: "docs/architecture.md", source: architectureSource },
  { name: "backlog references", path: "docs/backlog/BACKLOG.md", source: backlogSource },
  { name: "internal generator comments/labels", path: "internal/synthetic/generator.go", source: generatorSource },
];

const cleanRoomVocabularyExceptions = [
  {
    file: "docs/backlog/BACKLOG.md",
    sectionPattern: /## GOSS-11[\s\S]*?(?=\n## GOSS-\d|$)/,
    requires: /cleanup rule/i,
    allowedPhrase: "legacy identifier",
  },
];

function collectForbiddenMatches(value) {
  const matches = [];
  for (const term of forbiddenVocabulary) {
    let cursor = 0;
    while (true) {
      const index = value.indexOf(term, cursor);
      if (index === -1) break;
      matches.push({ term, index, end: index + term.length });
      cursor = index + term.length;
    }
  }
  return matches;
}

function isIndexInRange(index, start, end) {
  return start <= index && index < end;
}

function assertNoForbiddenVocabulary(name, path, value) {
  const matches = collectForbiddenMatches(value);
  if (!matches.length) return;

  const exception = cleanRoomVocabularyExceptions.find((item) => item.file === path);
  if (!exception) {
    throw new Error(`${name} contains restricted clean-room wording: ${forbiddenVocabulary.join(", ")}`);
  }

  const allowedSection = value.match(exception.sectionPattern);
  if (!allowedSection || allowedSection.index === undefined || allowedSection.index === null || allowedSection.index < 0) {
    throw new Error(`${path} has no documented exception scope for restricted bus vocabulary`);
  }
  if (!exception.requires.test(allowedSection[0])) {
    throw new Error(`${path} has restricted bus vocabulary without documented cleanup rule context`);
  }
  if (!allowedSection[0].toLowerCase().includes(exception.allowedPhrase)) {
    throw new Error(`${path} exception context for restricted bus vocabulary is missing ${exception.allowedPhrase}`);
  }
  const allowedStart = allowedSection.index;
  const allowedEnd = allowedSection.index + allowedSection[0].length;
  if (matches.some((match) => !isIndexInRange(match.index, allowedStart, allowedEnd))) {
    throw new Error(`${name} contains restricted clean-room wording outside documented exception scope: ${forbiddenVocabulary.join(", ")}`);
  }
}

function includesForbiddenVocabulary(value) {
  return forbiddenVocabulary.some((term) => value.includes(term));
}

function requireNoForbiddenVocabularyInLabels(name, value, context = "label") {
  if (!value) return;
  if (typeof value === "string") {
    if (includesForbiddenVocabulary(value)) {
      throw new Error(`${name} ${context} contains restricted clean-room wording: ${forbiddenVocabulary.join(", ")}`);
    }
    return;
  }
  if (Array.isArray(value)) {
    value.forEach((item, index) => requireNoForbiddenVocabularyInLabels(name, item, `${context}[${index}]`));
    return;
  }
  if (typeof value !== "object") return;
  for (const [key, item] of Object.entries(value)) {
    if (key === "label") {
      requireNoForbiddenVocabularyInLabels(name, item, `${context}.label`);
      continue;
    }
    if (typeof item === "string") continue;
    if (Array.isArray(item) || typeof item === "object") {
      requireNoForbiddenVocabularyInLabels(name, item, `${context}.${key}`);
    }
  }
}

function requireReportTraceability(campaignID, campaign, report) {
  if (report.campaign_id !== campaignID) {
    throw new Error(`${campaignID} report campaign_id mismatch`);
  }
  if (!Array.isArray(report.requirements) || report.requirements.length === 0) {
    throw new Error(`${campaignID} report requirements must be a non-empty array`);
  }
  if (!Array.isArray(report.anomalies)) {
    throw new Error(`${campaignID} report anomalies must be an array`);
  }
  if (!Array.isArray(report.reproducibility)) {
    throw new Error(`${campaignID} report reproducibility must be an array`);
  }
  if (typeof report.synthetic_data_note !== "string" || !/fixture|synthetic|deterministic/i.test(report.synthetic_data_note)) {
    throw new Error(`${campaignID} report should describe synthetic data basis`);
  }
  if (typeof report.summary !== "string" || !/generic reference dut/i.test(report.summary)) {
    throw new Error(`${campaignID} report should retain generic summary wording`);
  }
}

async function readJSON(path) {
  return JSON.parse(await readFile(join(publicDir, path), "utf8"));
}

const manifest = await readJSON("manifest.json");
requireEnvelope("manifest", manifest);
if (manifest.name !== "Gossamer") throw new Error("manifest name mismatch");
if (!manifest.synthetic_only) throw new Error("manifest must be synthetic_only");
for (const route of ["landing", "supervisor", "bus-tap"]) {
  if (!appSource.includes(route)) throw new Error(`missing ${route} route in App.tsx`);
}
if (!appSource.includes("report")) throw new Error("missing report route in App.tsx");
for (const check of forbiddenVocabularyChecks) {
  assertNoForbiddenVocabulary(check.name, check.path, check.source);
}
for (const term of [
  "backend-owned role",
  "authority",
  "freshness",
  "provenance",
  "source and target identity",
  "fixture/live distinction",
  "no browser-only semantic derivation",
]) {
  if (!semanticsChecklist.includes(term)) throw new Error(`semantics checklist missing ${term}`);
}
if (!viewsCSS.includes('.graph-wall-card[data-render-kind="event_rail"] .graph-card-plot-shell')) throw new Error("event rail cards need a plot-shell height cap");
if (!viewsCSS.includes("max-height: 480px") || !viewsCSS.includes("overflow-y: auto")) throw new Error("event rail plot shell must cap height and scroll overflow");
if (!graphWallSource.includes('data-tile-backed="true"')) throw new Error("operator graph wall must advertise tile-backed rendering");
if (!graphWallSource.includes("api.tileManifest(campaignId)")) throw new Error("operator graph wall must load the tile manifest before graph cards");
if (!graphWallSource.includes("manifestError") || !graphWallSource.includes("api.invalidateTileManifest(campaignId)")) throw new Error("operator graph wall must expose retryable manifest load errors");
if (!graphWallSource.includes("graphResetIdentity") || graphWallSource.includes("[campaignId, defaultTimeRange, manifestRetryToken]")) throw new Error("operator graph wall must not reload manifests or clear operator state on viewport-only range changes");
if (!graphWallSource.includes('api.tile(campaignId, cardID, "minute")')) throw new Error("operator graph wall must materialize graph cards through the tile API");
if (!graphWallSource.includes("orderedSections") || !graphWallSource.includes(".sort(graphSectionPriority)") || !graphWallSource.includes(".sort(graphCardPriority)")) throw new Error("operator graph wall must derive primary semantics from ordered sections and cards");
if (!graphWallSource.includes("window.clearTimeout(timeoutID)")) throw new Error("operator graph wall must cancel delayed tile work on cleanup");
if (!graphWallSource.includes("if (cancelled || loadGeneration.current !== generation) return;\n        requestedTiles.current.add(cardID);")) throw new Error("operator graph wall must only mark tile requests after delayed work starts");
if (!graphWallSource.includes("if (cancelled || loadGeneration.current !== generation) return;\n            requestedTiles.current.delete(cardID);")) throw new Error("stale tile failures must not clear current-generation request markers");
if (!graphWallSource.includes("const safeStart = Number.isFinite(start)") || !graphWallSource.includes("end > safeStart ? end : safeStart + 1")) throw new Error("graph time range must derive fallback end from sanitized start");
if (!timeAxisSource.includes("finiteTimeRange") || !timeAxisSource.includes("Number.isFinite(parsedStart)") || !timeAxisSource.includes("Number.isFinite(parsedEnd)")) throw new Error("shared time axis must sanitize malformed backend ranges before ISO conversion");
if (!graphWallSource.includes('renderKind === "event_rail" ? 360')) throw new Error("event rail resize height must stay bounded");
if (!graphCardCSS.includes(".tile-event-rail")) throw new Error("event rail tile styles missing");
if (!markerSource.includes("slice(0, 8)")) throw new Error("event marker labels must default to eight characters or fewer");
if (!graphWallSource.includes("shortGateLabel(marker.label)")) throw new Error("event rail markers must use short label rendering");
if (!graphWallSource.includes("railLabelPlacements")) throw new Error("event rail marker labels must use collision-aware row placement");
if (!graphWallSource.includes("showLabel")) throw new Error("event rail marker labels must suppress labels without hiding marker dots");
if (!graphWallSource.includes("eventRailEvents") || !graphWallSource.includes("!markerIDs.has(event.id)")) throw new Error("event rail marker-derived events must not render twice");
if (!graphWallSource.includes('title={`${marker.label} ${marker.timestamp}`}')) throw new Error("event rail markers must preserve full hover titles");
if (!markerSource.includes("LABEL_COLLISION_PADDING = 8")) throw new Error("marker labels need a visible collision margin");
if (!markerSource.includes("displayableLegendValue") || !markerSource.includes('axisID === "pressure_log"')) throw new Error("pressure legends must follow canonical log-axis validity policy");
if (!markerSource.includes("function pressureLogAxis") || !markerSource.includes("!pressureLogAxis(series.axis_id) || value > 0")) throw new Error("linear pressure legends must preserve zero readouts while log pressure axes reject non-positive values");
if (!markerSource.includes("rawValueAt(series, timeMs, tile)") || !markerSource.includes("commandCenterTraceGapMs(tile, series)")) throw new Error("legend readouts must share command-center tile gap semantics with plotted traces");
if (!markerSource.includes("isDiscreteSeries(series)") || !markerSource.includes("valueFromInterpolation(series, interpolated)")) throw new Error("legend readouts must share discrete counter and pressure log interpolation semantics with plotted traces");
if (!markerSource.includes("const unit = series.unit || series.units || unitForAxis(series.axis_id)")) throw new Error("legend readouts must honor both unit and units aliases");
if (!markerSource.includes("stateLabel(series") || !markerSource.includes("series.value_table")) throw new Error("state span readouts must resolve value_table labels before numeric fallbacks");
if (!markerSource.includes("timeMs < end") || !markerSource.includes("selectedStart")) throw new Error("state spans must use half-open transition boundaries and prefer the latest matching span");
if (!visualPolicySource.includes('Pick<TileSeries, "id" | "role" | "render_kind" | "kind" | "color">') || !visualPolicySource.includes("configuredColor.trim()")) throw new Error("graph signal colors must honor backend contract-provided colors before local palettes");
if (!visualPolicySource.includes("const roleColor = roleColors[signal.role]") || !visualPolicySource.includes("if (roleColor) return roleColor")) throw new Error("graph signal colors must use canonical role colors before palette fallbacks");
if (!graphWallSource.includes("stateLabel(series, block.value")) throw new Error("swimlane labels must resolve value_table labels before generic active/idle fallbacks");
if (!graphWallSource.includes("function stateBlockIsActive") || !graphWallSource.includes("stateBlockDisplayLabel(series, block)") || !graphWallSource.includes("return !inactiveLabels.has(label)")) throw new Error("swimlane state block activity must use semantic labels instead of numeric positivity only");
if (!graphWallSource.includes("function heroFooterStateSeries") || !graphWallSource.includes("heroFooterStateIDs.has(series.id)") || !graphWallSource.includes('renderKind === "swimlane"')) throw new Error("hero state footer must discover state-like span series instead of only literal trace IDs");
if (graphWallSource.includes("background: block.value > 0 ? colorForSignal(series)")) throw new Error("swimlane state block fill must not classify string-valued states as idle");
if (!graphWallSource.includes("function observedStateBlocks") || !graphWallSource.includes("Math.min(block.left + block.width, observedPct)")) throw new Error("swimlane replay state blocks must be clipped to observed replay time");
if (!graphWallSource.includes("event-marker-overflow") || !graphWallSource.includes("event-chip-overflow")) throw new Error("dense event rails must keep overflow labels visible instead of silently dropping them");
if (!graphWallSource.includes('if (baseNow >= endMs)') || !graphWallSource.includes("if (next >= endMs)") || !graphWallSource.includes("window.clearInterval(timer);\n        return;")) throw new Error("accelerated replay timers must stop after reaching the replay end");
if (!graphWallSource.includes('querySelector<HTMLElement>(".graph-card-plot-shell")') || graphWallSource.includes("const startHeight = cardRefEl.current?.getBoundingClientRect().height")) throw new Error("graph card resize must use plot-shell height, not whole-card height");
if (!markerSource.includes("-5 * gap") || !markerSource.includes("5 * gap")) throw new Error("marker labels need enough fallback stack positions for dense graphs");
if (!graphWallSource.includes('from "signalforge-web"')) throw new Error("operator graph wall must consume SignalForge web graph primitives");
if (graphWallSource.includes("./tiles/uPlotAdapter")) throw new Error("operator graph wall must not use a local uPlot adapter copy");
if (graphWallSource.includes("./tiles/decimation")) throw new Error("operator graph wall must use SignalForge decimation helpers, not local forks");
for (const helper of ["CANONICAL_TILE_RENDERER", "uplotData", "drawTileOverlays", "stateBlocks", "inTimeRange", "renderKindFor", "scaleForSeries"]) {
  if (!graphWallSource.includes(helper)) throw new Error(`operator graph wall missing SignalForge helper ${helper}`);
}
for (const helper of ["viewportSeries", "lttb", "decimationValue", "resampleSeries", "commandCenterGapBreaks", "commandCenterTraceGapMs", "commandCenterProjectedSeries", "displayValue"]) {
  if (!graphWallSource.includes(helper)) throw new Error(`operator graph wall missing SignalForge decimation helper ${helper}`);
}
for (const helper of ["interpolationValue", "isDiscreteSeries", "valueFromInterpolation"]) {
  if (!markerSource.includes(helper)) throw new Error(`marker readouts missing SignalForge interpolation helper ${helper}`);
}
if (!timeAxisSource.includes("const requested = Number.isFinite(count) ? Math.round(count) : TIME_GRID_TICK_COUNT_DEFAULT") || !timeAxisSource.includes("const target = Math.max(2, Math.min(20, requested))")) throw new Error("shared time axis must honor compact/mobile tick budgets");
if (!graphWallSource.includes("data-graph-renderer={CANONICAL_TILE_RENDERER}")) throw new Error("operator graph wall must advertise the canonical SignalForge renderer");
if (!graphWallSource.includes('import uPlot from "uplot"')) throw new Error("Gossamer interaction shell must keep uPlot as the SignalForge graph engine");
if (!viteConfigSource.includes('"signalforge-web"') || !viteConfigSource.includes("./vendor/signalforge-web/dist/signalforge-web.es.js")) throw new Error("Vite must resolve signalforge-web to the vendored public SignalForge dist build");
if (webTsConfig.compilerOptions?.paths?.["signalforge-web"]?.[0] !== "vendor/signalforge-web/dist/index.d.ts") throw new Error("TypeScript must resolve signalforge-web to vendored SignalForge public dist types");
if (webPackage.dependencies?.["signalforge-web"] !== "file:vendor/signalforge-web") throw new Error("web package must use the vendored SignalForge web package for reproducible builds");
if (!signalForgeUPlotAdapterSource) throw new Error("vendored SignalForge package must include uPlot adapter source-map evidence");
if (!signalForgeUPlotAdapterSource.includes("function commandAnchoredMarker")) throw new Error("SignalForge event markers need explicit command-line anchoring policy");
if (!signalForgeUPlotAdapterSource.includes('commandAnchored && series.role === "command"')) throw new Error("SignalForge command-like event markers must prefer command role series");
if (!signalForgeUPlotAdapterSource.includes("function drawExactMarkerAnchorLine")) throw new Error("SignalForge event markers need an exact timestamp anchor line");
if (!signalForgeUPlotAdapterSource.includes("ctx.moveTo(x, top)") || !signalForgeUPlotAdapterSource.includes("ctx.lineTo(x, top + height)")) throw new Error("SignalForge marker anchor line must stay exactly on marker timestamp");
if (signalForgeUPlotAdapterSource.includes('anchorY - (marker.kind === "functional_gate" ? 5 : 0)')) throw new Error("SignalForge functional gate glyph must not be vertically offset from command-line anchor");
if (!signalForgeUPlotAdapterSource.includes("drawMarkerLeader(ctx, x, anchorY")) throw new Error("SignalForge attached marker labels must move independently with a leader back to the exact anchor");
if (!signalForgeUPlotAdapterSource.includes('import uPlot from "uplot"')) throw new Error("SignalForge tile renderer must use uPlot as the canonical graph engine");
if (!signalForgeUPlotAdapterSource.includes('(band.kind ?? "").toLowerCase()')) throw new Error("SignalForge uPlot adapter must preserve band-kind null safety");
if (webPackage.dependencies?.uplot !== "^1.6.32") throw new Error("web package must keep uPlot dependency explicit");
if (!apiSource.includes('currentBundle: () => getJSON<TileBundleManifest>("/data/current/manifest.json")')) throw new Error("frontend must use static tile bundle manifest");
if (!apiSource.includes('getJSON<GraphTileManifest>(`/data/current/campaigns/${id}/manifest.json`)')) throw new Error("frontend must load campaign tile manifests from static bundle");
if (!apiSource.includes('getJSON<GraphModel>(`/data/current/campaigns/${id}/graph-shell.json`)')) throw new Error("frontend must load graph shells from static bundle");
if (!apiSource.includes("return arrowTile(id, card, manifest, graph")) throw new Error("frontend tile API must materialize Arrow-backed GraphTile objects");
if (apiSource.includes("/api/campaigns/${id}/tiles")) throw new Error("frontend must not fetch legacy JSON tile endpoints for graph wall rendering");
if (!arrowTilesSource.includes("tableFromIPC")) throw new Error("Arrow tiles must decode Apache Arrow IPC");
if (!arrowTilesSource.includes("telemetry.arrow.gz")) throw new Error("Arrow tiles must load compressed telemetry archives from the static tile bundle");
if (!arrowTilesSource.includes('source: "arrow_telemetry"')) throw new Error("GraphTile diagnostics must preserve Arrow telemetry provenance");
if (!arrowTilesSource.includes('mode: "browser_native_arrow"')) throw new Error("GraphTile diagnostics must identify browser-native Arrow mode");

const supervisor = await readJSON("supervisor_overview.json");
requireEnvelope("supervisor", supervisor);
if (!Array.isArray(supervisor.lanes) || supervisor.lanes.length < 4) throw new Error("supervisor requires at least four lanes");
let hasTemperatureHero = false;
for (const lane of supervisor.lanes) {
  if (!Array.isArray(lane.hero_graphs) || lane.hero_graphs.length === 0) throw new Error(`supervisor lane ${lane.id} missing hero graphs`);
  for (const graph of lane.hero_graphs) {
    if (graph.units === "degC" && Array.isArray(graph.values) && graph.values.length > 0) hasTemperatureHero = true;
  }
}
if (!hasTemperatureHero) throw new Error("supervisor requires a temperature hero graph");

const busTap = await readJSON("bus_virtualization_tap.json");
requireEnvelope("bus tap", busTap);
if (!busTap.events?.some((event) => event.direction === "TM")) throw new Error("bus tap missing TM events");
if (!busTap.events?.some((event) => event.direction === "TC")) throw new Error("bus tap missing TC events");

const sources = await readJSON("source_catalogue.json");
requireEnvelope("sources", sources);
requireNoForbiddenVocabularyInLabels("source catalogue", sources);
if (!Array.isArray(sources.sources) || sources.sources.length < 4) throw new Error("source catalogue too small");
const sourceIDs = new Set(sources.sources.map((source) => source.id));
const treeLeafIDs = new Set();
function visitSourceTree(nodes) {
  for (const node of nodes ?? []) {
    if (!node.id || !node.label || !node.kind) throw new Error("source tree node missing id, label, or kind");
    if (node.kind === "stream") {
      if (!node.source_id) throw new Error(`source tree stream ${node.id} missing source_id`);
      treeLeafIDs.add(node.source_id);
    }
    visitSourceTree(node.children);
  }
}
visitSourceTree(sources.tree);
for (const source of sources.sources) {
  if (!source.owner || !source.bus || !source.quality) throw new Error(`source ${source.id} missing required field`);
  if (!["exclusive_connection", "shared_monitor", "external_master", "derived", "fallback"].includes(source.owner_mode)) throw new Error(`source ${source.id} has invalid owner_mode`);
  if (!["primary", "shared", "derivative", "fallback"].includes(source.use)) throw new Error(`source ${source.id} has invalid use`);
  if (!["decoded", "raw_legacy"].includes(source.format_preference)) throw new Error(`source ${source.id} has invalid format_preference`);
  if (!source.discovery_path?.node || !source.discovery_path?.device || !source.discovery_path?.subsystem || !source.discovery_path?.stream) throw new Error(`source ${source.id} missing discovery_path`);
  if (!treeLeafIDs.has(source.id)) throw new Error(`source ${source.id} missing from source tree`);
}
if (!sourceIDs.has("tvac_tmtc_primary") || !sourceIDs.has("tvac_tmtc_backup")) throw new Error("source catalogue missing TMTC primary/backup examples");
if (!sources.sources.some((source) => source.format_preference === "raw_legacy")) throw new Error("source catalogue missing raw legacy fallback example");
const graphWallManifest = await readJSON("graph_wall_manifest.json");
requireEnvelope("graph wall manifest", graphWallManifest);
if (!Array.isArray(graphWallManifest.targets) || graphWallManifest.targets.length < 3) throw new Error("graph wall manifest requires targets");
for (const target of graphWallManifest.targets) {
  if (!target.target_id || !target.lane || !target.role || !target.source_id || !target.timestamp) throw new Error("graph wall target missing required fields");
  if (!sourceIDs.has(target.source_id)) throw new Error(`graph wall target references unknown source ${target.source_id}`);
}

for (const campaignID of manifest.campaigns) {
  const campaign = await readJSON(`campaigns/${campaignID}.json`);
  requireEnvelope(campaignID, campaign);
  if (!Array.isArray(campaign.requirements) || campaign.requirements.length < 6) throw new Error(`${campaignID} requirements missing`);
  if (!Array.isArray(campaign.anomalies)) throw new Error(`${campaignID} anomalies must be an array`);
  const graph = await readJSON(`graph_models/${campaignID}.json`);
  requireEnvelope(`${campaignID} graph`, graph);
  requireNoForbiddenVocabularyInLabels(`graph model ${campaignID}`, graph);
  const report = await readJSON(`reports/${campaignID}_report.json`);
  requireEnvelope(`${campaignID} report`, report);
  requireReportTraceability(campaignID, campaign, report);
  if (!Array.isArray(report.requirements)) throw new Error(`${campaignID} report requirements must be an array`);
  if (!Array.isArray(report.sources)) throw new Error(`${campaignID} report sources must be an array`);
  if (!Array.isArray(report.graph_evidence)) throw new Error(`${campaignID} report graph evidence must be an array`);
  if (!Array.isArray(report.anomalies)) throw new Error(`${campaignID} report anomalies must be an array`);
  if (!Array.isArray(report.reproducibility)) throw new Error(`${campaignID} report reproducibility must be an array`);
  requireNoForbiddenVocabularyInLabels(`report ${campaignID}`, report);
  for (const lane of graph.lanes) {
    for (const series of lane.series) {
      if (!series.units || !series.role) throw new Error(`${series.id} missing units or role`);
    }
  }
}

console.log("contract fixtures ok");
