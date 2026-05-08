/**
 * Playwright layout + visual check for gossamer.jmeyer.space
 */
import { chromium } from "playwright";

const BASE = "https://gossamer.jmeyer.space";
const results = [];

function pass(name, detail) {
  results.push({ ok: true, name, detail });
  console.log(`  ✓ ${name}: ${detail}`);
}

function fail(name, detail) {
  results.push({ ok: false, name, detail });
  console.error(`  ✗ ${name}: ${detail}`);
}

async function screenshot(page, label) {
  const path = `/tmp/gossamer-${label}.png`;
  await page.screenshot({ path, fullPage: false });
  console.log(`  📸 ${path}`);
}

async function goto(page, url) {
  await page.goto(url, { waitUntil: "domcontentloaded", timeout: 30000 });
  // wait for React hydration
  await page.waitForFunction(() => document.querySelector(".shell") !== null, { timeout: 8000 }).catch(() => {});
  await page.waitForTimeout(800);
}

async function checkLanding(page) {
  console.log("\n── Landing ──────────────────────────────────────────");
  await goto(page, BASE + "/");
  await screenshot(page, "landing");

  const h1 = await page.$eval("h1", el => el.textContent.trim()).catch(() => null);
  h1 ? pass("h1", h1) : fail("h1", "missing");

  const ctaLinks = await page.$$eval(".hero-actions a", els => els.map(el => el.textContent.trim())).catch(() => []);
  ctaLinks.length >= 3
    ? pass("hero CTAs", ctaLinks.join(" | "))
    : fail("hero CTAs", `found ${ctaLinks.length}, want ≥3`);

  const cards = await page.$$(".landing-experience-card");
  cards.length === 3 ? pass("experience cards", "3 present") : fail("experience cards", `found ${cards.length}`);

  if (cards.length > 0) {
    const box = await cards[0].boundingBox();
    box?.height >= 270
      ? pass("card height", `${Math.round(box.height)}px ≥ 270px`)
      : fail("card height", `${Math.round(box?.height ?? 0)}px, want ≥270px`);
  }

  // Bottom void: shell should not have massive empty space below last child
  const voidCheck = await page.evaluate(() => {
    const shell = document.querySelector(".shell");
    const lastChild = shell?.lastElementChild;
    if (!shell || !lastChild) return null;
    const shellBottom = shell.getBoundingClientRect().bottom;
    const lastBottom = lastChild.getBoundingClientRect().bottom;
    return { shellBottom: Math.round(shellBottom), lastBottom: Math.round(lastBottom), gap: Math.round(shellBottom - lastBottom) };
  });
  if (voidCheck) {
    voidCheck.gap < 80
      ? pass("no bottom void", `gap=${voidCheck.gap}px (shell=${voidCheck.shellBottom} last-child=${voidCheck.lastBottom})`)
      : fail("bottom void", `${voidCheck.gap}px empty below last content`);
  }
}

