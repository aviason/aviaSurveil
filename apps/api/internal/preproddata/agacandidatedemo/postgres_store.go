package agacandidatedemo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

// OverlaySchemaDDL is executed only by the disposable preprod bootstrap owner.
// It intentionally names no governed-domain table.
const OverlaySchemaDDL = `
CREATE SCHEMA IF NOT EXISTS preprod_aga_demo;
CREATE TABLE IF NOT EXISTS preprod_aga_demo.package_intents (intent_digest text PRIMARY KEY, run_id text NOT NULL UNIQUE, target_digest text NOT NULL, payload jsonb NOT NULL, canonical_payload text NOT NULL, row_digest text NOT NULL, created_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS preprod_aga_demo.packages (package_digest text PRIMARY KEY, singleton boolean NOT NULL DEFAULT true UNIQUE CHECK (singleton), intent_digest text NOT NULL REFERENCES preprod_aga_demo.package_intents(intent_digest), zip_digest text NOT NULL, json_digest text NOT NULL, manifest_digest text NOT NULL, status text NOT NULL CHECK (status = 'SEALED_PREPROD_DEMO_PROJECTION'), payload jsonb NOT NULL, canonical_payload text NOT NULL, row_digest text NOT NULL);
CREATE TABLE IF NOT EXISTS preprod_aga_demo.forms (package_digest text NOT NULL REFERENCES preprod_aga_demo.packages(package_digest), form_code text NOT NULL, form_digest text NOT NULL, question_extraction_state text NOT NULL CHECK (question_extraction_state IN ('EXTRACTED_CANDIDATE_BOUNDARIES', 'NO_PROTOCOL_QUESTION_BOUNDARY_DETECTED')), candidate_state text NOT NULL CHECK (candidate_state = 'NOT_IMPORTED'), source_mapping_state text NOT NULL CHECK (source_mapping_state = 'SOURCE_MAPPING_REQUIRED'), payload jsonb NOT NULL, canonical_payload text NOT NULL, row_digest text NOT NULL, PRIMARY KEY (package_digest, form_code));
CREATE TABLE IF NOT EXISTS preprod_aga_demo.form_source_proposals (package_digest text NOT NULL, form_code text NOT NULL, ordinal integer NOT NULL CHECK (ordinal > 0), payload jsonb NOT NULL, canonical_payload text NOT NULL, row_digest text NOT NULL, PRIMARY KEY (package_digest, form_code, ordinal), FOREIGN KEY (package_digest, form_code) REFERENCES preprod_aga_demo.forms(package_digest, form_code));
CREATE TABLE IF NOT EXISTS preprod_aga_demo.source_reference_catalog (package_digest text NOT NULL REFERENCES preprod_aga_demo.packages(package_digest), reference text NOT NULL, payload jsonb NOT NULL, canonical_payload text NOT NULL, row_digest text NOT NULL, PRIMARY KEY (package_digest, reference));
CREATE TABLE IF NOT EXISTS preprod_aga_demo.questions (package_digest text NOT NULL REFERENCES preprod_aga_demo.packages(package_digest), proposal_id text NOT NULL, form_code text NOT NULL, ordinal integer NOT NULL CHECK (ordinal > 0), text_digest text NOT NULL, source_gap_category text NOT NULL CHECK (source_gap_category IN ('PROPOSAL_PRESENT_REVIEW_REQUIRED', 'UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL')), candidate_state text NOT NULL CHECK (candidate_state = 'NON_AUTHORITATIVE_CANDIDATE'), source_mapping_state text NOT NULL CHECK (source_mapping_state = 'SOURCE_MAPPING_REQUIRED'), risk_review_state text NOT NULL CHECK (risk_review_state = 'CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW'), payload jsonb NOT NULL, canonical_payload text NOT NULL, row_digest text NOT NULL, PRIMARY KEY (package_digest, proposal_id), FOREIGN KEY (package_digest, form_code) REFERENCES preprod_aga_demo.forms(package_digest, form_code));
CREATE TABLE IF NOT EXISTS preprod_aga_demo.question_source_proposals (package_digest text NOT NULL, proposal_id text NOT NULL, ordinal integer NOT NULL CHECK (ordinal > 0), payload jsonb NOT NULL, canonical_payload text NOT NULL, row_digest text NOT NULL, PRIMARY KEY (package_digest, proposal_id, ordinal), FOREIGN KEY (package_digest, proposal_id) REFERENCES preprod_aga_demo.questions(package_digest, proposal_id));
CREATE TABLE IF NOT EXISTS preprod_aga_demo.package_seals (package_digest text PRIMARY KEY REFERENCES preprod_aga_demo.packages(package_digest), intent_digest text NOT NULL REFERENCES preprod_aga_demo.package_intents(intent_digest), target_digest text NOT NULL, reconciliation_digest text NOT NULL, seal_digest text NOT NULL UNIQUE, relationship_digests jsonb NOT NULL, sealed_at timestamptz NOT NULL);
ALTER SCHEMA preprod_aga_demo OWNER TO preprod_aga_demo_owner;
ALTER TABLE preprod_aga_demo.package_intents OWNER TO preprod_aga_demo_owner;
ALTER TABLE preprod_aga_demo.packages OWNER TO preprod_aga_demo_owner;
ALTER TABLE preprod_aga_demo.forms OWNER TO preprod_aga_demo_owner;
ALTER TABLE preprod_aga_demo.form_source_proposals OWNER TO preprod_aga_demo_owner;
ALTER TABLE preprod_aga_demo.source_reference_catalog OWNER TO preprod_aga_demo_owner;
ALTER TABLE preprod_aga_demo.questions OWNER TO preprod_aga_demo_owner;
ALTER TABLE preprod_aga_demo.question_source_proposals OWNER TO preprod_aga_demo_owner;
ALTER TABLE preprod_aga_demo.package_seals OWNER TO preprod_aga_demo_owner;
CREATE OR REPLACE FUNCTION preprod_aga_demo.reject_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'preprod AGA demo rows are immutable'; END $$;
CREATE TRIGGER package_intents_immutable BEFORE UPDATE OR DELETE ON preprod_aga_demo.package_intents FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_mutation();
CREATE TRIGGER packages_immutable BEFORE UPDATE OR DELETE ON preprod_aga_demo.packages FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_mutation();
CREATE TRIGGER forms_immutable BEFORE UPDATE OR DELETE ON preprod_aga_demo.forms FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_mutation();
CREATE TRIGGER form_source_proposals_immutable BEFORE UPDATE OR DELETE ON preprod_aga_demo.form_source_proposals FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_mutation();
CREATE TRIGGER source_reference_catalog_immutable BEFORE UPDATE OR DELETE ON preprod_aga_demo.source_reference_catalog FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_mutation();
CREATE TRIGGER questions_immutable BEFORE UPDATE OR DELETE ON preprod_aga_demo.questions FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_mutation();
CREATE TRIGGER question_source_proposals_immutable BEFORE UPDATE OR DELETE ON preprod_aga_demo.question_source_proposals FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_mutation();
CREATE TRIGGER package_seals_immutable BEFORE UPDATE OR DELETE ON preprod_aga_demo.package_seals FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_mutation();
CREATE OR REPLACE FUNCTION preprod_aga_demo.reject_child_after_seal() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF EXISTS (SELECT 1 FROM preprod_aga_demo.package_seals WHERE package_digest = NEW.package_digest) THEN RAISE EXCEPTION 'sealed preprod AGA demo package cannot accept child rows'; END IF; RETURN NEW; END $$;
CREATE TRIGGER forms_after_seal BEFORE INSERT ON preprod_aga_demo.forms FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_child_after_seal();
CREATE TRIGGER form_source_proposals_after_seal BEFORE INSERT ON preprod_aga_demo.form_source_proposals FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_child_after_seal();
CREATE TRIGGER source_reference_catalog_after_seal BEFORE INSERT ON preprod_aga_demo.source_reference_catalog FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_child_after_seal();
CREATE TRIGGER questions_after_seal BEFORE INSERT ON preprod_aga_demo.questions FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_child_after_seal();
CREATE TRIGGER question_source_proposals_after_seal BEFORE INSERT ON preprod_aga_demo.question_source_proposals FOR EACH ROW EXECUTE FUNCTION preprod_aga_demo.reject_child_after_seal();
CREATE VIEW preprod_aga_demo.sealed_packages AS SELECT p.*, s.reconciliation_digest, s.seal_digest, s.relationship_digests, s.sealed_at FROM preprod_aga_demo.packages p JOIN preprod_aga_demo.package_seals s USING (package_digest);
CREATE VIEW preprod_aga_demo.sealed_forms AS SELECT f.* FROM preprod_aga_demo.forms f JOIN preprod_aga_demo.package_seals s USING (package_digest);
CREATE VIEW preprod_aga_demo.sealed_questions AS SELECT q.* FROM preprod_aga_demo.questions q JOIN preprod_aga_demo.package_seals s USING (package_digest);
ALTER FUNCTION preprod_aga_demo.reject_mutation() OWNER TO preprod_aga_demo_owner;
ALTER FUNCTION preprod_aga_demo.reject_child_after_seal() OWNER TO preprod_aga_demo_owner;
ALTER VIEW preprod_aga_demo.sealed_packages OWNER TO preprod_aga_demo_owner;
ALTER VIEW preprod_aga_demo.sealed_forms OWNER TO preprod_aga_demo_owner;
ALTER VIEW preprod_aga_demo.sealed_questions OWNER TO preprod_aga_demo_owner;
REVOKE ALL ON ALL TABLES IN SCHEMA preprod_aga_demo FROM PUBLIC, preprod_normal_api, preprod_aga_demo_reader;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA preprod_aga_demo FROM PUBLIC, preprod_normal_api, preprod_aga_demo_reader;
GRANT USAGE ON SCHEMA preprod_aga_demo TO preprod_aga_demo_writer;
GRANT USAGE ON SCHEMA preprod_aga_demo TO preprod_aga_demo_reader;
GRANT SELECT ON preprod_aga_demo.sealed_packages TO preprod_aga_demo_reader;
GRANT SELECT ON preprod_aga_demo.sealed_forms, preprod_aga_demo.sealed_questions TO preprod_aga_demo_reader;
GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA preprod_aga_demo TO preprod_aga_demo_writer;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA preprod_aga_demo TO preprod_aga_demo_writer;
`

