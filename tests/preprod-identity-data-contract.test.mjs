import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { isDeepStrictEqual } from "node:util";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const specificationPath = path.join(
  repositoryRoot,
  "docs",
  "product-specs",
  "data-and-rules",
  "PREPROD_IDENTITY_AND_DATA_PROFILE.md",
);
const routeCatalogPath = path.join(
  repositoryRoot,
  "apps",
  "web",
  "src",
  "parity",
  "legacy-screen-source.json",
);
const openAPIPath = path.join(
  repositoryRoot,
  "api",
  "openapi",
  "source",
  "openapi.json",
);

const contractBegin = "<!-- PREPROD_IDENTITY_DATA_CONTRACT:BEGIN -->";
const contractEnd = "<!-- PREPROD_IDENTITY_DATA_CONTRACT:END -->";

const expectedRoles = [
  "inspector",
  "leadInspector",
  "manager",
  "finance",
  "gm",
  "executiveDirector",
  "auditee",
  "admin",
];
const expectedMembershipStates = [
  "requested",
  "invited",
  "active",
  "suspended",
  "deactivated",
  "reactivation-pending",
];
const expectedStateMachines = [
  "providerAccount",
  "desiredMembership",
  "applicationProfile",
  "invitationRecovery",
  "mfa",
  "session",
];
const expectedProfiles = ["smoke", "acceptance", "realistic", "stress"];
const expectedLifecycleScenarios = [
  "planned",
  "active",
  "overdue",
  "returned",
  "rejected",
  "corrected",
  "superseded",
  "reopened",
  "partially-closed",
  "not-closed",
  "authorized-closed",
  "verified-closed",
];
const expectedOwnerDecisions = [
  "INVITATION_CHANNEL_EXPIRY_RESEND",
  "AUDITEE_MFA",
  "RECOVERY_AND_MFA_RESET",
  "SUSPENSION_DEACTIVATION_REACTIVATION",
  "ORGANIZATION_TRANSFER",
  "IDENTIFIER_RETENTION_REUSE",
  "PERMISSIBLE_MULTI_ROLE_COMBINATIONS",
  "BOOTSTRAP_ADMIN_BREAK_GLASS",
  "KEYCLOAK_SERVICE_ACCOUNT_PRIVILEGES",
  "PROVIDER_OBSERVATION_FRESHNESS_DEADLINE",
  "PROFILE_VOLUMES_RESOURCE_LIMITS",
];
const expectedAuthorityMutations = [
  "activate",
  "suspend",
  "deactivate",
  "request-reactivation",
  "reactivate",
  "change-roles",
  "transfer-organization",
  "reset-mfa",
  "force-logout",
  "resend-invitation",
  "start-recovery",
];
const approvedDecisionStatus = "approved — owner decision recorded";
const rolePolicy = (value) =>
  Object.fromEntries(expectedRoles.map((role) => [role, value]));
