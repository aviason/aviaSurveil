import assert from "node:assert/strict";
import test from "node:test";

import { assembleOpenApi } from "../../../scripts/bundle-openapi.mjs";

// Break caught: transport clients could receive a governed question without
// either mandatory view, or a source gap could collapse into empty citation
// fields rather than the literal SOURCE_MAPPING_REQUIRED state.
test("governed checklist questions require scope, origin, and regulatory-trace views", () => {
  const schemas = assembleOpenApi().components.schemas;
  const question = schemas.GovernedQuestionView;
  const scope = schemas.GovernedScopeRecommendationView;
  const trace = schemas.GovernedRegulatoryTraceView;
  const reconciliation = schemas.GovernedQuestionReconciliationView;

  assert.ok(question.required.includes("origin"));
  assert.ok(question.required.includes("scopeRecommendation"));
  assert.ok(question.required.includes("regulatoryTrace"));
  assert.deepEqual(schemas.GovernedQuestionOrigin.enum, [
    "REGULATORY_TRACE",
    "EXISTING_CHECKLIST_CANDIDATE",
    "HYBRID_RECONCILED",
  ]);
  assert.deepEqual(scope.required, [
    "classification",
    "inputSignals",
    "operationalHistoryBasis",
    "rationale",
    "guardrails",
    "approvalReviewState",
    "automaticDeferral",
  ]);
  assert.deepEqual(schemas.GovernedScopeGuardrailsView.required, [
    "mandatoryControl",
    "safetyCritical",
    "unknownHistory",
    "sourceChanged",
    "overdueControl",
    "automaticDeferralPermitted",
  ]);
  assert.ok(trace.required.includes("state"));
  assert.deepEqual(trace.properties.state.enum, ["RESOLVED", "SOURCE_MAPPING_REQUIRED"]);
  assert.ok(trace.properties.sha256);
  assert.ok(trace.properties.controlledCaaProcedureMapping);
  assert.ok(trace.properties.currentnessState);
  assert.ok(trace.properties.technicalReviewState);
  assert.ok(question.properties.reconciliation);
  assert.deepEqual(reconciliation.required, [
    "legacyQuestionId",
    "legacyWording",
    "legacyOperationalIntent",
    "legacyResultHistory",
    "legacyExpectedEvidence",
    "legacyApplicability",
    "legacyScopeClassification",
    "currentWording",
    "currentExpectedEvidence",
    "currentApplicability",
    "currentScopeClassification",
    "wordingChanged",
    "evidenceChanged",
    "applicabilityChanged",
    "scopeChanged",
  ]);
});

// Break caught: a raw newer source row could become implicitly current during
// import. The canonical transport must instead require an explicit immutable
// predecessor/current activation before any candidate may bind it.
test("source-currentness activation has an explicit immutable transport contract", () => {
  const openApi = assembleOpenApi();
  const schemas = openApi.components.schemas;
  const binding = schemas.GovernedSourceCurrentnessBindingInput;
  const activation = schemas.GovernedSourceCurrentnessActivationInput;
  const receipt = schemas.GovernedSourceCurrentnessActivationView;
  const operation = openApi.paths["/v1/admin/governed-checklist/source-currentness-activations"].post;

  assert.deepEqual(binding.required, [
    "currentSourceSnapshotId",
    "currentSourceHash",
    "previousSourceSnapshotId",
    "previousSourceHash",
  ]);
  assert.deepEqual(activation.required, [
    "operationId",
    "idempotencyKey",
    "currentSourceSnapshotId",
    "currentSourceHash",
    "previousSourceSnapshotId",
    "previousSourceHash",
    "reason",
  ]);
  assert.equal(activation.additionalProperties, false);
  assert.equal(activation.properties.previousSourceSnapshotId.type.includes("null"), true);
  assert.equal(activation.properties.previousSourceHash.type.includes("null"), true);
  assert.deepEqual(receipt.required, [
    "eventId",
    "impactReviewDraftId",
    "sourceIdentity",
    "previousSourceSnapshotId",
    "previousSourceHash",
    "currentSourceSnapshotId",
    "currentSourceHash",
    "status",
    "activatedAt",
  ]);
  assert.deepEqual(receipt.properties.status.enum, ["BASELINE_ACTIVATED", "IMPACT_REVIEW_DRAFT"]);
  assert.equal(operation.operationId, "activateAdminGovernedSourceCurrentness");
  assert.equal(operation.responses["201"].description, "Explicit immutable source-currentness activation");
});
