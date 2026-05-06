import { readFile } from "node:fs/promises";
import { join } from "node:path";

const root = new URL("../..", import.meta.url).pathname;
const publicDir = join(root, "fixtures", "public");
const appSource = await readFile(join(root, "web", "src", "App.tsx"), "utf8");

function requireEnvelope(name, value) {
  if (value.schema_version !== 1) throw new Error(`${name} has invalid schema_version`);
  if (!value.generated_at) throw new Error(`${name} missing generated_at`);
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
if (!Array.isArray(sources.sources) || sources.sources.length < 4) throw new Error("source catalogue too small");
for (const source of sources.sources) {
  if (!source.owner || !source.bus || !source.quality) throw new Error(`source ${source.id} missing required field`);
}

for (const campaignID of manifest.campaigns) {
  const campaign = await readJSON(`campaigns/${campaignID}.json`);
  requireEnvelope(campaignID, campaign);
  if (!Array.isArray(campaign.requirements) || campaign.requirements.length < 8) throw new Error(`${campaignID} requirements missing`);
  if (!Array.isArray(campaign.anomalies)) throw new Error(`${campaignID} anomalies must be an array`);
  const graph = await readJSON(`graph_models/${campaignID}.json`);
  requireEnvelope(`${campaignID} graph`, graph);
  const report = await readJSON(`reports/${campaignID}_report.json`);
  requireEnvelope(`${campaignID} report`, report);
  if (!Array.isArray(report.requirements)) throw new Error(`${campaignID} report requirements must be an array`);
  if (!Array.isArray(report.sources)) throw new Error(`${campaignID} report sources must be an array`);
  if (!Array.isArray(report.graph_evidence)) throw new Error(`${campaignID} report graph evidence must be an array`);
  if (!Array.isArray(report.anomalies)) throw new Error(`${campaignID} report anomalies must be an array`);
  if (!Array.isArray(report.reproducibility)) throw new Error(`${campaignID} report reproducibility must be an array`);
  for (const lane of graph.lanes) {
    for (const series of lane.series) {
      if (!series.units || !series.role) throw new Error(`${series.id} missing units or role`);
    }
  }
}

console.log("contract fixtures ok");
