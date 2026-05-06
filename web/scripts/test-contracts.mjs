import { readFile } from "node:fs/promises";
import { join } from "node:path";

const root = new URL("../..", import.meta.url).pathname;
const publicDir = join(root, "fixtures", "public");

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
  const graph = await readJSON(`graph_models/${campaignID}.json`);
  requireEnvelope(`${campaignID} graph`, graph);
  for (const lane of graph.lanes) {
    for (const series of lane.series) {
      if (!series.units || !series.role) throw new Error(`${series.id} missing units or role`);
    }
  }
}

console.log("contract fixtures ok");

