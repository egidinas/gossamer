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
];

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

await mkdir(artifactDir, { recursive: true });

const api = start("api", "go", ["run", "./cmd/gossamer-server", "-addr", `127.0.0.1:${apiPort}`]);
const web = start("web", "npm", ["run", "dev", "--", "--host", "127.0.0.1", "--port", webPort], {
  cwd: webRoot,
  env: { VITE_GOSSAMER_API_BASE: apiURL },
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
        if (routeName === "acceptance" || routeName === "qualification" || routeName === "command-center") {
          await page.locator(".operator-graph-wall").waitFor({ state: "visible", timeout: 10000 });
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
