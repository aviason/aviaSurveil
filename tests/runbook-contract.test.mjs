import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const operationsRoot = path.join(repositoryRoot, "docs", "operations");
const runbooksRoot = path.join(operationsRoot, "runbooks");
const operationsIndexPath = path.join(operationsRoot, "index.md");
const alertCatalogPath = path.join(operationsRoot, "ALERT_CATALOG.md");
const prometheusRulesPath = path.join(
  repositoryRoot,
  "deploy",
  "observability",
  "rules",
  "aviasurveil360.yaml",
);

const requiredRunbooks = [
  "START_STOP.md",
  "INCIDENT_RESPONSE.md",
  "IDENTITY_MFA.md",
  "EVIDENCE_SCAN.md",
  "EMAIL_DOCUMENT_WORKERS.md",
  "BACKUP.md",
  "RESTORE.md",
  "DISASTER_RECOVERY.md",
  "RELEASE_ROLLBACK.md",
  "SECRET_ROTATION.md",
];

const requiredSections = [
  "Scope And Owner",
  "Preconditions",
  "Symptoms",
  "Safety Boundary",
  "Diagnosis",
  "Expected Output",
  "Reversible Mitigation",
  "Recovery Verification",
  "Evidence Capture",
  "Escalation",
  "Authorization Required",
];

function readRequired(filePath) {
  assert.equal(
    existsSync(filePath),
    true,
    `${path.relative(repositoryRoot, filePath)} must exist`,
  );
  return readFileSync(filePath, "utf8");
}

function markdownTableAfterHeading(markdown, heading) {
  const headingStart = markdown.indexOf(`## ${heading}`);
  assert.notEqual(headingStart, -1, `missing ${heading} section`);
  const lines = markdown.slice(headingStart).split("\n");
  const tableStart = lines.findIndex((line) => line.startsWith("|"));
  assert.notEqual(tableStart, -1, `${heading} must contain a table`);

  const tableLines = [];
  for (const line of lines.slice(tableStart)) {
    if (!line.startsWith("|")) break;
    tableLines.push(line);
  }
  const cells = tableLines.map((line) =>
    line
      .slice(1, -1)
      .split("|")
      .map((cell) => cell.trim()),
  );
  const headers = cells[0];
  return cells.slice(2).map((row) =>
    Object.fromEntries(
      headers.map((header, index) => [header, row[index] ?? ""]),
    ),
  );
}

function repositoryPath(relativePath) {
  assert.match(
    relativePath,
    /^docs\/operations\/runbooks\/[A-Z0-9_]+\.md$/,
    `invalid runbook path ${relativePath}`,
  );
  return path.join(repositoryRoot, relativePath);
}

function shellBlocks(markdown) {
  return [...markdown.matchAll(/```(?:bash|sh)\n([\s\S]*?)```/g)].map(
    ([, block]) => block,
  );
}

test("operations index links every required runbook", () => {
  const index = readRequired(operationsIndexPath);
  for (const runbook of requiredRunbooks) {
    assert.match(
      index,
      new RegExp(`\\(runbooks/${runbook.replace(".", "\\.")}\\)`),
      `operations index must link ${runbook}`,
    );
  }
});

test("alert catalog and Prometheus rules resolve to existing runbooks", () => {
  const alerts = markdownTableAfterHeading(
    readRequired(alertCatalogPath),
    "Alert catalog",
  );
  assert.equal(alerts.length, 8);
  for (const alert of alerts) {
    readRequired(repositoryPath(alert.Runbook));
  }

  const rules = readRequired(prometheusRulesPath);
  const ruleLinks = [
    ...rules.matchAll(
      /^\s+runbook_url:\s*(docs\/operations\/runbooks\/[A-Z0-9_]+\.md)\s*$/gm,
    ),
  ].map(([, link]) => link);
  assert.equal(ruleLinks.length, 8);
  for (const link of ruleLinks) {
    readRequired(repositoryPath(link));
  }
});

test("every runbook defines the complete candidate-only operating contract", () => {
  for (const runbook of requiredRunbooks) {
    const source = readRequired(path.join(runbooksRoot, runbook));
    const normalizedSource = source.replace(/\s+/g, " ");
    for (const section of requiredSections) {
      assert.match(source, new RegExp(`^## ${section}$`, "m"), `${runbook}: ${section}`);
    }
    assert.match(source, /^Owner:\s*\S.+$/m, `${runbook}: owner is required`);
    assert.match(
      source,
      /^Escalation owner:\s*\S.+$/m,
      `${runbook}: escalation owner is required`,
    );
    assert.match(source, /`candidate-only`/);
    assert.match(normalizedSource, /not production-ready/);
    assert.match(source, /`(?:verified locally|not run)`/);
    assert.doesNotMatch(
      normalizedSource.replaceAll("not production-ready", ""),
      /\bproduction-ready\b|\bproduction (?:validated|verified)\b/i,
    );
  }
});

test("runbook shell procedures reject broad destructive and secret-reading commands", () => {
  const forbiddenCommands = [
    /\bdocker\s+system\s+prune\b/,
    /\bdocker\s+volume\s+prune\b/,
    /\bdocker\s+network\s+prune\b/,
    /\bdocker\s+compose\b[^\n]*\b(?:down|rm)\b/,
    /\brm\s+-rf\b/,
    /\bterraform\s+destroy\b/,
    /\bterragrunt\b[^\n]*\bdestroy\b/,
    /\baws\s+s3\s+rm\b/,
    /\b(?:cat|head|tail|less|more)\b[^\n]*(?:\/secrets\/|_password|_secret|_key)\b/,
  ];

  for (const runbook of requiredRunbooks) {
    const source = readRequired(path.join(runbooksRoot, runbook));
    for (const block of shellBlocks(source)) {
      for (const forbidden of forbiddenCommands) {
        assert.doesNotMatch(block, forbidden, `${runbook}: unsafe shell procedure`);
      }
      if (/scripts\/local-stack\.sh\s+(?:down|status|logs|check)\b/.test(block)) {
        assert.match(block, /^export AVIA_LOCAL_PROJECT=/m);
        assert.match(block, /^export AVIASURVEIL_LOCAL_STATE_DIR=/m);
      }
      if (
        /(?:scripts\/local-stack\.sh|docker\s+compose\b[^\n]*)\s+up\b/.test(
          block,
        )
      ) {
        assert.match(block, /^export AVIA_LOCAL_HTTPS_PORT=/m);
        assert.match(block, /^export AVIA_LOCAL_PUBLIC_ORIGIN=/m);
      }
    }
  }
});

test("drill matrix covers tabletop and live recovery for warning and critical incidents", () => {
  const drills = markdownTableAfterHeading(
    readRequired(operationsIndexPath),
    "Runbook Drill Matrix",
  );
  const combinations = new Set(
    drills.map((row) => `${row.Severity}:${row.Mode}`),
  );
  assert.deepEqual(
    combinations,
    new Set([
      "warning:tabletop",
      "warning:live",
      "critical:tabletop",
      "critical:live",
    ]),
  );
  for (const drill of drills) {
    assert.notEqual(drill.Owner, "");
    assert.match(drill.Scenario, /\S/);
    assert.match(drill.Recovery, /\S/);
    assert.match(drill.Evidence, /^`(?:not run|verified locally)`$/);
  }
});
