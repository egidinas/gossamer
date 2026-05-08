import { chromium } from "playwright";

const baseURL = process.env.GOSSAMER_HOSTED_URL || "https://gossamer.jmeyer.space/";
const expectedDataVersion = process.env.GOSSAMER_EXPECT_DATA_VERSION || "";

const routes = [
  ["landing", "#landing", 0, "Complete Operator Surfaces"],
  ["acceptance", "#acceptance", 8, "Thermal Chamber FAT -"],
  ["command-center", "#command-center-fat", 4, "Command Center FAT"],
  ["qualification", "#qualification", 12, "TVac Qualification -"],
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
    failedRequests.push(`${request.url()} ${request.failure()?.errorText}`);
  });

  for (const [name, hash, expectedGraphs, textNeedle] of routes) {
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
      return { loadingText, canvases, largeText };
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

    console.log(`${name}: ok graphs=${graphCount}`);
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
