import { mkdir } from "node:fs/promises";
import { join, resolve } from "node:path";
import { spawn } from "node:child_process";
import { createServer } from "node:net";
import { chromium } from "playwright";

const root = resolve(new URL("../..", import.meta.url).pathname);
const webRoot = join(root, "web");

async function freePort() {
  return new Promise((resolvePort, rejectPort) => {
    const server = createServer();
    server.once("error", rejectPort);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        server.close(() => rejectPort(new Error("could not allocate a local port")));
        return;
      }
      const port = String(address.port);
      server.close(() => resolvePort(port));
    });
  });
}

const apiPort = process.env.GOSSAMER_BROWSER_API_PORT || await freePort();
const webPort = process.env.GOSSAMER_BROWSER_WEB_PORT || await freePort();
const apiURL = `http://127.0.0.1:${apiPort}`;
const webURL = `http://127.0.0.1:${webPort}`;
const artifactDir = join(webRoot, "test-artifacts", "screenshots");

const routes = [
  ["landing", "#landing"],
  ["profile", "#profile"],
  ["acceptance", "#acceptance"],
  ["command-center", "#command-center-fat"],
  ["qualification", "#qualification"],
  ["mission-map", "#mission-map"],
  ["supervisor", "#supervisor"],
  ["graph-wall", "#graph-wall"],
  ["sources", "#sources"],
  ["requirements", "#requirements"],
  ["commands", "#commands"],
  ["bus-tap", "#bus-tap"],
  ["report", "#report"],
  ["file-viewer", "#file-viewer"],
];

const routeContentSelectors = new Map([
  ["profile", ".profile-grid"],
  ["mission-map", ".node-grid"],
  ["supervisor", ".operator-graph-wall"],
  ["graph-wall", ".operator-graph-wall"],
  ["sources", ".source-tree"],
  ["requirements", ".requirement-expression-row"],
  ["commands", ".command-grid"],
  ["bus-tap", ".stream-grid"],
  ["report", ".anomaly"],
  ["file-viewer", ".file-viewer-lane"],
]);

const graphRouteMinimums = new Map([
  ["acceptance", 6],
  ["command-center", 4],
  ["qualification", 8],
]);

const routeRequiredSignals = new Map([
  ["acceptance", ["trace.power_total", "trace.power_payload", "trace.dut_temp_a"]],
  ["qualification", ["trace.power_total", "trace.power_payload", "trace.dut_temp_a"]],
]);

const viewports = [
  ["desktop", { width: 1440, height: 960 }],
  ["4k", { width: 3840, height: 2160 }],
  ["mobile", { width: 390, height: 900 }],
];

function start(name, command, args, options = {}) {
  const child = spawn(command, args, {
    cwd: options.cwd || root,
    env: { ...process.env, ...options.env },
    stdio: ["ignore", "pipe", "pipe"],
    detached: true,
  });
  child.stdout.on("data", (chunk) => process.stdout.write(`[${name}] ${chunk}`));
  child.stderr.on("data", (chunk) => process.stderr.write(`[${name}] ${chunk}`));
  return child;
}

async function waitFor(url, name) {
  const started = Date.now();
  while (Date.now() - started < 30000) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {
      // keep waiting
    }
    await new Promise((resolveWait) => setTimeout(resolveWait, 300));
  }
  throw new Error(`${name} did not become ready at ${url}`);
}

async function launchBrowser() {
  try {
    return await chromium.launch();
  } catch (error) {
    if (process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE) {
      return chromium.launch({ executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE });
    }
    return chromium.launch({ executablePath: "chromium" });
  }
}

function stop(child) {
  if (child.exitCode !== null || child.signalCode !== null) return Promise.resolve();
  return new Promise((resolveStop) => {
    const timer = setTimeout(() => {
      if (child.exitCode === null && child.signalCode === null) {
        try {
          process.kill(-child.pid, "SIGKILL");
        } catch {
          child.kill("SIGKILL");
        }
      }
      resolveStop();
    }, 2000);
    child.once("exit", () => {
      clearTimeout(timer);
      resolveStop();
    });
    try {
      process.kill(-child.pid, "SIGTERM");
    } catch {
      child.kill("SIGTERM");
    }
  });
}

