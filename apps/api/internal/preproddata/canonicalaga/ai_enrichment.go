package canonicalaga

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

const (
	aiRecommendationSchemaVersion        = "aga-ai-checklist-recommendations/v1"
	aiRecommendationPolicyVersion        = "AGA_AI_RECOMMENDATION_POLICY_V1"
	aiRecommendationCatalogVersion       = "aga-approved-source@2.0.0"
	aiRecommendationRootDigest           = "sha256:972f06005ba7befecb480d477334ea8cee542555d39d2604607f082deeee6e48"
	aiRecommendationPackageJSONDigest    = "sha256:57abb7b87ce91dc7383fb3b24426ccd542811ce979f33c6b17bcf938c3907973"
	aiRecommendationSourceManifestDigest = "sha256:53679bd6eccb77b2d4bf1c909cb16a3b925ca1433584d206daeaf212031877f8"
	aiRecommendationQuestionCount        = 1310
)

type AIRecommendationSourceCatalog struct {
	CatalogVersion       string `json:"catalogVersion"`
	PackageVersion       string `json:"packageVersion"`
	PackageZipSHA256     string `json:"packageZipSha256"`
	PackageJSONSHA256    string `json:"packageJsonSha256"`
	SourceManifestSHA256 string `json:"sourceManifestSha256"`
	CatalogRootDigest    string `json:"catalogRootDigest"`
	QuestionCount        int    `json:"questionCount"`
}

type AIRecommendationModelRun struct {
	ClassificationRunID     string   `json:"classificationRunId"`
	ClassificationRunDigest string   `json:"classificationRunDigest"`
	PromptDigest            string   `json:"promptDigest"`
	TaxonomyVersion         string   `json:"taxonomyVersion"`
	TaxonomyDigest          string   `json:"taxonomyDigest"`
	InputDigest             string   `json:"inputDigest"`
	ModelDescriptorDigests  []string `json:"modelDescriptorDigests"`
}

type AIRecommendationRiskPolicy struct {
	Tier             string `json:"tier"`
	RecurrenceMonths int    `json:"recurrenceMonths"`
	DefaultBucket    string `json:"defaultBucket"`
}

type AIRecommendationPolicy struct {
	Version                    string                                `json:"version"`
	UnknownRiskNeverSuppressed bool                                  `json:"unknownRiskNeverSuppressed"`
	PriorAuditEvidenceRequired bool                                  `json:"priorAuditEvidenceRequiredForDeferral"`
	RuntimeModelCalls          bool                                  `json:"runtimeModelCalls"`
	RiskBands                  map[string]AIRecommendationRiskPolicy `json:"riskBands"`
	Explanation                string                                `json:"explanation"`
}

type AIRecommendationItem struct {
	QuestionVersionID                       string   `json:"questionVersionId"`
	FormCode                                string   `json:"formCode"`
	ProposalID                              string   `json:"proposalId"`
	Ordinal                                 int      `json:"ordinal"`
	QuestionDigest                          string   `json:"questionDigest"`
	SourceLocator                           string   `json:"sourceLocator"`
	DomainCode                              string   `json:"domainCode"`
	TopicCodes                              []string `json:"topicCodes"`
	InspectionTypeCodes                     []string `json:"inspectionTypeCodes"`
	InspectionProfileCodes                  []string `json:"inspectionProfileCodes"`
	ApplicabilityDisposition                string   `json:"applicabilityDisposition"`
	RiskBand                                string   `json:"riskBand"`
	RiskTier                                string   `json:"riskTier"`
	SafetyCritical                          bool     `json:"safetyCritical"`
	AgreementConfidence                     string   `json:"agreementConfidence"`
	SourceClassificationRecommendationState string   `json:"sourceClassificationRecommendationState"`
	AdvisoryState                           string   `json:"advisoryState"`
	DefaultRecommendationBucket             string   `json:"defaultRecommendationBucket"`
	RecurrenceMonths                        int      `json:"recurrenceMonths"`
	RationaleCodes                          []string `json:"rationaleCodes"`
	ExternalApplicabilityUnresolved         bool     `json:"externalApplicabilityUnresolved"`
	RiskSignalSource                        string   `json:"riskSignalSource"`
}

