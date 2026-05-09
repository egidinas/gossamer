import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { decodeAgentContext, encodeAgentContext } from "@loom-gossamer/shared/agent-context-codec";

const root = new URL("../..", import.meta.url).pathname;
const fixturePath = join(root, "fixtures", "public", "agent_context_codec_benchmark.json");

const REQUIRED_FIELDS = [
  "id",
  "provenance",
  "authority",
  "fixture_status",
  "source_id",
];

function fail(message) {
  throw new Error(`agent context codec: ${message}`);
}

function stableJSONString(value) {
  if (Array.isArray(value)) {
    return `[${value.map(stableJSONString).join(",")}]`;
  }
  if (value && typeof value === "object") {
    return `{${Object.keys(value)
      .sort()
      .map((key) => `${JSON.stringify(key)}:${stableJSONString(value[key])}`)
      .join(",")}}`;
  }
  return JSON.stringify(value);
}

function encodeRows(rows, fields) {
  return {
    encoding: "gossamer.agent_context_table.v1",
    fields,
    rows: rows.map((row) => fields.map((field) => row[field] ?? null)),
  };
}

function decodeRows(compact) {
  if (compact.encoding !== "gossamer.agent_context_table.v1") {
    fail(`unsupported compact encoding ${compact.encoding}`);
  }
  return compact.rows.map((row) => Object.fromEntries(compact.fields.map((field, index) => [field, row[index]])));
}

const fixture = JSON.parse(await readFile(fixturePath, "utf8"));

if (fixture.schema_version !== 1) fail("fixture schema_version must be 1");
if (fixture.synthetic_only !== true) fail("fixture must be synthetic_only");
if (!Array.isArray(fixture.samples) || fixture.samples.length < 2) fail("fixture requires at least two samples");

for (const sample of fixture.samples) {
  if (!sample.name) fail("sample missing name");
  if (!Array.isArray(sample.canonical_rows) || sample.canonical_rows.length === 0) fail(`${sample.name} missing canonical rows`);
  for (const field of REQUIRED_FIELDS) {
    if (!sample.fields.includes(field)) fail(`${sample.name} does not preserve required field ${field}`);
  }
  for (const row of sample.canonical_rows) {
    for (const field of REQUIRED_FIELDS) {
      if (row[field] === undefined || row[field] === "") fail(`${sample.name} row missing required field ${field}`);
    }
  }

  const compact = encodeRows(sample.canonical_rows, sample.fields);
  const decoded = decodeRows(compact);
  const sharedDecoded = decodeAgentContext(encodeAgentContext(sample.canonical_rows));
  const canonicalJSON = stableJSONString(sample.canonical_rows);
  const compactJSON = stableJSONString(compact);
  const reduction = 1 - compactJSON.length / canonicalJSON.length;

  if (stableJSONString(decoded) !== canonicalJSON) fail(`${sample.name} compact round-trip changed canonical rows`);
  if (stableJSONString(sharedDecoded) !== canonicalJSON) fail(`${sample.name} shared codec round-trip changed canonical rows`);
  if (compactJSON !== stableJSONString(sample.compact)) fail(`${sample.name} compact fixture is stale`);
  if (sample.metrics?.canonical_bytes !== canonicalJSON.length) fail(`${sample.name} canonical byte count is stale`);
  if (sample.metrics?.compact_bytes !== compactJSON.length) fail(`${sample.name} compact byte count is stale`);
  if (Math.abs(sample.metrics?.byte_reduction_ratio - reduction) > 0.000001) fail(`${sample.name} reduction ratio is stale`);
  if (reduction < 0.20) fail(`${sample.name} reduction below 20%`);
}

console.log("agent-context-codec-ok");
