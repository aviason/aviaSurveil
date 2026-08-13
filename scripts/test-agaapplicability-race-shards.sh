#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
API_ROOT="${REPOSITORY_ROOT}/apps/api"

# Keep the heavy immutable-graph race tests in bounded shards. Running this
# package inside the repository-wide race command can exceed its wall budget.
export GOMODCACHE="/private/tmp/avia-aga-race-modcache"
export GOPATH="/private/tmp/avia-aga-race-gopath"
export GOCACHE="/private/tmp/avia-aga-race-gocache"
export GOTMPDIR="/private/tmp/avia-aga-race-tmp"
mkdir -p "${GOTMPDIR}"

EXPECTED_TEST_COUNT=45
test_count="$(go -C "${API_ROOT}" test -list '^Test' ./internal/agaapplicability | awk '$1 ~ /^Test/ { count++ } END { print count + 0 }')"
if [[ "${test_count}" != "${EXPECTED_TEST_COUNT}" ]]; then
  echo "agaapplicability test inventory changed: found ${test_count}, expected ${EXPECTED_TEST_COUNT}; update the shard map" >&2
  exit 1
fi

run_shard() {
  local name="$1"
  local timeout="$2"
  local pattern="$3"

  printf '\n=== race shard: %s (%s) ===\n' "${name}" "${timeout}"
  go -C "${API_ROOT}" test \
    -race \
    -p 1 \
    -count=1 \
    -timeout "${timeout}" \
    -v \
    ./internal/agaapplicability \
    -run "${pattern}"
  printf 'verified locally: race shard %s\n' "${name}"
}

run_shard "authority-inputs" "360s" \
  '^(TestFrozenAuthorityPins|TestFrozenTaxonomyReconstructsAuthoritySelfDigest|TestModelDescriptorAcceptsTruthfulPlatformUnavailableMetadata|TestClassifySealedBaseContract|TestPassInputsAndSealsBindRoleRunModelAndAcceptedInventory|TestPassInputAndReceiptUseFrozenClosedPreimages|TestFrozen25BatchClassificationManifestRuntimePin)$'

run_shard "classification-boundaries" "360s" \
  '^(TestCandidateAndChallengeRequireRoleNeutralPrivateSnapshots|TestClassificationDerivesEvidenceFactsFromPrivateInput|TestDerivedResearchFactUsesFrozenDomain|TestTaxonomyDiagnosticsNeverEchoUntrustedValues|TestClassificationResultJSONRoundTripIsTextFreeAndDraftable|TestClassificationResultHandoffRejectsUnknownAndBodyFields|TestSealedClassificationItemRejectsSemanticDigestTampering)$'

run_shard "sealed-classification" "360s" \
  '^(TestClassificationDiagnosticsNeverEchoUntrustedPassRole|TestSealedClassificationItemJSONAndSafeDigestBoundary|TestClassificationFatalErrorsAbortRun|TestConfidenceRecommendationPrecedence|TestConfidenceEvidenceBindsEveryProposal|TestExternalInvolvementEdgesAreIndependentAndOptional|TestPassProposalRecordsAreCompleteAndResolvable|TestClassificationDigestGraphIsNonCircular)$'

run_shard "draft-core" "360s" \
  '^(TestProjectionNormalizationIsDeterministicAndOptionalSetsStayArrays|TestDraftUnknownActionDiagnosticDoesNotEchoUntrustedAction|TestDraftCommandsCreateImmutableSuccessors|TestSealedBaseStateBindsImmutableProjectionAndGlobalOrder|TestDraftBatchRejectsInactiveResetSupersededAndUnsealed|TestDraftSemanticEditDemotesAutoPreselection|TestDraftResolvesEveryProposalFamily|TestQuestionReferenceUnionIsClosed|TestValidatedRuntimeIncludeMatchesFullCommandSemantics)$'

run_shard "runtime-public-roundtrip" "360s" \
  '^TestValidatedRuntimeIncludesSurviveFullScalePublicRoundTrip$'

run_shard "runtime-hydration" "420s" \
  '^TestHydrateDraftForRuntimeRestoresPrivateStateAfterPublicRoundTrip$'

run_shard "workspace-identity" "360s" \
  '^(TestAddAllocatesFreshWorkspaceRootVersionAndProposal|TestDraftRewordReplacesCurrentLeaf|TestWorkspaceQuestionIdentityRejectsAliases|TestQuestionSnapshotReconstructsExactLeaves)$'

run_shard "batch-readiness" "360s" \
  '^(TestDraftBatchPreviewIsAtomic|TestDraftRequiresResolvableSealedPasses|TestDraftReadinessExhaustsExactBaseAndPreservesUnchangedSourceGapConfidence)$'

run_shard "recommendation-current-leaf" "420s" \
  '^TestRecommendRequiresCurrentIncludedLeaf$'

run_shard "recommendation-derived-facts" "420s" \
  '^TestRecommendationRequiresServerDerivedFacts$'

run_shard "recommendation-kind-profile" "420s" \
  '^TestRecommendationRejectsKindProfileMismatch$'

run_shard "recommendation-ambiguous-leaf" "420s" \
  '^TestRecommendationRejectsAmbiguousQuestionLeafGraph$'

run_shard "recommendation-readiness-snapshot" "420s" \
  '^TestRecommendationSnapshotPinsReadiness$'

printf '\nverified locally: agaapplicability race suite, %s tests across 13 shards\n' "${EXPECTED_TEST_COUNT}"
