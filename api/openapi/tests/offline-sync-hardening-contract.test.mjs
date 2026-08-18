import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const source = JSON.parse(fs.readFileSync(path.join(repositoryRoot, "api/openapi/source/openapi.json"), "utf8"));

test("offline sync operations carry stable delivery identity and causal metadata", () => {
  const required = [
    "operationId",
    "protocolVersion",
    "offlineGrantId",
    "packageId",
    "packageVersion",
    "packageRevision",
    "entityId",
    "commandType",
    "baseRevision",
    "deviceInstanceId",
    "actorSubject",
    "operationSequence",
    "payloadHash",
    "requestHash",
    "profileKeyId",
    "authorityProof",
    "dependencies",
    "clientOccurredAt",
    "payload",
  ];
  for (const name of [
    "UpsertChecklistResponseOperation",
    "CreatePotentialFindingOperation",
    "SubmitChecklistOperation",
    "RegisterInspectionAttachmentOperation",
    "ResolveFieldConflictOperation",
  ]) {
    assert.deepEqual(
      source.components.schemas[name].required,
      required,
      `${name} must use the complete at-least-once operation envelope`,
    );
  }
});

test("sync pull is bound to the exact device profile metadata", () => {
  const parameters = source.paths["/v1/sync/changes"].get.parameters;
  assert.ok(parameters.some((parameter) => parameter.name === "deviceInstanceId" && parameter.required === true));
  assert.ok(parameters.some((parameter) => parameter.name === "profileKeyId" && parameter.required === true));
});

test("conflict resolution is an explicit typed operation", () => {
  assert.ok(source.components.schemas.FieldCommandType.enum.includes("RESOLVE_FIELD_CONFLICT"));
  const resolution = source.components.schemas.ResolveFieldConflictOperation;
  assert.deepEqual(resolution.properties.payload.properties.resolution.enum, [
    "ACCEPT_SERVER",
    "KEEP_LOCAL_AS_NEW_REVISION",
    "AUTHORIZED_REVIEWER",
  ]);
  assert.ok(resolution.required.includes("operationSequence"));
  assert.ok(source.components.schemas.FieldSyncOperation.oneOf.some((entry) =>
    entry.$ref === "#/components/schemas/ResolveFieldConflictOperation",
  ));
});

test("checkout and grant contracts bind the profile authority and lease revision", () => {
  const checkout = source.components.schemas.CheckoutInspectionPackageInput;
  assert.ok(checkout.required.includes("profileKeyId"));
  assert.ok(checkout.required.includes("profilePublicJwk"));
  const grant = source.components.schemas.OfflineGrant;
  assert.ok(grant.required.includes("profileKeyId"));
  assert.ok(grant.required.includes("assignmentRevision"));
  assert.ok(grant.required.includes("leaseIssuedAt"));
  assert.ok(grant.required.includes("leaseExpiresAt"));
});

test("inspection attachment upload contract is resumable and receipt-first", () => {
  const begin = source.components.schemas.BeginInspectionAttachmentUploadOutput;
  assert.ok(begin.required.includes("sessionEpoch"));
  assert.ok(begin.required.includes("partSize"));
  assert.ok(begin.required.includes("receivedParts"));
  assert.ok(begin.required.includes("partHashes"));
  const complete = source.components.schemas.CompleteInspectionAttachmentUploadInput;
  assert.ok(complete.required.includes("parts"));
  assert.equal(source.paths["/v1/inspection-attachments/uploads/{uploadId}/parts"].post.operationId, "beginInspectionAttachmentPartUpload");
  assert.equal(source.paths["/v1/inspection-attachments/uploads/{uploadId}/parts/acknowledge"].post.operationId, "acknowledgeInspectionAttachmentPart");
  assert.ok(source.components.schemas.UploadPartReceipt.required.includes("objectVersion"));
});

test("finalization is a server receipt rather than an empty-outbox claim", () => {
  const operation = source.paths["/v1/inspections/{inspectionId}/finalize"].post;
  assert.equal(operation.operationId, "finalizeInspection");
  const receipt = source.components.schemas.InspectionFinalizationReceipt;
  assert.ok(receipt.required.includes("answerManifestHash"));
  assert.ok(receipt.required.includes("attachmentManifestHash"));
  assert.deepEqual(receipt.properties.canonicalizationVersion.enum, ["avia-finalization-manifest/v1"]);
});

console.log("offline-sync-hardening-contract: tests loaded");
