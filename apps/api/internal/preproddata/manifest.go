package preproddata

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/preproddata/profiles"
)

var (
	ErrInvalidIntent = errors.New("invalid preprod intent")
	digestPattern    = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	systemIDPattern  = regexp.MustCompile(`^[0-9]{10,24}$`)
	runIDPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{5,95}$`)
)

const canonicalizationContract = "AVIA_CANONICAL_JSON_V1"

type TargetFingerprint struct {
	Environment              string `json:"environment"`
	DatabaseName             string `json:"databaseName"`
	DatabaseOwner            string `json:"databaseOwner"`
	PostgresSystemIdentifier string `json:"postgresSystemIdentifier"`
	PostgresHost             string `json:"postgresHost"`
	PostgresPort             int    `json:"postgresPort"`
	ComposeProject           string `json:"composeProject"`
	KeycloakRealm            string `json:"keycloakRealm"`
	KeycloakDatabase         string `json:"keycloakDatabase"`
	KeycloakServiceClientID  string `json:"keycloakServiceClientId"`
	MailpitNamespace         string `json:"mailpitNamespace"`
	ObjectBucket             string `json:"objectBucket"`
	ObjectPrefix             string `json:"objectPrefix"`
	LoaderQueueNamespace     string `json:"loaderQueueNamespace"`
	ProfileName              string `json:"profileName"`
	ProfileVersion           string `json:"profileVersion"`
	RunID                    string `json:"runId"`
	IntentDigest             string `json:"intentDigest"`
}

func (target TargetFingerprint) Validate() error {
	switch {
	case target.Environment != "local-preprod":
		return fmt.Errorf("%w: environment must be local-preprod", ErrInvalidIntent)
	case target.DatabaseName != "aviasurveil360_local_preprod":
		return fmt.Errorf("%w: database is not on the disposable allowlist", ErrInvalidIntent)
	case target.DatabaseOwner != "aviasurveil360_preprod_loader":
		return fmt.Errorf("%w: database owner is not loader-exclusive", ErrInvalidIntent)
	case !systemIDPattern.MatchString(target.PostgresSystemIdentifier):
		return fmt.Errorf("%w: PostgreSQL system identifier is required", ErrInvalidIntent)
	case target.PostgresHost != "preprod-postgres" || target.PostgresPort != 5432:
		return fmt.Errorf("%w: PostgreSQL endpoint is not the isolated target", ErrInvalidIntent)
	case target.ComposeProject != "aviasurveil360-local-preprod":
		return fmt.Errorf("%w: Compose project is not the isolated target", ErrInvalidIntent)
	case target.KeycloakRealm != "aviasurveil360-local-preprod":
		return fmt.Errorf("%w: Keycloak realm is not isolated", ErrInvalidIntent)
	case target.KeycloakDatabase != "keycloak_local_preprod":
		return fmt.Errorf("%w: Keycloak database is not isolated", ErrInvalidIntent)
	case target.KeycloakServiceClientID !=
		"aviasurveil360-local-preprod-lifecycle":
		return fmt.Errorf("%w: lifecycle service client is not isolated", ErrInvalidIntent)
	case target.MailpitNamespace != "aviasurveil360-local-preprod":
		return fmt.Errorf("%w: Mailpit namespace is not isolated", ErrInvalidIntent)
	case target.ObjectBucket != "aviasurveil360-local-preprod":
		return fmt.Errorf("%w: object bucket is not isolated", ErrInvalidIntent)
	case target.LoaderQueueNamespace != "aviasurveil360-local-preprod":
		return fmt.Errorf("%w: loader queue namespace is not isolated", ErrInvalidIntent)
	case !runIDPattern.MatchString(target.RunID):
		return fmt.Errorf("%w: run ID is invalid", ErrInvalidIntent)
	case target.ObjectPrefix != "runs/"+target.RunID+"/":
		return fmt.Errorf("%w: object prefix is not bound to the run ID", ErrInvalidIntent)
	case target.ProfileName == "" || target.ProfileVersion == "":
		return fmt.Errorf("%w: profile identity is required", ErrInvalidIntent)
	}
	if target.IntentDigest != "" && !digestPattern.MatchString(target.IntentDigest) {
		return fmt.Errorf("%w: intent digest is malformed", ErrInvalidIntent)
	}
	return nil
}

type IntentManifest struct {
	SchemaVersion           string                      `json:"schemaVersion"`
	RunID                   string                      `json:"runId"`
	ProfileName             string                      `json:"profile"`
	ProfileVersion          string                      `json:"profileVersion"`
	SeedHash                string                      `json:"seedHash"`
	ClockOrigin             time.Time                   `json:"clockOrigin"`
	ExpectedCounts          map[string]int64            `json:"expectedCounts"`
	ExactDistributions      map[string]map[string]int64 `json:"exactDistributions"`
	CodeDigest              string                      `json:"codeDigest"`
	ContractDigest          string                      `json:"contractDigest"`
	Canonicalization        string                      `json:"canonicalization"`
	Target                  TargetFingerprint           `json:"target"`
	IntentDigest            string                      `json:"intentDigest"`
	TargetFingerprintDigest string                      `json:"targetFingerprintDigest"`
}

type IntentInput struct {
	RunID          string
	Profile        profiles.Profile
	SeedHash       string
	CodeDigest     string
	ContractDigest string
	Target         TargetFingerprint
}

func BuildIntent(input IntentInput) (IntentManifest, error) {
	if err := profiles.ValidateFrozen(input.Profile); err != nil {
		return IntentManifest{}, fmt.Errorf("%w: %v", ErrInvalidIntent, err)
	}
	if input.RunID != input.Target.RunID ||
		input.Profile.Name != input.Target.ProfileName ||
		input.Profile.Version != input.Target.ProfileVersion {
		return IntentManifest{}, fmt.Errorf(
			"%w: run/profile target binding mismatch",
			ErrInvalidIntent,
		)
	}
	if !digestPattern.MatchString(input.SeedHash) ||
		!digestPattern.MatchString(input.CodeDigest) ||
		!digestPattern.MatchString(input.ContractDigest) {
		return IntentManifest{}, fmt.Errorf(
			"%w: seed, code, and contract digests must be SHA-256",
			ErrInvalidIntent,
		)
	}
	input.Target.IntentDigest = ""
	if err := input.Target.Validate(); err != nil {
		return IntentManifest{}, err
	}
	distributions, err := completeDistributions(
		input.Profile.ExpectedCounts,
		input.Profile.ExactDistributions,
	)
	if err != nil {
		return IntentManifest{}, err
	}
	intent := IntentManifest{
		SchemaVersion: "preprod-intent-manifest/v1",
		RunID:         input.RunID, ProfileName: input.Profile.Name,
		ProfileVersion: input.Profile.Version, SeedHash: input.SeedHash,
		ClockOrigin:        input.Profile.ResourceEnvelope.ClockOrigin,
		ExpectedCounts:     cloneCounts(input.Profile.ExpectedCounts),
		ExactDistributions: distributions,
		CodeDigest:         input.CodeDigest, ContractDigest: input.ContractDigest,
		Canonicalization: canonicalizationContract,
		Target:           input.Target,
	}
	intentPayload, err := intentDigestPayload(intent)
	if err != nil {
		return IntentManifest{}, fmt.Errorf("%w: canonicalize intent: %v", ErrInvalidIntent, err)
	}
	intent.IntentDigest = sha256Digest(intentPayload)
	intent.Target.IntentDigest = intent.IntentDigest
	targetPayload, err := canonicalJSON(intent.Target)
	if err != nil {
		return IntentManifest{}, fmt.Errorf("%w: canonicalize target: %v", ErrInvalidIntent, err)
	}
	intent.TargetFingerprintDigest = sha256Digest(targetPayload)
	return intent, nil
}

func completeDistributions(
	counts map[string]int64,
	source map[string]map[string]int64,
) (map[string]map[string]int64, error) {
	output := cloneDistributionMap(source)
	for family, count := range counts {
		if count < 0 {
			return nil, fmt.Errorf("%w: negative count for %s", ErrInvalidIntent, family)
		}
		distribution, ok := output[family]
		if !ok {
			output[family] = map[string]int64{"generated": count}
			continue
		}
		var total int64
		for _, value := range distribution {
			if value < 0 {
				return nil, fmt.Errorf(
					"%w: negative distribution for %s",
					ErrInvalidIntent,
					family,
				)
			}
			total += value
		}
		if total != count {
			return nil, fmt.Errorf(
				"%w: distribution total for %s is %d, expected %d",
				ErrInvalidIntent,
				family,
				total,
				count,
			)
		}
	}
	for family := range output {
		if _, ok := counts[family]; !ok {
			return nil, fmt.Errorf(
				"%w: distribution has no count for %s",
				ErrInvalidIntent,
				family,
			)
		}
	}
	return output, nil
}

func (intent IntentManifest) Validate() error {
	if intent.SchemaVersion != "preprod-intent-manifest/v1" ||
		!runIDPattern.MatchString(intent.RunID) ||
		!digestPattern.MatchString(intent.SeedHash) ||
		!digestPattern.MatchString(intent.CodeDigest) ||
		!digestPattern.MatchString(intent.ContractDigest) ||
		intent.ClockOrigin.IsZero() ||
		intent.Canonicalization != canonicalizationContract ||
		intent.RunID != intent.Target.RunID ||
		intent.ProfileName != intent.Target.ProfileName ||
		intent.ProfileVersion != intent.Target.ProfileVersion ||
		intent.Target.IntentDigest != intent.IntentDigest ||
		!digestPattern.MatchString(intent.IntentDigest) ||
		!digestPattern.MatchString(intent.TargetFingerprintDigest) {
		return ErrInvalidIntent
	}
	if err := intent.Target.Validate(); err != nil {
		return err
	}
	profile, err := profiles.Lookup(
		intent.ProfileName,
		intent.ProfileVersion,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidIntent, err)
	}
	distributions, err := completeDistributions(
		profile.ExpectedCounts,
		profile.ExactDistributions,
	)
	if err != nil ||
		!intent.ClockOrigin.Equal(profile.ResourceEnvelope.ClockOrigin) ||
		!reflect.DeepEqual(intent.ExpectedCounts, profile.ExpectedCounts) ||
		!reflect.DeepEqual(intent.ExactDistributions, distributions) {
		return fmt.Errorf(
			"%w: profile content differs from frozen catalog",
			ErrInvalidIntent,
		)
	}
	payload, err := intentDigestPayload(intent)
	if err != nil || sha256Digest(payload) != intent.IntentDigest {
		return fmt.Errorf("%w: content digest mismatch", ErrInvalidIntent)
	}
	targetPayload, err := canonicalJSON(intent.Target)
	if err != nil || sha256Digest(targetPayload) != intent.TargetFingerprintDigest {
		return fmt.Errorf("%w: target fingerprint digest mismatch", ErrInvalidIntent)
	}
	return nil
}

func intentDigestPayload(intent IntentManifest) ([]byte, error) {
	intent.IntentDigest = ""
	intent.TargetFingerprintDigest = ""
	intent.Target.IntentDigest = ""
	return canonicalJSON(intent)
}

type ResultManifest struct {
	SchemaVersion       string            `json:"schemaVersion"`
	RunID               string            `json:"runId"`
	IntentDigest        string            `json:"intentDigest"`
	ActualCounts        map[string]int64  `json:"actualCounts"`
	RelationshipDigests map[string]string `json:"relationshipDigests"`
	Checkpoints         []string          `json:"checkpoints"`
	Failures            []string          `json:"failures"`
	Outcome             string            `json:"outcome"`
	CompletedAt         time.Time         `json:"completedAt"`
	ResultDigest        string            `json:"resultDigest"`
}

type ResultInput struct {
	RunID               string
	IntentDigest        string
	ActualCounts        map[string]int64
	RelationshipDigests map[string]string
	Checkpoints         []string
	Failures            []string
	CompletedAt         time.Time
}

func BuildResult(input ResultInput) (ResultManifest, error) {
	if !runIDPattern.MatchString(input.RunID) ||
		!digestPattern.MatchString(input.IntentDigest) ||
		input.CompletedAt.IsZero() {
		return ResultManifest{}, fmt.Errorf("invalid run result identity")
	}
	for family, digest := range input.RelationshipDigests {
		if strings.TrimSpace(family) == "" || !digestPattern.MatchString(digest) {
			return ResultManifest{}, fmt.Errorf(
				"invalid relationship digest for %q",
				family,
			)
		}
	}
	for family, count := range input.ActualCounts {
		if strings.TrimSpace(family) == "" || count < 0 {
			return ResultManifest{}, fmt.Errorf(
				"invalid actual count for %q",
				family,
			)
		}
	}
	outcome := "SUCCEEDED"
	if len(input.Failures) > 0 {
		outcome = "FAILED"
	}
	result := ResultManifest{
		SchemaVersion: "preprod-run-result/v1", RunID: input.RunID,
		IntentDigest:        input.IntentDigest,
		ActualCounts:        cloneCounts(input.ActualCounts),
		RelationshipDigests: cloneStrings(input.RelationshipDigests),
		Checkpoints:         append([]string(nil), input.Checkpoints...),
		Failures:            append([]string(nil), input.Failures...),
		Outcome:             outcome, CompletedAt: input.CompletedAt.UTC(),
	}
	payload, err := resultDigestPayload(result)
	if err != nil {
		return ResultManifest{}, err
	}
	result.ResultDigest = sha256Digest(payload)
	return result, nil
}

func (result ResultManifest) Validate() error {
	if result.SchemaVersion != "preprod-run-result/v1" ||
		!runIDPattern.MatchString(result.RunID) ||
		!digestPattern.MatchString(result.IntentDigest) ||
		!digestPattern.MatchString(result.ResultDigest) {
		return fmt.Errorf("invalid run result")
	}
	if (result.Outcome == "SUCCEEDED" && len(result.Failures) != 0) ||
		(result.Outcome == "FAILED" && len(result.Failures) == 0) ||
		(result.Outcome != "SUCCEEDED" && result.Outcome != "FAILED") {
		return fmt.Errorf("run result outcome does not match failures")
	}
	for family, count := range result.ActualCounts {
		if strings.TrimSpace(family) == "" || count < 0 {
			return fmt.Errorf("invalid run result count")
		}
	}
	for family, digest := range result.RelationshipDigests {
		if strings.TrimSpace(family) == "" || !digestPattern.MatchString(digest) {
			return fmt.Errorf("invalid run result relationship digest")
		}
	}
	payload, err := resultDigestPayload(result)
	if err != nil || sha256Digest(payload) != result.ResultDigest {
		return fmt.Errorf("run result digest mismatch")
	}
	return nil
}

func resultDigestPayload(result ResultManifest) ([]byte, error) {
	result.ResultDigest = ""
	return canonicalJSON(result)
}

type Checkpoint struct {
	SchemaVersion   string    `json:"schemaVersion"`
	RunID           string    `json:"runId"`
	IntentDigest    string    `json:"intentDigest"`
	Sequence        int64     `json:"sequence"`
	Name            string    `json:"name"`
	AppliedCommands int64     `json:"appliedCommands"`
	LastOperationID string    `json:"lastOperationId,omitempty"`
	RecordedAt      time.Time `json:"recordedAt"`
}

type CleanupAttestation struct {
	SchemaVersion     string    `json:"schemaVersion"`
	RunID             string    `json:"runId"`
	IntentDigest      string    `json:"intentDigest"`
	ResultDigest      string    `json:"resultDigest"`
	TargetDigest      string    `json:"targetFingerprintDigest"`
	AuthorizationHash string    `json:"authorizationHash"`
	CleanedAt         time.Time `json:"cleanedAt"`
	AttestationDigest string    `json:"attestationDigest"`
}

type CleanupAttestationInput struct {
	RunID             string
	IntentDigest      string
	ResultDigest      string
	TargetDigest      string
	AuthorizationHash string
	CleanedAt         time.Time
}

func BuildCleanupAttestation(
	input CleanupAttestationInput,
) (CleanupAttestation, error) {
	attestation := CleanupAttestation{
		SchemaVersion:     "preprod-cleanup-attestation/v1",
		RunID:             input.RunID,
		IntentDigest:      input.IntentDigest,
		ResultDigest:      input.ResultDigest,
		TargetDigest:      input.TargetDigest,
		AuthorizationHash: input.AuthorizationHash,
		CleanedAt:         input.CleanedAt.UTC(),
	}
	if !runIDPattern.MatchString(attestation.RunID) ||
		!digestPattern.MatchString(attestation.IntentDigest) ||
		!digestPattern.MatchString(attestation.ResultDigest) ||
		!digestPattern.MatchString(attestation.TargetDigest) ||
		!digestPattern.MatchString(attestation.AuthorizationHash) ||
		attestation.CleanedAt.IsZero() {
		return CleanupAttestation{}, fmt.Errorf(
			"invalid cleanup attestation input",
		)
	}
	payload, err := canonicalJSON(attestation)
	if err != nil {
		return CleanupAttestation{}, err
	}
	attestation.AttestationDigest = sha256Digest(payload)
	return attestation, nil
}

func sha256Digest(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func cloneCounts(source map[string]int64) map[string]int64 {
	output := make(map[string]int64, len(source))
	for key, value := range source {
		output[key] = value
	}
	return output
}

func cloneStrings(source map[string]string) map[string]string {
	output := make(map[string]string, len(source))
	for key, value := range source {
		output[key] = value
	}
	return output
}

func cloneDistributionMap(
	source map[string]map[string]int64,
) map[string]map[string]int64 {
	output := make(map[string]map[string]int64, len(source))
	for family, distribution := range source {
		output[family] = cloneCounts(distribution)
	}
	return output
}
