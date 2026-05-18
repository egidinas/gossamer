import { readFile, readdir, stat } from "node:fs/promises";
import { extname, join, relative } from "node:path";

const repoRoot = new URL("../..", import.meta.url).pathname;

const privateRepoNames = [
  "loom",
  "mynaric_telemetry",
  "work_time",
  "Jobsearch",
  "kvaser-dual-bridge",
];

const forbiddenFragments = [
  "loom-gossamer-shared",
  "@loom-gossamer/shared",
  ...privateRepoNames.map((name) => `github.com/egidinas/${name}`),
  ...privateRepoNames.flatMap((name) => [
    `/home/svc_pmg_testbed_b/${name}`,
    `../${name}`,
  ]),
];

const scanTargets = [
  {
    name: "root manifests",
    root: repoRoot,
    exts: new Set([".json"]),
    maxDepth: 0,
  },
  {
    name: "go source",
    root: repoRoot,
    exts: new Set([".go", ".mod", ".sum"]),
  },
  {
    name: "web manifests and scripts",
    root: join(repoRoot, "web"),
    exts: new Set([".json", ".mjs", ".js", ".ts", ".tsx"]),
  },
];

const allowedMentionFiles = new Set([
  "web/scripts/test-information-barrier.mjs",
]);

const ignoredPathParts = [
  "/.git/",
  "/node_modules/",
  "/dist/",
  "/test-artifacts/",
];

function toUnixPath(value) {
  return value.replaceAll("\\", "/");
}

function shouldSkip(path) {
  const normalized = toUnixPath(path);
  return ignoredPathParts.some((part) => normalized.includes(part));
}

function lineForOffset(content, index) {
  return content.slice(0, index).split(/\r?\n/).length;
}

async function collectFiles(root, allowedExts, maxDepth = Infinity, depth = 0) {
  const files = [];
  const entries = await readdir(root, { withFileTypes: true });
  for (const entry of entries) {
    const path = join(root, entry.name);
    if (entry.name.startsWith(".") || entry.name === "node_modules" || entry.name === "dist" || entry.name === "test-artifacts") {
      continue;
    }
    if (entry.isDirectory()) {
      if (depth < maxDepth) files.push(...(await collectFiles(path, allowedExts, maxDepth, depth + 1)));
      continue;
    }
    if (!entry.isFile()) continue;
    if (!allowedExts.has(extname(entry.name).toLowerCase())) continue;
    files.push(path);
  }
  return files;
}

function findForbiddenFragments(normalizedPath, content, findings) {
  if (allowedMentionFiles.has(normalizedPath)) return;
  for (const fragment of forbiddenFragments) {
    let index = content.indexOf(fragment);
    while (index !== -1) {
      findings.push({
        path: normalizedPath,
        line: lineForOffset(content, index),
        kind: "forbidden private dependency/path reference",
        match: fragment,
      });
      index = content.indexOf(fragment, index + fragment.length);
    }
  }
}

function checkGoMod(content, findings) {
  const normalizedPath = "go.mod";
  if (!content.includes("github.com/egidinas/signalforge ")) {
    findings.push({
      path: normalizedPath,
      line: 1,
      kind: "missing public SignalForge dependency",
      match: "github.com/egidinas/signalforge",
    });
  }
  const localReplace = /^replace\s+github\.com\/egidinas\/signalforge\s+=>\s+(?:\.{1,2}\/|\/home\/)/gm;
  for (const match of content.matchAll(localReplace)) {
    findings.push({
      path: normalizedPath,
      line: lineForOffset(content, match.index ?? 0),
      kind: "local SignalForge replace breaks public boundary",
      match: match[0],
    });
  }
}

function checkPackageJSON(normalizedPath, content, findings) {
  let parsed;
  try {
    parsed = JSON.parse(content);
  } catch {
    return;
  }
  for (const section of ["dependencies", "devDependencies", "optionalDependencies"]) {
    const deps = parsed[section] ?? {};
    for (const [name, spec] of Object.entries(deps)) {
      const dependencyName = name.toLowerCase();
      const dependencySpec = String(spec);
      const isApprovedSignalForgeWebFile = dependencyName === "signalforge-web" && dependencySpec === "file:vendor/signalforge-web";
      if (dependencyName.includes("loom") || (dependencySpec.startsWith("file:") && !isApprovedSignalForgeWebFile)) {
        findings.push({
          path: normalizedPath,
          line: 1,
          kind: "forbidden web dependency boundary",
          match: `${section}.${name}=${spec}`,
        });
      }
    }
  }
}

async function run() {
  const findings = [];
  let scannedFiles = 0;

  for (const target of scanTargets) {
    const exists = await stat(target.root).then(() => true).catch(() => false);
    if (!exists) continue;
    const files = await collectFiles(target.root, target.exts, target.maxDepth ?? Infinity);
    for (const path of files) {
      if (shouldSkip(path)) continue;
      const normalizedPath = toUnixPath(relative(repoRoot, path));
      const content = await readFile(path, "utf8");
      scannedFiles += 1;
      findForbiddenFragments(normalizedPath, content, findings);
      if (normalizedPath === "go.mod") checkGoMod(content, findings);
      if (normalizedPath.endsWith("package.json")) checkPackageJSON(normalizedPath, content, findings);
    }
  }

  if (findings.length === 0) {
    console.log(`information-barrier scan ok (${scannedFiles} files checked)`);
    return;
  }

  console.error(`information-barrier scan failed (${findings.length} findings):`);
  for (const item of findings) {
    console.error(`- ${item.path}:line ${item.line} ${item.kind}: ${item.match}`);
  }
  process.exit(1);
}

await run();