const expectedProfileResources = {
  smoke: {
    cpuCores: 2,
    memoryMiB: 1024,
    diskMiB: 2048,
    objectBytes: 134217728,
    durationSeconds: 120,
    cleanupSeconds: 60,
  },
  acceptance: {
    cpuCores: 4,
    memoryMiB: 4096,
    diskMiB: 20480,
    objectBytes: 2147483648,
    durationSeconds: 1200,
    cleanupSeconds: 600,
  },
  realistic: {
    cpuCores: 8,
    memoryMiB: 12288,
    diskMiB: 51200,
    objectBytes: 21474836480,
    durationSeconds: 7200,
    cleanupSeconds: 2700,
  },
  stress: {
    cpuCores: 12,
    memoryMiB: 12288,
    diskMiB: 65536,
    objectBytes: 8589934592,
    durationSeconds: 28800,
    cleanupSeconds: 5400,
  },
};
const expectedLocalQualificationRevisions = {
  realistic: {
    version: "1.1.0",
    status: approvedDecisionStatus,
    approvalReference: "OWNER-DIRECTIVE-2026-07-28-P5T8-01",
    approvedAt: "2026-07-28",
    purpose: "local-qualification",
    sourceProfile: "acceptance@1.0.0",
    scaleMultiplier: 2,
    preservedCatalogCountFamilies: [
      "routeDispositions",
      "visibleActionDispositions",
    ],
    organizationDistribution: { caa: 1, auditee: 49 },
    resourceEnvelope: {
      seedRequired: true,
      clockOrigin: "2026-01-01T00:00:00Z",
      identityNamespace: "synthetic-realistic-local-v1-1",
      cpuCores: 8,
      memoryMiB: 8192,
      diskMiB: 20480,
      objectBytes: 2147483648,
      durationSeconds: 900,
      qualificationSeconds: 900,
      cleanupSeconds: 300,
    },
  },
  stress: {
    version: "1.1.0",
    status: approvedDecisionStatus,
    approvalReference: "OWNER-DIRECTIVE-2026-07-28-P5T8-01",
    approvedAt: "2026-07-28",
    purpose: "local-qualification",
    sourceProfile: "acceptance@1.0.0",
    scaleMultiplier: 4,
    preservedCatalogCountFamilies: [
      "routeDispositions",
      "visibleActionDispositions",
    ],
    organizationDistribution: { caa: 1, auditee: 99 },
    resourceEnvelope: {
      seedRequired: true,
      clockOrigin: "2026-01-01T00:00:00Z",
      identityNamespace: "synthetic-stress-local-v1-1",
      cpuCores: 12,
      memoryMiB: 12288,
      diskMiB: 32768,
      objectBytes: 536870912,
      durationSeconds: 1800,
      qualificationSeconds: 1800,
      cleanupSeconds: 300,
    },
  },
};
const expectedOwnerImplementationValues = {
  INVITATION_CHANNEL_EXPIRY_RESEND: {
    channel: "authenticated-local-smtp-mailpit",
    expirySeconds: 86400,
    resend: "invalidate-prior-action",
    maximumResendsPer24Hours: 3,
    verifyEmailByRole: rolePolicy(true),
  },
  AUDITEE_MFA: {
    policy: "optional-all-roles",
    configureTotpByRole: rolePolicy(false),
    selfEnrollment: true,
  },
  RECOVERY_AND_MFA_RESET: {
    initiation: "reasoned-admin-assisted",
    providerModel: "keycloak-execute-actions",
    expirySeconds: 900,
    revokeSessionsBeforeAction: true,
    accountRecoveryRequiredActions: ["UPDATE_PASSWORD"],
    mfaReset: "clear-existing-enrollment",
    mfaReenrollment: "optional",
    applicationSecretHandling: "forbidden",
  },
  SUSPENSION_DEACTIVATION_REACTIVATION: {
    suspension: "temporary-until-explicit-reactivation",
    deactivation: "retained-tombstone-no-future-authority",
    reactivation: "owner-approved-reactivation-pending",
    automaticExpiry: false,
    revokeSessionsAndOfflineGrants: true,
    reviveOldSessions: false,
  },
  ORGANIZATION_TRANSFER: {
    mode: "reasoned-effective-dated",
    requireFutureEffectiveAt: true,
    historicalOwnershipRewrite: "forbidden",
    providerOrganizationChange: "atomic-at-effective-time",
    reconciliationFailure: "fail-closed",
    revokeSessionsAndOfflineGrants: true,
    forbiddenRoleOrganizationCombination: "reject",
  },
  IDENTIFIER_RETENTION_REUSE: {
    policy: "permanent-non-reuse",
    identifiers: [
      "membershipId",
      "providerSubject",
      "username",
      "loginIdentifier",
    ],
    reactivationUsesExistingMembership: true,
    automaticRelease: "forbidden",
  },
  PERMISSIBLE_MULTI_ROLE_COMBINATIONS: {
    maximumRolesPerMembership: 1,
    roleChange: "atomic-replacement",
    auditeeCaaMix: "forbidden",
    futureCaaCombinationPolicy: "new-versioned-allowlist-required",
  },
  BOOTSTRAP_ADMIN_BREAK_GLASS: {
    bootstrap: "one-shot-removed-from-runtime",
    breakGlassApplicationMembership: false,
    custodyApprovals: 2,
    windowSeconds: 900,
    alarmRequired: true,
    auditRequired: true,
    incidentRequired: true,
    rotateAfterUse: true,
    closeSessionsAfterUse: true,
    sharedCredential: "forbidden",
    permanentRealmAdmin: "forbidden",
  },
  KEYCLOAK_SERVICE_ACCOUNT_PRIVILEGES: {
    clientType: "confidential",
    grantType: "client_credentials",
    allowedRealmRoles: [
      "query-users",
      "view-users",
      "manage-users",
      "view-realm",
    ],
    deniedCapabilities: [
      "realm-admin",
      "manage-realm",
      "manage-clients",
      "impersonation",
      "cross-realm-access",
    ],
    interactiveLogin: false,
    applicationMembership: false,
    credentialStorage: "environment-specific-secret-management",
  },
  PROVIDER_OBSERVATION_FRESHNESS_DEADLINE: {
    heartbeatSeconds: 30,
    maximumAgeSeconds: 60,
    reconciliationDeadlineSeconds: 120,
    ageIsUserInactivity: false,
    staleNewLogin: "deny",
    staleAuthorityMutation: "deny",
    staleExistingSessions: "revocation-pending-then-revoked-by-deadline",
    driftOrDisablement: "immediate-fail-closed",
    recovery: "fresh-exact-observation-and-new-login",
  },
  PROFILE_VOLUMES_RESOURCE_LIMITS: {
    profileVersions: {
      smoke: "1.0.0",
      acceptance: "1.0.0",
      realistic: "1.0.0",
      stress: "1.0.0",
    },
    exactCounts: "machine-profile-manifests",
    resourceEnvelopes: expectedProfileResources,
    runtimeFeasibility: "not run",
    silentReduction: "forbidden",
    localQualificationRevision: {
      approvalReference: "OWNER-DIRECTIVE-2026-07-28-P5T8-01",
      approvedAt: "2026-07-28",
      profileVersions: {
        realistic: "1.1.0",
        stress: "1.1.0",
      },
      maximumQualificationSeconds: {
        realistic: 900,
        stress: 1800,
      },
      retainedGates: [
        "all-data-families",
        "relationship-reconciliation",
        "privacy",
        "resume",
        "resource-envelope",
        "whole-namespace-cleanup",
      ],
      fullVolumeEndurancePurpose: "release-readiness-evidence",
      fullVolumeEnduranceStatus: "not run",
    },
  },
};