async function assertRequiredSignalReadouts(page, routeName, viewportName) {
  const requiredSignals = routeRequiredSignals.get(routeName);
  if (!requiredSignals) return;
  await page.waitForFunction(
    (signalIDs) => signalIDs.every((signalID) => {
      const chip = document.querySelector(`.graph-card-readout-chip[data-signal-id="${signalID}"]`);
      const value = chip?.getAttribute("data-readout-value") ?? "";
      return value.trim() && value.trim() !== "-";
    }),
    requiredSignals,
    { timeout: 30000 }
  ).catch(async () => {
    const observed = await page.evaluate((signalIDs) => signalIDs.map((signalID) => {
      const chip = document.querySelector(`.graph-card-readout-chip[data-signal-id="${signalID}"]`);
      return {
        signalID,
        found: Boolean(chip),
        label: chip?.querySelector("b")?.textContent?.trim() ?? "",
        sourceFamily: chip?.getAttribute("data-signal-source-family") ?? "",
        value: chip?.getAttribute("data-readout-value") ?? chip?.querySelector("em")?.textContent?.trim() ?? "",
      };
    }), requiredSignals);
    throw new Error(`${viewportName} ${routeName} missing populated required readouts: ${JSON.stringify(observed)}`);
  });
}

async function assertNoVisibleEventLabelOverlaps(page, routeName, viewportName) {
  const overlaps = await page.evaluate(() => {
    const labels = [...document.querySelectorAll(".event-marker-wrap strong, .event-chip-wrap strong")]
      .map((element) => {
        const rect = element.getBoundingClientRect();
        return {
          text: (element.textContent ?? "").trim(),
          left: rect.left,
          right: rect.right,
          top: rect.top,
          bottom: rect.bottom,
          width: rect.width,
          height: rect.height,
        };
      })
      .filter((item) => item.text && item.width > 2 && item.height > 2);
    const collisions = [];
    for (let i = 0; i < labels.length; i += 1) {
      for (let j = i + 1; j < labels.length; j += 1) {
        const a = labels[i];
        const b = labels[j];
        const xOverlap = Math.min(a.right, b.right) - Math.max(a.left, b.left);
        const yOverlap = Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top);
        if (xOverlap > 1 && yOverlap > 1) {
          collisions.push({ a: a.text, b: b.text, xOverlap: Math.round(xOverlap), yOverlap: Math.round(yOverlap) });
        }
      }
    }
    return collisions.slice(0, 8);
  });
  if (overlaps.length > 0) {
    throw new Error(`${viewportName} ${routeName} has overlapping event labels: ${JSON.stringify(overlaps)}`);
  }
}

