import { readdir, readFile, stat } from "node:fs/promises";
import { extname, join, relative } from "node:path";

const repoRoot = new URL("../..", import.meta.url).pathname;

const textTargets = [
  {
    name: "docs",
    root: join(repoRoot, "docs"),
    exts: new Set([".md", ".txt", ".yml", ".yaml", ".toml", ".json", ".jsonl", ".sh", ".go", ".ts", ".tsx", ".js", ".mjs", ".css", ".html"]),
  },
  {
    name: "public fixtures",
    root: join(repoRoot, "fixtures/public"),
    exts: new Set([".json"]),
  },
  {
    name: "source (internal)",
    root: join(repoRoot, "internal"),
    exts: new Set([".go", ".json"]),
  },
  {
    name: "source (web)",
    root: join(repoRoot, "web/src"),
    exts: new Set([".ts", ".tsx", ".css", ".json"]),
  },
  {
    name: "source (scripts)",
    root: join(repoRoot, "scripts"),
    exts: new Set([".sh", ".ps1", ".yaml", ".yml", ".go", ".json", ".md", ".js", ".ts"]),
  },
  {
    name: "source (cmd)",
    root: join(repoRoot, "cmd"),
    exts: new Set([".go"]),
  },
  {
    name: "web scripts",
    root: join(repoRoot, "web/scripts"),
    exts: new Set([".mjs", ".js"]),
  },
];

// Raster screenshots are intentionally excluded from this static text scan:
// compressed pixels cannot prove rendered text content. Hosted/browser smoke
// checks rendered DOM text and canvas pixels before screenshots are trusted.
const binaryTargets = [];

