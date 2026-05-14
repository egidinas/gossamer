import { chromium } from "playwright";

const baseURL = process.env.GOSSAMER_HOSTED_URL || "https://gossamer.jmeyer.space/";
const expectedDataVersion = process.env.GOSSAMER_EXPECT_DATA_VERSION || "";

const routes = [
  { name: "landing", hash: "#landing", expectedGraphs: 0, textNeedle: "Complete Operator Surfaces", minReadouts: 0 },
  {
    name: "acceptance",
    hash: "#acceptance",
    expectedGraphs: 8,
    textNeedle: "Thermal Chamber FAT",
    minReadouts: 10,
    requiredSignals: ["trace.power_total", "trace.power_payload", "trace.dut_temp_a"],
  },
  { name: "command-center", hash: "#command-center-fat", expectedGraphs: 4, textNeedle: "Command Center FAT", minReadouts: 8 },
  {
    name: "qualification",
    hash: "#qualification",
    expectedGraphs: 12,
    textNeedle: "TVac Qualification",
    minReadouts: 12,
    requiredSignals: ["trace.power_total", "trace.power_payload", "trace.dut_temp_a", "trace.tvac_pressure"],
  },
];

async function launchBrowser() {
  try {
    return await chromium.launch();
  } catch {
    if (process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE) {
      return chromium.launch({ executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE });
    }
    return chromium.launch({ executablePath: "chromium" });
  }
}

function url(path) {
  return new URL(path, baseURL).toString();
}

async function assertRequiredSignalReadouts(page, route) {
  if (!route.requiredSignals?.length) return;
  await page.waitForFunction(
    (signalIDs) => signalIDs.every((signalID) => {
      const chip = document.querySelector(`.graph-card-readout-chip[data-signal-id="${signalID}"]`);
      const value = chip?.getAttribute("data-readout-value") ?? "";
      return value.trim() && value.trim() !== "-";
    }),
    route.requiredSignals,
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
    }), route.requiredSignals);
    throw new Error(`${route.name}: missing populated required readouts: ${JSON.stringify(observed)}`);
  });
}

async function assertNoVisibleEventLabelOverlaps(page, routeName) {
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
    throw new Error(`${routeName}: overlapping event labels: ${JSON.stringify(overlaps)}`);
  }
}

const manifestResponse = await fetch(url("/data/current/manifest.json"));
if (!manifestResponse.ok) {
  throw new Error(`hosted manifest failed: ${manifestResponse.status} ${manifestResponse.statusText}`);
}

const manifest = await manifestResponse.json();
if (expectedDataVersion && manifest.data_version !== expectedDataVersion) {
  throw new Error(`hosted data version ${manifest.data_version}, expected ${expectedDataVersion}`);
}

