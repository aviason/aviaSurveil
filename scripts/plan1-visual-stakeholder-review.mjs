#!/usr/bin/env node
import { createServer } from "node:http";
import { spawn } from "node:child_process";
import {
  copyFileSync,
  createReadStream,
  existsSync,
  mkdirSync,
  readFileSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { basename, dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "..");
const reviewRoot = resolve(repoRoot, ".local/aviasurveil360/visual-review");
const framesDir = resolve(reviewRoot, "frames");
const ledgerPath = resolve(repoRoot, "docs/demo-evidence/REACT_86_SCREEN_VISUAL_REVIEW_2026-07-25.md");
const decisionJsonPath = resolve(
  repoRoot,
  "docs/demo-evidence/stakeholder/PLAN1_VISUAL_STAKEHOLDER_DECISIONS_2026-07-27.json",
);
const decisionMarkdownPath = resolve(
  repoRoot,
  "docs/demo-evidence/stakeholder/PLAN1_VISUAL_STAKEHOLDER_DECISIONS_2026-07-27.md",
);
const triageJsonPath = resolve(
  repoRoot,
  "docs/demo-evidence/stakeholder/PLAN1_VISUAL_CODEX_TRIAGE_2026-07-27.json",
);
const triageMarkdownPath = resolve(
  repoRoot,
  "docs/demo-evidence/stakeholder/PLAN1_VISUAL_CODEX_TRIAGE_2026-07-27.md",
);
const baselineRoot = resolve(repoRoot, "apps/web/tests/visual-baselines/react-legacy-parity");
const viewports = new Set(["desktop", "tablet", "mobile"]);
const npmBinaryCandidates = [
  process.env.NPM_BINARY,
  "/private/tmp/node-v24.16.0-darwin-arm64/bin/npm",
  "/private/tmp/aviasurveil360-node/bin/npm",
  "npm",
].filter(Boolean);
const nodePathPrefix = [
  "/private/tmp/aviasurveil360-node/bin",
  "/private/tmp/node-v24.16.0-darwin-arm64/bin",
  process.env.PATH ?? "",
].join(":");
const statusByDecision = {
  fix: "pending-fix",
  accept: "accepted",
  defer: "deferred",
};
const markdownDecisionByValue = {
  fix: "Fix",
  accept: "Accept",
  defer: "Defer",
};
const valueByMarkdownDecision = Object.fromEntries(
  Object.entries(markdownDecisionByValue).map(([value, label]) => [label.toLowerCase(), value]),
);
const defaultRationaleByDecision = {
  fix: "Stakeholder requested correction for this visual deviation before Plan 1 sign-off.",
  accept: "Stakeholder accepted this visual deviation for Plan 1 sign-off.",
  defer: "Stakeholder deferred this visual decision for later review.",
};
const codexAutoDisposition = {
  authorizedByUser: true,
  authorizationDate: "2026-07-28",
  decision: "accept",
  scope: "Codex triage records with codexRecommendation=accept and userReviewRequired=false",
  source: "docs/demo-evidence/stakeholder/PLAN1_VISUAL_CODEX_TRIAGE_2026-07-27.json",
};

mkdirSync(framesDir, { recursive: true });

function decodeHtmlBreaks(value) {
  return value.replaceAll("<br>", "\n");
}

function parseFailedRegions(pixelCell) {
  const failed = [];
  const normalized = decodeHtmlBreaks(pixelCell);
  const pattern = /([a-z-]+)\s+([0-9.]+)\/([0-9.]+)\s+fail/g;
  let match;
  while ((match = pattern.exec(normalized))) {
    failed.push({
      region: match[1],
      ratio: match[2],
      threshold: match[3],
      label: `${match[1]} ${match[2]}/${match[3]}`,
    });
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
    const sequence = pairs.length + 1;
    pairs.push({
      sequence,
      auditId,
      surface,
      viewport,
      failedRegions,
      failedRegionAndRatio: failedRegions.map((region) => region.label).join("; "),
      baselinePath: resolve(baselineRoot, viewport, `${surface}.png`),
      candidatePath: resolve(framesDir, `${surface}-${viewport}-react-candidate-viewport.png`),
      decodedPath: resolve(framesDir, `${surface}-${viewport}-decoded-pixel-region-results.json`),
    });
  }
  return pairs;
}

function loadDecisionData() {
  if (!existsSync(decisionJsonPath)) {
    const markdownDecisions = parseExistingMarkdownDecisions();
    return {
      metadata: {
        plan: "Full React 86-Screen Migration",
        decisionDateStarted: "2026-07-27",
        sourceLedger: "docs/demo-evidence/REACT_86_SCREEN_VISUAL_REVIEW_2026-07-25.md",
        total: 170,
        reviewed: markdownDecisions.length,
      },
      decisions: markdownDecisions,
    };
  }
  return JSON.parse(readFileSync(decisionJsonPath, "utf8"));
}

function loadTriageData() {
  if (!existsSync(triageJsonPath)) {
    return {
      metadata: {
        total: 170,
        reviewed: 0,
        userReviewRequired: 0,
        codexAccept: 0,
        codexFix: 0,
      },
      records: [],
    };
  }
  return JSON.parse(readFileSync(triageJsonPath, "utf8"));
}

function parseExistingMarkdownDecisions() {
  if (!existsSync(decisionMarkdownPath)) return [];
  const decisions = [];
  for (const line of readFileSync(decisionMarkdownPath, "utf8").split("\n")) {
    if (!/^\d+\s+\|/.test(line)) continue;
    const cells = line.split("|").map((cell) => cell.trim());
    if (cells.length < 9) continue;
    const decision = valueByMarkdownDecision[cells[5].toLowerCase()] ?? cells[5].toLowerCase();
    decisions.push({
      sequence: Number(cells[0]),
      auditId: cells[1],
      surface: cells[2].replaceAll("`", ""),
      viewport: cells[3],
      failedRegionAndRatio: cells[4],
      decision,
      decisionLabel: markdownDecisionByValue[decision] ?? cells[5],
      rationale: cells[6],
      decisionDate: cells[7],
      implementationStatus: cells[8],
    });
  }
  return decisions.filter((decision) => Number.isFinite(decision.sequence));
}

function sortDecisions(decisions) {
  return [...decisions].sort((a, b) => a.sequence - b.sequence);
}

function dispositionSummary(decisions, triageData = loadTriageData()) {
  const decisionsBySequence = new Map(decisions.map((decision) => [decision.sequence, decision]));
  const records = triageData.records ?? [];
  const codexAutoAcceptedSequences = new Set(
    records
      .filter((record) => record.codexRecommendation === "accept" && !record.userReviewRequired)
      .map((record) => record.sequence),
  );
  const explicitlyResolvedSequences = new Set(
    decisions
      .filter(
        (decision) =>
          decision.decision !== "fix" || decision.implementationStatus === "fixed-verified-locally",
      )
      .map((decision) => decision.sequence),
  );
  const resolvedSequences = new Set([
    ...codexAutoAcceptedSequences,
    ...explicitlyResolvedSequences,
  ]);
  const completedFixes = decisions.filter(
    (decision) => decision.decision === "fix" && decision.implementationStatus === "fixed-verified-locally",
  ).length;
  const manualRecords = records.filter((record) => record.userReviewRequired);
  const manualResolved = manualRecords.filter((record) => resolvedSequences.has(record.sequence)).length;
  return {
    codexAutoAccepted: codexAutoAcceptedSequences.size,
    completedFixes,
    manualReviewTotal: manualRecords.length,
    manualReviewRemaining: manualRecords.length - manualResolved,
    resolved: resolvedSequences.size,
  };
}

function saveDecisionData(data) {
  const decisions = sortDecisions(data.decisions);
  const summary = dispositionSummary(decisions);
  const nextData = {
    metadata: {
      ...data.metadata,
      total: 170,
      reviewed: decisions.length,
      resolved: summary.resolved,
      manualReviewRemaining: summary.manualReviewRemaining,
      codexAutoDisposition: {
        ...codexAutoDisposition,
        count: summary.codexAutoAccepted,
      },
      updatedAt: new Date().toISOString(),
    },
    decisions,
  };
  writeFileSync(decisionJsonPath, `${JSON.stringify(nextData, null, 2)}\n`);
  writeDecisionMarkdown(nextData);
  return nextData;
}

function markdownEscape(value) {
  return String(value).replaceAll("|", "\\|").replaceAll("\n", " ");
}

function writeDecisionMarkdown(data) {
  const triageData = loadTriageData();
  const summary = dispositionSummary(data.decisions, triageData);
  const rows = sortDecisions(data.decisions)
    .map((decision) => {
      const value = [
        decision.sequence,
        decision.auditId,
        `\`${decision.surface}\``,
        decision.viewport,
        markdownEscape(decision.failedRegionAndRatio),
        markdownDecisionByValue[decision.decision] ?? decision.decision,
        markdownEscape(decision.rationale),
        decision.decisionDate,
        decision.implementationStatus,
      ];
      return value.join(" | ");
    })
    .join("\n");
  const resolvedDecisionSequences = new Set(
    data.decisions
      .filter(
        (decision) =>
          decision.decision !== "fix" || decision.implementationStatus === "fixed-verified-locally",
      )
      .map((decision) => decision.sequence),
  );
  const manualRows = (triageData.records ?? [])
    .filter((record) => record.userReviewRequired && !resolvedDecisionSequences.has(record.sequence))
    .map((record) =>
      [
        record.sequence,
        record.auditId,
        `\`${record.surface}\``,
        record.viewport,
        markdownEscape(record.failedRegionAndRatio),
        markdownEscape(record.rationale),
      ].join(" | "),
    )
    .join("\n");
  const body = `# Plan 1 Visual Stakeholder Decisions

**Decision date started:** 27 July 2026

**Plan:** [Full React 86-Screen Migration](../../exec-plans/completed/2026-07-22-full-react-86-screen-migration-plan.md)

**Source ledger:** [React 86-Screen Visual Review Ledger](../REACT_86_SCREEN_VISUAL_REVIEW_2026-07-25.md)

**Scope:** The 170 route/viewport pairs that retained decoded-pixel failures in
the final exact-runtime visual command.

**Disposition progress:** ${summary.resolved}/170 resolved

**Manual review remaining:** ${summary.manualReviewRemaining}/170

On 28 July 2026, the user directed the Plan 1 closure to apply Codex's
high-confidence recommendations and leave only the records explicitly marked
as requiring manual stakeholder review. This authorizes an aggregate Accept
disposition for ${summary.codexAutoAccepted} records whose triage has
\`codexRecommendation=accept\` and \`userReviewRequired=false\`.

The user then directed the remaining nine risk-surface records to be fixed:
make the Manager Risk Dashboard heatmap more visible, show the Organization
Risk Profile health score, and preserve the GM Risk Dashboard risk matrix.
All nine implementations are \`fixed-verified-locally\`. The focused visual
command remains literally non-green for the nine route/viewport pixel
comparisons; no baseline, threshold, mask, authority, or semantic truth was
weakened.

## Decision Records

Sequence | Audit ID | Surface | Viewport | Failed region and ratio | User decision | Brief rationale | Decision date | Implementation status
---|---|---|---|---|---|---|---|---
${rows}

## Manual Decisions Remaining

Sequence | Audit ID | Surface | Viewport | Failed region and ratio | Decision question
---|---|---|---|---|---
${manualRows}
`;
  writeFileSync(decisionMarkdownPath, body);
}

function enrichPairs(pairs, data, triageData = loadTriageData()) {
  const decisionsBySequence = new Map(data.decisions.map((decision) => [decision.sequence, decision]));
  const triageBySequence = new Map((triageData.records ?? []).map((record) => [record.sequence, record]));
  return pairs.map((pair) => ({
    ...pair,
    baselinePathLabel: pair.baselinePath,
    candidatePathLabel: pair.candidatePath,
    baselineExists: existsSync(pair.baselinePath),
    candidateExists: existsSync(pair.candidatePath),
    decodedExists: existsSync(pair.decodedPath),
    decision: decisionsBySequence.get(pair.sequence) ?? null,
    triage: triageBySequence.get(pair.sequence) ?? null,
  }));
}

function getPair(sequence) {
  const pair = parseLedgerPairs().find((item) => item.sequence === sequence);
  if (!pair) throw Object.assign(new Error(`Unknown pair sequence ${sequence}`), { status: 404 });
  return pair;
}

function ensureAllowedImagePath(path) {
  const absolute = resolve(path);
  const allowedRoots = [baselineRoot, framesDir];
  if (!allowedRoots.some((root) => absolute.startsWith(`${root}/`) || absolute === root)) {
    throw Object.assign(new Error("Image path is outside the visual review allowlist."), { status: 403 });
  }
  if (!existsSync(absolute)) {
    throw Object.assign(new Error(`Image does not exist: ${absolute}`), { status: 404 });
  }
  return absolute;
}

function collectSpecs(node, specs = []) {
  if (!node || typeof node !== "object") return specs;
  if (Array.isArray(node.specs)) specs.push(...node.specs);
  if (Array.isArray(node.suites)) {
    for (const suite of node.suites) collectSpecs(suite, specs);
  }
  return specs;
}

function attachmentBuffer(attachment) {
  if (attachment.body) return Buffer.from(attachment.body, "base64");
  if (attachment.path && existsSync(attachment.path)) return readFileSync(attachment.path);
  return null;
}

function extractFramesFromReport(surface, reportPath) {
  const report = JSON.parse(readFileSync(reportPath, "utf8"));
  const extracted = [];
  for (const spec of collectSpecs(report)) {
    const match = new RegExp(`for ${surface} at (desktop|tablet|mobile)$`).exec(spec.title ?? "");
    if (!match) continue;
    const viewport = match[1];
    for (const testCase of spec.tests ?? []) {
      for (const result of testCase.results ?? []) {
        for (const attachment of result.attachments ?? []) {
          const body = attachmentBuffer(attachment);
          if (!body) continue;
          if (attachment.name === "react-candidate-viewport") {
            const outputPath = resolve(framesDir, `${surface}-${viewport}-react-candidate-viewport.png`);
            writeFileSync(outputPath, body);
            extracted.push(outputPath);
          }
          if (attachment.name === "decoded-pixel-region-results") {
            const outputPath = resolve(framesDir, `${surface}-${viewport}-decoded-pixel-region-results.json`);
            writeFileSync(outputPath, `${body.toString("utf8")}\n`);
            extracted.push(outputPath);
          }
        }
      }
    }
  }
  return extracted;
}

function runVisualSurface(surface) {
  return new Promise((resolveRun) => {
    const outputDir = resolve(reviewRoot, `${surface}-json`);
    const reportPath = resolve(reviewRoot, `${surface}-report.json`);
    mkdirSync(outputDir, { recursive: true });
    const npmBinary = npmBinaryCandidates.find((candidate) => candidate === "npm" || existsSync(candidate));
    const child = spawn(
      npmBinary,
      ["--prefix", "apps/web", "run", "test:e2e:visual-parity", "--", "--reporter=json"],
      {
        cwd: repoRoot,
        env: {
          ...process.env,
          PATH: nodePathPrefix,
          AVIA_VISUAL_SURFACES: surface,
          AVIA_PLAYWRIGHT_OUTPUT_DIR: outputDir,
          PLAYWRIGHT_JSON_OUTPUT_NAME: reportPath,
        },
      },
    );
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk) => {
      stdout += chunk.toString();
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString();
    });
    child.on("close", (exitCode) => {
      let extracted = [];
      let extractionError = null;
      if (existsSync(reportPath)) {
        try {
          extracted = extractFramesFromReport(surface, reportPath);
        } catch (error) {
          extractionError = error instanceof Error ? error.message : String(error);
        }
      }
      resolveRun({
        exitCode,
        reportPath,
        outputDir,
        extracted,
        extractionError,
        stdout: stdout.slice(-4000),
        stderr: stderr.slice(-4000),
      });
    });
  });
}