type AIRecommendationArtifact struct {
	SchemaVersion        string                        `json:"schemaVersion"`
	Status               string                        `json:"status"`
	AdvisoryOnly         bool                          `json:"advisoryOnly"`
	GeneratedBy          string                        `json:"generatedBy"`
	SourceCatalog        AIRecommendationSourceCatalog `json:"sourceCatalog"`
	ModelRun             AIRecommendationModelRun      `json:"modelRun"`
	RecommendationPolicy AIRecommendationPolicy        `json:"recommendationPolicy"`
	Items                []AIRecommendationItem        `json:"items"`
	ItemCount            int                           `json:"itemCount"`
	ArtifactDigest       string                        `json:"artifactDigest"`
}

type AIEnrichmentLoadResult struct {
	CatalogID      string
	ItemCount      int
	ArtifactDigest string
}

type aiEnrichmentRow struct {
	QuestionVersionID               string
	ArtifactVersion                 string
	ArtifactDigest                  string
	SourceCatalogRootDigest         string
	ClassificationRunID             string
	ClassificationRunDigest         string
	PromptDigest                    string
	TaxonomyVersion                 string
	TaxonomyDigest                  string
	RecommendationPolicyVersion     string
	DomainCode                      string
	TopicCodes                      []string
	InspectionTypeCodes             []string
	InspectionProfileCodes          []string
	ApplicabilityDisposition        string
	RiskBand                        string
	RiskTier                        string
	SafetyCritical                  bool
	AgreementConfidence             string
	AdvisoryState                   string
	DefaultRecommendationBucket     string
	RecurrenceMonths                int
	RationaleCodes                  []string
	ExternalApplicabilityUnresolved bool
}