const forbiddenPatterns = [
  {
    id: "private-ipv4",
    regex: /\b(?:10\.(?:\d{1,3}\.){2}\d{1,3}|172\.(?:1[6-9]|2\d|3[0-1])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3}|169\.254\.\d{1,3}\.\d{1,3})\b/g,
    message: "private network address",
  },
  {
    id: "private-hostname",
    regex: /\b(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+(?:local|lan|internal|private|corp|home)\b/gi,
    message: "private hostname-like identifier",
  },
  {
    id: "credential-assignment",
    regex: /\b(?:api[_-]?key|access[_-]?token|auth(?:entication)?[_-]?token|refresh[_-]?token|bearer[_-]?token|password|passwd|private[_-]?key|secret)\s*[:=]\s*["']?[A-Za-z0-9+/._-]{12,}/gi,
    message: "credential-like assignment",
  },
  {
    id: "protocol-capture-artifact",
    regex: /\b[a-z0-9._-]+\.(?:dbc|arxml|kcd|pcap|pcapng|mf4|asc)\b/gi,
    message: "protocol DB / capture artifact",
  },
  {
    id: "protocol-capture-phrase",
    regex: /\b(?:protocol\s+database|packet\s+capture|trace\s+capture|bus\s+database|raw\s+capture)\b/gi,
    message: "protocol capture wording",
  },
  {
    id: "live-hardware-procedure",
    regex: /\b(?:commissioning|acceptance|safety|facility|chamber)\s+(?:procedure|runbook|sequence)\b|\blive\s+hardware\s+procedure\b/gi,
    message: "hardware procedure language",
  },
];

const allowedHostLiteral = new Set(["127.0.0.1", "localhost"]);
const allowedUrlHost = new Set(["gossamer.jmeyer.space", "jmeyer.space"]);
const documentationExceptionFiles = new Set([
  "docs/clean_room_import_checklist.md",
  "docs/ip_clean_room.md",
  "docs/backlog/shared_loom_gossamer_backlog.md",
  "web/scripts/test-clean-room.mjs",
]);

function toUnixPath(value) {
  return value.replaceAll("\\", "/");
}

function shouldIgnorePath(filePath) {
  return filePath.includes("/node_modules/") || filePath.includes("/dist/") || filePath.includes("/.git/");
}

function isAllowedMatch(term, source = "text") {
  if (allowedHostLiteral.has(term.toLowerCase())) return true;
  if (/\b(localhost|127\.0\.0\.1)\b/i.test(term)) return true;
  if (allowedUrlHost.has(term.toLowerCase())) return true;
  if (source === "binary" && term.includes("token")) return true;
  return false;
}

function lineForOffset(content, index) {
  return content.slice(0, index).split(/\r?\n/).length;
}

function addFinding(findings, path, kind, match, context, line) {
  findings.push({
    path,
    kind,
    match,
    line,
    context,
  });
}

async function collectFiles(root, allowedExts) {
  const files = [];
  const entries = await readdir(root, { withFileTypes: true });
  for (const entry of entries) {
    const path = join(root, entry.name);
    if (entry.name.startsWith(".") || entry.name === "node_modules" || entry.name === "dist" || entry.name === "test-artifacts") {
      continue;
    }
    if (entry.isDirectory()) {
      files.push(...(await collectFiles(path, allowedExts)));
      continue;
    }
    if (!entry.isFile()) continue;
    if (!allowedExts.has(extname(entry.name).toLowerCase())) continue;
    files.push(path);
  }
  return files;
}

function scanTextContent(path, content, findings) {
  const normalizedPath = toUnixPath(relative(repoRoot, path));
  const isDocumentedException = documentationExceptionFiles.has(normalizedPath);
  for (const rule of forbiddenPatterns) {
    if (isDocumentedException) break;
    for (const match of content.matchAll(rule.regex)) {
      const value = match[0];
      if (isAllowedMatch(value)) continue;
      const line = lineForOffset(content, match.index ?? 0);
      addFinding(findings, normalizedPath, rule.message, value, rule.id, line);
    }
  }
}

async function scanBinaryFile(path, findings) {
  const normalizedPath = toUnixPath(relative(repoRoot, path));
  let content;
  try {
    const raw = await readFile(path);
    content = raw.toString("latin1");
  } catch (err) {
    throw new Error(`failed to read screenshot ${normalizedPath}: ${err.message}`);
  }

  for (const rule of forbiddenPatterns) {
    for (const match of content.matchAll(rule.regex)) {
      const value = match[0];
      if (isAllowedMatch(value, "binary")) continue;
      addFinding(findings, normalizedPath, rule.message, value, rule.id, "binary");
    }
  }
}

async function run() {
  const findings = [];
  let scannedFiles = 0;

  for (const target of textTargets) {
    const exists = await stat(target.root).then(() => true).catch(() => false);
    if (!exists) {
      throw new Error(`missing scan root: ${toUnixPath(relative(repoRoot, target.root))}`);
    }
    const files = await collectFiles(target.root, target.exts);
    for (const path of files) {
      if (shouldIgnorePath(path)) continue;
      const content = await readFile(path, "utf8");
      scannedFiles += 1;
      scanTextContent(path, content, findings);
    }
  }

  for (const target of binaryTargets) {
    const exists = await stat(target.root).then(() => true).catch(() => false);
    if (!exists) continue;
    const files = await collectFiles(target.root, new Set([".png", ".jpg", ".jpeg", ".webp"]));
    for (const path of files) {
      if (shouldIgnorePath(path)) continue;
      scannedFiles += 1;
      await scanBinaryFile(path, findings);
    }
  }

  const invalid = findings.filter((item) => item.kind && item.match && item.match.length > 0);
  if (invalid.length === 0) {
    console.log(`clean-room guardrail scan ok (${scannedFiles} files checked)`);
    return;
  }

  console.error(`clean-room guardrail scan failed (${invalid.length} findings):`);
  for (const item of invalid) {
    const location = typeof item.line === "string" ? item.line : `line ${item.line}`;
    console.error(`- ${item.path}:${location} ${item.kind}: ${item.match}`);
  }
  process.exit(1);
}

await run();
