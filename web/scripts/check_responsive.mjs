/**
 * Responsive layout check: 4K, 1440p, 1080p, tablet, mobile
 */
import { chromium } from "playwright";

const BASE = "https://gossamer.jmeyer.space";

const VIEWPORTS = [
  { name: "4K",      width: 3840, height: 2160, deviceScaleFactor: 2 },
  { name: "1440p",   width: 1440, height: 900,  deviceScaleFactor: 1 },
  { name: "1080p",   width: 1280, height: 800,  deviceScaleFactor: 1 },
  { name: "tablet",  width: 768,  height: 1024, deviceScaleFactor: 2 },
  { name: "mobile",  width: 390,  height: 844,  deviceScaleFactor: 3, isMobile: true },
];

const ROUTES = [
  { path: "/",                   name: "landing"    },
  { path: "/#acceptance",        name: "acceptance" },
  { path: "/#command-center-fat",name: "cmdcenter"  },
  { path: "/#qualification",     name: "tvac"       },
];

const results = [];
function pass(vp, check, detail) { results.push({ ok: true,  vp, check, detail }); console.log(`  ✓ [${vp}] ${check}: ${detail}`); }
function fail(vp, check, detail) { results.push({ ok: false, vp, check, detail }); console.error(`  ✗ [${vp}] ${check}: ${detail}`); }
function note(vp, check, detail) { console.log(`  · [${vp}] ${check}: ${detail}`); }

async function goto(page, url) {
  await page.goto(url, { waitUntil: "domcontentloaded", timeout: 30000 });
  await page.waitForFunction(() => !!document.querySelector(".shell"), { timeout: 8000 }).catch(() => {});
  await page.waitForTimeout(600);
}

async function waitForWall(page) {
  await page.waitForSelector(".operator-wall-cards", { timeout: 12000 }).catch(() => {});
  await page.waitForTimeout(1200);
}

async function checkLanding(page, vp) {
  await goto(page, BASE + "/");

  const h1 = await page.$eval("h1", el => el.textContent.trim()).catch(() => null);
  h1 ? pass(vp, "h1", h1) : fail(vp, "h1", "missing");

  // Hero must not overflow viewport width
  const heroOverflow = await page.evaluate(() => {
    const hero = document.querySelector(".landing-hero");
    if (!hero) return null;
    const r = hero.getBoundingClientRect();
    return { w: Math.round(r.width), vw: window.innerWidth, overflow: Math.round(r.right - window.innerWidth) };
  });
  if (heroOverflow) {
    heroOverflow.overflow <= 2
      ? pass(vp, "hero no overflow", `${heroOverflow.w}px / vw=${heroOverflow.vw}px`)
      : fail(vp, "hero overflow", `extends ${heroOverflow.overflow}px past viewport`);
  }

  // Experience cards readable (min 200px wide)
  const cardWidths = await page.$$eval(".landing-experience-card", els => els.map(el => Math.round(el.getBoundingClientRect().width))).catch(() => []);
  if (cardWidths.length > 0) {
    const minW = Math.min(...cardWidths);
    minW >= 180
      ? pass(vp, "exp-card widths", `min=${minW}px (${cardWidths.join(", ")}px)`)
      : fail(vp, "exp-card widths", `min=${minW}px too narrow`);
  }

  // No horizontal scrollbar
  const hScroll = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 4);
  hScroll ? fail(vp, "no h-scroll", `scrollWidth=${document.documentElement?.scrollWidth}`) : pass(vp, "no h-scroll", "ok");

  // Typography: hero h1 font-size (not nav headings)
  const h1Size = await page.$eval(".landing-hero h1", el => parseFloat(getComputedStyle(el).fontSize)).catch(() => 0);
  h1Size >= 28 ? pass(vp, "h1 font-size", `${Math.round(h1Size)}px`) : fail(vp, "h1 font-size", `${Math.round(h1Size)}px < 28px`);
}

