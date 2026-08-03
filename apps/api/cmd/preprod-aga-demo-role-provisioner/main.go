package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agacandidatedemo"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		slog.Error("AGA demo role provisioner failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string) error {
	if len(arguments) > 1 || (len(arguments) == 1 && arguments[0] != "verify-least-privilege" && arguments[0] != "verify-sealed-projection") {
		return fmt.Errorf("usage: preprod-aga-demo-role-provisioner [verify-least-privilege|verify-sealed-projection]")
	}
	ownerPassword, err := readSecret("/run/secrets/preprod_app_database_password")
	if err != nil {
		return err
	}
	normalPassword, err := readSecret("/run/secrets/preprod_normal_api_database_password")
	if err != nil {
		return err
	}
	readerPassword, err := readSecret("/run/secrets/preprod_aga_demo_reader_database_password")
	if err != nil {
		return err
	}
	writerPassword, err := readSecret("/run/secrets/preprod_aga_demo_writer_database_password")
	if err != nil {
		return err
	}
	connection := databaseURL("aviasurveil360_preprod_loader", ownerPassword)
	pool, err := database.Open(ctx, connection)
	if err != nil {
		return fmt.Errorf("open AGA demo bootstrap PostgreSQL connection: %w", err)
	}
	defer pool.Close()
	if len(arguments) == 0 {
		if err := agacandidatedemo.ProvisionOverlaySchema(ctx, pool, agacandidatedemo.RolePasswords{NormalAPI: normalPassword, Reader: readerPassword, Writer: writerPassword}); err != nil {
			return err
		}
		fmt.Println("AGA demo roles and empty immutable schema provisioned")
		return nil
	}
	if arguments[0] == "verify-least-privilege" {
		return verifyLeastPrivilege(ctx, normalPassword, readerPassword, writerPassword)
	}
	return verifySealedProjection(ctx, readerPassword)
}

func databaseURL(role, password string) string {
	return (&url.URL{Scheme: "postgres", User: url.UserPassword(role, password), Host: net.JoinHostPort("preprod-postgres", "5432"), Path: "aviasurveil360_local_preprod", RawQuery: "sslmode=disable"}).String()
}

func verifyLeastPrivilege(ctx context.Context, normalPassword, readerPassword, writerPassword string) error {
	normal, err := database.Open(ctx, databaseURL("preprod_normal_api", normalPassword))
	if err != nil {
		return fmt.Errorf("open normal API privilege probe: %w", err)
	}
	defer normal.Close()
	if err := verifyNormalAPIAuthenticationSurface(ctx, normal); err != nil {
		return err
	}
	if err := mustFail(ctx, normal, "SELECT count(*) FROM preprod_aga_demo.sealed_packages"); err != nil {
		return fmt.Errorf("normal API overlay isolation: %w", err)
	}

	reader, err := database.Open(ctx, databaseURL("preprod_aga_demo_reader", readerPassword))
	if err != nil {
		return fmt.Errorf("open reader privilege probe: %w", err)
	}
	defer reader.Close()
	var sealedPackages int
	if err := reader.QueryRow(ctx, "SELECT count(*) FROM preprod_aga_demo.sealed_packages").Scan(&sealedPackages); err != nil || sealedPackages != 1 {
		return fmt.Errorf("reader sealed-view receipt: count=%d error=%v", sealedPackages, err)
	}
	for _, statement := range []string{
		"SELECT count(*) FROM preprod_aga_demo.packages",
		"INSERT INTO preprod_aga_demo.forms (package_digest, form_code, form_digest, question_extraction_state, candidate_state, source_mapping_state, payload, canonical_payload, row_digest) SELECT package_digest, 'PRIVILEGE-PROBE', 'sha256:0000000000000000000000000000000000000000000000000000000000000000', 'EXTRACTED_CANDIDATE_BOUNDARIES', 'NOT_IMPORTED', 'SOURCE_MAPPING_REQUIRED', '{}'::jsonb, '{}', 'sha256:0000000000000000000000000000000000000000000000000000000000000000' FROM preprod_aga_demo.sealed_packages",
		"SET ROLE preprod_aga_demo_writer",
	} {
		if err := mustFail(ctx, reader, statement); err != nil {
			return fmt.Errorf("reader least privilege: %w", err)
		}
	}

	writer, err := database.Open(ctx, databaseURL("preprod_aga_demo_writer", writerPassword))
	if err != nil {
		return fmt.Errorf("open writer privilege probe: %w", err)
	}
	defer writer.Close()
	for _, statement := range []string{
		"INSERT INTO preprod_aga_demo.form_source_proposals (package_digest, form_code, ordinal, payload, canonical_payload, row_digest) SELECT package_digest, form_code, 9999, '{}'::jsonb, '{}', 'sha256:0000000000000000000000000000000000000000000000000000000000000000' FROM preprod_aga_demo.sealed_forms LIMIT 1",
		"DELETE FROM preprod_aga_demo.forms WHERE package_digest = (SELECT package_digest FROM preprod_aga_demo.sealed_packages)",
		"CREATE TABLE preprod_aga_demo.privilege_probe (id integer)",
		"SET ROLE preprod_aga_demo_owner",
	} {
		if err := mustFail(ctx, writer, statement); err != nil {
			return fmt.Errorf("writer least privilege: %w", err)
		}
	}
	fmt.Printf("AGA demo least-privilege credential matrix verified: sealedPackages=%d\n", sealedPackages)
	return nil
}

