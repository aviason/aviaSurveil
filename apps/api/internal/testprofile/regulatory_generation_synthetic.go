package testprofile

import (
	"context"
	"fmt"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/regulatory"
	"github.com/jackc/pgx/v5"
)

// BootstrapSyntheticRegulatoryGenerationInputs is an internal test-profile-only
// fixture. It creates only an explicit synthetic baseline-currentness receipt;
// it never creates a source-change impact Draft, technical review,
// publication, Audit package, or real-source claim.
func BootstrapSyntheticRegulatoryGenerationInputs(ctx context.Context, pool *database.Pool) error {
	if pool == nil {
		return fmt.Errorf("synthetic regulatory bootstrap requires database")
	}
	var sourceCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_source_versions WHERE id IN ('SOURCE-SYNTHETIC-OPS-AOC','SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2')`).Scan(&sourceCount); err != nil {
		return err
	}
	if sourceCount == 2 {
		var clauses, rows, inputRows, holdoutRows, impactClauses, impactRows, impactInputRows, impactHoldoutRows int
		if err := pool.QueryRow(ctx, `SELECT
            (SELECT COUNT(*) FROM regulatory_normalized_clauses WHERE regulatory_source_version_id='SOURCE-SYNTHETIC-OPS-AOC'),
            (SELECT COUNT(*) FROM state_compliance_crosswalk_rows WHERE regulatory_source_version_id='SOURCE-SYNTHETIC-OPS-AOC'),
			(SELECT COUNT(*) FROM regulatory_evaluation_partition_rows WHERE partition_id='PARTITION-SYNTHETIC-INPUT'),
			(SELECT COUNT(*) FROM regulatory_evaluation_partition_rows WHERE partition_id='PARTITION-SYNTHETIC-HOLDOUT'),
			(SELECT COUNT(*) FROM regulatory_normalized_clauses WHERE regulatory_source_version_id='SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2'),
			(SELECT COUNT(*) FROM state_compliance_crosswalk_rows WHERE regulatory_source_version_id='SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2'),
			(SELECT COUNT(*) FROM regulatory_evaluation_partition_rows WHERE partition_id='PARTITION-SYNTHETIC-IMPACT-INPUT'),
			(SELECT COUNT(*) FROM regulatory_evaluation_partition_rows WHERE partition_id='PARTITION-SYNTHETIC-IMPACT-HOLDOUT')`).Scan(&clauses, &rows, &inputRows, &holdoutRows, &impactClauses, &impactRows, &impactInputRows, &impactHoldoutRows); err != nil {
			return err
		}
		if clauses == 2 && rows == 2 && inputRows == 1 && holdoutRows == 1 && impactClauses == 2 && impactRows == 2 && impactInputRows == 1 && impactHoldoutRows == 1 {
			if err := ensureSyntheticRegulatoryGenerationFoundation(ctx, pool); err != nil {
				return err
			}
			return ensureSyntheticBaselineSourceCurrentness(ctx, pool)
		}
		return fmt.Errorf("synthetic regulatory test profile is incomplete or conflicting")
	}
	if sourceCount != 0 {
		return fmt.Errorf("synthetic regulatory test profile has conflicting source identities")
	}
	if err := ensureSyntheticRegulatoryGenerationFoundation(ctx, pool); err != nil {
		return err
	}
	statements := []string{
		`INSERT INTO regulatory_source_versions (id, source_identity, version_identity, title, source_class, source_status, source_locator, source_hash, effective_from, source_metadata) VALUES ('SOURCE-SYNTHETIC-OPS-AOC', 'SYNTHETIC-OPS-AOC', '1', 'Synthetic test-profile source', 'STATE_COMPLIANCE_CROSSWALK', 'SUPPLIED_WORKING_COPY', 'Synthetic OPS/AOC source', 'sha256:1111111111111111111111111111111111111111111111111111111111111111', '2025-01-01', '{"testProfileOnly":true}')`,
		`INSERT INTO regulatory_normalized_clauses (id, regulatory_source_version_id, clause_identity, annex_identity, section_identity, clause_locator, source_hash, normalized_digest) VALUES ('CLAUSE-SYNTHETIC-OPS-AOC-1', 'SOURCE-SYNTHETIC-OPS-AOC', 'SYNTHETIC-OPS-AOC-1', 'SYNTHETIC', '1', 'Synthetic OPS/AOC 1', 'sha256:1111111111111111111111111111111111111111111111111111111111111111', 'sha256:2222222222222222222222222222222222222222222222222222222222222222')`,
		`INSERT INTO regulatory_normalized_clauses (id, regulatory_source_version_id, clause_identity, annex_identity, section_identity, clause_locator, source_hash, normalized_digest) VALUES ('CLAUSE-SYNTHETIC-OPS-AOC-HOLDOUT-1', 'SOURCE-SYNTHETIC-OPS-AOC', 'SYNTHETIC-OPS-AOC-HOLDOUT-1', 'SYNTHETIC', 'HOLDOUT-1', 'Synthetic OPS/AOC holdout 1', 'sha256:1111111111111111111111111111111111111111111111111111111111111111', 'sha256:3333333333333333333333333333333333333333333333333333333333333333')`,
		`INSERT INTO state_compliance_crosswalk_rows (id, regulatory_source_version_id, normalized_clause_id, stable_row_identity, annex_identity, section_identity, row_digest) VALUES ('CCROW-SYNTHETIC-OPS-AOC-1', 'SOURCE-SYNTHETIC-OPS-AOC', 'CLAUSE-SYNTHETIC-OPS-AOC-1', 'CC:SYNTHETIC:OPS:AOC:1', 'SYNTHETIC', '1', 'sha256:2222222222222222222222222222222222222222222222222222222222222222'), ('CCROW-SYNTHETIC-OPS-AOC-HOLDOUT-1', 'SOURCE-SYNTHETIC-OPS-AOC', 'CLAUSE-SYNTHETIC-OPS-AOC-HOLDOUT-1', 'CC:SYNTHETIC:OPS:AOC:HOLDOUT:1', 'SYNTHETIC', 'HOLDOUT-1', 'sha256:3333333333333333333333333333333333333333333333333333333333333333')`,
		`INSERT INTO regulatory_evaluations (id, evaluation_identity, purpose) VALUES ('EVAL-SYNTHETIC-OPS-AOC', 'SYNTHETIC-OPS-AOC', 'test-profile only')`,
		`INSERT INTO regulatory_evaluation_partitions (id, evaluation_id, partition_kind) VALUES ('PARTITION-SYNTHETIC-INPUT', 'EVAL-SYNTHETIC-OPS-AOC', 'GENERATION_INPUT'), ('PARTITION-SYNTHETIC-HOLDOUT', 'EVAL-SYNTHETIC-OPS-AOC', 'BLIND_HOLDOUT')`,
		`INSERT INTO regulatory_evaluation_partition_rows (evaluation_id, partition_id, state_compliance_crosswalk_row_id, stable_row_identity) VALUES ('EVAL-SYNTHETIC-OPS-AOC', 'PARTITION-SYNTHETIC-INPUT', 'CCROW-SYNTHETIC-OPS-AOC-1', 'CC:SYNTHETIC:OPS:AOC:1'), ('EVAL-SYNTHETIC-OPS-AOC', 'PARTITION-SYNTHETIC-HOLDOUT', 'CCROW-SYNTHETIC-OPS-AOC-HOLDOUT-1', 'CC:SYNTHETIC:OPS:AOC:HOLDOUT:1')`,
		`INSERT INTO regulatory_source_versions (id, source_identity, version_identity, title, source_class, source_status, source_locator, source_hash, effective_from, source_metadata) VALUES ('SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2', 'SYNTHETIC-OPS-AOC', '2', 'Synthetic test-profile impact source', 'STATE_COMPLIANCE_CROSSWALK', 'SUPPLIED_WORKING_COPY', 'Synthetic OPS/AOC impact source', 'sha256:4444444444444444444444444444444444444444444444444444444444444444', '2025-02-01', '{"testProfileOnly":true,"impactProfile":true}')`,
		`INSERT INTO regulatory_normalized_clauses (id, regulatory_source_version_id, clause_identity, annex_identity, section_identity, clause_locator, source_hash, normalized_digest) VALUES ('CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-2', 'SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2', 'SYNTHETIC-OPS-AOC-IMPACT-2', 'SYNTHETIC', '2', 'Synthetic OPS/AOC impact 2', 'sha256:4444444444444444444444444444444444444444444444444444444444444444', 'sha256:5555555555555555555555555555555555555555555555555555555555555555'), ('CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-HOLDOUT-2', 'SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2', 'SYNTHETIC-OPS-AOC-IMPACT-HOLDOUT-2', 'SYNTHETIC', 'HOLDOUT-2', 'Synthetic OPS/AOC impact holdout 2', 'sha256:4444444444444444444444444444444444444444444444444444444444444444', 'sha256:6666666666666666666666666666666666666666666666666666666666666666')`,
		`INSERT INTO state_compliance_crosswalk_rows (id, regulatory_source_version_id, normalized_clause_id, stable_row_identity, annex_identity, section_identity, row_digest) VALUES ('CCROW-SYNTHETIC-OPS-AOC-IMPACT-2', 'SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2', 'CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-2', 'CC:SYNTHETIC:OPS:AOC:IMPACT:2', 'SYNTHETIC', '2', 'sha256:5555555555555555555555555555555555555555555555555555555555555555'), ('CCROW-SYNTHETIC-OPS-AOC-IMPACT-HOLDOUT-2', 'SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2', 'CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-HOLDOUT-2', 'CC:SYNTHETIC:OPS:AOC:IMPACT:HOLDOUT:2', 'SYNTHETIC', 'HOLDOUT-2', 'sha256:6666666666666666666666666666666666666666666666666666666666666666')`,
		`INSERT INTO regulatory_evaluations (id, evaluation_identity, purpose) VALUES ('EVAL-SYNTHETIC-OPS-AOC-IMPACT-V2', 'SYNTHETIC-OPS-AOC-IMPACT-V2', 'test-profile only source-change impact review')`,
		`INSERT INTO regulatory_evaluation_partitions (id, evaluation_id, partition_kind) VALUES ('PARTITION-SYNTHETIC-IMPACT-INPUT', 'EVAL-SYNTHETIC-OPS-AOC-IMPACT-V2', 'GENERATION_INPUT'), ('PARTITION-SYNTHETIC-IMPACT-HOLDOUT', 'EVAL-SYNTHETIC-OPS-AOC-IMPACT-V2', 'BLIND_HOLDOUT')`,
		`INSERT INTO regulatory_evaluation_partition_rows (evaluation_id, partition_id, state_compliance_crosswalk_row_id, stable_row_identity) VALUES ('EVAL-SYNTHETIC-OPS-AOC-IMPACT-V2', 'PARTITION-SYNTHETIC-IMPACT-INPUT', 'CCROW-SYNTHETIC-OPS-AOC-IMPACT-2', 'CC:SYNTHETIC:OPS:AOC:IMPACT:2'), ('EVAL-SYNTHETIC-OPS-AOC-IMPACT-V2', 'PARTITION-SYNTHETIC-IMPACT-HOLDOUT', 'CCROW-SYNTHETIC-OPS-AOC-IMPACT-HOLDOUT-2', 'CC:SYNTHETIC:OPS:AOC:IMPACT:HOLDOUT:2')`,
	}
	if err := database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		for _, statement := range statements {
			if _, err := tx.Exec(ctx, statement); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return ensureSyntheticBaselineSourceCurrentness(ctx, pool)
}

// ensureSyntheticBaselineSourceCurrentness deliberately invokes the same
// controlled command as the canonical boundary. V2 remains merely supplied
// until an explicit source-change activation occurs in a test; keeping it
// inert proves that a raw newer source row cannot silently make a candidate
// current or publishable.
func ensureSyntheticBaselineSourceCurrentness(ctx context.Context, pool *database.Pool) error {
	_, err := regulatory.NewAdminService(pool, func() time.Time {
		return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	}).ActivateSourceCurrentness(ctx, identity.Principal{
		SubjectID: "SYNTHETIC-REGULATORY-GENERATOR",
		Roles:     []identity.Role{identity.RoleAdmin},
	}, regulatory.SourceCurrentnessActivationCommand{
		OperationID:             "TESTPROFILE-SOURCE-CURRENTNESS-BASELINE-V1",
		IdempotencyKey:          "TESTPROFILE-SOURCE-CURRENTNESS-BASELINE-V1",
		CurrentSourceSnapshotID: "SOURCE-SYNTHETIC-OPS-AOC",
		CurrentSourceHash:       "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		Reason:                  "Synthetic internal test-profile baseline currentness declaration.",
	})
	return err
}

// ensureSyntheticRegulatoryGenerationFoundation repairs the fixture's
// non-regulatory identity boundary after a canonical-profile reset. The reset
// deliberately removes organizations and their targets/scopes, while the
// immutable synthetic source rows remain so source-history tests can run. A
// complete fixture therefore requires both sets of records before a generated
// candidate may be imported.
func ensureSyntheticRegulatoryGenerationFoundation(ctx context.Context, pool *database.Pool) error {
	statements := []string{
		`INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('SYNTHETIC-REGULATORY-GENERATOR', 'test-profile', 'Synthetic Regulatory Generator') ON CONFLICT (subject_id) DO NOTHING`,
		`INSERT INTO organizations (id, legal_name, organization_type, status) VALUES ('ORG-SYNTHETIC-AOC', 'Synthetic AOC', 'OPERATOR', 'ACTIVE') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES ('TARGET-SYNTHETIC-AOC', 'ORGANIZATION', 'ORG-SYNTHETIC-AOC') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id) VALUES ('SCOPE-SYNTHETIC-AOC', 'ORG-SYNTHETIC-AOC', 'AIR_OPERATOR', 'AOC-SYNTHETIC', 'ACTIVE', '2025-01-01', 'TARGET-SYNTHETIC-AOC') ON CONFLICT (id) DO NOTHING`,
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		for _, statement := range statements {
			if _, err := tx.Exec(ctx, statement); err != nil {
				return err
			}
		}
		var identity, organization, target, scope bool
		if err := tx.QueryRow(ctx, `SELECT
			EXISTS (SELECT 1 FROM identity_references WHERE subject_id='SYNTHETIC-REGULATORY-GENERATOR' AND issuer='test-profile' AND display_name='Synthetic Regulatory Generator'),
			EXISTS (SELECT 1 FROM organizations WHERE id='ORG-SYNTHETIC-AOC' AND legal_name='Synthetic AOC' AND organization_type='OPERATOR' AND status='ACTIVE'),
			EXISTS (SELECT 1 FROM regulated_targets WHERE id='TARGET-SYNTHETIC-AOC' AND target_kind='ORGANIZATION' AND organization_id='ORG-SYNTHETIC-AOC'),
			EXISTS (SELECT 1 FROM organization_service_provider_scopes WHERE id='SCOPE-SYNTHETIC-AOC' AND organization_id='ORG-SYNTHETIC-AOC' AND service_provider_type_id='AIR_OPERATOR' AND authorization_identifier='AOC-SYNTHETIC' AND status='ACTIVE' AND effective_from='2025-01-01' AND primary_target_id='TARGET-SYNTHETIC-AOC')`).Scan(&identity, &organization, &target, &scope); err != nil {
			return err
		}
		if !identity || !organization || !target || !scope {
			return fmt.Errorf("synthetic regulatory test profile has conflicting identity, organization, target, or scope foundation")
		}
		return nil
	})
}

// BootstrapBlockedRealOPSAOCGenerationInputs creates only the predecessor
// facts needed to resolve the deliberately blocked real OPS/AOC request. It
// never creates generation output, decisions, publication, Audit data, or any
// source-owner confirmation.
func BootstrapBlockedRealOPSAOCGenerationInputs(ctx context.Context, pool *database.Pool) error {
	if pool == nil {
		return fmt.Errorf("blocked real OPS/AOC bootstrap requires database")
	}
	statements := []string{
		`INSERT INTO organizations (id, legal_name, organization_type, status) VALUES ('ORG-FLY-NAMIBIA', 'Fly Namibia', 'OPERATOR', 'ACTIVE') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES ('TARGET-OPS-AOC-SOURCE-BOUND', 'ORGANIZATION', 'ORG-FLY-NAMIBIA') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id) VALUES ('SCOPE-OPS-AOC-SOURCE-BOUND', 'ORG-FLY-NAMIBIA', 'AIR_OPERATOR', 'AOC-FLY-NAMIBIA-SOURCE-BOUND', 'ACTIVE', '2025-01-01', 'TARGET-OPS-AOC-SOURCE-BOUND') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO regulatory_evaluations (id, evaluation_identity, purpose) VALUES ('EVAL-OPS-AOC-SOURCE-BOUND', 'OPS-AOC-SOURCE-BOUND', 'blocked real request identity resolution only') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO regulatory_evaluation_partitions (id, evaluation_id, partition_kind) VALUES ('CC-OPS-TRAIN-1', 'EVAL-OPS-AOC-SOURCE-BOUND', 'GENERATION_INPUT'), ('CC-OPS-HOLDOUT-1', 'EVAL-OPS-AOC-SOURCE-BOUND', 'BLIND_HOLDOUT') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO regulatory_evaluation_partition_rows (evaluation_id, partition_id, state_compliance_crosswalk_row_id, stable_row_identity) VALUES ('EVAL-OPS-AOC-SOURCE-BOUND', 'CC-OPS-TRAIN-1', 'NCAA-CC-A610-ROW-4.2.2.2', 'CC:NAMB:ANNEX6:4.2.2.2'), ('EVAL-OPS-AOC-SOURCE-BOUND', 'CC-OPS-HOLDOUT-1', 'NCAA-CC-A610-ROW-4.2.12.1', 'CC:NAMB:ANNEX6:4.2.12.1') ON CONFLICT DO NOTHING`,
		`INSERT INTO regulatory_source_gap_facts (id, regulatory_source_version_id, gap_id, reason, ordinal) VALUES
			('GAP-OPS-AOC-CONTROLLED-PROCEDURE', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'CONTROLLED_PROCEDURE', 'The controlled NCAA Operations surveillance/ramp-inspection procedure has not been supplied.', 0),
			('GAP-OPS-AOC-PART-140', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'PART_140_AUTHORITY', 'Current Part 140 authority and supersession require source-owner confirmation.', 1),
			('GAP-OPS-AOC-PART-127', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'PART_127_APPLICABILITY', 'Exact Part 127 operation/configuration applicability requires Department Manager confirmation.', 2),
			('GAP-OPS-AOC-AMBIGUOUS-OWNERSHIP', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'AMBIGUOUS_OWNERSHIP', 'Exact source ownership and controlled-procedure stewardship remain unresolved.', 3)
		 ON CONFLICT (id) DO NOTHING`,
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		for _, statement := range statements {
			if _, err := tx.Exec(ctx, statement); err != nil {
				return err
			}
		}
		var scope, source, input, holdout bool
		if err := tx.QueryRow(ctx, `SELECT
            EXISTS (SELECT 1 FROM organization_service_provider_scopes WHERE id='SCOPE-OPS-AOC-SOURCE-BOUND' AND organization_id='ORG-FLY-NAMIBIA' AND service_provider_type_id='AIR_OPERATOR' AND authorization_identifier='AOC-FLY-NAMIBIA-SOURCE-BOUND' AND status='ACTIVE' AND primary_target_id='TARGET-OPS-AOC-SOURCE-BOUND'),
            EXISTS (SELECT 1 FROM regulatory_normalized_clauses clause JOIN regulatory_source_versions source ON source.id=clause.regulatory_source_version_id WHERE source.id='NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28' AND source.source_hash='sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2' AND clause.id='NCAA-CC-A610-4.2.2.2' AND clause.clause_locator='Annex 6 Part I 4.2.2.2'),
            EXISTS (SELECT 1 FROM regulatory_evaluation_partition_rows row WHERE row.partition_id='CC-OPS-TRAIN-1' AND row.state_compliance_crosswalk_row_id='NCAA-CC-A610-ROW-4.2.2.2' AND row.stable_row_identity='CC:NAMB:ANNEX6:4.2.2.2'),
            EXISTS (SELECT 1 FROM regulatory_evaluation_partition_rows row WHERE row.partition_id='CC-OPS-HOLDOUT-1' AND row.state_compliance_crosswalk_row_id='NCAA-CC-A610-ROW-4.2.12.1' AND row.stable_row_identity='CC:NAMB:ANNEX6:4.2.12.1')`).Scan(&scope, &source, &input, &holdout); err != nil {
			return err
		}
		if !scope || !source || !input || !holdout {
			return fmt.Errorf("blocked real OPS/AOC bootstrap is incomplete or conflicting")
		}
		return nil
	})
}
