import { readFile } from "node:fs/promises";
import { join } from "node:path";

const root = new URL("../..", import.meta.url).pathname;
const publicDir = join(root, "fixtures", "public");
const appSource = await readFile(join(root, "web", "src", "App.tsx"), "utf8");
const architectureSource = await readFile(join(root, "docs", "architecture.md"), "utf8");
const generatorSource = await readFile(join(root, "internal", "synthetic", "generator.go"), "utf8");
const backlogSource = await readFile(join(root, "docs", "backlog", "BACKLOG.md"), "utf8");
const graphWallSource = await readFile(join(root, "web", "src", "components", "OperatorGraphWall.tsx"), "utf8");
const graphCardCSS = await readFile(join(root, "web", "src", "styles", "graph-card.css"), "utf8");
const markerSource = await readFile(join(root, "web", "src", "components", "tiles", "markers.ts"), "utf8");
const viewsCSS = await readFile(join(root, "web", "src", "styles", "views.css"), "utf8");
const semanticsChecklist = await readFile(join(root, "docs", "backend_semantics_checklist.md"), "utf8");

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
if (!graphWallSource.includes('renderKind === "event_rail" ? 360')) throw new Error("event rail resize height must stay bounded");
if (!graphCardCSS.includes(".tile-event-rail")) throw new Error("event rail tile styles missing");
if (!markerSource.includes("slice(0, 8)")) throw new Error("event marker labels must default to eight characters or fewer");
if (!graphWallSource.includes("shortGateLabel(marker.label)")) throw new Error("event rail markers must use short label rendering");
if (!graphWallSource.includes("labeledEventRailMarkerIDs")) throw new Error("event rail marker labels must suppress overlapping labels without hiding marker dots");
if (!graphWallSource.includes('title={`${marker.label} ${marker.timestamp}`}')) throw new Error("event rail markers must preserve full hover titles");

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