func ReadAIRecommendationArtifact(path string) (AIRecommendationArtifact, string, error) {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return AIRecommendationArtifact{}, "", errors.New("AI recommendation artifact path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() < 2 || info.Size() > 8<<20 {
		return AIRecommendationArtifact{}, "", errors.New("AI recommendation artifact must be a bounded regular non-symlink file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return AIRecommendationArtifact{}, "", fmt.Errorf("read AI recommendation artifact: %w", err)
	}
	fileDigest := aiSHA256Digest(data)
	var artifact AIRecommendationArtifact
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&artifact); err != nil {
		return AIRecommendationArtifact{}, "", fmt.Errorf("decode AI recommendation artifact: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return AIRecommendationArtifact{}, "", errors.New("AI recommendation artifact contains trailing data")
	}
	if err := ValidateAIRecommendationArtifact(artifact); err != nil {
		return AIRecommendationArtifact{}, "", err
	}
	return artifact, fileDigest, nil
}

func ValidateAIRecommendationArtifact(artifact AIRecommendationArtifact) error {
	if artifact.SchemaVersion != aiRecommendationSchemaVersion || artifact.Status != "SEALED" || !artifact.AdvisoryOnly || artifact.GeneratedBy != "CODEX_OFFLINE_AI" {
		return errors.New("AI recommendation artifact envelope is invalid")
	}
	source := artifact.SourceCatalog
	if source.CatalogVersion != aiRecommendationCatalogVersion || source.PackageVersion != "AGA_ALL_FORMS_APPROVED_SOURCE_V2" || source.PackageJSONSHA256 != aiRecommendationPackageJSONDigest || source.SourceManifestSHA256 != aiRecommendationSourceManifestDigest || source.CatalogRootDigest != aiRecommendationRootDigest || source.QuestionCount != aiRecommendationQuestionCount || !validSHA256(source.PackageZipSHA256) {
		return errors.New("AI recommendation artifact source binding is invalid")
	}
	model := artifact.ModelRun
	if strings.TrimSpace(model.ClassificationRunID) == "" || !validSHA256(model.ClassificationRunDigest) || !validSHA256(model.PromptDigest) || strings.TrimSpace(model.TaxonomyVersion) == "" || !validSHA256(model.TaxonomyDigest) || !validSHA256(model.InputDigest) || len(model.ModelDescriptorDigests) == 0 {
		return errors.New("AI recommendation artifact model binding is invalid")
	}
	for _, digest := range model.ModelDescriptorDigests {
		if !validSHA256(digest) {
			return errors.New("AI recommendation artifact model descriptor digest is invalid")
		}
	}
	policy := artifact.RecommendationPolicy
	if policy.Version != aiRecommendationPolicyVersion || !policy.UnknownRiskNeverSuppressed || !policy.PriorAuditEvidenceRequired || policy.RuntimeModelCalls || strings.TrimSpace(policy.Explanation) == "" || len(policy.RiskBands) != 4 {
		return errors.New("AI recommendation policy binding is invalid")
	}
	expectedPolicies := map[string]AIRecommendationRiskPolicy{
		"PROPOSED_SAFETY_CRITICAL":   {Tier: "HIGH", RecurrenceMonths: 12, DefaultBucket: "SUGGESTED_NOW"},
		"PROPOSED_HIGH_OPERATIONAL":  {Tier: "MEDIUM", RecurrenceMonths: 18, DefaultBucket: "MATCHING_OPTIONAL"},
		"PROPOSED_CONTROL_ASSURANCE": {Tier: "LOW", RecurrenceMonths: 24, DefaultBucket: "MATCHING_OPTIONAL"},
		"PROPOSED_REVIEW_REQUIRED":   {Tier: "UNKNOWN", RecurrenceMonths: 12, DefaultBucket: "UNCERTAIN_SIGNAL"},
	}
	for band, expected := range expectedPolicies {
		if actual, ok := policy.RiskBands[band]; !ok || actual != expected {
			return errors.New("AI recommendation policy risk band drifted")
		}
	}
	if artifact.ItemCount != aiRecommendationQuestionCount || len(artifact.Items) != aiRecommendationQuestionCount || !validSHA256(artifact.ArtifactDigest) {
		return errors.New("AI recommendation artifact count or digest is invalid")
	}
	seen := make(map[string]struct{}, len(artifact.Items))
	for _, item := range artifact.Items {
		if !strings.HasPrefix(item.QuestionVersionID, "qv:aga-approved-source-v2:") || item.FormCode == "" || item.ProposalID == "" || item.Ordinal < 1 || !validSHA256(item.QuestionDigest) || item.DomainCode == "" || item.ApplicabilityDisposition == "" || item.AgreementConfidence == "" || item.AdvisoryState == "" || item.DefaultRecommendationBucket == "" || item.RiskSignalSource == "" || len(item.InspectionTypeCodes) == 0 {
			return errors.New("AI recommendation item identity is invalid")
		}
		if _, exists := seen[item.QuestionVersionID]; exists {
			return errors.New("AI recommendation item identity is duplicated")
		}
		seen[item.QuestionVersionID] = struct{}{}
		expected, ok := expectedPolicies[item.RiskBand]
		if !ok || item.RiskTier != expected.Tier || item.AdvisoryState != expected.DefaultBucket || item.DefaultRecommendationBucket != expected.DefaultBucket || item.RecurrenceMonths != expected.RecurrenceMonths || (item.RiskBand == "PROPOSED_SAFETY_CRITICAL" && !item.SafetyCritical) {
			return errors.New("AI recommendation item policy values are invalid")
		}
		if item.AgreementConfidence != "HIGH" && item.AgreementConfidence != "MEDIUM" && item.AgreementConfidence != "LOW" {
			return errors.New("AI recommendation item confidence is invalid")
		}
		if !sortedUnique(item.TopicCodes) || !sortedUnique(item.InspectionTypeCodes) || !sortedUnique(item.InspectionProfileCodes) || !sortedUnique(item.RationaleCodes) {
			return errors.New("AI recommendation item set fields are not normalized")
		}
	}
	if len(seen) != aiRecommendationQuestionCount {
		return errors.New("AI recommendation item bijection is incomplete")
	}
	if got := artifactDigest(artifact); got != artifact.ArtifactDigest {
		return errors.New("AI recommendation artifact digest mismatch")
	}
	return nil
}

func LoadAIRecommendationEnrichment(ctx context.Context, pool *database.Pool, artifact AIRecommendationArtifact, catalogVersion string, advisoryLockKey int64, now time.Time) (AIEnrichmentLoadResult, error) {
	if pool == nil || advisoryLockKey <= 0 || strings.TrimSpace(catalogVersion) == "" {
		return AIEnrichmentLoadResult{}, errors.New("AI enrichment loader requires database, catalog version, and advisory lock")
	}
	if err := ValidateAIRecommendationArtifact(artifact); err != nil {
		return AIEnrichmentLoadResult{}, err
	}
	if catalogVersion != artifact.SourceCatalog.CatalogVersion {
		return AIEnrichmentLoadResult{}, errors.New("AI enrichment catalog version does not match the artifact")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result := AIEnrichmentLoadResult{CatalogID: "catalog:" + catalogVersion, ItemCount: len(artifact.Items), ArtifactDigest: artifact.ArtifactDigest}
	err := database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, advisoryLockKey); err != nil {
			return fmt.Errorf("lock AI enrichment import: %w", err)
		}
		var root string
		var status, origin string
		var questionCount int
		if err := tx.QueryRow(ctx, `SELECT catalog_root_digest,status,source_origin,question_count FROM canonical_question_catalogs WHERE id=$1`, result.CatalogID).Scan(&root, &status, &origin, &questionCount); err != nil {
			return fmt.Errorf("read approved catalog for AI enrichment: %w", err)
		}
		if root != artifact.SourceCatalog.CatalogRootDigest || status != "SEALED" || origin != "IMPORTED_APPROVED_SOURCE" || questionCount != aiRecommendationQuestionCount {
			return errors.New("AI enrichment catalog binding is not the exact sealed approved catalog")
		}
		membershipDigests := make(map[string]string, aiRecommendationQuestionCount)
		rows, err := tx.Query(ctx, `SELECT question_version_id,question_digest FROM canonical_question_catalog_memberships WHERE catalog_id=$1 ORDER BY question_version_id`, result.CatalogID)
		if err != nil {
			return fmt.Errorf("read AI enrichment memberships: %w", err)
		}
		for rows.Next() {
			var id, digest string
			if err := rows.Scan(&id, &digest); err != nil {
				rows.Close()
				return fmt.Errorf("scan AI enrichment membership: %w", err)
			}
			membershipDigests[id] = digest
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("read AI enrichment membership rows: %w", err)
		}
		rows.Close()
		if len(membershipDigests) != aiRecommendationQuestionCount {
			return errors.New("AI enrichment membership count is not 1,310")
		}
		var existing int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM canonical_question_catalog_ai_enrichments WHERE catalog_id=$1`, result.CatalogID).Scan(&existing); err != nil {
			return fmt.Errorf("count AI enrichment rows: %w", err)
		}
		if existing != 0 {
			if existing != aiRecommendationQuestionCount {
				return errors.New("partial AI enrichment import exists; refusing repair or overwrite")
			}
			return verifyAIEnrichmentReplay(ctx, tx, result.CatalogID, artifact, membershipDigests)
		}
		for _, item := range artifact.Items {
			if digest, ok := membershipDigests[item.QuestionVersionID]; !ok || digest != item.QuestionDigest {
				return fmt.Errorf("AI enrichment question identity or digest does not match catalog: %s", item.QuestionVersionID)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO canonical_question_catalog_ai_enrichments (
					catalog_id,question_version_id,artifact_version,artifact_digest,source_catalog_root_digest,
					classification_run_id,classification_run_digest,prompt_digest,taxonomy_version,taxonomy_digest,
					recommendation_policy_version,domain_code,topic_codes,inspection_type_codes,inspection_profile_codes,
					applicability_disposition,risk_band,risk_tier,safety_critical,agreement_confidence,
				advisory_state,default_recommendation_bucket,recurrence_months,rationale_codes,
					external_applicability_unresolved,loaded_at
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)`,
				result.CatalogID, item.QuestionVersionID, artifact.SchemaVersion, artifact.ArtifactDigest, artifact.SourceCatalog.CatalogRootDigest,
				artifact.ModelRun.ClassificationRunID, artifact.ModelRun.ClassificationRunDigest, artifact.ModelRun.PromptDigest, artifact.ModelRun.TaxonomyVersion, artifact.ModelRun.TaxonomyDigest,
				artifact.RecommendationPolicy.Version, item.DomainCode, item.TopicCodes, item.InspectionTypeCodes, item.InspectionProfileCodes,
				item.ApplicabilityDisposition, item.RiskBand, item.RiskTier, item.SafetyCritical, item.AgreementConfidence,
				item.AdvisoryState, item.DefaultRecommendationBucket, item.RecurrenceMonths, item.RationaleCodes,
				item.ExternalApplicabilityUnresolved, now.UTC()); err != nil {
				return fmt.Errorf("insert AI enrichment %s: %w", item.QuestionVersionID, err)
			}
		}
		return nil
	})
	if err != nil {
		return AIEnrichmentLoadResult{}, err
	}
	return result, nil
}

