import type {
  CreateAdminGovernedCandidateRevisionInput,
  GovernedCandidateBundleInput,
  SubmitAdminGovernedCandidateReviewInput,
} from "./backend";

function canonicalValue(value: unknown): string {
  if (value === null || typeof value !== "object") return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map(canonicalValue).join(", ")}]`;
  const entries = Object.entries(value).sort(([left], [right]) =>
    left.length - right.length || (left < right ? -1 : left > right ? 1 : 0));
  return `{${entries.map(([key, child]) => `${JSON.stringify(key)}: ${canonicalValue(child)}`).join(", ")}}`;
}

// Review projections are server-derived views over immutable candidate bytes.
// They must never change the content or command digests that bind a candidate
// to its persisted source/mapping snapshots.
function withoutServerDerivedMappingReviewProjection(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(withoutServerDerivedMappingReviewProjection);
  if (value === null || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value)
      .filter(([key]) => key !== "mappingReviewState" &&
        !(key === "technicalReviewState" && (value as { state?: unknown }).state === "SOURCE_MAPPING_REQUIRED"))
      .map(([key, child]) => [key, withoutServerDerivedMappingReviewProjection(child)]),
  );
}

function withoutNullableSourcePredecessor(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(withoutNullableSourcePredecessor);
  if (value === null || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value)
      .filter(([key, child]) => !["previousSourceSnapshotId", "previousSourceHash"].includes(key) || child !== null)
      .map(([key, child]) => [key, withoutNullableSourcePredecessor(child)]),
  );
}

export function governedCanonicalJSON(value: unknown): string {
  return canonicalValue(JSON.parse(JSON.stringify(value)) as unknown);
}

export async function governedCanonicalSHA256(value: unknown): Promise<string> {
  const bytes = new TextEncoder().encode(governedCanonicalJSON(value));
  const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
  return `sha256:${Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

export async function governedRequestDigest(
  request: GovernedCandidateBundleInput["generationRequest"],
): Promise<string> {
  const unsigned = structuredClone(request) as unknown as Record<string, unknown>;
  Reflect.deleteProperty(unsigned, "canonicalInputDigest");
  return governedCanonicalSHA256(unsigned);
}

export function governedCandidateContentDigest(candidateOutput: {
  complianceMappings: unknown;
  inspectionChecklist: unknown;
}): Promise<string> {
  return governedCanonicalSHA256(withoutServerDerivedMappingReviewProjection(candidateOutput));
}

export function governedImportSemanticDigest(
  operationId: string,
  candidateBundle: GovernedCandidateBundleInput,
): Promise<string> {
  return governedCanonicalSHA256(withoutNullableSourcePredecessor(
    withoutServerDerivedMappingReviewProjection({ operationId, candidateBundle }),
  ));
}

export function governedEditSemanticDigest(
  command: Omit<CreateAdminGovernedCandidateRevisionInput, "operationId" | "idempotencyKey">,
): Promise<string> {
  return governedCanonicalSHA256(withoutServerDerivedMappingReviewProjection({
    candidateId: command.candidateId,
    expectedRevision: command.expectedRevision,
    expectedContentDigest: command.expectedContentDigest,
    changeReason: command.changeReason,
    mappings: command.mappings,
    questions: command.questions,
    requiredOwners: command.requiredOwners,
  }));
}

export function governedSubmitSemanticDigest(
  command: SubmitAdminGovernedCandidateReviewInput,
): Promise<string> {
  return governedCanonicalSHA256({
    operationId: command.operationId,
    idempotencyKey: command.idempotencyKey,
    candidateId: command.candidateId,
    expectedContentDigest: command.expectedContentDigest,
    reason: command.reason,
    expectedRevision: command.expectedRevision,
  });
}