// PostgresStore writes only the disposable preprod_aga_demo schema. It does
// not import a domain store and contains no provider client.
type PostgresStore struct{ pool *database.Pool }

func NewPostgresStore(pool *database.Pool) (*PostgresStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("AGA demo PostgreSQL pool is required")
	}
	return &PostgresStore{pool: pool}, nil
}

func (store *PostgresStore) Preflight(ctx context.Context, intent IntentManifest) error {
	if err := intent.Validate(); err != nil {
		return err
	}
	var databaseName, databaseUser, systemID string
	if err := store.pool.QueryRow(ctx, `SELECT current_database(), current_user, system_identifier::text FROM pg_control_system()`).Scan(&databaseName, &databaseUser, &systemID); err != nil {
		return fmt.Errorf("read PostgreSQL identity: %w", err)
	}
	if databaseName != intent.Target.DatabaseName || databaseUser != "preprod_aga_demo_writer" || systemID != intent.Target.PostgresSystemIdentifier {
		return fmt.Errorf("PostgreSQL target or writer identity mismatch")
	}
	var schemaOwner, relationCount, rowCount int64
	if err := store.pool.QueryRow(ctx, `
		SELECT (SELECT COUNT(*) FROM pg_namespace WHERE nspname = 'preprod_aga_demo' AND pg_get_userbyid(nspowner) = 'preprod_aga_demo_owner'),
		       (SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'preprod_aga_demo'),
		       (SELECT COUNT(*) FROM preprod_aga_demo.packages)`).Scan(&schemaOwner, &relationCount, &rowCount); err != nil {
		return fmt.Errorf("inspect AGA demo schema: %w", err)
	}
	if schemaOwner != 1 || relationCount != 8 || rowCount != 0 {
		return fmt.Errorf("AGA demo schema is absent, dirty, or divergent")
	}
	return nil
}

