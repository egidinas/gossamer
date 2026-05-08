import { chromium } from "playwright";

const baseURL = process.env.GOSSAMER_HOSTED_URL || "https://gossamer.jmeyer.space/";
const expectedDataVersion = process.env.GOSSAMER_EXPECT_DATA_VERSION || "";

const routes = [
  ["landing", "#landing", 0, "Complete Operator Surfaces"],
  ["acceptance", "#acceptance", 8, "Acceptance FAT"],
  ["command-center", "#command-center-fat", 4, "Command Center"],
  ["qualification", "#qualification", 11, "Qualification TVac"],
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
    }

    const graphCount = await page.locator("canvas").count();
    if (graphCount < expectedGraphs) {
      throw new Error(`${name}: expected at least ${expectedGraphs} canvases, got ${graphCount}`);
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