type normalAPIAuthenticationProbe struct {
	label     string
	statement string
}

func normalAPIAuthenticationProbes() []normalAPIAuthenticationProbe {
	return []normalAPIAuthenticationProbe{
		{
			label: "transaction-trace-context",
			statement: `SELECT set_config('avia.traceparent',
				'00-00000000000000000000000000000001-0000000000000001-01', true)`,
		},
		{
			label:     "session-lock-read",
			statement: "SELECT 1 FROM session_references WHERE false FOR UPDATE",
		},
		{
			label: "desired-authority-read",
			statement: `SELECT 1
				FROM desired_membership_versions version
				JOIN desired_membership_sync sync
				  ON sync.membership_id = version.membership_id
				WHERE false`,
		},
		{
			label:     "identity-reference-read",
			statement: "SELECT 1 FROM identity_references WHERE false",
		},
		{
			label:     "profile-read",
			statement: "SELECT 1 FROM user_profiles WHERE false",
		},
		{
			label: "department-authority-read",
			statement: `SELECT 1
				FROM caa_department_memberships membership
				JOIN caa_department_status_facts department_status
				  ON department_status.department_id = membership.department_id
				JOIN caa_organizational_unit_status_facts unit_status
				  ON unit_status.organizational_unit_id = membership.organizational_unit_id
				JOIN caa_organizational_units unit
				  ON unit.id = membership.organizational_unit_id
				WHERE false`,
		},
		{
			label:     "session-zero-row-refresh",
			statement: "UPDATE session_references SET last_seen_at = last_seen_at WHERE false",
		},
	}
}

func verifyNormalAPIAuthenticationSurface(
	ctx context.Context,
	pool *database.Pool,
) error {
	transaction, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("normal API auth-surface probe could not begin")
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	for _, probe := range normalAPIAuthenticationProbes() {
		if _, err := transaction.Exec(ctx, probe.statement); err != nil {
			return fmt.Errorf(
				"normal API auth-surface privilege is missing: %s",
				probe.label,
			)
		}
	}
	if err := transaction.Rollback(ctx); err != nil {
		return fmt.Errorf("normal API auth-surface probe rollback failed")
	}
	return nil
}

func mustFail(ctx context.Context, pool *database.Pool, statement string) error {
	if _, err := pool.Exec(ctx, statement); err == nil {
		return fmt.Errorf("unexpectedly permitted operation")
	}
	return nil
}