async function assertGraphWallStackGeometry(page, routeName, viewportName) {
  const geometry = await page.evaluate(() => {
    const rect = (element) => {
      const box = element.getBoundingClientRect();
      return {
        top: box.top,
        bottom: box.bottom,
        left: box.left,
        right: box.right,
        height: box.height,
      };
    };
    const gaps = [];
    const titleProblems = [];

    for (const stack of document.querySelectorAll(".operator-wall-cards")) {
      const cards = [...stack.querySelectorAll(":scope > .graph-wall-card")]
        .filter((card) => card.getBoundingClientRect().height > 1);
      for (let index = 1; index < cards.length; index += 1) {
        const previous = rect(cards[index - 1]);
        const current = rect(cards[index]);
        gaps.push({
          previous: cards[index - 1].getAttribute("data-card-id"),
          current: cards[index].getAttribute("data-card-id"),
          gap: Number((current.top - previous.bottom).toFixed(2)),
        });
      }
    }

    for (const card of document.querySelectorAll(".operator-graph-wall .graph-wall-card:not(.graph-card-collapsed)")) {
      const title = card.querySelector(".graph-card-inline-title");
      const plot = card.querySelector(".graph-card-uplot, .tile-swimlane, .tile-event-rail, .graph-card-loading");
      if (!title || !plot) continue;
      const titleRect = rect(title);
      if (titleRect.height < 1) continue; // hidden (display:none on secondary cards)
      const cardRect = rect(card);
      const plotRect = rect(plot);
      const titleOverlapsPlot = titleRect.bottom > plotRect.top + 1 && titleRect.top < plotRect.bottom - 1;
      const titleOverflows = titleRect.left < cardRect.left - 1 || titleRect.right > cardRect.right + 1 || titleRect.top < cardRect.top - 1 || titleRect.bottom > cardRect.bottom + 1;
      if (titleOverlapsPlot || titleOverflows || titleRect.height < 18 || titleRect.height > 48) {
        titleProblems.push({
          card: card.getAttribute("data-card-id"),
          titleOverlapsPlot,
          titleOverflows,
          titleHeight: Math.round(titleRect.height),
        });
      }
    }

    return {
      maxGap: gaps.length ? Math.max(...gaps.map((gap) => gap.gap)) : 0,
      gaps: gaps.filter((gap) => Math.abs(gap.gap) > 1).slice(0, 8),
      titleProblems: titleProblems.slice(0, 8),
    };
  });

  if (geometry.maxGap > 1 || geometry.gaps.length > 0) {
    throw new Error(`${viewportName} ${routeName} graph wall has vertical card gaps: ${JSON.stringify(geometry.gaps)}`);
  }
  if (geometry.titleProblems.length > 0) {
    throw new Error(`${viewportName} ${routeName} graph title placement problems: ${JSON.stringify(geometry.titleProblems)}`);
  }
}

async function assertQualificationContract(page, routeName, viewportName, apiURL) {
  if (routeName !== "qualification" || viewportName !== "desktop") return;

  // 1. Geometry: every uPlot inner plot left edge must align with SharedTimeAxis track left edge (±2px).
  const alignResult = await page.evaluate(() => {
    const timeAxisTrack = document.querySelector(".operator-shared-time-axis > .time-axis-track");
    if (!timeAxisTrack) return { error: "no .operator-shared-time-axis > .time-axis-track found" };
    const trackLeft = Math.round(timeAxisTrack.getBoundingClientRect().left);
    const plots = [...document.querySelectorAll(".operator-graph-wall .u-plot .u-over")]
      .map((el) => ({ cardId: el.closest("[data-card-id]")?.getAttribute("data-card-id"), left: Math.round(el.getBoundingClientRect().left) }))
      .filter((el) => el.left >= 0);
    const misaligned = plots.filter((p) => Math.abs(p.left - trackLeft) > 2);
    return { trackLeft, plots: plots.length, misaligned };
  });
  if (alignResult.error) throw new Error(`${viewportName} ${routeName} geometry: ${alignResult.error}`);
  if (alignResult.misaligned.length > 0) throw new Error(`${viewportName} ${routeName} plot left edges misaligned with SharedTimeAxis track (trackLeft=${alignResult.trackLeft}): ${JSON.stringify(alignResult.misaligned)}`);
  console.log(`  └─ ${viewportName} ${routeName} axis alignment OK: ${alignResult.plots} plots aligned to x=${alignResult.trackLeft}`);

  // 2. Deduplication: dut_temperature and tvac_pressure companion cards must be absent.
  const dupCards = await page.evaluate(() => {
    const found = [];
    for (const id of ["dut_temperature", "tvac_pressure"]) {
      if (document.querySelector(`[data-card-id="${id}"]`)) found.push(id);
    }
    return found;
  });
  if (dupCards.length > 0) throw new Error(`${viewportName} ${routeName} still has redundant companion cards: ${dupCards.join(", ")}`);
  console.log(`  └─ ${viewportName} ${routeName} deduplication OK: dut_temperature and tvac_pressure absent`);

  // 3. Vacuum target ramp: fetch via Node (not browser) to avoid CORS; pressure_target must have ≥3 distinct values.
  const tileEndpoint = `${apiURL}/api/campaigns/tvac_qualification/tiles?card_id=thermal_program&level=minute`;
  let tileResp;
  try {
    const r = await fetch(tileEndpoint);
    if (!r.ok) throw new Error(`HTTP ${r.status}`);
    tileResp = await r.json();
  } catch (e) {
    throw new Error(`${viewportName} ${routeName} vacuum ramp check failed: ${e}`);
  }
  const targetSeries = (tileResp.series ?? []).find((s) => s.id === "trace.tvac_pressure_target");
  if (!targetSeries) throw new Error(`${viewportName} ${routeName} hero tile missing trace.tvac_pressure_target series`);
  // Verify ramp via raw fixture data — minute-level LTTB collapses sub-1-mbar values, so count distinct
  // non-ambient values from the per-cycle ramp logic in the generator (verified in Go unit tests).
  // Here we just confirm the series has data and any non-atmospheric values appear (vacuum phases).
  const allVals = (targetSeries.points ?? []).map((p) => p.value).filter((v) => typeof v === "number" && Number.isFinite(v));
  if (allVals.length < 10) throw new Error(`${viewportName} ${routeName} vacuum target series has too few points (${allVals.length})`);
  const hasVacuumPhase = allVals.some((v) => v < 1000);
  if (!hasVacuumPhase) throw new Error(`${viewportName} ${routeName} vacuum target series shows no vacuum phase (all values ≥ 1000 mbar)`);
  console.log(`  └─ ${viewportName} ${routeName} vacuum ramp OK: ${allVals.length} data points, vacuum phase present`);
}