func (store *PostgresStore) Materialize(ctx context.Context, intent IntentManifest, pkg AcceptedPackage, expected map[string]string) (SealReceipt, error) {
	if err := intent.Validate(); err != nil {
		return SealReceipt{}, err
	}
	if err := packageMatchesIntent(pkg, intent); err != nil {
		return SealReceipt{}, err
	}
	computed, err := RelationshipDigests(pkg)
	if err != nil {
		return SealReceipt{}, err
	}
	if err := exactDigestSet(expected, computed); err != nil {
		return SealReceipt{}, err
	}
	var receipt SealReceipt
	err = database.WithinTransaction(ctx, store.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := postgresPreflightTx(ctx, tx, intent); err != nil {
			return err
		}
		if err := insertProjection(ctx, tx, intent, pkg); err != nil {
			return err
		}
		reconciled, err := reconcileProjection(ctx, tx)
		if err != nil {
			return err
		}
		if err := exactDigestSet(expected, reconciled); err != nil {
			return err
		}
		// PostgreSQL stores timestamptz values at microsecond precision. Bind the
		// seal to that exact persisted precision before hashing it, so a sealed
		// receipt can be reconstructed byte-for-byte by a later reader.
		sealedAt := postgresTimestamp(time.Now())
		reconciliationDigest := reconciled["projection"]
		sealDigest := projectionSealDigest(pkg.Identity.JSONSHA256, intent.IntentDigest, intent.TargetFingerprintDigest, reconciliationDigest, sealedAt)
		encodedDigests, err := json.Marshal(reconciled)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO preprod_aga_demo.package_seals (package_digest, intent_digest, target_digest, reconciliation_digest, seal_digest, relationship_digests, sealed_at) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7)`, pkg.Identity.JSONSHA256, intent.IntentDigest, intent.TargetFingerprintDigest, reconciliationDigest, sealDigest, string(encodedDigests), sealedAt); err != nil {
			return fmt.Errorf("insert final AGA demo reconciliation seal: %w", err)
		}
		receipt = SealReceipt{PackageDigest: pkg.Identity.JSONSHA256, IntentDigest: intent.IntentDigest, TargetDigest: intent.TargetFingerprintDigest, ReconciliationDigest: reconciliationDigest, SealDigest: sealDigest, SealedAt: sealedAt}
		return nil
	})
	if err != nil {
		return SealReceipt{}, err
	}
	return receipt, receipt.Validate(intent)
}

func (store *PostgresStore) VerifySeal(ctx context.Context, intent IntentManifest) (SealReceipt, error) {
	if err := intent.Validate(); err != nil {
		return SealReceipt{}, err
	}
	var receipt SealReceipt
	err := database.WithinTransaction(ctx, store.pool, func(ctx context.Context, tx pgx.Tx) error {
		var encodedDigests []byte
		if err := tx.QueryRow(ctx, `SELECT package_digest, intent_digest, target_digest, reconciliation_digest, seal_digest, relationship_digests, sealed_at FROM preprod_aga_demo.package_seals WHERE intent_digest = $1`, intent.IntentDigest).Scan(&receipt.PackageDigest, &receipt.IntentDigest, &receipt.TargetDigest, &receipt.ReconciliationDigest, &receipt.SealDigest, &encodedDigests, &receipt.SealedAt); err != nil {
			return fmt.Errorf("read AGA demo final seal: %w", err)
		}
		var sealedDigests map[string]string
		if err := json.Unmarshal(encodedDigests, &sealedDigests); err != nil {
			return fmt.Errorf("decode AGA demo seal reconciliation: %w", err)
		}
		reconciled, err := reconcileProjection(ctx, tx)
		if err != nil {
			return err
		}
		if err := exactDigestSet(sealedDigests, reconciled); err != nil || receipt.ReconciliationDigest != reconciled["projection"] {
			return fmt.Errorf("AGA demo seal reconciliation drift")
		}
		return nil
	})
	if err != nil {
		return SealReceipt{}, err
	}
	if err := receipt.Validate(intent); err != nil {
		return SealReceipt{}, err
	}
	if receipt.SealDigest != projectionSealDigest(receipt.PackageDigest, receipt.IntentDigest, receipt.TargetDigest, receipt.ReconciliationDigest, receipt.SealedAt) {
		return SealReceipt{}, fmt.Errorf("AGA demo seal digest mismatch")
	}
	return receipt, nil
}

func postgresPreflightTx(ctx context.Context, tx pgx.Tx, intent IntentManifest) error {
	var packageCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM preprod_aga_demo.packages`).Scan(&packageCount); err != nil {
		return err
	}
	if packageCount != 0 {
		return fmt.Errorf("AGA demo schema is not empty")
	}
	return nil
}