async function checkGraphWall(page, hash, label) {
  console.log(`\n── ${label} ──────────────────────────────────────────`);
  await goto(page, BASE + "/" + hash);

  // Wait for wall cards to appear
  await page.waitForSelector(".operator-wall-cards", { timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(2000); // tiles start loading

  await screenshot(page, label.toLowerCase().replace(/\s+/g, "-"));

  const wall = await page.$(".operator-graph-wall");
  wall ? pass("wall container", "present") : fail("wall container", "missing");

  const cards = await page.$$(".graph-wall-card");
  cards.length > 0 ? pass("cards", `${cards.length} cards`) : fail("cards", "none found");

  // Card heights
  const cardMetrics = await page.$$eval(
    ".graph-wall-card:not(.graph-card-collapsed)",
    cards => cards.slice(0, 8).map(c => ({
      id: c.dataset.cardId ?? "?",
      priority: c.dataset.cardPriority ?? "?",
      h: Math.round(c.getBoundingClientRect().height),
      w: Math.round(c.getBoundingClientRect().width),
    }))
  ).catch(() => []);

  if (cardMetrics.length > 0) {
    const primary = cardMetrics.find(c => c.priority === "primary");
    const normalCards = cardMetrics.filter(c =>
      !["swimlane", "event_rail"].some(k => c.id.includes("swimlane") || c.id.includes("event"))
    );
    const minH = normalCards.length > 0 ? Math.min(...normalCards.map(c => c.h)) : 0;
    const detail = cardMetrics.map(c => `${c.id}(${c.priority})=${c.h}px`).join(", ");

    minH >= 200
      ? pass("card heights ≥200px", `min normal=${minH}px — ${detail}`)
      : fail("card heights", `min normal=${minH}px too short — ${detail}`);

    if (primary) {
      primary.h >= 280
        ? pass("primary card height", `${primary.h}px ≥ 280px`)
        : fail("primary card height", `${primary.h}px < 280px`);
    }
  }

  // Full-width primary card
  const primaryCard = await page.$(".graph-wall-card[data-card-priority='primary']");
  if (primaryCard) {
    const pBox = await primaryCard.boundingBox();
    const wallBox = await page.$eval(".operator-wall-cards", el => {
      const r = el.getBoundingClientRect();
      return { w: Math.round(r.width) };
    }).catch(() => null);
    if (pBox && wallBox) {
      const ratio = pBox.width / wallBox.w;
      ratio >= 0.94
        ? pass("primary full-width", `${Math.round(pBox.width)}px / ${wallBox.w}px (${(ratio*100).toFixed(0)}%)`)
        : fail("primary full-width", `${Math.round(pBox.width)}px / ${wallBox.w}px (${(ratio*100).toFixed(0)}%)`);
    }
  }

  // 2-column: within any .operator-wall-cards that has ≥2 secondary cards, they should be side-by-side
  const twoColResult = await page.evaluate(() => {
    const grids = Array.from(document.querySelectorAll(".operator-wall-cards"));
    for (const grid of grids) {
      const secCards = Array.from(grid.querySelectorAll(".graph-wall-card[data-card-priority='secondary']:not(.graph-card-collapsed)"));
      if (secCards.length >= 2) {
        const b0 = secCards[0].getBoundingClientRect();
        const b1 = secCards[1].getBoundingClientRect();
        return { y0: Math.round(b0.y), y1: Math.round(b1.y), count: secCards.length };
      }
    }
    return null;
  });
  if (twoColResult) {
    Math.abs(twoColResult.y0 - twoColResult.y1) < 30
      ? pass("2-col layout", `${twoColResult.count} sec cards side-by-side at y≈${twoColResult.y0}`)
      : fail("2-col layout", `cards stacked: y0=${twoColResult.y0} y1=${twoColResult.y1}`);
  } else {
    pass("2-col layout", "no multi-card section (each card in own section — expected for this campaign)");
  }

  // Time axis
  const timeAxis = await page.$(".operator-shared-time-axis");
  timeAxis ? pass("time axis", "present") : fail("time axis", "missing");
}

async function main() {
  const browser = await chromium.launch({ headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();

  // Suppress noisy console output from the page
  page.on("console", () => {});

  try {
    await checkLanding(page);
    await checkGraphWall(page, "#acceptance", "Acceptance FAT");
    await checkGraphWall(page, "#command-center-fat", "Operator Center");
    await checkGraphWall(page, "#qualification", "Qualification TVac");
  } finally {
    await browser.close();
  }

  const passed = results.filter(r => r.ok).length;
  const failed = results.filter(r => !r.ok).length;
  console.log(`\n── Result: ${passed} passed  ${failed} failed  (${results.length} total) ──────────`);
  if (failed > 0) {
    console.log("Failed:");
    results.filter(r => !r.ok).forEach(r => console.log(`  ✗ ${r.name}: ${r.detail}`));
    process.exit(1);
  }
}

main().catch(err => { console.error(err); process.exit(1); });
