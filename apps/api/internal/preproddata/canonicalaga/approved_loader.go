package canonicalaga

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/questioncatalog"
	"github.com/jackc/pgx/v5"
)

const (
	approvedCatalogProfile        = "approved-aga"
	approvedCatalogProfileVersion = "2.0.0"
)

// ScopeBinding identifies one exact authorized provider-scope/regulated-target
// pair for which the sealed approved catalog is eligible.
type ScopeBinding struct {
	ProviderScopeID   string
	RegulatedTargetID string
}

// LoadApprovedCatalog imports the immutable source-approved catalog directly
// into the governed operational class. Foundation rows are read and checked;
// this loader never creates or repairs organizations, scopes, or targets.
func LoadApprovedCatalog(ctx context.Context, pool *database.Pool, pkg ApprovedSourcePackage, catalogVersion, actorSubjectID string, bindings []ScopeBinding, advisoryLockKey int64, now time.Time) (LoadResult, error) {
	if pool == nil || strings.TrimSpace(actorSubjectID) == "" || len(bindings) == 0 || advisoryLockKey <= 0 {
		return LoadResult{}, fmt.Errorf("approved catalog loader requires database, actor, provider scope, and regulated target")
	}
	if err := ValidateScopeBindings(bindings); err != nil {
		return LoadResult{}, err
	}
	manifest, err := BuildApprovedImportManifest(pkg, catalogVersion)
	if err != nil {
		return LoadResult{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result := LoadResult{CatalogID: "catalog:" + catalogVersion, CatalogVersion: catalogVersion, QuestionCount: len(manifest.Rows), FormCount: len(manifest.Forms), ImportDigest: manifest.CatalogRootDigest}
	err = database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, advisoryLockKey); err != nil {
			return fmt.Errorf("lock approved catalog import: %w", err)
		}
		for _, binding := range bindings {
			var compatible bool
			if err := tx.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM organization_service_provider_scopes scope
					JOIN regulated_targets target ON target.id = $2
				WHERE scope.id = $1 AND scope.status = 'ACTIVE'
				  AND scope.effective_from <= CURRENT_DATE
				  AND (scope.effective_to IS NULL OR scope.effective_to > CURRENT_DATE)
				  AND (scope.primary_target_id = target.id OR EXISTS (
					SELECT 1 FROM organization_service_provider_scope_targets linked
					WHERE linked.organization_service_provider_scope_id = scope.id
					  AND linked.regulated_target_id = target.id
				))
				  AND (target.organization_id IS NULL OR target.organization_id = scope.organization_id)
				  AND (target.owner_organization_id IS NULL OR target.owner_organization_id = scope.organization_id)
				)`, binding.ProviderScopeID, binding.RegulatedTargetID).Scan(&compatible); err != nil {
				return err
			}
			if !compatible {
				return fmt.Errorf("approved catalog foundation scope/target is not an active compatible pair: %s/%s", binding.ProviderScopeID, binding.RegulatedTargetID)
			}
		}

		var existingVersion, existingUsage, existingOrigin, existingProfile, existingProfileVersion, existingRoot, existingCatalogRoot, existingSourceManifest string
		var existingQuestionCount, existingFormCount int
		catalogErr := tx.QueryRow(ctx, `
			SELECT catalog_version, usage_class, source_origin, profile_name, profile_version,
			       root_digest, catalog_root_digest, source_manifest_sha256, question_count, form_count
			FROM canonical_question_catalogs WHERE id=$1`, result.CatalogID).Scan(
			&existingVersion, &existingUsage, &existingOrigin, &existingProfile, &existingProfileVersion,
			&existingRoot, &existingCatalogRoot, &existingSourceManifest, &existingQuestionCount, &existingFormCount)
		if catalogErr == nil {
			if existingVersion != catalogVersion || existingUsage != string(questioncatalog.UsageClassGovernedOperational) || existingOrigin != string(questioncatalog.SourceOriginImportedApproved) || existingProfile != approvedCatalogProfile || existingProfileVersion != approvedCatalogProfileVersion || existingRoot != manifest.CatalogRootDigest || existingCatalogRoot != manifest.CatalogRootDigest || existingSourceManifest != manifest.SourceManifestSHA256 || existingQuestionCount != len(manifest.Rows) || existingFormCount != len(manifest.Forms) {
				return fmt.Errorf("existing approved catalog does not match the release-pinned source")
			}
			if err := appendMissingApprovedApplicabilities(ctx, tx, result.CatalogID, bindings, actorSubjectID, now); err != nil {
				return err
			}
			return verifyApprovedReplay(ctx, tx, result.CatalogID, manifest, pkg, actorSubjectID, bindings)
		}
		if !errors.Is(catalogErr, pgx.ErrNoRows) {
			return catalogErr
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO canonical_question_catalogs
			(id,catalog_version,usage_class,profile_name,profile_version,status,source_package_version,
			 source_package_json_sha256,source_package_zip_sha256,root_digest,catalog_root_digest,
			 source_origin,source_manifest_sha256,question_count,form_count,created_by_subject_id,created_at)
			VALUES ($1,$2,'GOVERNED_OPERATIONAL',$3,$4,'SEALED',$5,$6,$7,$8,$8,$9,$10,$11,$12,$13,$14)`,
			result.CatalogID, catalogVersion, approvedCatalogProfile, approvedCatalogProfileVersion,
			pkg.Identity.PackageVersion, pkg.Identity.JSONSHA256, pkg.Identity.ZipSHA256,
			manifest.CatalogRootDigest, string(questioncatalog.SourceOriginImportedApproved), manifest.SourceManifestSHA256,
			len(manifest.Rows), len(manifest.Forms), actorSubjectID, now.UTC()); err != nil {
			return fmt.Errorf("insert approved catalog: %w", err)
		}
		for _, form := range manifest.Forms {
			if _, err := tx.Exec(ctx, `INSERT INTO canonical_question_catalog_forms (catalog_id,form_code,form_digest,archive_digest,question_count,source_gap_state,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, result.CatalogID, form.FormCode, form.FormDigest, form.ArchiveDigest, form.QuestionCount, form.SourceGapState, now.UTC()); err != nil {
				return fmt.Errorf("insert approved form %s: %w", form.FormCode, err)
			}
		}
		for _, question := range manifest.QuestionVersions {
			if _, err := tx.Exec(ctx, `INSERT INTO question_versions (id,question_id,version,prompt,configured_reference,expected_evidence,created_by_subject_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, question.ID, question.QuestionID, question.Version, question.Prompt, question.ConfiguredReference, question.ExpectedEvidence, actorSubjectID, now.UTC()); err != nil {
				return fmt.Errorf("insert approved question version %s: %w", question.ID, err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO canonical_question_version_provenance (question_version_id,usage_class,source_origin,catalog_id,recorded_at) VALUES ($1,'GOVERNED_OPERATIONAL','IMPORTED_APPROVED_SOURCE',$2,$3)`, question.ID, result.CatalogID, now.UTC()); err != nil {
				return fmt.Errorf("insert approved provenance %s: %w", question.ID, err)
			}
		}
		questionByID := make(map[string]QuestionVersionImport, len(manifest.QuestionVersions))
		for _, question := range manifest.QuestionVersions {
			questionByID[question.ID] = question
		}
		for _, row := range manifest.Rows {
			question, ok := questionByID[row.QuestionVersionID]
			if !ok {
				return fmt.Errorf("approved membership %s has no question version", row.QuestionVersionID)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO canonical_question_catalog_memberships (catalog_id,question_version_id,usage_class,source_origin,form_code,proposal_id,ordinal,question_digest,source_locator,source_gap_state,proposed_domain,proposed_topic,proposed_risk_band,created_at) VALUES ($1,$2,'GOVERNED_OPERATIONAL','IMPORTED_APPROVED_SOURCE',$3,$4,$5,$6,$7,$8,NULL,NULL,NULL,$9)`, result.CatalogID, row.QuestionVersionID, row.FormCode, row.ProposalID, row.Ordinal, row.QuestionDigest, question.SourceLocator, "OPTIONAL_ENRICHMENT_NOT_PROVIDED", now.UTC()); err != nil {
				return fmt.Errorf("insert approved membership %s: %w", row.QuestionVersionID, err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO canonical_question_catalog_membership_events (event_id,catalog_id,question_version_id,status,reason,actor_subject_id,occurred_at) VALUES ($1,$2,$3,'AVAILABLE','approved source import membership',$4,$5)`, "catalog-membership:"+result.CatalogID+":"+row.QuestionVersionID+":available", result.CatalogID, row.QuestionVersionID, actorSubjectID, now.UTC()); err != nil {
				return fmt.Errorf("insert approved membership event %s: %w", row.QuestionVersionID, err)
			}
			for _, binding := range bindings {
				if _, err := tx.Exec(ctx, `INSERT INTO canonical_question_catalog_applicabilities (catalog_id,question_version_id,provider_scope_id,regulated_target_id,status,reason,actor_subject_id,created_at) VALUES ($1,$2,$3,$4,'ELIGIBLE','approved catalog active foundation binding',$5,$6)`, result.CatalogID, row.QuestionVersionID, binding.ProviderScopeID, binding.RegulatedTargetID, actorSubjectID, now.UTC()); err != nil {
					return fmt.Errorf("insert approved applicability %s/%s/%s: %w", row.QuestionVersionID, binding.ProviderScopeID, binding.RegulatedTargetID, err)
				}
			}
		}
		_, err := tx.Exec(ctx, `INSERT INTO canonical_question_catalog_import_runs (id,catalog_id,operation_id,idempotency_key,package_zip_sha256,package_json_sha256,source_origin,source_manifest_sha256,catalog_root_digest,import_digest,row_count,form_count,status,actor_subject_id,created_at) VALUES ($1,$2,$3,$3,$4,$5,'IMPORTED_APPROVED_SOURCE',$6,$7,$7,$8,$9,'SEALED',$10,$11)`, "import:"+catalogVersion, result.CatalogID, "import:approved-aga:"+manifest.CatalogRootDigest, pkg.Identity.ZipSHA256, pkg.Identity.JSONSHA256, manifest.SourceManifestSHA256, manifest.CatalogRootDigest, len(manifest.Rows), len(manifest.Forms), actorSubjectID, now.UTC())
		return err
	})
	if err != nil {
		return LoadResult{}, err
	}
	return result, nil
}

// appendMissingApprovedApplicabilities extends an existing sealed catalog only
// with new authorized foundation bindings. Existing catalog memberships,
// question versions, and applicability rows remain append-only and are never
// updated or deleted.
func appendMissingApprovedApplicabilities(ctx context.Context, tx pgx.Tx, catalogID string, bindings []ScopeBinding, actorSubjectID string, now time.Time) error {
	for _, binding := range bindings {
		if _, err := tx.Exec(ctx, `
			INSERT INTO canonical_question_catalog_applicabilities
				(catalog_id,question_version_id,provider_scope_id,regulated_target_id,status,reason,actor_subject_id,created_at)
			SELECT catalog_id,question_version_id,$2,$3,'ELIGIBLE','approved catalog active foundation binding',$4,$5
			FROM canonical_question_catalog_memberships
			WHERE catalog_id=$1
			ON CONFLICT (catalog_id,question_version_id,provider_scope_id,regulated_target_id) DO NOTHING`,
			catalogID, binding.ProviderScopeID, binding.RegulatedTargetID, actorSubjectID, now.UTC()); err != nil {
			return fmt.Errorf("append approved applicability %s/%s: %w", binding.ProviderScopeID, binding.RegulatedTargetID, err)
		}
	}
	return nil
}

func verifyApprovedReplay(ctx context.Context, tx pgx.Tx, catalogID string, manifest ImportManifest, pkg ApprovedSourcePackage, actorSubjectID string, bindings []ScopeBinding) error {
	var packageZip, packageJSON, origin, sourceManifest, root, digest, receiptActor string
	var rows, forms, receiptCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM canonical_question_catalog_import_runs WHERE catalog_id=$1`, catalogID).Scan(&receiptCount); err != nil {
		return fmt.Errorf("count approved catalog replay receipts: %w", err)
	}
	if receiptCount != 1 {
		return fmt.Errorf("approved catalog replay requires exactly one import receipt, got %d", receiptCount)
	}
	if err := tx.QueryRow(ctx, `SELECT package_zip_sha256,package_json_sha256,source_origin,source_manifest_sha256,catalog_root_digest,import_digest,row_count,form_count,actor_subject_id FROM canonical_question_catalog_import_runs WHERE catalog_id=$1 ORDER BY created_at DESC LIMIT 1`, catalogID).Scan(&packageZip, &packageJSON, &origin, &sourceManifest, &root, &digest, &rows, &forms, &receiptActor); err != nil {
		return fmt.Errorf("approved catalog replay receipt is missing: %w", err)
	}
	if packageZip != pkg.Identity.ZipSHA256 || packageJSON != pkg.Identity.JSONSHA256 || origin != string(questioncatalog.SourceOriginImportedApproved) || sourceManifest != manifest.SourceManifestSHA256 || root != manifest.CatalogRootDigest || digest != manifest.CatalogRootDigest || rows != len(manifest.Rows) || forms != len(manifest.Forms) || receiptActor != actorSubjectID {
		return fmt.Errorf("approved catalog replay receipt drifted")
	}
	var catalogPackageVersion, catalogPackageJSON, catalogPackageZip, catalogActor string
	if err := tx.QueryRow(ctx, `SELECT source_package_version,source_package_json_sha256,source_package_zip_sha256,created_by_subject_id FROM canonical_question_catalogs WHERE id=$1`, catalogID).Scan(&catalogPackageVersion, &catalogPackageJSON, &catalogPackageZip, &catalogActor); err != nil {
		return fmt.Errorf("read approved catalog identity: %w", err)
	}
	if catalogPackageVersion != pkg.Identity.PackageVersion || catalogPackageJSON != pkg.Identity.JSONSHA256 || catalogPackageZip != pkg.Identity.ZipSHA256 || catalogActor != actorSubjectID {
		return fmt.Errorf("approved catalog package identity drifted")
	}

	var membershipCount, formCount, applicabilityCount, eventCount, provenanceCount int
	if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM canonical_question_catalog_memberships WHERE catalog_id=$1),(SELECT count(*) FROM canonical_question_catalog_forms WHERE catalog_id=$1),(SELECT count(*) FROM canonical_question_catalog_applicabilities WHERE catalog_id=$1),(SELECT count(*) FROM canonical_question_catalog_membership_events WHERE catalog_id=$1),(SELECT count(*) FROM canonical_question_version_provenance WHERE catalog_id=$1)`, catalogID).Scan(&membershipCount, &formCount, &applicabilityCount, &eventCount, &provenanceCount); err != nil {
		return err
	}
	if membershipCount != len(manifest.Rows) || formCount != len(manifest.Forms) || applicabilityCount != len(manifest.Rows)*len(bindings) || eventCount != len(manifest.Rows) || provenanceCount != len(manifest.Rows) {
		return fmt.Errorf("approved catalog replay row counts drifted")
	}

	formByCode := make(map[string]CatalogFormImport, len(manifest.Forms))
	for _, form := range manifest.Forms {
		formByCode[form.FormCode] = form
	}
	formRows, err := tx.Query(ctx, `SELECT form_code,form_digest,archive_digest,question_count,source_gap_state FROM canonical_question_catalog_forms WHERE catalog_id=$1 ORDER BY form_code`, catalogID)
	if err != nil {
		return fmt.Errorf("read approved catalog forms: %w", err)
	}
	seenForms := make(map[string]struct{}, len(manifest.Forms))
	for formRows.Next() {
		var code, formDigest, archiveDigest, sourceGap string
		var questionCount int
		if err := formRows.Scan(&code, &formDigest, &archiveDigest, &questionCount, &sourceGap); err != nil {
			formRows.Close()
			return fmt.Errorf("scan approved catalog form: %w", err)
		}
		expected, ok := formByCode[code]
		if !ok || expected.FormDigest != formDigest || expected.ArchiveDigest != archiveDigest || expected.QuestionCount != questionCount || expected.SourceGapState != sourceGap {
			formRows.Close()
			return fmt.Errorf("approved catalog form %s drifted", code)
		}
		if _, duplicate := seenForms[code]; duplicate {
			formRows.Close()
			return fmt.Errorf("approved catalog form %s is duplicated", code)
		}
		seenForms[code] = struct{}{}
	}
	if err := formRows.Err(); err != nil {
		formRows.Close()
		return fmt.Errorf("read approved catalog forms: %w", err)
	}
	formRows.Close()
	if len(seenForms) != len(formByCode) {
		return fmt.Errorf("approved catalog form membership is incomplete")
	}

	expectedByKey := make(map[string]QuestionVersionImport, len(manifest.QuestionVersions))
	for _, question := range manifest.QuestionVersions {
		expectedByKey[question.FormCode+"\x00"+strconv.Itoa(question.Ordinal)] = question
	}
	questionRows, err := tx.Query(ctx, `
		SELECT membership.form_code,membership.ordinal,membership.question_version_id,membership.proposal_id,
		       membership.question_digest,membership.source_locator,membership.source_gap_state,
		       question.question_id,question.version,question.prompt,question.configured_reference,question.expected_evidence,
		       membership.usage_class,membership.source_origin,
		       provenance.usage_class,provenance.source_origin
		FROM canonical_question_catalog_memberships membership
		JOIN question_versions question ON question.id=membership.question_version_id
		JOIN canonical_question_version_provenance provenance
		  ON provenance.question_version_id=membership.question_version_id
		 AND provenance.catalog_id=membership.catalog_id
		WHERE membership.catalog_id=$1
		ORDER BY membership.form_code,membership.ordinal`, catalogID)
	if err != nil {
		return fmt.Errorf("read approved catalog question rows: %w", err)
	}
	seenQuestions := make(map[string]struct{}, len(manifest.QuestionVersions))
	for questionRows.Next() {
		var formCode, questionVersionID, proposalID, questionDigest, sourceLocator, sourceGap string
		var questionID, prompt, configuredReference, expectedEvidence string
		var usageClass, sourceOrigin, provenanceUsage, provenanceOrigin string
		var ordinal, version int
		if err := questionRows.Scan(&formCode, &ordinal, &questionVersionID, &proposalID, &questionDigest, &sourceLocator, &sourceGap, &questionID, &version, &prompt, &configuredReference, &expectedEvidence, &usageClass, &sourceOrigin, &provenanceUsage, &provenanceOrigin); err != nil {
			questionRows.Close()
			return fmt.Errorf("scan approved catalog question row: %w", err)
		}
		expected, ok := expectedByKey[formCode+"\x00"+strconv.Itoa(ordinal)]
		if !ok || expected.ID != questionVersionID || expected.ProposalID != proposalID || expected.TextDigest != questionDigest || expected.SourceLocator != sourceLocator || expected.SourceGapState != sourceGap || expected.QuestionID != questionID || expected.Version != version || expected.Prompt != prompt || expected.ConfiguredReference != configuredReference || expected.ExpectedEvidence != expectedEvidence || usageClass != string(questioncatalog.UsageClassGovernedOperational) || sourceOrigin != string(questioncatalog.SourceOriginImportedApproved) || provenanceUsage != usageClass || provenanceOrigin != sourceOrigin {
			questionRows.Close()
			return fmt.Errorf("approved catalog question %s/%d drifted", formCode, ordinal)
		}
		if _, duplicate := seenQuestions[questionVersionID]; duplicate {
			questionRows.Close()
			return fmt.Errorf("approved catalog question %s is duplicated", questionVersionID)
		}
		seenQuestions[questionVersionID] = struct{}{}
	}
	if err := questionRows.Err(); err != nil {
		questionRows.Close()
		return fmt.Errorf("read approved catalog question rows: %w", err)
	}
	questionRows.Close()
	if len(seenQuestions) != len(expectedByKey) {
		return fmt.Errorf("approved catalog question membership is incomplete")
	}

	eventRows, err := tx.Query(ctx, `SELECT event_id,question_version_id,status,reason,actor_subject_id FROM canonical_question_catalog_membership_events WHERE catalog_id=$1 ORDER BY question_version_id`, catalogID)
	if err != nil {
		return fmt.Errorf("read approved catalog membership events: %w", err)
	}
	seenEvents := make(map[string]struct{}, len(manifest.Rows))
	for eventRows.Next() {
		var eventID, questionVersionID, status, reason, actor string
		if err := eventRows.Scan(&eventID, &questionVersionID, &status, &reason, &actor); err != nil {
			eventRows.Close()
			return fmt.Errorf("scan approved catalog membership event: %w", err)
		}
		expectedID := "catalog-membership:" + catalogID + ":" + questionVersionID + ":available"
		if eventID != expectedID || status != "AVAILABLE" || reason != "approved source import membership" || actor != actorSubjectID {
			eventRows.Close()
			return fmt.Errorf("approved catalog membership event %s drifted", questionVersionID)
		}
		if _, duplicate := seenEvents[questionVersionID]; duplicate {
			eventRows.Close()
			return fmt.Errorf("approved catalog membership event %s is duplicated", questionVersionID)
		}
		seenEvents[questionVersionID] = struct{}{}
	}
	if err := eventRows.Err(); err != nil {
		eventRows.Close()
		return fmt.Errorf("read approved catalog membership events: %w", err)
	}
	eventRows.Close()
	if len(seenEvents) != len(manifest.Rows) {
		return fmt.Errorf("approved catalog membership events are incomplete")
	}

	applicabilityRows, err := tx.Query(ctx, `SELECT question_version_id,provider_scope_id,regulated_target_id,status,reason,actor_subject_id FROM canonical_question_catalog_applicabilities WHERE catalog_id=$1 ORDER BY question_version_id`, catalogID)
	if err != nil {
		return fmt.Errorf("read approved catalog applicability: %w", err)
	}
	seenApplicability := make(map[string]struct{}, len(manifest.Rows)*len(bindings))
	for applicabilityRows.Next() {
		var questionVersionID, scopeID, targetID, status, reason, actor string
		if err := applicabilityRows.Scan(&questionVersionID, &scopeID, &targetID, &status, &reason, &actor); err != nil {
			applicabilityRows.Close()
			return fmt.Errorf("scan approved catalog applicability: %w", err)
		}
		if !scopeBindingContains(bindings, scopeID, targetID) || status != "ELIGIBLE" || reason != "approved catalog active foundation binding" || actor != actorSubjectID {
			applicabilityRows.Close()
			return fmt.Errorf("approved catalog applicability %s drifted", questionVersionID)
		}
		key := questionVersionID + "\x00" + scopeID + "\x00" + targetID
		if _, duplicate := seenApplicability[key]; duplicate {
			applicabilityRows.Close()
			return fmt.Errorf("approved catalog applicability %s is duplicated", key)
		}
		seenApplicability[key] = struct{}{}
	}
	if err := applicabilityRows.Err(); err != nil {
		applicabilityRows.Close()
		return fmt.Errorf("read approved catalog applicability: %w", err)
	}
	applicabilityRows.Close()
	if len(seenApplicability) != len(manifest.Rows)*len(bindings) {
		return fmt.Errorf("approved catalog applicability is incomplete")
	}

	return nil
}

// ValidateScopeBindings checks the manifest-level binding identity before a
// database is touched. The approved catalog loader and its validate command
// share this fail-closed contract.
func ValidateScopeBindings(bindings []ScopeBinding) error {
	seen := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		if strings.TrimSpace(binding.ProviderScopeID) == "" || strings.TrimSpace(binding.RegulatedTargetID) == "" {
			return fmt.Errorf("approved catalog scope binding is incomplete")
		}
		key := binding.ProviderScopeID + "\x00" + binding.RegulatedTargetID
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("approved catalog scope bindings contain a duplicate pair")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func scopeBindingContains(bindings []ScopeBinding, providerScopeID, regulatedTargetID string) bool {
	for _, binding := range bindings {
		if binding.ProviderScopeID == providerScopeID && binding.RegulatedTargetID == regulatedTargetID {
			return true
		}
	}
	return false
}