function sendJson(response, body, status = 200) {
  response.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Cache-Control": "no-store",
  });
  response.end(`${JSON.stringify(body, null, 2)}\n`);
}

function sendText(response, body, status = 200, contentType = "text/plain; charset=utf-8") {
  response.writeHead(status, {
    "Content-Type": contentType,
    "Cache-Control": "no-store",
  });
  response.end(body);
}

function readRequestJson(request) {
  return new Promise((resolveBody, rejectBody) => {
    let body = "";
    request.on("data", (chunk) => {
      body += chunk.toString();
      if (body.length > 1_000_000) {
        rejectBody(Object.assign(new Error("Request body too large."), { status: 413 }));
      }
    });
    request.on("end", () => {
      try {
        resolveBody(body ? JSON.parse(body) : {});
      } catch (error) {
        rejectBody(Object.assign(error, { status: 400 }));
      }
    });
  });
}

function html() {
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Plan 1 Visual Stakeholder Review</title>
    <style>
      :root {
        color-scheme: light;
        --bg: #eef2f7;
        --panel: #ffffff;
        --panel-soft: #f8fafc;
        --ink: #172033;
        --muted: #607089;
        --line: #cfd9e8;
        --blue: #2563eb;
        --green: #16965f;
        --red: #d94141;
        --amber: #b7791f;
        --shadow: 0 18px 44px rgba(23, 32, 51, 0.12);
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        background: var(--bg);
        color: var(--ink);
        font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
        line-height: 1.45;
      }
      button, input, textarea {
        font: inherit;
      }
      .app {
        display: grid;
        grid-template-columns: 300px minmax(0, 1fr);
        min-height: 100vh;
      }
      .rail {
        background: #0b1b2f;
        color: #e7edf7;
        padding: 18px;
        display: flex;
        flex-direction: column;
        gap: 16px;
        min-height: 100vh;
      }
      .brand h1 {
        font-size: 18px;
        margin: 0;
        letter-spacing: 0;
      }
      .brand p {
        margin: 4px 0 0;
        color: #a9b7cb;
        font-size: 13px;
      }
      .progress-card {
        background: rgba(255, 255, 255, 0.08);
        border: 1px solid rgba(255, 255, 255, 0.14);
        border-radius: 8px;
        padding: 12px;
      }
      .progress-line {
        height: 8px;
        border-radius: 999px;
        background: rgba(255, 255, 255, 0.16);
        overflow: hidden;
        margin: 10px 0;
      }
      .progress-line span {
        display: block;
        height: 100%;
        width: 0%;
        background: #60a5fa;
      }
      .progress-detail {
        color: #a9b7cb;
        font-size: 12px;
        margin: 6px 0 0;
      }
      .link-stack {
        display: grid;
        gap: 8px;
        margin-top: 10px;
      }
      .filters {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 8px;
      }
      .filters button,
      .utility-link {
        min-height: 38px;
        border-radius: 8px;
        border: 1px solid rgba(255, 255, 255, 0.16);
        background: rgba(255, 255, 255, 0.08);
        color: #e7edf7;
        cursor: pointer;
      }
      .filters button.active {
        border-color: #93c5fd;
        background: rgba(96, 165, 250, 0.22);
      }
      .pair-list {
        overflow: auto;
        display: flex;
        flex-direction: column;
        gap: 8px;
        padding-right: 4px;
      }
      .pair-item {
        border: 1px solid rgba(255, 255, 255, 0.12);
        border-radius: 8px;
        background: rgba(255, 255, 255, 0.06);
        color: #e7edf7;
        cursor: pointer;
        padding: 10px;
        text-align: left;
      }
      .pair-item.active {
        border-color: #93c5fd;
        background: rgba(37, 99, 235, 0.34);
      }
      .pair-item.done {
        opacity: 0.68;
      }
      .pair-item strong {
        display: block;
        font-size: 13px;
      }
      .pair-item span {
        display: block;
        color: #afbdd1;
        font-size: 12px;
        margin-top: 3px;
      }
      .main {
        min-width: 0;
        padding: 18px;
        display: grid;
        grid-template-rows: auto minmax(0, 1fr);
        gap: 14px;
      }
      .top {
        background: var(--panel);
        border: 1px solid var(--line);
        border-radius: 8px;
        box-shadow: var(--shadow);
        padding: 14px;
        display: grid;
        grid-template-columns: minmax(0, 1fr) auto;
        gap: 16px;
        align-items: center;
      }
      .meta {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        align-items: center;
      }
      .pill {
        border: 1px solid var(--line);
        border-radius: 999px;
        background: var(--panel-soft);
        padding: 5px 9px;
        color: #324158;
        font-size: 12px;
        font-weight: 700;
      }
      .title {
        margin: 0 0 8px;
        font-size: 22px;
        letter-spacing: 0;
      }
      .actions {
        display: flex;
        gap: 8px;
        align-items: center;
      }
      .decision-button,
      .nav-button,
      .generate-button {
        min-height: 42px;
        border-radius: 8px;
        border: 1px solid var(--line);
        background: var(--panel);
        color: var(--ink);
        padding: 0 14px;
        font-weight: 800;
        cursor: pointer;
      }
      .decision-button.fix { background: var(--red); border-color: var(--red); color: #fff; }
      .decision-button.accept { background: var(--green); border-color: var(--green); color: #fff; }
      .decision-button.defer { background: #fff7ed; border-color: #fed7aa; color: #8a4b12; }
      .generate-button { background: #eff6ff; border-color: #bfdbfe; color: #1d4ed8; }
      .decision-button:disabled,
      .nav-button:disabled,
      .generate-button:disabled {
        cursor: wait;
        opacity: 0.6;
      }
      .review-grid {
        min-height: 0;
        display: grid;
        grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
        gap: 14px;
      }
      .pane {
        min-width: 0;
        min-height: 0;
        background: var(--panel);
        border: 1px solid var(--line);
        border-radius: 8px;
        box-shadow: var(--shadow);
        display: grid;
        grid-template-rows: auto minmax(0, 1fr);
      }
      .pane header {
        border-bottom: 1px solid var(--line);
        padding: 10px 12px;
        display: flex;
        justify-content: space-between;
        gap: 10px;
        align-items: center;
      }
      .pane h2 {
        margin: 0;
        font-size: 14px;
        letter-spacing: 0;
      }
      .path-label {
        color: var(--muted);
        font-size: 11px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        max-width: 34vw;
      }
      .image-shell {
        min-height: 0;
        overflow: auto;
        background: #d8e0eb;
      }
      .image-shell img {
        display: block;
        width: 100%;
        height: auto;
        image-rendering: auto;
      }
      .missing {
        min-height: 140px;
        display: flex;
        align-items: flex-start;
        justify-content: center;
        padding: 18px;
        color: var(--muted);
        text-align: center;
      }
      .note-row {
        display: grid;
        grid-template-columns: minmax(0, 1fr) auto;
        gap: 8px;
        margin-top: 10px;
      }
      .note-row input {
        min-height: 38px;
        border: 1px solid var(--line);
        border-radius: 8px;
        padding: 0 10px;
      }
      .status-text {
        color: var(--muted);
        font-size: 12px;
        margin-top: 8px;
        min-height: 18px;
      }
      .triage-note {
        color: #42526a;
        font-size: 13px;
        margin: 8px 0 0;
        max-width: 1100px;
      }
      @media (max-width: 1100px) {
        .app { grid-template-columns: 1fr; }
        .rail { min-height: auto; max-height: 360px; }
        .review-grid { grid-template-columns: 1fr; }
        .top { grid-template-columns: 1fr; }
        .actions { flex-wrap: wrap; }
        .path-label { max-width: 74vw; }
      }
    </style>
  </head>
  <body>
    <div class="app">
      <aside class="rail">
        <div class="brand">
          <h1>Plan 1 Visual Review</h1>
          <p id="progressText">Loading</p>
        </div>
        <div class="progress-card">
          <strong id="countText">0/170</strong>
          <div class="progress-line"><span id="progressBar"></span></div>
          <p class="progress-detail" id="triageText">Codex triage loading</p>
          <div class="link-stack">
            <a class="utility-link" href="/api/decisions" target="_blank" rel="noreferrer" style="display:grid;place-items:center;text-decoration:none;">Export Decisions JSON</a>
            <a class="utility-link" href="/api/triage" target="_blank" rel="noreferrer" style="display:grid;place-items:center;text-decoration:none;">Export Codex Triage</a>
          </div>
        </div>
        <div class="filters">
          <button data-filter="needs-user" class="active">Needs User</button>
          <button data-filter="pending">Pending</button>
          <button data-filter="all">All</button>
          <button data-filter="fix">Fix</button>
          <button data-filter="accept">Accept</button>
          <button data-filter="defer">Later</button>
          <button data-filter="codex-accept">Codex Accept</button>
          <button data-filter="codex-fix">Codex Fix</button>
        </div>
        <div class="pair-list" id="pairList"></div>
      </aside>
      <main class="main">
        <section class="top">
          <div>
            <h2 class="title" id="pairTitle">Loading</h2>
            <div class="meta" id="pairMeta"></div>
            <p class="triage-note" id="triageNote"></p>
            <div class="note-row">
              <input id="rationaleInput" type="text" placeholder="Brief rationale">
              <button class="nav-button" id="saveNoteButton" type="button">Save Note</button>
            </div>
            <div class="status-text" id="statusText"></div>
          </div>
          <div class="actions">
            <button class="nav-button" id="prevButton" type="button">Prev</button>
            <button class="decision-button fix" data-decision="fix" type="button">Düzelt</button>
            <button class="decision-button accept" data-decision="accept" type="button">Kabul et</button>
            <button class="decision-button defer" data-decision="defer" type="button">Sonra bak</button>
            <button class="nav-button" id="nextButton" type="button">Next</button>
          </div>
        </section>
        <section class="review-grid">
          <article class="pane">
            <header>
              <h2>Legacy Baseline</h2>
              <span class="path-label" id="baselinePath"></span>
            </header>
            <div class="image-shell" id="baselinePane"></div>
          </article>
          <article class="pane">
            <header>
              <h2>React Candidate</h2>
              <span class="path-label" id="candidatePath"></span>
            </header>
            <div class="image-shell" id="candidatePane"></div>
          </article>
        </section>
      </main>
    </div>
    <script>
      const state = {
        pairs: [],
        metadata: null,
        currentSequence: 1,
        filter: "needs-user",
        busy: false,
      };

      const elements = {
        pairList: document.getElementById("pairList"),
        pairTitle: document.getElementById("pairTitle"),
        pairMeta: document.getElementById("pairMeta"),
        triageNote: document.getElementById("triageNote"),
        baselinePane: document.getElementById("baselinePane"),
        candidatePane: document.getElementById("candidatePane"),
        baselinePath: document.getElementById("baselinePath"),
        candidatePath: document.getElementById("candidatePath"),
        progressText: document.getElementById("progressText"),
        countText: document.getElementById("countText"),
        triageText: document.getElementById("triageText"),
        progressBar: document.getElementById("progressBar"),
        rationaleInput: document.getElementById("rationaleInput"),
        statusText: document.getElementById("statusText"),
        prevButton: document.getElementById("prevButton"),
        nextButton: document.getElementById("nextButton"),
        saveNoteButton: document.getElementById("saveNoteButton"),
      };

      async function api(path, options = {}) {
        const response = await fetch(path, {
          headers: { "Content-Type": "application/json" },
          ...options,
        });
        const data = await response.json();
        if (!response.ok) throw new Error(data.error || "Request failed");
        return data;
      }

      function decisionLabel(value) {
        return { fix: "Fix", accept: "Accept", defer: "Later" }[value] || "Pending";
      }

      function codexLabel(value) {
        return { fix: "Codex: Fix", accept: "Codex: Accept", defer: "Codex: Later" }[value] || "Codex: Untriaged";
      }

      function currentPair() {
        return state.pairs.find((pair) => pair.sequence === state.currentSequence) || state.pairs[0];
      }

      function filteredPairs() {
        if (state.filter === "all") return state.pairs;
        if (state.filter === "needs-user") return state.pairs.filter((pair) => pair.triage?.userReviewRequired && !pair.decision);
        if (state.filter === "pending") return state.pairs.filter((pair) => !pair.decision);
        if (state.filter === "codex-accept") return state.pairs.filter((pair) => pair.triage?.codexRecommendation === "accept");
        if (state.filter === "codex-fix") return state.pairs.filter((pair) => pair.triage?.codexRecommendation === "fix");
        return state.pairs.filter((pair) => pair.decision?.decision === state.filter);
      }

      function chooseInitialPair() {
        const needsUser = state.pairs.find((pair) => pair.triage?.userReviewRequired && !pair.decision);
        if (needsUser) {
          state.filter = "needs-user";
          state.currentSequence = needsUser.sequence;
          syncActiveFilter();
          return;
        }
        state.filter = "pending";
        syncActiveFilter();
        const pending = state.pairs.find((pair) => !pair.decision);
        state.currentSequence = pending ? pending.sequence : (state.pairs[0]?.sequence || 1);
      }

      function syncActiveFilter() {
        document.querySelectorAll("[data-filter]").forEach((item) => {
          item.classList.toggle("active", item.dataset.filter === state.filter);
        });
      }

      function renderList() {
        const pairs = filteredPairs();
        elements.pairList.innerHTML = "";
        for (const pair of pairs) {
          const button = document.createElement("button");
          button.className = "pair-item" +
            (pair.sequence === state.currentSequence ? " active" : "") +
            (pair.decision ? " done" : "");
          button.type = "button";
          button.innerHTML =
            "<strong>" + pair.sequence + ". " + pair.surface + " / " + pair.viewport + "</strong>" +
            "<span>" + pair.auditId + " · " + pair.failedRegionAndRatio + "</span>" +
            "<span>" + codexLabel(pair.triage?.codexRecommendation) +
              (pair.triage?.userReviewRequired ? " · needs user" : "") + "</span>" +
            "<span>" + decisionLabel(pair.decision?.decision) + "</span>";
          button.addEventListener("click", () => {
            state.currentSequence = pair.sequence;
            render();
          });
          elements.pairList.append(button);
        }
      }

      function imageMarkup(pair, kind) {
        const exists = kind === "baseline" ? pair.baselineExists : pair.candidateExists;
        if (!exists) {
          if (kind === "candidate") {
            return '<div class="missing"><button class="generate-button" id="generateButton" type="button">Generate surface</button></div>';
          }
          return '<div class="missing">Missing baseline</div>';
        }
        return '<img src="/api/image/' + pair.sequence + '/' + kind + '?t=' + Date.now() + '" alt="' + kind + ' ' + pair.surface + ' ' + pair.viewport + '">';
      }

      function renderPair() {
        const pair = currentPair();
        if (!pair) return;
        elements.pairTitle.textContent = pair.sequence + "/170 · " + pair.surface + " · " + pair.viewport;
        elements.pairMeta.innerHTML = [
          pair.auditId,
          pair.failedRegionAndRatio,
          pair.candidateExists ? "candidate ready" : "candidate missing",
          pair.triage ? codexLabel(pair.triage.codexRecommendation) : "Codex: untriaged",
          pair.triage?.userReviewRequired ? "needs user review" : "Codex reviewed",
          pair.decision ? decisionLabel(pair.decision.decision) : "pending",
        ].map((value) => '<span class="pill">' + value + '</span>').join("");
        elements.triageNote.textContent = pair.triage ? pair.triage.rationale : "";
        elements.baselinePath.textContent = pair.baselinePathLabel;
        elements.candidatePath.textContent = pair.candidatePathLabel;
        elements.baselinePane.innerHTML = imageMarkup(pair, "baseline");
        elements.candidatePane.innerHTML = imageMarkup(pair, "candidate");
        elements.rationaleInput.value = pair.decision?.rationale || pair.triage?.rationale || "";
        const generateButton = document.getElementById("generateButton");
        if (generateButton) {
          generateButton.addEventListener("click", () => generateSurface(pair.surface));
        }
      }

      function renderProgress() {
        const reviewed = state.pairs.filter((pair) => pair.decision).length;
        const total = state.pairs.length;
        const triage = state.metadata?.triage ?? {};
        const needsUserLeft = state.pairs.filter((pair) => pair.triage?.userReviewRequired && !pair.decision).length;
        elements.progressText.textContent = reviewed + "/" + total + " reviewed";
        elements.countText.textContent = reviewed + "/" + total;
        elements.triageText.textContent = "Codex triage: " + (triage.reviewed || 0) + "/" + total +
          " reviewed · " + needsUserLeft + " needs user";
        elements.progressBar.style.width = total ? ((reviewed / total) * 100).toFixed(1) + "%" : "0%";
      }

      function setBusy(value, message = "") {
        state.busy = value;
        elements.statusText.textContent = message;
        document.querySelectorAll("button").forEach((button) => {
          button.disabled = value;
        });
      }

      function render() {
        renderProgress();
        renderList();
        renderPair();
      }

      async function loadState(keepCurrent = false) {
        const data = await api("/api/state");
        state.pairs = data.pairs;
        state.metadata = data.metadata;
        if (!keepCurrent) chooseInitialPair();
        render();
      }

      async function saveDecision(decision) {
        const pair = currentPair();
        if (!pair) return;
        setBusy(true, "Saving");
        try {
          await api("/api/decision", {
            method: "POST",
            body: JSON.stringify({
              sequence: pair.sequence,
              decision,
              rationale: elements.rationaleInput.value.trim(),
            }),
          });
          await loadState(true);
          const visible = filteredPairs();
          const nextPending = visible.find((item) => item.sequence > pair.sequence)
            || visible[0]
            || state.pairs.find((item) => !item.decision && item.sequence > pair.sequence)
            || state.pairs.find((item) => !item.decision);
          if (nextPending) state.currentSequence = nextPending.sequence;
          render();
          elements.statusText.textContent = "Saved";
        } catch (error) {
          elements.statusText.textContent = error.message;
        } finally {
          setBusy(false, elements.statusText.textContent);
        }
      }

      async function saveNote() {
        const pair = currentPair();
        if (!pair || !pair.decision) return;
        await saveDecision(pair.decision.decision);
      }

      async function generateSurface(surface) {
        setBusy(true, "Generating " + surface);
        try {
          const result = await api("/api/generate", {
            method: "POST",
            body: JSON.stringify({ surface }),
          });
          await loadState(true);
          render();
          elements.statusText.textContent = "Generated " + result.extracted.length + " attachments; exit " + result.exitCode;
        } catch (error) {
          elements.statusText.textContent = error.message;
        } finally {
          setBusy(false, elements.statusText.textContent);
        }
      }

      function move(delta) {
        const visible = filteredPairs();
        const index = visible.findIndex((pair) => pair.sequence === state.currentSequence);
        const next = visible[index + delta];
        if (next) {
          state.currentSequence = next.sequence;
          render();
        }
      }

      document.querySelectorAll("[data-decision]").forEach((button) => {
        button.addEventListener("click", () => saveDecision(button.dataset.decision));
      });
      document.querySelectorAll("[data-filter]").forEach((button) => {
        button.addEventListener("click", () => {
          state.filter = button.dataset.filter;
          document.querySelectorAll("[data-filter]").forEach((item) => item.classList.toggle("active", item === button));
          const visible = filteredPairs();
          if (!visible.some((pair) => pair.sequence === state.currentSequence) && visible[0]) {
            state.currentSequence = visible[0].sequence;
          }
          render();
        });
      });
      elements.prevButton.addEventListener("click", () => move(-1));
      elements.nextButton.addEventListener("click", () => move(1));
      elements.saveNoteButton.addEventListener("click", saveNote);
      document.addEventListener("keydown", (event) => {
        if (event.target instanceof HTMLInputElement) return;
        if (event.key === "ArrowRight") move(1);
        if (event.key === "ArrowLeft") move(-1);
        if (event.key.toLowerCase() === "d") saveDecision("fix");
        if (event.key.toLowerCase() === "k") saveDecision("accept");
        if (event.key.toLowerCase() === "s") saveDecision("defer");
      });

      loadState().catch((error) => {
        elements.statusText.textContent = error.message;
      });
    </script>
  </body>
</html>`;
}

async function route(request, response) {
  const url = new URL(request.url ?? "/", "http://127.0.0.1");
  try {
    if (request.method === "GET" && url.pathname === "/") {
      sendText(response, html(), 200, "text/html; charset=utf-8");
      return;
    }
    if (request.method === "GET" && url.pathname === "/api/state") {
      const pairs = parseLedgerPairs();
      const data = loadDecisionData();
      const triageData = loadTriageData();
      sendJson(response, {
        metadata: {
          total: pairs.length,
          reviewed: data.decisions.length,
          markdownPath: decisionMarkdownPath,
          jsonPath: decisionJsonPath,
          triage: {
            ...triageData.metadata,
            markdownPath: triageMarkdownPath,
            jsonPath: triageJsonPath,
          },
        },
        pairs: enrichPairs(pairs, data, triageData),
      });
      return;
    }
    if (request.method === "GET" && url.pathname === "/api/decisions") {
      sendJson(response, loadDecisionData());
      return;
    }
    if (request.method === "GET" && url.pathname === "/api/triage") {
      sendJson(response, loadTriageData());
      return;
    }
    const imageMatch = /^\/api\/image\/(\d+)\/(baseline|candidate)$/.exec(url.pathname);
    if (request.method === "GET" && imageMatch) {
      const pair = getPair(Number(imageMatch[1]));
      const imagePath = ensureAllowedImagePath(imageMatch[2] === "baseline" ? pair.baselinePath : pair.candidatePath);
      response.writeHead(200, {
        "Content-Type": "image/png",
        "Content-Length": statSync(imagePath).size,
        "Cache-Control": "no-store",
      });
      createReadStream(imagePath).pipe(response);
      return;
    }
    if (request.method === "POST" && url.pathname === "/api/decision") {
      const payload = await readRequestJson(request);
      const pair = getPair(Number(payload.sequence));
      const decision = String(payload.decision ?? "");
      if (!Object.hasOwn(statusByDecision, decision)) {
        throw Object.assign(new Error("Decision must be fix, accept, or defer."), { status: 400 });
      }
      const currentData = loadDecisionData();
      const existing = currentData.decisions.find((item) => item.sequence === pair.sequence);
      const rationale = String(payload.rationale ?? "").trim() || existing?.rationale || defaultRationaleByDecision[decision];
      const nextDecision = {
        sequence: pair.sequence,
        auditId: pair.auditId,
        surface: pair.surface,
        viewport: pair.viewport,
        failedRegionAndRatio: pair.failedRegionAndRatio,
        decision,
        decisionLabel: markdownDecisionByValue[decision],
        rationale,
        decisionDate: "2026-07-27",
        implementationStatus:
          existing?.decision === decision && existing.implementationStatus === "fixed-verified-locally"
            ? existing.implementationStatus
            : statusByDecision[decision],
      };
      const decisions = currentData.decisions.filter((item) => item.sequence !== pair.sequence);
      decisions.push(nextDecision);
      const saved = saveDecisionData({ ...currentData, decisions });
      sendJson(response, { ok: true, decision: nextDecision, metadata: saved.metadata });
      return;
    }
    if (request.method === "POST" && url.pathname === "/api/generate") {
      const payload = await readRequestJson(request);
      const surface = String(payload.surface ?? "").trim();
      const pairs = parseLedgerPairs();
      if (!pairs.some((pair) => pair.surface === surface)) {
        throw Object.assign(new Error(`Unknown failed surface: ${surface}`), { status: 404 });
      }
      const result = await runVisualSurface(surface);
      if (!result.extracted.length && result.extractionError) {
        throw Object.assign(new Error(result.extractionError), { status: 500 });
      }
      if (!result.extracted.length && !existsSync(result.reportPath)) {
        throw Object.assign(new Error(`Visual report was not written: ${result.reportPath}`), { status: 500 });
      }
      sendJson(response, {
        ok: true,
        exitCode: result.exitCode,
        reportPath: result.reportPath,
        outputDir: result.outputDir,
        extracted: result.extracted,
        extractionError: result.extractionError,
      });
      return;
    }
    sendJson(response, { error: "Not found" }, 404);
  } catch (error) {
    const status = Number(error.status) || 500;
    sendJson(response, { error: error instanceof Error ? error.message : String(error) }, status);
  }
}

function listen(port) {
  const server = createServer((request, response) => {
    route(request, response);
  });
  server.on("error", (error) => {
    if (error.code === "EADDRINUSE" && port < 4210) {
      listen(port + 1);
      return;
    }
    throw error;
  });
  server.listen(port, "127.0.0.1", () => {
    console.log(`Plan 1 visual stakeholder review: http://127.0.0.1:${port}`);
    console.log(`Decisions JSON: ${decisionJsonPath}`);
    console.log(`Decisions Markdown: ${decisionMarkdownPath}`);
  });
}

if (parseLedgerPairs().length !== 170) {
  throw new Error(`Expected 170 failed visual pairs, found ${parseLedgerPairs().length}.`);
}

if (!existsSync(decisionJsonPath)) {
  saveDecisionData(loadDecisionData());
}

if (process.argv.includes("--sync")) {
  saveDecisionData(loadDecisionData());
  process.exit(0);
}

const requestedPort = Number(process.env.PORT || 4197);
listen(Number.isFinite(requestedPort) ? requestedPort : 4197);