func insertProjection(ctx context.Context, tx pgx.Tx, intent IntentManifest, pkg AcceptedPackage) error {
	if err := insertCanonical(ctx, tx, `INSERT INTO preprod_aga_demo.package_intents (intent_digest, run_id, target_digest, payload, canonical_payload, row_digest, created_at) VALUES ($1,$2,$3,$5::jsonb,$6,$7,$4)`, intent, intent.IntentDigest, intent.RunID, intent.TargetFingerprintDigest, intent.CreatedAt); err != nil {
		return fmt.Errorf("insert package intent: %w", err)
	}
	if err := insertCanonical(ctx, tx, `INSERT INTO preprod_aga_demo.packages (package_digest, intent_digest, zip_digest, json_digest, manifest_digest, status, payload, canonical_payload, row_digest) VALUES ($1,$2,$3,$4,$5,'SEALED_PREPROD_DEMO_PROJECTION',$6::jsonb,$7,$8)`, projectionPackage(pkg), pkg.Identity.JSONSHA256, intent.IntentDigest, pkg.Identity.ZipSHA256, pkg.Identity.JSONSHA256, pkg.Identity.ManifestSHA256); err != nil {
		return fmt.Errorf("insert package: %w", err)
	}
	for index, source := range pkg.SourceCoverage {
		row := sourceCatalogProjectionRow{index + 1, source}
		if err := insertCanonical(ctx, tx, `INSERT INTO preprod_aga_demo.source_reference_catalog (package_digest, reference, payload, canonical_payload, row_digest) VALUES ($1,$2,$3::jsonb,$4,$5)`, row, pkg.Identity.JSONSHA256, source.Ref); err != nil {
			return fmt.Errorf("insert source reference %s: %w", source.Ref, err)
		}
	}
	for formIndex, form := range pkg.Forms {
		if err := insertCanonical(ctx, tx, `INSERT INTO preprod_aga_demo.forms (package_digest, form_code, form_digest, question_extraction_state, candidate_state, source_mapping_state, payload, canonical_payload, row_digest) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9)`, projectionForm(formIndex+1, form), pkg.Identity.JSONSHA256, form.FormCode, form.FormSHA256, form.QuestionExtractionState, form.CandidateState, form.SourceMappingState); err != nil {
			return fmt.Errorf("insert form %s: %w", form.FormCode, err)
		}
		for index, source := range form.FormSourceProposals {
			row := formSourceProjectionRow{form.FormCode, index + 1, source}
			if err := insertCanonical(ctx, tx, `INSERT INTO preprod_aga_demo.form_source_proposals (package_digest, form_code, ordinal, payload, canonical_payload, row_digest) VALUES ($1,$2,$3,$4::jsonb,$5,$6)`, row, pkg.Identity.JSONSHA256, form.FormCode, index+1); err != nil {
				return fmt.Errorf("insert form source proposal: %w", err)
			}
		}
		for _, question := range form.Questions {
			gap := "PROPOSAL_PRESENT_REVIEW_REQUIRED"
			if len(question.SourceProposals) == 0 {
				gap = "UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL"
			}
			row := projectionQuestion(form.FormCode, formIndex+1, question)
			if err := insertCanonical(ctx, tx, `INSERT INTO preprod_aga_demo.questions (package_digest, proposal_id, form_code, ordinal, text_digest, source_gap_category, candidate_state, source_mapping_state, risk_review_state, payload, canonical_payload, row_digest) VALUES ($1,$2,$3,$4,$5,$6,'NON_AUTHORITATIVE_CANDIDATE',$7,'CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW',$8::jsonb,$9,$10)`, row, pkg.Identity.JSONSHA256, question.ProposalID, form.FormCode, question.Ordinal, question.TextDigest, gap, question.SourceMappingState); err != nil {
				return fmt.Errorf("insert question %s: %w", question.ProposalID, err)
			}
			for index, source := range question.SourceProposals {
				proposalRow := questionSourceProjectionRow{question.ProposalID, index + 1, source}
				if err := insertCanonical(ctx, tx, `INSERT INTO preprod_aga_demo.question_source_proposals (package_digest, proposal_id, ordinal, payload, canonical_payload, row_digest) VALUES ($1,$2,$3,$4::jsonb,$5,$6)`, proposalRow, pkg.Identity.JSONSHA256, question.ProposalID, index+1); err != nil {
					return fmt.Errorf("insert question source proposal: %w", err)
				}
			}
		}
	}
	return nil
}