const browser = await launchBrowser();
try {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1100 }, deviceScaleFactor: 1 });
  const staleRequests = [];
  const failedRequests = [];
  const pageErrors = [];

  page.on("pageerror", (error) => pageErrors.push(error.message));
  page.on("request", (request) => {
    const requestURL = request.url();
    if (/\/api\/campaigns\/.*\/telemetry|\/telemetry\.arrow(\?|$)|\/live|\/assets\/gossamer\//.test(requestURL)) {
      staleRequests.push(requestURL);
    }
  });
  page.on("requestfailed", (request) => {
    console.error("REQUEST FAILED:", request.url(), request.failure()?.errorText);
    failedRequests.push(`${request.url()} ${request.failure()?.errorText}`);
  });

  for (const route of routes) {
    const { name, hash, expectedGraphs, textNeedle, minReadouts } = route;
    await page.goto(url(hash), { waitUntil: "networkidle", timeout: 60000 });
    await page.locator(".shell").waitFor({ state: "visible", timeout: 30000 });
    await page.getByText(textNeedle, { exact: false }).first().waitFor({ timeout: 30000 });

    if (expectedGraphs > 0) {
      await page.locator("canvas").first().waitFor({ timeout: 30000 });
      await page.waitForFunction(
        (minimumGraphs) => document.querySelectorAll("canvas").length >= minimumGraphs,
        expectedGraphs,
        { timeout: 30000 }
      );
    }

    if (minReadouts > 0) {
      // Readouts populate after tiles fetch and the animated replay cursor reaches the data window.
      await page.waitForFunction(
        (minimum) => {
          const values = [...document.querySelectorAll(".graph-card-legend-rail em")]
            .map((el) => (el.textContent ?? "").trim())
            .filter((v) => v && v !== "-");
          return values.length >= minimum;
        },
        minReadouts,
        { timeout: 30000 }
      );
      await assertRequiredSignalReadouts(page, route);
      await assertNoVisibleEventLabelOverlaps(page, name);
    }

    const routeState = await page.evaluate(() => {
      const loadingText = document.body.innerText.match(/Loading [^\n]+/g) ?? [];
      const canvases = [...document.querySelectorAll("canvas")].map((canvas) => {
        const context = canvas.getContext("2d");
        let nonBlankRatio = 0;
        if (context && canvas.width > 0 && canvas.height > 0) {
          const width = Math.min(canvas.width, 240);
          const height = Math.min(canvas.height, 120);
          const sample = context.getImageData(0, 0, width, height).data;
          let nonBlank = 0;
          for (let i = 0; i < sample.length; i += 4) {
            if (sample[i] > 18 || sample[i + 1] > 18 || sample[i + 2] > 18) nonBlank += 1;
          }
          nonBlankRatio = nonBlank / (sample.length / 4);
        }
        const rect = canvas.getBoundingClientRect();
        return { width: canvas.width, height: canvas.height, cssWidth: rect.width, cssHeight: rect.height, nonBlankRatio };
      });
      const largeText = [...document.querySelectorAll(".shell *")]
        .map((element) => {
          const style = getComputedStyle(element);
          const rect = element.getBoundingClientRect();
          return {
            text: (element.textContent ?? "").trim().slice(0, 80),
            fontSize: Number.parseFloat(style.fontSize),
            width: rect.width,
            height: rect.height
          };
        })
        .filter((item) => item.fontSize > 42 && item.height > 20);
      const readouts = [...document.querySelectorAll(".graph-card-legend-rail em")]
        .map((element) => (element.textContent ?? "").trim())
        .filter((value) => value && value !== "-");
      return { loadingText, canvases, largeText, readouts };
    });

    const graphCount = routeState.canvases.length;
    if (graphCount < expectedGraphs) {
      throw new Error(`${name}: expected at least ${expectedGraphs} canvases, got ${graphCount}`);
    }
    if (expectedGraphs > 0 && routeState.loadingText.length > 0) {
      throw new Error(`${name}: still showing loading text: ${routeState.loadingText.join(" | ")}`);
    }
    for (const [index, canvas] of routeState.canvases.entries()) {
      if (canvas.width < 200 || canvas.height < 80 || canvas.nonBlankRatio < 0.01) {
        throw new Error(`${name}: canvas ${index} is too small or blank: ${JSON.stringify(canvas)}`);
      }
    }
    if (name === "command-center" && routeState.largeText.length > 0) {
      throw new Error(`${name}: oversized operator text: ${JSON.stringify(routeState.largeText.slice(0, 4))}`);
    }
    if (routeState.readouts.length < minReadouts) {
      throw new Error(`${name}: expected at least ${minReadouts} populated legend readouts, got ${routeState.readouts.length}`);
    }

    console.log(`${name}: ok graphs=${graphCount} readouts=${routeState.readouts.length}`);
  }

  if (pageErrors.length > 0) {
    throw new Error(`hosted page errors: ${pageErrors.join(" | ")}`);
  }
  if (failedRequests.length > 0) {
    throw new Error(`hosted request failures: ${[...new Set(failedRequests)].join(" | ")}`);
  }
  if (staleRequests.length > 0) {
    throw new Error(`stale hosted requests: ${[...new Set(staleRequests)].join(" | ")}`);
  }

  console.log(`hosted smoke ok; data_version=${manifest.data_version}; url=${baseURL}`);
} finally {
  await browser.close();
}