function readContract() {
  assert.equal(
    existsSync(specificationPath),
    true,
    "PREPROD_IDENTITY_AND_DATA_PROFILE.md must exist before Task 1 can pass",
  );
  const source = readFileSync(specificationPath, "utf8");
  const begin = source.indexOf(contractBegin);
  const end = source.indexOf(contractEnd);
  assert.ok(begin >= 0 && end > begin, "machine-readable contract markers must exist");
  const fenced = source.slice(begin + contractBegin.length, end).trim();
  const match = fenced.match(/^```json\s*\n([\s\S]+)\n```$/u);
  assert.ok(match, "machine-readable contract must be one fenced JSON object");
  return JSON.parse(match[1]);
}

function clone(value) {
  return structuredClone(value);
}

function sameMembers(actual, expected) {
  return Array.isArray(actual) &&
    actual.length === expected.length &&
    new Set(actual).size === expected.length &&
    expected.every((item) => actual.includes(item));
}

function validateContract(contract) {
  const errors = [];
  const add = (condition, message) => {
    if (!condition) errors.push(message);
  };

  add(contract.schemaVersion === "preprod-identity-data-contract/v1", "schema version");
  add(
    typeof contract.contractVersion === "string" &&
      /^\d+\.\d+\.\d+$/u.test(contract.contractVersion),
    "contract version",
  );
  add(contract.status === "active — Task 1 authorized", "contract status");

  const roles = contract.identity?.roles ?? [];
  add(
    sameMembers(roles.map((role) => role.id), expectedRoles),
    "eight-role coverage",
  );
  add(
    roles.every((role) =>
      role.id === "auditee"
        ? role.organizationScope === "exactly-one-non-CAA"
        : role.organizationScope === "exact-CAA"
    ),
    "role organization scope",
  );
  add(
    contract.identity?.roleAuthority === "server-owned-desired-membership",
    "client-authored roles",
  );
  add(
    contract.identity?.clientAuthorityInput === "reject",
    "client-authored organization",
  );
  const forbiddenCombinationIds = new Set(
    (contract.identity?.forbiddenCombinations ?? []).map((item) => item.id),
  );
  for (const required of [
    "AUDITEE_WITH_CAA_ROLE",
    "AUDITEE_IN_CAA",
    "CAA_ROLE_OUTSIDE_CAA",
    "MULTIPLE_ORGANIZATIONS",
    "UNAPPROVED_MULTI_ROLE_SET",
  ]) {
    add(forbiddenCombinationIds.has(required), `forbidden combination ${required}`);
  }

  add(
    sameMembers(Object.keys(contract.stateMachines ?? {}), expectedStateMachines),
    "separate authority state machines",
  );
  add(
    sameMembers(
      contract.stateMachines?.desiredMembership?.states,
      expectedMembershipStates,
    ),
    "membership lifecycle coverage",
  );
  add(
    contract.stateMachines?.desiredMembership?.businessKey === "membershipId" &&
      contract.stateMachines?.desiredMembership?.revision ===
        "monotonic-positive-integer",
    "stable revisioned membership identity",
  );
  for (const field of [
    "membershipId",
    "revision",
    "requestedAt",
    "effectiveAt",
    "actorSubjectId",
    "reason",
    "roles",
    "organizationId",
  ]) {
    add(
      contract.stateMachines?.desiredMembership?.requiredFields?.includes(field),
      `membership field ${field}`,
    );
  }

  const mutations = contract.authorityMutations ?? [];
  add(
    sameMembers(mutations.map((mutation) => mutation.id), expectedAuthorityMutations),
    "authority mutation coverage",
  );
  add(
    mutations.every((mutation) =>
      mutation.expectedMembershipRevision === "required" &&
      typeof mutation.sessionOutcome === "string" &&
      typeof mutation.membershipOutcome === "string"
    ),
    "expectedMembershipRevision on every authority mutation",
  );
  add(
    contract.revisionContract?.initialExpectedMembershipRevision === 0 &&
      contract.revisionContract?.staleRevisionOutcome ===
        "conflict-with-no-side-effects",
    "initial and stale membership revision semantics",
  );

  add(contract.registration?.public === false, "public registration");
  add(
    contract.invitation?.providerModel === "keycloak-execute-actions" &&
      contract.invitation?.requiredActions?.includes("UPDATE_PASSWORD"),
    "Keycloak execute-actions UPDATE_PASSWORD invitation",
  );
  add(
    contract.invitation?.deliveryPolicy?.status === approvedDecisionStatus &&
      isDeepStrictEqual(
        contract.invitation?.deliveryPolicy?.effectiveValue,
        {
          channel: "authenticated-local-smtp-mailpit",
          expirySeconds: 86400,
          resend: "invalidate-prior-action",
          maximumResendsPer24Hours: 3,
        },
      ),
    "approved invitation channel expiry resend owner-decision field",
  );
  add(
    contract.invitation?.verifyEmail?.status === approvedDecisionStatus &&
      contract.invitation?.verifyEmail?.effectiveValue ===
        "required-all-roles" &&
      isDeepStrictEqual(
        contract.invitation?.verifyEmail?.effectiveByRole,
        rolePolicy(true),
      ),
    "approved verifyEmail owner-decision field",
  );
  add(
    contract.invitation?.configureTotp?.status === approvedDecisionStatus &&
      contract.invitation?.configureTotp?.effectiveValue ===
        "optional-all-roles" &&
      isDeepStrictEqual(
        contract.invitation?.configureTotp?.effectiveByRole,
        rolePolicy(false),
      ),
    "approved configureTotp owner-decision field",
  );
  add(
    contract.providerObservation?.status === approvedDecisionStatus &&
      isDeepStrictEqual(
        contract.providerObservation?.effectiveValue,
        expectedOwnerImplementationValues
          .PROVIDER_OBSERVATION_FRESHNESS_DEADLINE,
      ) &&
      contract.providerObservation?.failClosedOn?.length >= 4,
    "approved provider observation decision and fail-closed behavior",
  );

  const normalArtifact = contract.normalArtifact ?? {};
  add(
    normalArtifact.publicRegistration === false,
    "normal artifact public registration",
  );
  add(
    (normalArtifact.authentication ?? []).every(
      (method) => !/canonical|header/iu.test(method),
    ),
    "normal artifact canonical-header authentication",
  );
  for (const forbidden of [
    "seed-route",
    "reset-route",
    "seed-startup-hook",
    "reset-startup-hook",
    "loader-command",
  ]) {
    add(
      normalArtifact.forbiddenSurfaces?.includes(forbidden),
      `normal artifact ${forbidden}`,
    );
  }
  for (const forbiddenImport of [
    "apps/api/internal/testprofile",
    "apps/api/internal/preproddata",
  ]) {
    add(
      normalArtifact.forbiddenImports?.includes(forbiddenImport),
      `normal artifact import ${forbiddenImport}`,
    );
  }

  add(
    contract.catalogs?.routeCatalog?.exactCount === 86,
    "86-route coverage",
  );
  add(
    contract.catalogs?.visibleActionCatalog?.coverage === "complete",
    "visible-action coverage",
  );
  add(
    sameMembers(contract.catalogs?.roleCatalog?.roles, expectedRoles),
    "catalog eight-role coverage",
  );
  add(
    sameMembers(
      contract.catalogs?.lifecycleScenarioCatalog?.scenarios,
      expectedLifecycleScenarios,
    ),
    "lifecycle scenario catalog",
  );

  const familyDefinitions = contract.dataProfiles?.familyDefinitions ?? [];
  const familyIds = familyDefinitions.map((family) => family.id);
  add(
    familyIds.length > 0 && new Set(familyIds).size === familyIds.length,
    "closed generated-family catalog",
  );
  add(
    familyDefinitions.every((family) =>
      family.relationshipDigest?.required === true &&
      family.relationshipDigest?.algorithm === "sha256" &&
      family.relationshipDigest?.placeholderAllowed === false &&
      Array.isArray(family.relationshipDigest?.tupleFields) &&
      family.relationshipDigest.tupleFields.length > 0
    ),
    "relationship digest for every generated family",
  );

  const profiles = contract.dataProfiles?.profiles ?? [];
  add(
    sameMembers(profiles.map((profile) => profile.name), expectedProfiles),
    "four profile coverage",
  );
  for (const profile of profiles) {
    add(
      typeof profile.version === "string" &&
        /^\d+\.\d+\.\d+$/u.test(profile.version),
      `versioned profile ${profile.name}`,
    );
    add(
      profile.status === approvedDecisionStatus &&
        profile.implementationAllowed === false,
      `approved profile ${profile.name} remains runtime blocked`,
    );
    add(
      profile.changePolicy === "new-version-required",
      `unversioned profile change ${profile.name}`,
    );
    add(
      profile.catalogs?.routeCount === 86 &&
        profile.catalogs?.visibleActionCoverage === "complete" &&
        sameMembers(profile.catalogs?.roles, expectedRoles) &&
        sameMembers(profile.catalogs?.lifecycleScenarios, expectedLifecycleScenarios),
      `complete catalogs for ${profile.name}`,
    );
    for (const familyId of familyIds) {
      const count = profile.expectedCounts?.[familyId];
      add(
        Number.isSafeInteger(count) && count >= 0,
        `exact count ${profile.name}/${familyId}`,
      );
      const distribution = profile.exactDistributions?.[familyId] ??
        (contract.dataProfiles?.defaultDistributionFamilies?.includes(familyId)
          ? { generated: count }
          : null);
      add(
        distribution !== null &&
          Object.values(distribution).every((value) =>
            Number.isSafeInteger(value) && value >= 0
          ) &&
          Object.values(distribution).reduce((sum, value) => sum + value, 0) ===
            count,
        `exact distribution ${profile.name}/${familyId}`,
      );
    }
    for (const field of [
      "seedRequired",
      "clockOrigin",
      "identityNamespace",
      "cpuCores",
      "memoryMiB",
      "diskMiB",
      "objectBytes",
      "durationSeconds",
      "cleanupSeconds",
    ]) {
      add(profile.resourceEnvelope?.[field] !== undefined, `resource ${profile.name}/${field}`);
    }
    add(
      Object.entries(expectedProfileResources[profile.name] ?? {}).every(
        ([field, value]) => profile.resourceEnvelope?.[field] === value,
      ),
      `approved resource envelope ${profile.name}`,
    );
    add(
      profile.exactDistributions?.mfaEnrollments?.["enrollment-required"] ===
        0,
      `optional TOTP distribution ${profile.name}`,
    );
    if (profile.name === "realistic" || profile.name === "stress") {
      add(
        isDeepStrictEqual(
          profile.localQualification,
          expectedLocalQualificationRevisions[profile.name],
        ),
        `owner-approved local qualification revision ${profile.name}`,
      );
      add(
        profile.evidencePurpose === "release-readiness-endurance" &&
          profile.runtimeStatus === "not run",
        `deferred full-volume endurance evidence ${profile.name}`,
      );
    }
  }

  const dictionary = contract.syntheticData ?? {};
  add(dictionary.piiAllowed === false, "PII prohibition");
  add(dictionary.secretsAllowed === false, "secret prohibition");
  add(
    (dictionary.allowedEmailDomains ?? []).every((domain) =>
      domain.endsWith(".invalid")
    ),
    "synthetic-only email domains",
  );
  for (const email of dictionary.exampleEmails ?? []) {
    const domain = email.split("@")[1] ?? "";
    add(
      dictionary.allowedEmailDomains?.includes(domain),
      `real-looking PII ${email}`,
    );
  }
  add(
    (dictionary.forbiddenSources ?? []).includes("repository-history") &&
      (dictionary.forbiddenSources ?? []).includes("logs") &&
      (dictionary.forbiddenSources ?? []).includes("customer-exports") &&
      (dictionary.forbiddenSources ?? []).includes("local-address-books"),
    "synthetic source boundaries",
  );
  for (const secretField of [
    "password",
    "totpSecret",
    "recoveryCode",
    "accessToken",
    "refreshToken",
    "privateKey",
    "clientSecret",
  ]) {
    add(
      dictionary.forbiddenFields?.includes(secretField),
      `secret field ${secretField}`,
    );
  }

  add(
    contract.loaderBoundary?.target === "whole-disposable-namespace" &&
      contract.loaderBoundary?.selectiveAppendOnlyDeletion === "forbidden" &&
      contract.loaderBoundary?.retainedControlStore === "outside-target",
    "whole-namespace cleanup boundary",
  );

  const ownerDecisions = contract.ownerDecisions ?? [];
  add(
    sameMembers(ownerDecisions.map((decision) => decision.id), expectedOwnerDecisions),
    "owner-decision package coverage",
  );
  add(
    new Set(ownerDecisions.map((decision) => decision.approvalReference))
        .size === expectedOwnerDecisions.length,
    "unique owner approval references",
  );
  for (const decision of ownerDecisions) {
    add(
      decision.status === approvedDecisionStatus,
      `owner decision status ${decision.id}`,
    );
    add(
      decision.approved === true &&
        decision.frozen === true &&
        decision.approvedAt === "2026-07-28" &&
        typeof decision.approvalReference === "string" &&
        /^OWNER-DIRECTIVE-2026-07-28-P5T1-\d{2}$/u.test(
          decision.approvalReference,
        ) &&
        decision.effectiveContractVersion === contract.contractVersion &&
        isDeepStrictEqual(
          decision.implementationValue,
          expectedOwnerImplementationValues[decision.id],
        ),
      `unresolved owner decision used as implementation assumption ${decision.id}`,
    );
    add(
      typeof decision.owner === "string" &&
        decision.owner.length > 0 &&
        Array.isArray(decision.options) &&
        decision.options.length >= 2 &&
        typeof decision.recommended === "string" &&
        decision.recommended.length > 0 &&
        typeof decision.rationale === "string" &&
        decision.rationale.length > 0 &&
        Array.isArray(decision.affectedTasks) &&
        decision.affectedTasks.length > 0 &&
        typeof decision.blockerIfUnresolved === "string" &&
        decision.blockerIfUnresolved.length > 0,
      `owner decision fields ${decision.id}`,
    );
  }

  return errors;
}