func insertCanonical(ctx context.Context, tx pgx.Tx, query string, value any, args ...any) error {
	canonical := string(mustCanonical(value))
	args = append(args, canonical, canonical, digestBytes([]byte(canonical)))
	_, err := tx.Exec(ctx, query, args...)
	return err
}

func reconcileProjection(ctx context.Context, tx pgx.Tx) (map[string]string, error) {
	relations := map[string]struct{ table, ordering string }{
		"package": {"packages", "package_digest"}, "forms": {"forms", "form_code"}, "formSourceProposals": {"form_source_proposals", "form_code, ordinal"}, "sourceReferenceCatalog": {"source_reference_catalog", "reference"}, "questions": {"questions", "form_code, ordinal, proposal_id"}, "questionSourceProposals": {"question_source_proposals", "proposal_id, ordinal"},
	}
	output := make(map[string]string, len(relations)+2)
	for relation, details := range relations {
		rows, err := tx.Query(ctx, "SELECT canonical_payload, row_digest FROM preprod_aga_demo."+details.table+" ORDER BY "+details.ordering)
		if err != nil {
			return nil, fmt.Errorf("read %s for reconciliation: %w", relation, err)
		}
		var values [][]byte
		for rows.Next() {
			var canonical, rowDigest string
			if err := rows.Scan(&canonical, &rowDigest); err != nil {
				rows.Close()
				return nil, err
			}
			if digestBytes([]byte(canonical)) != rowDigest {
				rows.Close()
				return nil, fmt.Errorf("stored %s row digest mismatch", relation)
			}
			values = append(values, []byte(canonical))
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
		output[relation] = relationDigest(relation, values)
	}
	requirementRows := make([][]byte, 0, len(sourceResolutionRequirements))
	for _, requirement := range sourceResolutionRequirements {
		requirementRows = append(requirementRows, []byte(requirement))
	}
	output["sourceResolutionRequirements"] = relationDigest("sourceResolutionRequirements", requirementRows)
	output["projection"] = relationDigest("projection", orderedDigestValues(output))
	return output, nil
}

func projectionSealDigest(packageDigest, intentDigest, targetDigest, reconciliationDigest string, sealedAt time.Time) string {
	return digestBytes(mustCanonical(struct {
		PackageDigest        string    `json:"packageDigest"`
		IntentDigest         string    `json:"intentDigest"`
		TargetDigest         string    `json:"targetDigest"`
		ReconciliationDigest string    `json:"reconciliationDigest"`
		SealedAt             time.Time `json:"sealedAt"`
	}{packageDigest, intentDigest, targetDigest, reconciliationDigest, sealedAt.UTC()}))
}

func postgresTimestamp(value time.Time) time.Time {
	return value.UTC().Truncate(time.Microsecond)
}