func verifySealedProjection(ctx context.Context, readerPassword string) error {
	reader, err := database.Open(ctx, databaseURL("preprod_aga_demo_reader", readerPassword))
	if err != nil {
		return fmt.Errorf("open sealed projection reader: %w", err)
	}
	defer reader.Close()
	expected := agacandidatedemo.ExactAcceptedPackage()
	var packageDigest, reconciliationDigest, sealDigest string
	var relationshipJSON []byte
	var sourceReferences int
	if err := reader.QueryRow(ctx, `
		SELECT package_digest, reconciliation_digest, seal_digest,
		       relationship_digests, (payload->>'sourceReferenceCount')::integer
		FROM preprod_aga_demo.sealed_packages
	`).Scan(&packageDigest, &reconciliationDigest, &sealDigest, &relationshipJSON, &sourceReferences); err != nil {
		return fmt.Errorf("read sole sealed-package receipt: %w", err)
	}
	if packageDigest != expected.JSONSHA256 || !validDigest(reconciliationDigest) || !validDigest(sealDigest) || sourceReferences != expected.ExpectedCounts.UniqueSourceReferences {
		return fmt.Errorf("sealed-package identity or source catalog mismatch")
	}
	var relationships map[string]string
	if err := json.Unmarshal(relationshipJSON, &relationships); err != nil || len(relationships) != 8 {
		return fmt.Errorf("sealed relationship receipt mismatch")
	}
	for _, digest := range relationships {
		if !validDigest(digest) {
			return fmt.Errorf("sealed relationship digest is invalid")
		}
	}
	var forms, extractedForms, reviewForms, formLinks int
	if err := reader.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE question_extraction_state = 'EXTRACTED_CANDIDATE_BOUNDARIES'),
		       count(*) FILTER (WHERE question_extraction_state = 'NO_PROTOCOL_QUESTION_BOUNDARY_DETECTED'),
		       sum((payload->>'formSourceProposalCount')::integer)
		FROM preprod_aga_demo.sealed_forms
	`).Scan(&forms, &extractedForms, &reviewForms, &formLinks); err != nil {
		return fmt.Errorf("read sealed form reconciliation: %w", err)
	}
	if forms != expected.ExpectedCounts.Forms || extractedForms != expected.ExpectedCounts.FormsWithCandidateBoundaries ||
		reviewForms != len(expected.ZeroFormCodes) || formLinks != expected.ExpectedCounts.FormSourceProposalLinks {
		return fmt.Errorf("sealed form count mismatch")
	}
	rows, err := reader.Query(ctx, `SELECT form_code FROM preprod_aga_demo.sealed_forms ORDER BY form_code`)
	if err != nil {
		return err
	}
	var formCodes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			rows.Close()
			return err
		}
		formCodes = append(formCodes, code)
	}
	rows.Close()
	if !sameStrings(formCodes, expected.FormCodes) {
		return fmt.Errorf("sealed form identity set mismatch")
	}
	var questions, proposed, unmapped, exactSource, extracted, blockers, questionLinks int
	var fixedStates, nullAuthority bool
	if err := reader.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE source_gap_category = 'PROPOSAL_PRESENT_REVIEW_REQUIRED'),
		       count(*) FILTER (WHERE source_gap_category = 'UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL'),
		       count(*) FILTER (WHERE payload->'question'->>'extractionState' = 'EXACT_SOURCE_BACKED'),
		       count(*) FILTER (WHERE payload->'question'->>'extractionState' = 'EXTRACTED_CANDIDATE'),
		       count(*) FILTER (WHERE payload->'question'->'proposedRisk'->>'band' = 'PROPOSED_REVIEW_REQUIRED'),
		       sum((payload->>'sourceProposalCount')::integer),
		       bool_and(candidate_state = 'NON_AUTHORITATIVE_CANDIDATE'
		           AND source_mapping_state = 'SOURCE_MAPPING_REQUIRED'
		           AND risk_review_state = 'CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW'),
		       bool_and(NOT (payload->'question' ? 'approvedRisk')
		           AND NOT (payload->'question' ? 'safetyCriticalDecision')
		           AND NOT (payload->'question' ? 'findingSeverity')
		           AND payload->'question'->>'decisionState' = 'NOT_SUPPLIED'
		           AND payload->'question'->>'sourceAuthorityState' = 'NOT_ATTESTED')
		FROM preprod_aga_demo.sealed_questions
	`).Scan(&questions, &proposed, &unmapped, &exactSource, &extracted, &blockers, &questionLinks, &fixedStates, &nullAuthority); err != nil {
		return fmt.Errorf("read sealed question reconciliation: %w", err)
	}
	counts := expected.ExpectedCounts
	if questions != counts.Questions || proposed != counts.QuestionsWithProposals || unmapped != counts.UnmappedQuestions ||
		exactSource != 28 || extracted != 1282 || blockers != counts.ExpertRiskReviewBlockers ||
		questionLinks != counts.QuestionSourceProposalLinks || !fixedStates || !nullAuthority {
		return fmt.Errorf("sealed question count, state, or null-authority mismatch")
	}
	if err := verifyDistribution(ctx, reader, `SELECT payload->'form'->'proposedRisk'->>'band', count(*) FROM preprod_aga_demo.sealed_forms GROUP BY 1`, expected.FormRiskBands); err != nil {
		return fmt.Errorf("sealed form risk distribution: %w", err)
	}
	if err := verifyDistribution(ctx, reader, `SELECT payload->'question'->'proposedRisk'->>'band', count(*) FROM preprod_aga_demo.sealed_questions GROUP BY 1`, expected.RiskBands); err != nil {
		return fmt.Errorf("sealed question risk distribution: %w", err)
	}
	fmt.Printf("AGA demo sealed-view reconciliation verified: forms=%d questions=%d seal=%s reconciliation=%s\n", forms, questions, sealDigest, reconciliationDigest)
	return nil
}

func verifyDistribution(ctx context.Context, pool *database.Pool, query string, expected map[string]int) error {
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	actual := make(map[string]int, len(expected))
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return err
		}
		actual[key] = count
	}
	if len(actual) != len(expected) {
		return fmt.Errorf("distribution key count mismatch")
	}
	for key, count := range expected {
		if actual[key] != count {
			return fmt.Errorf("distribution mismatch")
		}
	}
	return rows.Err()
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, character := range value[len("sha256:"):] {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func readSecret(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read private AGA demo bootstrap secret: %w", err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", fmt.Errorf("private AGA demo bootstrap secret is empty")
	}
	return value, nil
}