function expectRejected(base, name, mutate, expectedError) {
  const candidate = clone(base);
  mutate(candidate);
  const errors = validateContract(candidate);
  assert.ok(
    errors.some((error) => error.includes(expectedError)),
    `${name} must be rejected with ${expectedError}; got ${errors.join(", ")}`,
  );
}

function validateNormalArtifactObservation(contract, observation) {
  const errors = [];
  const normalArtifact = contract.normalArtifact;
  if (observation.publicRegistration) {
    errors.push("public registration");
  }
  for (const method of observation.authentication ?? []) {
    if (/canonical|test-subject-header|test-token-header/iu.test(method)) {
      errors.push(`canonical-header authentication: ${method}`);
    }
  }
  for (const route of observation.routes ?? []) {
    if (/seed|reset|\/__test(?:\/|$)/iu.test(route)) {
      errors.push(`normal-mode seed/reset route: ${route}`);
    }
  }
  for (const hook of observation.startupHooks ?? []) {
    if (/seed|reset|fixture|canonical/iu.test(hook)) {
      errors.push(`normal-mode seed/reset startup hook: ${hook}`);
    }
  }
  for (const command of observation.commands ?? []) {
    if (/loader|seed|reset/iu.test(command)) {
      errors.push(`normal-mode loader command: ${command}`);
    }
  }
  for (const imported of observation.importGraph ?? []) {
    if (
      normalArtifact.forbiddenImports.some(
        (forbidden) =>
          imported === forbidden || imported.startsWith(`${forbidden}/`),
      )
    ) {
      errors.push(`normal artifact forbidden import graph: ${imported}`);
    }
  }
  if (observation.clientAuthoredRoles) {
    errors.push("client-authored roles");
  }
  return errors;
}