func verifyAIEnrichmentReplay(ctx context.Context, tx pgx.Tx, catalogID string, artifact AIRecommendationArtifact, membershipDigests map[string]string) error {
	rows, err := tx.Query(ctx, `
		SELECT question_version_id,artifact_version,artifact_digest,source_catalog_root_digest,
		       classification_run_id,classification_run_digest,prompt_digest,taxonomy_version,taxonomy_digest,
		       recommendation_policy_version,domain_code,topic_codes,inspection_type_codes,inspection_profile_codes,
		       applicability_disposition,risk_band,risk_tier,safety_critical,agreement_confidence,
			advisory_state,default_recommendation_bucket,recurrence_months,rationale_codes,
		       external_applicability_unresolved
		FROM canonical_question_catalog_ai_enrichments WHERE catalog_id=$1 ORDER BY question_version_id`, catalogID)
	if err != nil {
		return fmt.Errorf("read AI enrichment replay: %w", err)
	}
	defer rows.Close()
	byID := make(map[string]aiEnrichmentRow, len(artifact.Items))
	for rows.Next() {
		var row aiEnrichmentRow
		if err := rows.Scan(&row.QuestionVersionID, &row.ArtifactVersion, &row.ArtifactDigest, &row.SourceCatalogRootDigest,
			&row.ClassificationRunID, &row.ClassificationRunDigest, &row.PromptDigest, &row.TaxonomyVersion, &row.TaxonomyDigest,
			&row.RecommendationPolicyVersion, &row.DomainCode, &row.TopicCodes, &row.InspectionTypeCodes, &row.InspectionProfileCodes,
			&row.ApplicabilityDisposition, &row.RiskBand, &row.RiskTier, &row.SafetyCritical, &row.AgreementConfidence,
			&row.AdvisoryState, &row.DefaultRecommendationBucket, &row.RecurrenceMonths, &row.RationaleCodes,
			&row.ExternalApplicabilityUnresolved); err != nil {
			return fmt.Errorf("scan AI enrichment replay: %w", err)
		}
		byID[row.QuestionVersionID] = row
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read AI enrichment replay rows: %w", err)
	}
	if len(byID) != len(artifact.Items) {
		return errors.New("AI enrichment replay count drifted")
	}
	for _, item := range artifact.Items {
		if membershipDigests[item.QuestionVersionID] != item.QuestionDigest {
			return fmt.Errorf("AI enrichment replay membership drifted: %s", item.QuestionVersionID)
		}
		row, ok := byID[item.QuestionVersionID]
		if !ok || row.ArtifactVersion != artifact.SchemaVersion || row.ArtifactDigest != artifact.ArtifactDigest || row.SourceCatalogRootDigest != artifact.SourceCatalog.CatalogRootDigest || row.ClassificationRunID != artifact.ModelRun.ClassificationRunID || row.ClassificationRunDigest != artifact.ModelRun.ClassificationRunDigest || row.PromptDigest != artifact.ModelRun.PromptDigest || row.TaxonomyVersion != artifact.ModelRun.TaxonomyVersion || row.TaxonomyDigest != artifact.ModelRun.TaxonomyDigest || row.RecommendationPolicyVersion != artifact.RecommendationPolicy.Version || row.DomainCode != item.DomainCode || !reflect.DeepEqual(row.TopicCodes, item.TopicCodes) || !reflect.DeepEqual(row.InspectionTypeCodes, item.InspectionTypeCodes) || !reflect.DeepEqual(row.InspectionProfileCodes, item.InspectionProfileCodes) || row.ApplicabilityDisposition != item.ApplicabilityDisposition || row.RiskBand != item.RiskBand || row.RiskTier != item.RiskTier || row.SafetyCritical != item.SafetyCritical || row.AgreementConfidence != item.AgreementConfidence || row.AdvisoryState != item.AdvisoryState || row.DefaultRecommendationBucket != item.DefaultRecommendationBucket || row.RecurrenceMonths != item.RecurrenceMonths || !reflect.DeepEqual(row.RationaleCodes, item.RationaleCodes) || row.ExternalApplicabilityUnresolved != item.ExternalApplicabilityUnresolved {
			return fmt.Errorf("AI enrichment replay drifted: %s", item.QuestionVersionID)
		}
	}
	return nil
}

func artifactDigest(artifact AIRecommendationArtifact) string {
	value := map[string]any{}
	data, _ := json.Marshal(artifact)
	_ = json.Unmarshal(data, &value)
	delete(value, "artifactDigest")
	canonical, _ := json.Marshal(value)
	digest := sha256.Sum256(append(canonical, '\n'))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func aiSHA256Digest(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func sortedUnique(values []string) bool {
	if values == nil {
		return true
	}
	copyValues := append([]string(nil), values...)
	if !sort.StringsAreSorted(copyValues) {
		return false
	}
	for index := 1; index < len(copyValues); index++ {
		if copyValues[index-1] == copyValues[index] {
			return false
		}
	}
	return true
}
