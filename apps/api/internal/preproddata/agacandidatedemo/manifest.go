package agacandidatedemo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

var (
	runIDPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{5,95}$`)
	systemIDPattern = regexp.MustCompile(`^[0-9]{10,24}$`)
)

const overlayCanonicalization = "AVIA_AGA_CANDIDATE_DEMO_CANONICAL_V1"

type TargetFingerprint struct {
	Environment              string `json:"environment"`
	DatabaseName             string `json:"databaseName"`
	DatabaseOwner            string `json:"databaseOwner"`
	PostgresSystemIdentifier string `json:"postgresSystemIdentifier"`
	PostgresHost             string `json:"postgresHost"`
	PostgresPort             int    `json:"postgresPort"`
	ComposeProject           string `json:"composeProject"`
	OverlaySchema            string `json:"overlaySchema"`
}

func (target TargetFingerprint) Validate() error {
	if target.Environment != "local-preprod" || target.DatabaseName != "aviasurveil360_local_preprod" || target.DatabaseOwner != "aviasurveil360_preprod_loader" || !systemIDPattern.MatchString(target.PostgresSystemIdentifier) || target.PostgresHost != "preprod-postgres" || target.PostgresPort != 5432 || target.ComposeProject != "aviasurveil360-local-preprod" || target.OverlaySchema != "preprod_aga_demo" {
		return fmt.Errorf("invalid AGA demo target fingerprint")
	}
	return nil
}

type IntentManifest struct {
	SchemaVersion               string                      `json:"schemaVersion"`
	RunID                       string                      `json:"runId"`
	Operation                   string                      `json:"operation"`
	BaseRunID                   string                      `json:"baseRunId"`
	BaseIntentDigest            string                      `json:"baseIntentDigest"`
	BaseResultDigest            string                      `json:"baseResultDigest"`
	BaseTargetDigest            string                      `json:"baseTargetFingerprintDigest"`
	PackageZipDigest            string                      `json:"packageZipDigest"`
	PackageJSONDigest           string                      `json:"packageJsonDigest"`
	PackageManifestDigest       string                      `json:"packageManifestDigest"`
	ExpectedCounts              map[string]int64            `json:"expectedCounts"`
	ExpectedDistributions       map[string]map[string]int64 `json:"expectedDistributions"`
	ExpectedRelationshipDigests map[string]string           `json:"expectedRelationshipDigests"`
	CanonicalizationContract    string                      `json:"canonicalizationContract"`
	CodeDigest                  string                      `json:"codeDigest"`
	ContractDigest              string                      `json:"contractDigest"`
	Target                      TargetFingerprint           `json:"target"`
	CreatedAt                   time.Time                   `json:"createdAt"`
	IntentDigest                string                      `json:"intentDigest"`
	TargetFingerprintDigest     string                      `json:"targetFingerprintDigest"`
}

type IntentInput struct {
	RunID                       string
	BaseRunID                   string
	BaseIntentDigest            string
	BaseResultDigest            string
	BaseTargetDigest            string
	CodeDigest                  string
	ContractDigest              string
	ExpectedPackage             ExpectedPackage
	ExpectedRelationshipDigests map[string]string
	Target                      TargetFingerprint
	CreatedAt                   time.Time
}

func BuildIntent(input IntentInput) (IntentManifest, error) {
	if !runIDPattern.MatchString(input.RunID) || !runIDPattern.MatchString(input.BaseRunID) || len(input.ExpectedRelationshipDigests) == 0 || input.CreatedAt.IsZero() || !validDigest(input.BaseIntentDigest) || !validDigest(input.BaseResultDigest) || !validDigest(input.BaseTargetDigest) || !validDigest(input.CodeDigest) || !validDigest(input.ContractDigest) || !validDigest(input.ExpectedPackage.ZipSHA256) || !validDigest(input.ExpectedPackage.JSONSHA256) || !validDigest(input.ExpectedPackage.ManifestSHA256) {
		return IntentManifest{}, fmt.Errorf("invalid AGA demo intent input")
	}
	if err := input.Target.Validate(); err != nil {
		return IntentManifest{}, err
	}
	intent := IntentManifest{
		SchemaVersion: "preprod-aga-candidate-demo-intent/v1", RunID: input.RunID,
		Operation: "LOAD_AGA_CANDIDATE_DEMO_OVERLAY", BaseRunID: input.BaseRunID,
		BaseIntentDigest: input.BaseIntentDigest, BaseResultDigest: input.BaseResultDigest, BaseTargetDigest: input.BaseTargetDigest,
		PackageZipDigest: input.ExpectedPackage.ZipSHA256, PackageJSONDigest: input.ExpectedPackage.JSONSHA256, PackageManifestDigest: input.ExpectedPackage.ManifestSHA256,
		ExpectedCounts: expectedCounts(input.ExpectedPackage.ExpectedCounts), ExpectedDistributions: expectedDistributions(input.ExpectedPackage), ExpectedRelationshipDigests: cloneDigestMap(input.ExpectedRelationshipDigests),
		CanonicalizationContract: overlayCanonicalization, CodeDigest: input.CodeDigest, ContractDigest: input.ContractDigest, Target: input.Target, CreatedAt: input.CreatedAt.UTC(),
	}
	payload, err := intentPayload(intent)
	if err != nil {
		return IntentManifest{}, err
	}
	intent.IntentDigest = digestBytes(payload)
	targetPayload, err := json.Marshal(intent.Target)
	if err != nil {
		return IntentManifest{}, err
	}
	intent.TargetFingerprintDigest = digestBytes(targetPayload)
	return intent, intent.Validate()
}

func (intent IntentManifest) Validate() error {
	if intent.SchemaVersion != "preprod-aga-candidate-demo-intent/v1" || intent.Operation != "LOAD_AGA_CANDIDATE_DEMO_OVERLAY" || !runIDPattern.MatchString(intent.RunID) || !runIDPattern.MatchString(intent.BaseRunID) || len(intent.ExpectedRelationshipDigests) == 0 || intent.CreatedAt.IsZero() || intent.CanonicalizationContract != overlayCanonicalization || !validDigest(intent.BaseIntentDigest) || !validDigest(intent.BaseResultDigest) || !validDigest(intent.BaseTargetDigest) || !validDigest(intent.PackageZipDigest) || !validDigest(intent.PackageJSONDigest) || !validDigest(intent.PackageManifestDigest) || !validDigest(intent.CodeDigest) || !validDigest(intent.ContractDigest) || !validDigest(intent.IntentDigest) || !validDigest(intent.TargetFingerprintDigest) {
		return fmt.Errorf("invalid AGA demo intent")
	}
	if err := intent.Target.Validate(); err != nil {
		return err
	}
	for _, digest := range intent.ExpectedRelationshipDigests {
		if !validDigest(digest) {
			return fmt.Errorf("invalid AGA demo relationship digest")
		}
	}
	payload, err := intentPayload(intent)
	if err != nil || digestBytes(payload) != intent.IntentDigest {
		return fmt.Errorf("AGA demo intent digest mismatch")
	}
	targetPayload, err := json.Marshal(intent.Target)
	if err != nil || digestBytes(targetPayload) != intent.TargetFingerprintDigest {
		return fmt.Errorf("AGA demo target fingerprint digest mismatch")
	}
	return nil
}

func expectedCounts(counts ExpectedCounts) map[string]int64 {
	return map[string]int64{"forms": int64(counts.Forms), "formsWithCandidateBoundaries": int64(counts.FormsWithCandidateBoundaries), "questions": int64(counts.Questions), "questionsWithProposals": int64(counts.QuestionsWithProposals), "unmappedQuestions": int64(counts.UnmappedQuestions), "questionSourceProposalLinks": int64(counts.QuestionSourceProposalLinks), "formSourceProposalLinks": int64(counts.FormSourceProposalLinks), "uniqueSourceReferences": int64(counts.UniqueSourceReferences), "expertRiskReviewBlockers": int64(counts.ExpertRiskReviewBlockers)}
}
func expectedDistributions(expected ExpectedPackage) map[string]map[string]int64 {
	output := map[string]map[string]int64{}
	for name, values := range map[string]map[string]int{"questionProposedRiskBands": expected.RiskBands, "formProposedRiskBands": expected.FormRiskBands} {
		converted := map[string]int64{}
		for key, value := range values {
			converted[key] = int64(value)
		}
		output[name] = converted
	}
	return output
}
func cloneDigestMap(input map[string]string) map[string]string {
	output := map[string]string{}
	for key, value := range input {
		output[key] = value
	}
	return output
}
func intentPayload(intent IntentManifest) ([]byte, error) {
	intent.IntentDigest = ""
	intent.TargetFingerprintDigest = ""
	return json.Marshal(intent)
}
func digestBytes(content []byte) string {
	digest := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(digest[:])
}
func validDigest(value string) bool { return digestPattern.MatchString(value) }