test("Task 1 contract is machine-readable and matches canonical role and route sources", () => {
  const contract = readContract();
  assert.deepEqual(validateContract(contract), []);

  const openAPI = JSON.parse(readFileSync(openAPIPath, "utf8"));
  assert.deepEqual(openAPI.components.schemas.Role.enum, expectedRoles);
  const routes = JSON.parse(readFileSync(routeCatalogPath, "utf8"));
  assert.equal(routes.length, 86);
  assert.equal(new Set(routes.map((route) => route.auditId)).size, 86);
  assert.equal(contract.catalogs.routeCatalog.exactCount, routes.length);
});

test("contract rejects identity, artifact, privacy, coverage, and version mutations", async (t) => {
  const base = readContract();
  const mutations = [
    [
      "public registration",
      (contract) => {
        contract.registration.public = true;
      },
      "public registration",
    ],
    [
      "normal-mode seed route",
      (contract) => {
        contract.normalArtifact.forbiddenSurfaces =
          contract.normalArtifact.forbiddenSurfaces.filter(
            (surface) => surface !== "seed-route",
          );
      },
      "seed-route",
    ],
    [
      "normal-mode reset startup hook",
      (contract) => {
        contract.normalArtifact.forbiddenSurfaces =
          contract.normalArtifact.forbiddenSurfaces.filter(
            (surface) => surface !== "reset-startup-hook",
          );
      },
      "reset-startup-hook",
    ],
    [
      "normal artifact testprofile import graph",
      (contract) => {
        contract.normalArtifact.forbiddenImports =
          contract.normalArtifact.forbiddenImports.filter(
            (entry) => entry !== "apps/api/internal/testprofile",
          );
      },
      "apps/api/internal/testprofile",
    ],
    [
      "normal artifact loader import graph",
      (contract) => {
        contract.normalArtifact.forbiddenImports =
          contract.normalArtifact.forbiddenImports.filter(
            (entry) => entry !== "apps/api/internal/preproddata",
          );
      },
      "apps/api/internal/preproddata",
    ],
    [
      "normal artifact canonical-header authentication",
      (contract) => {
        contract.normalArtifact.authentication.push(
          "canonical-test-subject-header",
        );
      },
      "canonical-header authentication",
    ],
    [
      "client-authored roles",
      (contract) => {
        contract.identity.roleAuthority = "client-authored";
      },
      "client-authored roles",
    ],
    [
      "real-looking PII",
      (contract) => {
        contract.syntheticData.exampleEmails.push("john.smith@gmail.com");
      },
      "real-looking PII",
    ],
    [
      "missing role",
      (contract) => {
        contract.identity.roles.pop();
      },
      "eight-role coverage",
    ],
    [
      "missing membership lifecycle",
      (contract) => {
        contract.stateMachines.desiredMembership.states =
          contract.stateMachines.desiredMembership.states.filter(
            (state) => state !== "requested",
          );
      },
      "membership lifecycle coverage",
    ],
    [
      "missing 86-route coverage",
      (contract) => {
        contract.catalogs.routeCatalog.exactCount = 85;
      },
      "86-route coverage",
    ],
    [
      "missing exact-count manifest field",
      (contract) => {
        delete contract.dataProfiles.profiles[0].expectedCounts.organizations;
      },
      "exact count smoke/organizations",
    ],
    [
      "unversioned profile change",
      (contract) => {
        delete contract.dataProfiles.profiles[0].version;
      },
      "versioned profile smoke",
    ],
    [
      "authority mutation without membership revision",
      (contract) => {
        contract.authorityMutations[0].expectedMembershipRevision = "optional";
      },
      "expectedMembershipRevision",
    ],
    [
      "unresolved owner decision used as implemented assumption",
      (contract) => {
        contract.ownerDecisions[0].status =
          "proposed — owner decision required";
        contract.ownerDecisions[0].approved = false;
        contract.ownerDecisions[0].frozen = false;
        contract.ownerDecisions[0].approvalReference = null;
      },
      "unresolved owner decision used as implementation assumption",
    ],
  ];

  for (const [name, mutate, expectedError] of mutations) {
    await t.test(name, () => {
      expectRejected(base, name, mutate, expectedError);
    });
  }
});