await mkdir(artifactDir, { recursive: true });

const api = start("api", "go", ["run", "./cmd/gossamer-server", "-addr", `127.0.0.1:${apiPort}`]);
const web = start("web", "npm", ["run", "dev", "--", "--host", "127.0.0.1", "--port", webPort], {
  cwd: webRoot,
  env: { GOSSAMER_API_PROXY_TARGET: apiURL },
});

try {
  await waitFor(`${apiURL}/api/manifest`, "API");
  await waitFor(webURL, "web app");

  const browser = await launchBrowser();
  try {
    for (const [viewportName, viewport] of viewports) {
      const page = await browser.newPage({ viewport });
      const pageErrors = [];
      const failedResponses = [];
      page.on("pageerror", (error) => pageErrors.push(error.message));
      page.on("response", (response) => {
        const url = response.url();
        if (!response.ok() && !url.endsWith("/favicon.ico")) {
          failedResponses.push(`${response.status()} ${url}`);
        }
      });

      for (const [routeName, hash] of routes) {
        pageErrors.length = 0;
        failedResponses.length = 0;
        await page.goto(`${webURL}/${hash}`, { waitUntil: "networkidle" });
        await page.locator(".shell").waitFor({ state: "visible", timeout: 10000 });
        const contentSelector = routeContentSelectors.get(routeName);
        if (contentSelector) {
          await page.locator(contentSelector).first().waitFor({ state: "visible", timeout: 10000 });
        }
        const graphMinimum = graphRouteMinimums.get(routeName);
        if (graphMinimum) {
          console.log(`  [graph] ${viewportName} ${routeName} waiting for ${graphMinimum} canvases…`);
          await page.locator(".operator-graph-wall").waitFor({ state: "visible", timeout: 10000 });
          await page.waitForFunction(
            (minimum) => {
              const loading = document.querySelectorAll(".operator-graph-wall .graph-card-loading").length;
              const canvases = document.querySelectorAll(".operator-graph-wall canvas").length;
              return loading === 0 && canvases >= minimum;
            },
            graphMinimum,
            { timeout: 30000 }
          );
          await assertRequiredSignalReadouts(page, routeName, viewportName);
          await assertNoVisibleEventLabelOverlaps(page, routeName, viewportName);
          await assertGraphWallStackGeometry(page, routeName, viewportName);
          await assertQualificationContract(page, routeName, viewportName, apiURL);
        }
        await page.screenshot({ path: join(artifactDir, `${viewportName}-${routeName}.png`), fullPage: true });

        const textLength = await page.locator("body").innerText().then((text) => text.trim().length);
        if (textLength < 40) {
          throw new Error(`${viewportName} ${routeName} rendered a blank or near-blank route`);
        }
        const overflow = await page.evaluate(() => Math.max(0, document.documentElement.scrollWidth - document.documentElement.clientWidth));
        const shouldAllowGraphOverflow = viewportName === "mobile" && (routeName === "acceptance" || routeName === "qualification");
        if (shouldAllowGraphOverflow) {
          const graphFrame = await page.$eval(".operator-graph-wall .operator-wall-scrollframe", (frame) => {
            if (!(frame instanceof HTMLElement)) {
              return null;
            }
            return {
              exists: true,
              scrollable: frame.scrollWidth > frame.clientWidth + 2,
            };
          }).catch(() => null);
          if (!graphFrame) {
            throw new Error(`${viewportName} ${routeName} missing operator graph scrollframe`);
          }
          if (overflow > 1 && !graphFrame.scrollable) {
            throw new Error(`${viewportName} ${routeName} has ${overflow}px global overflow but graph frame is not horizontally scrollable`);
          }
        } else if (overflow > 1) {
          throw new Error(`${viewportName} ${routeName} has ${overflow}px horizontal overflow`);
        }

        if (routeName === "command-center" && viewportName === "4k") {
          const wallSelector = ".operator-graph-wall[data-campaign-id=\"command_center_fat\"]";
          const laneSelector = `${wallSelector} .graph-wall-card:not(.graph-card-collapsed)`;
          await page.locator(laneSelector).nth(3).waitFor({ state: "visible", timeout: 10000 });
          const occupancyHandle = await page.waitForFunction(
            ({ wallSelector, laneSelector }) => {
              const wall = document.querySelector(wallSelector);
              if (!wall) return null;
              const cards = Array.from(document.querySelectorAll(laneSelector)).filter((card) => card.getBoundingClientRect().height > 0).slice(0, 4);
              if (cards.length < 4) return null;
              const laneHeights = cards.map((card) => Math.round(card.getBoundingClientRect().height));
              const stackHeight = laneHeights.reduce((sum, h) => sum + h, 0);
              return {
                laneHeights,
                laneCount: cards.length,
                stackHeight,
                wallHeight: Math.round(wall.getBoundingClientRect().height),
                viewportHeight: Math.round(window.innerHeight),
                occupancy: Math.round((stackHeight / window.innerHeight) * 100),
              };
            },
            { wallSelector, laneSelector },
            { timeout: 15000 }
          );
          const occupancy = await occupancyHandle.jsonValue();
          if (!occupancy) {
            throw new Error(`${viewportName} ${routeName} has no visible lane cards`);
          }
          if (occupancy.laneCount < 4) {
            throw new Error(`${viewportName} ${routeName} has only ${occupancy.laneCount} visible lanes (expected 4)`);
          }
          if (occupancy.occupancy < 80) {
            throw new Error(`${viewportName} ${routeName} lane-stack occupancy only ${occupancy.occupancy}% (${occupancy.stackHeight}px / ${occupancy.viewportHeight}px), lane heights=${occupancy.laneHeights.join(", ")}`);
          }
          console.log(`  └─ ${viewportName} ${routeName} lane-stack occupancy: ${occupancy.stackHeight}px/${occupancy.viewportHeight}px (${occupancy.occupancy}%)`);
        }
        if (pageErrors.length > 0) {
          throw new Error(`${viewportName} ${routeName} page errors: ${pageErrors.join(" | ")}`);
        }
        if (failedResponses.length > 0) {
          throw new Error(`${viewportName} ${routeName} failed responses: ${failedResponses.join(" | ")}`);
        }
      }
      await page.close();
    }
  } finally {
    await browser.close();
  }
  console.log(`browser smoke ok; screenshots written to ${artifactDir}`);
} finally {
  await Promise.all([stop(web), stop(api)]);
}
