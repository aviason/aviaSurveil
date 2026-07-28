#!/usr/bin/env node
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { pathToFileURL } from "node:url";

import { chromium } from "../apps/web/node_modules/playwright/index.mjs";

const repoRoot = resolve(import.meta.dirname, "..");
const ledgerPath = resolve(repoRoot, "docs/demo-evidence/REACT_86_SCREEN_VISUAL_REVIEW_2026-07-25.md");
const baselineRoot = resolve(repoRoot, "apps/web/tests/visual-baselines/react-legacy-parity");
const framesDir = resolve(repoRoot, ".local/aviasurveil360/visual-review/frames");
const outputDir = resolve(repoRoot, ".local/aviasurveil360/visual-review/triage-contact-sheets");
const viewports = new Set(["desktop", "tablet", "mobile"]);
const pairsPerSheet = 10;

mkdirSync(outputDir, { recursive: true });

function decodeHtmlBreaks(value) {
  return value.replaceAll("<br>", "\n");
}

function parseFailedRegions(pixelCell) {
  const failed = [];
  const normalized = decodeHtmlBreaks(pixelCell);
  const pattern = /([a-z-]+)\s+([0-9.]+)\/([0-9.]+)\s+fail/g;
  let match;
  while ((match = pattern.exec(normalized))) {
    failed.push(`${match[1]} ${match[2]}/${match[3]}`);
  }
  return failed;
}

function parseLedgerPairs() {
  const source = readFileSync(ledgerPath, "utf8");
  const pairs = [];
  for (const line of source.split("\n")) {
    if (!line.startsWith("ui-audit-")) continue;
    const cells = line.split("|").map((cell) => cell.trim());
    if (cells.length < 10) continue;
    const [auditId, surfaceCell, viewport, pixelCell] = cells;
    if (!viewports.has(viewport)) continue;
    const failedRegions = parseFailedRegions(pixelCell);
    if (failedRegions.length === 0) continue;
    const surface = surfaceCell.replaceAll("`", "");
    pairs.push({
      sequence: pairs.length + 1,
      auditId,
      surface,
      viewport,
      failedRegionAndRatio: failedRegions.join("; "),
      baselinePath: resolve(baselineRoot, viewport, `${surface}.png`),
      candidatePath: resolve(framesDir, `${surface}-${viewport}-react-candidate-viewport.png`),
    });
  }
  return pairs;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function imageUrl(path) {
  return pathToFileURL(path).href;
}

function sheetHtml(pairs, sheetNumber) {
  const rows = pairs
    .map(
      (pair) => `
        <section class="pair">
          <header>
            <strong>${pair.sequence}/170 · ${escapeHtml(pair.surface)} · ${pair.viewport}</strong>
            <span>${escapeHtml(pair.auditId)} · ${escapeHtml(pair.failedRegionAndRatio)}</span>
          </header>
          <div class="images">
            <figure>
              <figcaption>Legacy baseline</figcaption>
              <img src="${imageUrl(pair.baselinePath)}" alt="Legacy ${escapeHtml(pair.surface)} ${pair.viewport}">
            </figure>
            <figure>
              <figcaption>React candidate</figcaption>
              <img src="${imageUrl(pair.candidatePath)}" alt="React ${escapeHtml(pair.surface)} ${pair.viewport}">
            </figure>
          </div>
        </section>`,
    )
    .join("\n");
  return `<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <title>Plan 1 Visual Triage Sheet ${sheetNumber}</title>
    <style>
      body {
        margin: 0;
        background: #eef2f7;
        color: #172033;
        font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      }
      .sheet {
        padding: 18px;
      }
      h1 {
        margin: 0 0 14px;
        font-size: 22px;
        letter-spacing: 0;
      }
      .pair {
        background: #fff;
        border: 1px solid #cfd9e8;
        border-radius: 8px;
        margin-bottom: 16px;
        overflow: hidden;
      }
      header {
        display: flex;
        justify-content: space-between;
        gap: 16px;
        padding: 9px 12px;
        border-bottom: 1px solid #d8e0eb;
        font-size: 13px;
      }
      header span {
        color: #607089;
      }
      .images {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 1px;
        background: #cfd9e8;
      }
      figure {
        margin: 0;
        background: #d8e0eb;
      }
      figcaption {
        padding: 7px 10px;
        background: #f8fafc;
        border-bottom: 1px solid #d8e0eb;
        color: #42526a;
        font-size: 12px;
        font-weight: 700;
      }
      img {
        display: block;
        width: 100%;
        height: 330px;
        object-fit: contain;
        object-position: top center;
        background: #d8e0eb;
      }
    </style>
  </head>
  <body>
    <main class="sheet">
      <h1>Plan 1 Visual Triage · Sheet ${sheetNumber}</h1>
      ${rows}
    </main>
  </body>
</html>`;
}

const pairs = parseLedgerPairs();
if (pairs.length !== 170) {
  throw new Error(`Expected 170 failed visual pairs, found ${pairs.length}.`);
}

const htmlPaths = [];
for (let index = 0; index < pairs.length; index += pairsPerSheet) {
  const sheetNumber = Math.floor(index / pairsPerSheet) + 1;
  const chunk = pairs.slice(index, index + pairsPerSheet);
  const htmlPath = resolve(outputDir, `sheet-${String(sheetNumber).padStart(3, "0")}.html`);
  writeFileSync(htmlPath, sheetHtml(chunk, sheetNumber));
  htmlPaths.push(htmlPath);
}

const browser = await chromium.launch({ headless: true });
try {
  const page = await browser.newPage({ viewport: { width: 1800, height: 1200 }, deviceScaleFactor: 1 });
  for (const htmlPath of htmlPaths) {
    await page.goto(pathToFileURL(htmlPath).href, { waitUntil: "load" });
    await page.evaluate(async () => {
      const images = Array.from(document.images);
      await Promise.all(
        images.map((image) => {
          if (image.complete) return Promise.resolve();
          return new Promise((resolveImage, rejectImage) => {
            image.addEventListener("load", resolveImage, { once: true });
            image.addEventListener("error", rejectImage, { once: true });
          });
        }),
      );
    });
    const pngPath = htmlPath.replace(/\.html$/, ".png");
    await page.screenshot({ path: pngPath, fullPage: true });
  }
} finally {
  await browser.close();
}

console.log(
  JSON.stringify(
    {
      outputDir,
      sheets: htmlPaths.length,
      firstSheet: htmlPaths[0]?.replace(/\.html$/, ".png"),
      lastSheet: htmlPaths.at(-1)?.replace(/\.html$/, ".png"),
    },
    null,
    2,
  ),
);