test("normal artifact observation rejects runtime backdoors and client authority", async (t) => {
  const contract = readContract();
  const safeObservation = {
    publicRegistration: false,
    authentication: ["oidc", "secure-http-only-application-session"],
    routes: ["/health/live", "/v1/profile"],
    startupHooks: ["run-forward-only-migrations"],
    commands: ["api"],
    importGraph: ["apps/api/internal/httpapi", "apps/api/internal/identity"],
    clientAuthoredRoles: false,
  };
  assert.deepEqual(
    validateNormalArtifactObservation(contract, safeObservation),
    [],
  );

  const mutations = [
    [
      "public registration",
      (observation) => {
        observation.publicRegistration = true;
      },
      "public registration",
    ],
    [
      "seed route",
      (observation) => {
        observation.routes.push("/__test/reset");
      },
      "seed/reset route",
    ],
    [
      "seed startup hook",
      (observation) => {
        observation.startupHooks.push("canonical-seed-on-start");
      },
      "seed/reset startup hook",
    ],
    [
      "loader command",
      (observation) => {
        observation.commands.push("preprod-data-loader");
      },
      "loader command",
    ],
    [
      "testprofile import graph",
      (observation) => {
        observation.importGraph.push("apps/api/internal/testprofile");
      },
      "forbidden import graph",
    ],
    [
      "loader import graph",
      (observation) => {
        observation.importGraph.push(
          "apps/api/internal/preproddata/scenarios",
        );
      },
      "forbidden import graph",
    ],
    [
      "canonical-header authentication",
      (observation) => {
        observation.authentication.push("canonical-test-subject-header");
      },
      "canonical-header authentication",
    ],
    [
      "client-authored roles",
      (observation) => {
        observation.clientAuthoredRoles = true;
      },
      "client-authored roles",
    ],
  ];

  for (const [name, mutate, expectedError] of mutations) {
    await t.test(name, () => {
      const observation = clone(safeObservation);
      mutate(observation);
      const errors = validateNormalArtifactObservation(contract, observation);
      assert.ok(
        errors.some((error) => error.includes(expectedError)),
        `${name} must be rejected with ${expectedError}; got ${errors.join(", ")}`,
      );
    });
  }
});
