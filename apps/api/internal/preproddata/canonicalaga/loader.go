package canonicalaga

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
	"github.com/jackc/pgx/v5"
)

type LoadResult struct {
	CatalogID      string
	CatalogVersion string
	QuestionCount  int
	FormCount      int
	ImportDigest   string
}

// LoadSealedCatalog writes one whole catalog in a single transaction.  The
// package reader must have already verified the sealed ZIP/JSON identity; this
// method only creates immutable question_versions and reference-only catalog
// membership/lineage rows.  Rerunning the same package is idempotent, while a
// changed body or digest for an existing version fails closed.
func LoadSealedCatalog(ctx context.Context, pool *database.Pool, pkg agacandidatedemo.AcceptedPackage, catalogVersion, actorSubjectID string, now time.Time) (LoadResult, error) {
	if pool == nil {
		return LoadResult{}, fmt.Errorf("canonical AGA catalog database is required")
	}
	manifest, err := BuildImportManifest(pkg, catalogVersion)
	if err != nil {
		return LoadResult{}, err
	}
	if strings.TrimSpace(actorSubjectID) == "" {
		return LoadResult{}, fmt.Errorf("canonical AGA catalog loader actor is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result := LoadResult{CatalogID: "catalog:" + catalogVersion, CatalogVersion: catalogVersion, QuestionCount: len(manifest.Rows), FormCount: 52, ImportDigest: manifest.ImportDigest}
	err = database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		catalogID := result.CatalogID
		var existingVersion, existingUsage, existingProfile, existingProfileVersion, existingRoot string
		var existingQuestionCount, existingFormCount int
		catalogErr := tx.QueryRow(ctx, `
			SELECT catalog_version, usage_class, profile_name, profile_version,
			       root_digest, question_count, form_count
			FROM canonical_question_catalogs WHERE id = $1
		`, catalogID).Scan(&existingVersion, &existingUsage, &existingProfile,
			&existingProfileVersion, &existingRoot, &existingQuestionCount, &existingFormCount)
		switch {
		case catalogErr == nil:
			if existingVersion != catalogVersion || existingUsage != string(manifest.UsageClass) ||
				existingProfile != "aga-preprod" || existingProfileVersion != "1.0.0" ||
				existingRoot != manifest.ImportDigest || existingQuestionCount != len(manifest.Rows) ||
				existingFormCount != len(manifest.Forms) {
				return fmt.Errorf("existing canonical catalog %s does not match sealed import", catalogID)
			}
		case errors.Is(catalogErr, pgx.ErrNoRows):
			if _, err := tx.Exec(ctx, `
			INSERT INTO canonical_question_catalogs
			(id,catalog_version,usage_class,profile_name,profile_version,status,source_package_version,source_package_json_sha256,source_package_zip_sha256,root_digest,question_count,form_count,created_by_subject_id,created_at)
			VALUES ($1,$2,'PREPROD_EXERCISE','aga-preprod','1.0.0','SEALED',$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (id) DO NOTHING
			`, catalogID, catalogVersion, pkg.Identity.PackageVersion, pkg.Identity.JSONSHA256, pkg.Identity.ZipSHA256, manifest.ImportDigest, len(manifest.Rows), result.FormCount, actorSubjectID, now.UTC()); err != nil {
				return err
			}
		default:
			return catalogErr
		}
		for _, form := range manifest.Forms {
			var existingFormDigest, existingArchiveDigest, existingGap string
			var existingFormCount int
			formErr := tx.QueryRow(ctx, `SELECT form_digest, COALESCE(archive_digest,''), question_count, source_gap_state FROM canonical_question_catalog_forms WHERE catalog_id=$1 AND form_code=$2`, catalogID, form.FormCode).Scan(&existingFormDigest, &existingArchiveDigest, &existingFormCount, &existingGap)
			if formErr == nil {
				if existingFormDigest != form.FormDigest || existingArchiveDigest != form.ArchiveDigest || existingFormCount != form.QuestionCount || existingGap != form.SourceGapState {
					return fmt.Errorf("canonical form %s differs from sealed import", form.FormCode)
				}
				continue
			}
			if !errors.Is(formErr, pgx.ErrNoRows) {
				return formErr
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO canonical_question_catalog_forms
				(catalog_id,form_code,form_digest,archive_digest,question_count,source_gap_state,created_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7)
				ON CONFLICT (catalog_id,form_code) DO NOTHING
			`, catalogID, form.FormCode, form.FormDigest, form.ArchiveDigest,
				form.QuestionCount, form.SourceGapState, now.UTC()); err != nil {
				return err
			}
		}
		for _, question := range manifest.QuestionVersions {
			var existingQuestionID, existingPrompt, existingReference, existingEvidence string
			var existingVersion int
			err := tx.QueryRow(ctx, `SELECT question_id, version, prompt, configured_reference, expected_evidence FROM question_versions WHERE id=$1`, question.ID).Scan(&existingQuestionID, &existingVersion, &existingPrompt, &existingReference, &existingEvidence)
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				if _, err := tx.Exec(ctx, `INSERT INTO question_versions (id,question_id,version,prompt,configured_reference,expected_evidence,created_by_subject_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, question.ID, question.QuestionID, question.Version, question.Prompt, question.ConfiguredReference, question.ExpectedEvidence, actorSubjectID, now.UTC()); err != nil {
					return err
				}
			case err != nil:
				return err
			case existingQuestionID != question.QuestionID || existingVersion != question.Version || existingPrompt != question.Prompt || existingReference != question.ConfiguredReference || existingEvidence != question.ExpectedEvidence:
				return fmt.Errorf("question_versions immutable fields mismatch for %s", question.ID)
			}
		}
		for _, row := range manifest.Rows {
			var existingFormCode, existingProposalID, existingSourceLocator, existingGap, existingDomain, existingTopic, existingRisk, existingDigest string
			var existingOrdinal int
			membershipErr := tx.QueryRow(ctx, `SELECT form_code, proposal_id, ordinal, question_digest, COALESCE(source_locator,''), source_gap_state, COALESCE(proposed_domain,''), COALESCE(proposed_topic,''), COALESCE(proposed_risk_band,'') FROM canonical_question_catalog_memberships WHERE catalog_id=$1 AND question_version_id=$2`, catalogID, row.QuestionVersionID).Scan(&existingFormCode, &existingProposalID, &existingOrdinal, &existingDigest, &existingSourceLocator, &existingGap, &existingDomain, &existingTopic, &existingRisk)
			if membershipErr == nil {
				if existingFormCode != row.FormCode || existingProposalID != row.ProposalID || existingOrdinal != row.Ordinal || existingDigest != row.QuestionDigest || existingSourceLocator != findSourceLocator(manifest, row.QuestionVersionID) || existingGap != findSourceGap(manifest, row.QuestionVersionID) || existingDomain != findDomain(manifest, row.QuestionVersionID) || existingTopic != findTopic(manifest, row.QuestionVersionID) || existingRisk != findRiskBand(manifest, row.QuestionVersionID) {
					return fmt.Errorf("canonical membership immutable fields mismatch for %s", row.QuestionVersionID)
				}
				continue
			}
			if !errors.Is(membershipErr, pgx.ErrNoRows) {
				return membershipErr
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO canonical_question_catalog_memberships
				(catalog_id,question_version_id,usage_class,form_code,proposal_id,ordinal,question_digest,source_locator,source_gap_state,proposed_domain,proposed_topic,proposed_risk_band,created_at)
				VALUES ($1,$2,'PREPROD_EXERCISE',$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
				ON CONFLICT (catalog_id,question_version_id) DO NOTHING
			`, catalogID, row.QuestionVersionID, row.FormCode, row.ProposalID, row.Ordinal, row.QuestionDigest, findSourceLocator(manifest, row.QuestionVersionID), findSourceGap(manifest, row.QuestionVersionID), findDomain(manifest, row.QuestionVersionID), findTopic(manifest, row.QuestionVersionID), findRiskBand(manifest, row.QuestionVersionID), now.UTC()); err != nil {
				return err
			}
		}
		var existingImportDigest, existingZip, existingJSON string
		var existingRows, existingForms int
		importRunID := "import:" + catalogVersion
		importErr := tx.QueryRow(ctx, `SELECT package_zip_sha256, package_json_sha256, import_digest, row_count, form_count FROM canonical_question_catalog_import_runs WHERE id=$1`, importRunID).Scan(&existingZip, &existingJSON, &existingImportDigest, &existingRows, &existingForms)
		if importErr == nil {
			if existingZip != pkg.Identity.ZipSHA256 || existingJSON != pkg.Identity.JSONSHA256 || existingImportDigest != manifest.ImportDigest || existingRows != len(manifest.Rows) || existingForms != result.FormCount {
				return fmt.Errorf("existing canonical import run differs from sealed import")
			}
			return nil
		}
		if !errors.Is(importErr, pgx.ErrNoRows) {
			return importErr
		}
		_, err := tx.Exec(ctx, `INSERT INTO canonical_question_catalog_import_runs (id,catalog_id,operation_id,idempotency_key,package_zip_sha256,package_json_sha256,import_digest,row_count,form_count,status,actor_subject_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'SEALED',$10,$11)`, importRunID, catalogID, "import:"+manifest.ImportDigest, "import:"+manifest.ImportDigest, pkg.Identity.ZipSHA256, pkg.Identity.JSONSHA256, manifest.ImportDigest, len(manifest.Rows), result.FormCount, actorSubjectID, now.UTC())
		return err
	})
	if err != nil {
		return LoadResult{}, err
	}
	return result, nil
}

func findQuestion(manifest ImportManifest, id string) (QuestionVersionImport, bool) {
	for _, question := range manifest.QuestionVersions {
		if question.ID == id {
			return question, true
		}
	}
	return QuestionVersionImport{}, false
}

func findSourceLocator(manifest ImportManifest, id string) string {
	question, _ := findQuestion(manifest, id)
	return question.SourceLocator
}

func findSourceGap(manifest ImportManifest, id string) string {
	question, _ := findQuestion(manifest, id)
	return question.SourceGapState
}

func findDomain(manifest ImportManifest, id string) string {
	question, _ := findQuestion(manifest, id)
	return question.ProposedDomain
}

func findTopic(manifest ImportManifest, id string) string {
	question, _ := findQuestion(manifest, id)
	return question.ProposedTopic
}

func findRiskBand(manifest ImportManifest, id string) string {
	question, _ := findQuestion(manifest, id)
	return question.ProposedRiskBand
}
