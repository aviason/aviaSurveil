import type { FieldSyncOperation } from "../backend/backend";
import type { FieldOutboxState, OutboxRow } from "./db";
import { sha256Canonical } from "./schema-migrations";

const TERMINAL_STATES = new Set<FieldOutboxState>([
  "ACKNOWLEDGED",
  "SUPERSEDED",
  "REJECTED",
  "QUARANTINED",
]);

export function isTerminalOutboxState(state: FieldOutboxState): boolean {
  return TERMINAL_STATES.has(state);
}

export function isUnsentOutboxState(state: FieldOutboxState): boolean {
  return state === "PENDING" || state === "BLOCKED_ON_DEPENDENCY" || state === "FAILED_RETRYABLE";
}

export function isCausalDependencyState(state: FieldOutboxState): boolean {
  return !TERMINAL_STATES.has(state);
}

export async function fieldOperationRequestDigest(operation: FieldSyncOperation): Promise<string> {
  return sha256Canonical({
    operationId: operation.operationId,
    protocolVersion: operation.protocolVersion,
    offlineGrantId: operation.offlineGrantId,
    packageId: operation.packageId,
    packageVersion: operation.packageVersion,
    packageRevision: operation.packageRevision ?? 0,
    entityId: operation.entityId,
    commandType: operation.commandType,
    baseRevision: operation.baseRevision,
    deviceInstanceId: operation.deviceInstanceId,
    actorSubject: operation.actorSubject ?? "",
    operationSequence: operation.operationSequence ?? 0,
    payloadHash: operation.payloadHash ?? "",
    profileKeyId: operation.profileKeyId ?? "",
    dependencies: [...(operation.dependencies ?? [])].sort(),
    payload: operation.payload,
  });
}

export async function createOutboxRow(input: {
  subjectId: string;
  operation: FieldSyncOperation;
  state: FieldOutboxState;
  createdAt: string;
  dependsOnOperationIds?: string[];
  operationSequence?: number;
  entityRevision?: number;
  entityHash?: string;
}): Promise<OutboxRow> {
  const entityHash = input.entityHash ?? await sha256Canonical(input.operation.payload);
  const operationSequence = input.operationSequence ?? 0;
  return {
    operationId: input.operation.operationId,
    operationSequence,
    subjectId: input.subjectId,
    packageId: input.operation.packageId,
    commandType: input.operation.commandType,
    entityId: input.operation.entityId,
    baseRevision: input.operation.baseRevision,
    state: input.state,
    createdAt: input.createdAt,
    attemptCount: 0,
    nextAttemptAt: input.createdAt,
    dependsOnOperationIds: [...new Set(input.dependsOnOperationIds ?? [])].sort(),
    supersededByOperationId: null,
    requestDigest: await fieldOperationRequestDigest(input.operation),
    entityRevision: input.entityRevision ?? input.operation.baseRevision ?? 0,
    entityHash,
    commitReceiptKey: `commit-receipt:${input.subjectId}:${input.operation.operationId}`,
    lastErrorCode: null,
    operation: structuredClone(input.operation),
  };
}
