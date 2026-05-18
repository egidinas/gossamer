import { chromium } from "playwright";

const baseURL = process.env.GOSSAMER_HOSTED_URL || "https://gossamer.jmeyer.space/";
const expectedDataVersion = process.env.GOSSAMER_EXPECT_DATA_VERSION || "";

const routes = [
  { name: "landing", hash: "#landing", expectedCards: 0, expectedCanvases: 0, textNeedle: "Complete Operator Surfaces", minReadouts: 0 },
  {
    name: "acceptance",
    hash: "#acceptance",
    expectedCards: 9,
    expectedCanvases: 7,
    textNeedle: "Thermal Chamber FAT",
    minReadouts: 10,
    requiredSignals: ["trace.power_total", "trace.power_payload", "trace.dut_temp_a"],
  },
  { name: "command-center", hash: "#command-center-fat", expectedCards: 4, expectedCanvases: 4, textNeedle: "Command Center FAT", minReadouts: 8 },
  {
    name: "qualification",
    hash: "#qualification",
    graphShellPath: "/data/current/campaigns/tvac_qualification/graph-shell.json",
    expectedCards: 12,
    expectedCanvases: 10,
    textNeedle: "TVac Qualification",
    minReadouts: 12,
    requiredSignals: ["trace.power_total", "trace.power_payload", "trace.dut_temp_a", "trace.tvac_pressure_target"],
    requiredTileSignals: ["trace.tvac_pressure"],
  },
];

const forbiddenVisiblePatterns = [
  {
    id: "private-ipv4",
    regex: /\b(?:10\.(?:\d{1,3}\.){2}\d{1,3}|172\.(?:1[6-9]|2\d|3[0-1])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3}|169\.254\.\d{1,3}\.\d{1,3})\b/g,
  },
  {
    id: "credential-assignment",
    regex: /\b(?:api[_-]?key|access[_-]?token|auth(?:entication)?[_-]?token|refresh[_-]?token|bearer[_-]?token|password|passwd|private[_-]?key|secret)\s*[:=]\s*["']?[A-Za-z0-9+/._-]{12,}/gi,
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

async function assertRequiredTileSignals(route) {
  if (!route.requiredTileSignals?.length) return;
  if (!route.graphShellPath) {
    throw new Error(`${route.name}: requiredTileSignals need graphShellPath`);
  }
  const response = await fetch(url(route.graphShellPath));
  if (!response.ok) {
    throw new Error(`${route.name}: graph shell failed: ${response.status} ${response.statusText}`);
  }
  const shell = await response.json();
  const cards = shell.tile_manifest?.cards ?? [];
  const observed = new Set();
  for (const card of cards) {
    for (const signal of card.signals ?? card.traces ?? []) {
      observed.add(signal.signal_id ?? signal.id ?? signal);
    }
  }
  const missing = route.requiredTileSignals.filter((signalID) => !observed.has(signalID));
  if (missing.length > 0) {
    throw new Error(`${route.name}: graph shell missing required tile signals: ${missing.join(", ")}`);
  }
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

async function assertGraphWallStackGeometry(page, routeName) {
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
      const cardRect = rect(card);
      const titleRect = rect(title);
      const titleStyle = getComputedStyle(title);
      if (titleStyle.display === "none" || titleRect.width < 1 || titleRect.height < 1) continue;
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
    throw new Error(`${routeName}: hosted graph wall has vertical card gaps: ${JSON.stringify(geometry.gaps)}`);
  }
  if (geometry.titleProblems.length > 0) {
    throw new Error(`${routeName}: hosted graph title placement problems: ${JSON.stringify(geometry.titleProblems)}`);
  }
}

async function assertNoForbiddenVisibleText(page, routeName) {
  const bodyText = await page.evaluate(() => document.body.innerText);
  const findings = [];
  for (const rule of forbiddenVisiblePatterns) {
    for (const match of bodyText.matchAll(rule.regex)) {
      findings.push({ id: rule.id, match: match[0] });
    }
  }
  if (findings.length > 0) {
    throw new Error(`${routeName}: forbidden visible text: ${JSON.stringify(findings.slice(0, 8))}`);
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
    const { name, hash, expectedCards, expectedCanvases, textNeedle, minReadouts } = route;
    await page.goto(url(hash), { waitUntil: "networkidle", timeout: 60000 });
    await page.locator(".shell").waitFor({ state: "visible", timeout: 30000 });
    await page.getByText(textNeedle, { exact: false }).first().waitFor({ timeout: 30000 });
    await assertNoForbiddenVisibleText(page, name);

    if (expectedCanvases > 0) {
      await page.locator("canvas").first().waitFor({ timeout: 30000 });
      await page.waitForFunction(
        ({ minimumCards, minimumCanvases }) =>
          document.querySelectorAll(".graph-wall-card").length >= minimumCards &&
          document.querySelectorAll("canvas").length >= minimumCanvases,
        { minimumCards: expectedCards, minimumCanvases: expectedCanvases },
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
      await assertRequiredTileSignals(route);
      await assertNoVisibleEventLabelOverlaps(page, name);
      await assertGraphWallStackGeometry(page, name);
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
      const cardKinds = [...document.querySelectorAll(".graph-wall-card")]
        .map((element) => element.getAttribute("data-render-kind") ?? "");
      return { loadingText, canvases, largeText, readouts, cardKinds };
    });

    const cardCount = routeState.cardKinds.length;
    if (cardCount < expectedCards) {
      throw new Error(`${name}: expected at least ${expectedCards} graph cards, got ${cardCount}: ${routeState.cardKinds.join(", ")}`);
    }
    const canvasCount = routeState.canvases.length;
    if (canvasCount < expectedCanvases) {
      throw new Error(`${name}: expected at least ${expectedCanvases} canvases, got ${canvasCount}; cards: ${routeState.cardKinds.join(", ")}`);
    }
    if (expectedCanvases > 0 && routeState.loadingText.length > 0) {
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

    console.log(`${name}: ok cards=${cardCount} canvases=${canvasCount} readouts=${routeState.readouts.length}`);
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