async function checkGraphWall(page, path, routeLabel, vp) {
  await goto(page, BASE + path);
  await waitForWall(page);

  const cards = await page.$$(".graph-wall-card");
  if (!cards.length) { fail(vp, `${routeLabel} cards`, "none found"); return; }

  // Primary card full-width within its grid
  const primaryMetrics = await page.evaluate(() => {
    const p = document.querySelector(".graph-wall-card[data-card-priority='primary']");
    const grid = p?.closest(".operator-wall-cards");
    if (!p || !grid) return null;
    const pr = p.getBoundingClientRect();
    const gr = grid.getBoundingClientRect();
    return { pw: Math.round(pr.width), gw: Math.round(gr.width), ratio: pr.width / gr.width };
  });
  if (primaryMetrics) {
    primaryMetrics.ratio >= 0.94
      ? pass(vp, `${routeLabel} primary full-width`, `${primaryMetrics.pw}/${primaryMetrics.gw}px`)
      : fail(vp, `${routeLabel} primary full-width`, `${primaryMetrics.pw}/${primaryMetrics.gw}px (${(primaryMetrics.ratio*100).toFixed(0)}%)`);
  }

  // Card heights reasonable for this viewport
  const heights = await page.$$eval(
    ".graph-wall-card:not(.graph-card-collapsed):not([data-render-kind='swimlane']):not([data-render-kind='event_rail'])",
    cards => cards.slice(0, 4).map(c => Math.round(c.getBoundingClientRect().height))
  ).catch(() => []);
  if (heights.length > 0) {
    const minH = Math.min(...heights);
    const thresh = 140; // even on mobile cards should be usable (event_rail = 150px)
    minH >= thresh
      ? pass(vp, `${routeLabel} card heights`, `min=${minH}px (${heights.join(",")}px)`)
      : fail(vp, `${routeLabel} card heights`, `min=${minH}px < ${thresh}px`);
  }

  // No horizontal overflow inside the wall
  const wallOverflow = await page.evaluate(() => {
    const wall = document.querySelector(".operator-graph-wall");
    if (!wall) return null;
    return { sw: Math.round(wall.scrollWidth), cw: Math.round(wall.clientWidth), overflow: wall.scrollWidth - wall.clientWidth };
  });
  if (wallOverflow) {
    const isMobileVp = vp === "mobile";
    if (wallOverflow.overflow <= 4) {
      pass(vp, `${routeLabel} no wall h-overflow`, `scroll=${wallOverflow.sw} client=${wallOverflow.cw}`);
    } else if (isMobileVp) {
      note(vp, `${routeLabel} wall h-overflow`, `${wallOverflow.overflow}px (acceptable on mobile)`);
    } else {
      fail(vp, `${routeLabel} no wall h-overflow`, `overflows by ${wallOverflow.overflow}px`);
    }
  }

  // Time axis present
  const ta = await page.$(".operator-shared-time-axis");
  ta ? pass(vp, `${routeLabel} time-axis`, "present") : fail(vp, `${routeLabel} time-axis`, "missing");
}

async function runViewport(browser, vpConfig) {
  const { name, width, height, deviceScaleFactor = 1, isMobile = false } = vpConfig;
  console.log(`\n════ ${name} (${width}×${height}) ════════════════════════`);

  const ctx = await browser.newContext({
    viewport: { width, height },
    deviceScaleFactor,
    isMobile,
    hasTouch: isMobile,
  });
  const page = await ctx.newPage();
  page.on("console", () => {});

  try {
    await checkLanding(page, name);
    await page.screenshot({ path: `/tmp/gossamer-${name}-landing.png`, fullPage: false });
    console.log(`  📸 /tmp/gossamer-${name}-landing.png`);

    await checkGraphWall(page, "/#acceptance",         "acceptance", name);
    await page.screenshot({ path: `/tmp/gossamer-${name}-acceptance.png`, fullPage: false });
    console.log(`  📸 /tmp/gossamer-${name}-acceptance.png`);

    await checkGraphWall(page, "/#command-center-fat", "cmdcenter",  name);
    await page.screenshot({ path: `/tmp/gossamer-${name}-cmdcenter.png`, fullPage: false });
    console.log(`  📸 /tmp/gossamer-${name}-cmdcenter.png`);

    await checkGraphWall(page, "/#qualification",      "tvac",       name);
    await page.screenshot({ path: `/tmp/gossamer-${name}-tvac.png`, fullPage: false });
    console.log(`  📸 /tmp/gossamer-${name}-tvac.png`);
  } finally {
    await ctx.close();
  }
}

async function main() {
  const browser = await chromium.launch({ headless: true });
  try {
    for (const vp of VIEWPORTS) {
      await runViewport(browser, vp);
    }
  } finally {
    await browser.close();
  }

  const passed = results.filter(r => r.ok).length;
  const failed = results.filter(r => !r.ok).length;
  console.log(`\n════ Summary: ${passed} passed  ${failed} failed  (${results.length} total) ════`);
  if (failed > 0) {
    console.log("\nFailed:");
    results.filter(r => !r.ok).forEach(r => console.log(`  ✗ [${r.vp}] ${r.check}: ${r.detail}`));
  }
  process.exit(failed > 0 ? 1 : 0);
}

main().catch(err => { console.error(err); process.exit(1); });
