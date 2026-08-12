package scenarios

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"sort"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/preproddata"
	"github.com/aviason/aviaSurveil/internal/preproddata/profiles"
	"github.com/jackc/pgx/v5"
)

type PostgresStore struct {
	pool    *database.Pool
	profile profiles.Profile
	runID   string
}

func NewPostgresStore(
	pool *database.Pool,
	profile profiles.Profile,
	runID string,
) (*PostgresStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("PostgreSQL pool is required")
	}
	if err := profiles.ValidateFrozen(profile); err != nil {
		return nil, err
	}
	runID = strings.TrimSpace(runID)
	if runID == "" || len(runID) > 128 {
		return nil, fmt.Errorf("bounded run ID is required")
	}
	return &PostgresStore{pool: pool, profile: profile, runID: runID}, nil
}

func (store *PostgresStore) Initialize(ctx context.Context) error {
	if _, err := store.pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS preprod_loader;
		CREATE TABLE IF NOT EXISTS preprod_loader.scenario_records (
			run_id text NOT NULL,
			family text NOT NULL,
			operation_id text NOT NULL,
			record_id text NOT NULL,
			business_key text NOT NULL,
			revision bigint NOT NULL CHECK (revision > 0),
			predecessor_id text,
			distribution text NOT NULL,
			effective_at timestamptz NOT NULL,
			known_at timestamptz NOT NULL CHECK (known_at >= effective_at),
			actor_membership_id text NOT NULL,
			organization_id text NOT NULL,
			decision_reason text NOT NULL CHECK (btrim(decision_reason) <> ''),
			relationship_tuple text[] NOT NULL
				CHECK (cardinality(relationship_tuple) > 0),
			attributes jsonb NOT NULL,
			PRIMARY KEY (run_id, family, record_id)
		);
		CREATE INDEX IF NOT EXISTS scenario_records_family_idx
			ON preprod_loader.scenario_records (run_id, family, record_id);
		CREATE TABLE IF NOT EXISTS preprod_loader.applied_operations (
			run_id text NOT NULL,
			operation_id text NOT NULL,
			payload_digest text NOT NULL,
			applied_at timestamptz NOT NULL,
			PRIMARY KEY (run_id, operation_id)
		);
	`); err != nil {
		return fmt.Errorf("initialize scenario reconciliation schema: %w", err)
	}
	var retained int64
	if err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM preprod_loader.scenario_records
	`).Scan(&retained); err != nil {
		return err
	}
	if retained != 0 {
		return fmt.Errorf(
			"connected-scenario target retains %d prior records",
			retained,
		)
	}
	var organizationCount int64
	var exactCAA bool
	if err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*),
		       COALESCE(bool_and(
		           id = 'CAA'
		           AND organization_type = 'AUTHORITY'
		           AND status = 'ACTIVE'
		           AND tombstoned_at IS NULL
		       ), false)
		FROM organizations
	`).Scan(&organizationCount, &exactCAA); err != nil {
		return err
	}
	if organizationCount != 1 || !exactCAA {
		return fmt.Errorf(
			"connected-scenario target is not the clean post-migration baseline",
		)
	}
	for _, table := range []string{
		"identity_references",
		"desired_membership_versions",
		"session_references",
		"surveillance_plan_items",
		"inspections",
		"findings",
		"audit_events",
		"outbox_messages",
	} {
		var count int64
		if err := store.pool.QueryRow(
			ctx,
			"SELECT COUNT(*) FROM "+table,
		).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf(
				"connected-scenario target table %s is not empty",
				table,
			)
		}
	}
	return nil
}

func (store *PostgresStore) Resume(ctx context.Context) error {
	var recordCount, otherRunCount, unknownFamilyCount, excessFamilyCount int64
	if err := store.pool.QueryRow(ctx, `
		WITH family_counts AS (
			SELECT family, COUNT(*) AS actual_count
			FROM preprod_loader.scenario_records
			WHERE run_id = $1
			GROUP BY family
		),
		expected(family, expected_count) AS (
			SELECT key, value::bigint
			FROM jsonb_each_text($2::jsonb)
		)
		SELECT
			(SELECT COUNT(*)
			 FROM preprod_loader.scenario_records
			 WHERE run_id = $1),
			(SELECT COUNT(*)
			 FROM preprod_loader.scenario_records
			 WHERE run_id <> $1),
			(SELECT COUNT(*)
			 FROM family_counts
			 LEFT JOIN expected USING (family)
			 WHERE expected.family IS NULL),
			(SELECT COUNT(*)
			 FROM family_counts
			 JOIN expected USING (family)
			 WHERE family_counts.actual_count > expected.expected_count)
	`, store.runID, expectedCountsJSON(store.profile.ExpectedCounts)).Scan(
		&recordCount,
		&otherRunCount,
		&unknownFamilyCount,
		&excessFamilyCount,
	); err != nil {
		return fmt.Errorf("inspect resumable scenario rows: %w", err)
	}
	if recordCount == 0 ||
		otherRunCount != 0 ||
		unknownFamilyCount != 0 ||
		excessFamilyCount != 0 {
		return fmt.Errorf(
			"connected-scenario target is not an exact resumable run",
		)
	}
	var orphanRecords, otherRunOperations int64
	if err := store.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)
			 FROM preprod_loader.scenario_records records
			 LEFT JOIN preprod_loader.applied_operations operations
			   ON operations.run_id = records.run_id
			  AND operations.operation_id = records.operation_id
			 WHERE records.run_id = $1
			   AND operations.operation_id IS NULL),
			(SELECT COUNT(*)
			 FROM preprod_loader.applied_operations
			 WHERE run_id <> $1)
	`, store.runID).Scan(
		&orphanRecords,
		&otherRunOperations,
	); err != nil {
		return fmt.Errorf("inspect resumable scenario operations: %w", err)
	}
	if orphanRecords != 0 || otherRunOperations != 0 {
		return fmt.Errorf(
			"connected-scenario operations are not bound to the resumable run",
		)
	}
	return nil
}

func expectedCountsJSON(counts map[string]int64) string {
	encoded, _ := json.Marshal(counts)
	return string(encoded)
}

func (store *PostgresStore) Apply(
	ctx context.Context,
	command preproddata.AuthoritativeCommand,
) error {
	batch, err := DecodeBatch(command)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(command.Payload)
	payloadDigest := "sha256:" + hex.EncodeToString(digest[:])
	return database.WithinTransaction(
		ctx,
		store.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			if _, err := transaction.Exec(
				ctx,
				"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
				store.runID+":"+command.OperationID,
			); err != nil {
				return err
			}
			var existingDigest string
			err := transaction.QueryRow(ctx, `
				SELECT payload_digest
				FROM preprod_loader.applied_operations
				WHERE run_id = $1 AND operation_id = $2
			`, store.runID, command.OperationID).Scan(&existingDigest)
			if err == nil {
				if existingDigest != payloadDigest {
					return fmt.Errorf(
						"scenario operation ID was reused with different content",
					)
				}
				return nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			for _, record := range batch.Records {
				if err := store.materialize(
					ctx,
					transaction,
					record,
				); err != nil {
					return fmt.Errorf(
						"materialize %s/%s: %w",
						record.Family,
						record.RecordID,
						err,
					)
				}
				attributes, err := json.Marshal(record.Attributes)
				if err != nil {
					return err
				}
				if _, err := transaction.Exec(ctx, `
					INSERT INTO preprod_loader.scenario_records (
						run_id, family, operation_id, record_id,
						business_key, revision, predecessor_id,
						distribution, effective_at, known_at,
						actor_membership_id, organization_id,
						decision_reason, relationship_tuple, attributes
					) VALUES (
						$1, $2, $3, $4, $5, $6, NULLIF($7, ''),
						$8, $9, $10, $11, $12, $13, $14, $15
					)
				`, store.runID, record.Family, command.OperationID,
					record.RecordID, record.BusinessKey, record.Revision,
					record.PredecessorID, record.Distribution,
					record.EffectiveAt, record.KnownAt,
					record.ActorMembershipID, record.OrganizationID,
					record.DecisionReason, record.RelationshipTuple,
					attributes); err != nil {
					return err
				}
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO preprod_loader.applied_operations (
					run_id, operation_id, payload_digest, applied_at
				) VALUES ($1, $2, $3, $4)
			`, store.runID, command.OperationID, payloadDigest,
				time.Now().UTC()); err != nil {
				return err
			}
			return nil
		},
	)
}

func (store *PostgresStore) Reconcile(
	ctx context.Context,
) (preproddata.Reconciliation, error) {
	output := preproddata.Reconciliation{
		ActualCounts: make(map[string]int64, len(store.profile.ExpectedCounts)),
		RelationshipDigests: make(
			map[string]string,
			len(store.profile.ExpectedCounts),
		),
	}
	for family := range store.profile.ExpectedCounts {
		output.ActualCounts[family] = 0
		output.RelationshipDigests[family] =
			newRelationshipDigestAccumulator().Digest()
	}
	rows, err := store.pool.Query(ctx, `
		SELECT family, relationship_tuple
		FROM preprod_loader.scenario_records
		WHERE run_id = $1
		ORDER BY
			family COLLATE "C",
			array_to_string(relationship_tuple, chr(31)) COLLATE "C",
			record_id COLLATE "C"
	`, store.runID)
	if err != nil {
		return preproddata.Reconciliation{}, err
	}
	defer rows.Close()
	var currentFamily string
	var accumulator *relationshipDigestAccumulator
	flush := func() {
		if currentFamily != "" {
			output.RelationshipDigests[currentFamily] =
				accumulator.Digest()
		}
	}
	for rows.Next() {
		var family string
		var tuple []string
		if err := rows.Scan(&family, &tuple); err != nil {
			return preproddata.Reconciliation{}, err
		}
		if _, ok := output.ActualCounts[family]; !ok {
			return preproddata.Reconciliation{}, fmt.Errorf(
				"unexpected scenario family %s",
				family,
			)
		}
		if family != currentFamily {
			flush()
			currentFamily = family
			accumulator = newRelationshipDigestAccumulator()
		}
		output.ActualCounts[family]++
		accumulator.Add(tuple)
	}
	if err := rows.Err(); err != nil {
		return preproddata.Reconciliation{}, err
	}
	flush()
	var objectCount, objectBytes int64
	if err := store.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(size_bytes), 0)
		FROM object_metadata
		WHERE bucket_name = 'aviasurveil360-local-preprod'
		  AND object_key LIKE $1
	`, "runs/"+store.runID+"/objects/%").Scan(
		&objectCount,
		&objectBytes,
	); err != nil {
		return preproddata.Reconciliation{}, err
	}
	if objectCount != store.profile.ExpectedCounts["objectVersions"] ||
		objectBytes < 1 ||
		objectBytes > store.profile.ResourceEnvelope.ObjectBytes ||
		(store.profile.Name == "stress" &&
			objectBytes != store.profile.ResourceEnvelope.ObjectBytes) {
		return preproddata.Reconciliation{}, fmt.Errorf(
			"object payload count or byte envelope differs from frozen profile",
		)
	}
	return output, nil
}

func (store *PostgresStore) Records(
	ctx context.Context,
	family string,
) ([]Record, error) {
	expected, ok := store.profile.ExpectedCounts[family]
	if !ok {
		return nil, fmt.Errorf("unknown scenario family %s", family)
	}
	records := make([]Record, 0, expected)
	err := store.ScanRecords(ctx, family, func(record Record) error {
		records = append(records, record)
		return nil
	})
	return records, err
}

func (store *PostgresStore) ScanRecords(
	ctx context.Context,
	family string,
	yield func(Record) error,
) error {
	expected, ok := store.profile.ExpectedCounts[family]
	if !ok {
		return fmt.Errorf("unknown scenario family %s", family)
	}
	if yield == nil {
		return fmt.Errorf("scenario record scanner callback is required")
	}
	rows, err := store.pool.Query(ctx, `
		SELECT family, record_id, business_key, revision,
		       predecessor_id, distribution, effective_at, known_at,
		       actor_membership_id, organization_id, decision_reason,
		       relationship_tuple, attributes
		FROM preprod_loader.scenario_records
		WHERE run_id = $1 AND family = $2
		ORDER BY record_id
	`, store.runID, family)
	if err != nil {
		return err
	}
	defer rows.Close()
	var count int64
	for rows.Next() {
		var record Record
		var predecessorID *string
		var attributes []byte
		if err := rows.Scan(
			&record.Family,
			&record.RecordID,
			&record.BusinessKey,
			&record.Revision,
			&predecessorID,
			&record.Distribution,
			&record.EffectiveAt,
			&record.KnownAt,
			&record.ActorMembershipID,
			&record.OrganizationID,
			&record.DecisionReason,
			&record.RelationshipTuple,
			&attributes,
		); err != nil {
			return err
		}
		if predecessorID != nil {
			record.PredecessorID = *predecessorID
		}
		if err := json.Unmarshal(attributes, &record.Attributes); err != nil {
			return fmt.Errorf(
				"decode durable scenario attributes for %s/%s: %w",
				record.Family,
				record.RecordID,
				err,
			)
		}
		count++
		if count > expected {
			return fmt.Errorf(
				"durable scenario family %s exceeds profile bound",
				family,
			)
		}
		if err := yield(record); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func relationshipDigest(tuples [][]string) string {
	canonical := make([]string, len(tuples))
	for index, tuple := range tuples {
		canonical[index] = strings.Join(tuple, "\x1f")
	}
	sort.Strings(canonical)
	digest := sha256.Sum256([]byte(strings.Join(canonical, "\n")))
	return "sha256:" + hex.EncodeToString(digest[:])
}

type relationshipDigestAccumulator struct {
	digest   hash.Hash
	hasValue bool
}

func newRelationshipDigestAccumulator() *relationshipDigestAccumulator {
	return &relationshipDigestAccumulator{digest: sha256.New()}
}

func (accumulator *relationshipDigestAccumulator) Add(tuple []string) {
	if accumulator.hasValue {
		_, _ = accumulator.digest.Write([]byte{'\n'})
	}
	_, _ = accumulator.digest.Write(
		[]byte(strings.Join(tuple, "\x1f")),
	)
	accumulator.hasValue = true
}

func (accumulator *relationshipDigestAccumulator) Digest() string {
	return "sha256:" + hex.EncodeToString(
		accumulator.digest.Sum(nil),
	)
}

func (store *PostgresStore) materialize(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	switch record.Family {
	case "organizations":
		if record.RecordID == "CAA" {
			var exact bool
			return transaction.QueryRow(ctx, `
				SELECT organization_type = 'AUTHORITY'
				       AND status = 'ACTIVE'
				       AND tombstoned_at IS NULL
				FROM organizations WHERE id = 'CAA'
			`).Scan(&exact)
		}
		_, err := transaction.Exec(ctx, `
			INSERT INTO organizations (
				id, legal_name, organization_type, status, revision,
				created_at, updated_at
			) VALUES ($1, $2, 'OPERATOR', 'ACTIVE', 1, $3, $3)
		`, record.RecordID, attrString(record, "legalName"),
			record.KnownAt)
		return err
	case "providerAccounts":
		_, err := transaction.Exec(ctx, `
			INSERT INTO identity_references (
				subject_id, issuer, display_name, email, revision,
				created_at
			) VALUES (
				$1,
				'https://preprod-keycloak:8080/identity/realms/aviasurveil360-local-preprod',
				$2, $3, 1, $4
			)
		`, attrString(record, "providerSubjectId"),
			"SYNTHETIC "+strings.ToUpper(attrString(record, "role"))+
				" USER",
			attrString(record, "email"), record.KnownAt)
		return err
	case "desiredMembershipVersions":
		return store.materializeMembership(ctx, transaction, record)
	case "applicationProfiles":
		subjectID, err := subjectForMembership(
			ctx,
			transaction,
			attrString(record, "membershipId"),
		)
		if err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO user_profiles (
				subject_id, display_name, organization_id, revision,
				created_at, updated_at
			) VALUES ($1, $2, $3, 1, $4, $4)
		`, subjectID, attrString(record, "displayName"),
			record.OrganizationID, record.KnownAt); err != nil {
			return err
		}
		_, err = transaction.Exec(ctx, `
			INSERT INTO user_settings (
				subject_id, notification_preferences, locale, timezone,
				revision, updated_at
			) VALUES ($1, '{}'::jsonb, 'en', 'UTC', 1, $2)
		`, subjectID, record.KnownAt)
		return err
	case "invitations":
		return store.materializeIdentityFact(
			ctx,
			transaction,
			record,
			"INVITATION",
			invitationFactState(record.Distribution),
		)
	case "recoveryRequests":
		state := "RESET_PENDING"
		if record.RecordID[len(record.RecordID)-1]%2 == 0 {
			state = "RESET_COMPLETED"
		}
		return store.materializeIdentityFact(
			ctx,
			transaction,
			record,
			"RECOVERY",
			state,
		)
	case "mfaEnrollments", "planningApprovals",
		"evidenceReferences", "routeDispositions",
		"visibleActionDispositions", "identityLifecycleCases",
		"lifecycleScenarioCases":
		return nil
	case "sessions":
		return store.materializeSession(ctx, transaction, record)
	case "surveillancePlans":
		statuses := []struct {
			status string
			owner  string
			next   string
		}{
			{"FINANCE_REVIEW", "finance", "Finance budget review"},
			{"GM_REVIEW", "gm", "General Manager review"},
			{"EXECUTIVE_DIRECTOR_REVIEW", "executiveDirector", "Executive approval"},
			{"GM_RELEASE", "gm", "General Manager release"},
		}
		state := statuses[indexFromRecord(record)%len(statuses)]
		_, err := transaction.Exec(ctx, `
			INSERT INTO surveillance_plan_items (
				id, title, plan_year, organization_id, inspection_type,
				scheduled_date, estimated_budget, status,
				current_owner_role, next_action, revision,
				created_at, updated_at
			) VALUES (
				$1, $2, 2026, $3, 'ROUTINE',
				$4, 1000, $5, $6, $7, $8, $9, $9
			)
		`, record.RecordID, "SYNTHETIC SURVEILLANCE PLAN",
			record.OrganizationID, record.EffectiveAt.Format("2006-01-02"),
			state.status, state.owner, state.next, record.Revision,
			record.KnownAt)
		return err
	case "audits":
		subjectID, err := store.actorSubject(ctx, transaction, record)
		if err != nil {
			return err
		}
		_, err = transaction.Exec(ctx, `
			INSERT INTO inspections (
				id, organization_id, assigned_inspector_subject_id,
				title, inspection_type, status, due_date, revision,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, 'SYNTHETIC CONNECTED AUDIT', 'ROUTINE',
				$4, $5, $6, $7, $7
			)
		`, record.RecordID, record.OrganizationID, subjectID,
			strings.ToUpper(record.Distribution),
			record.EffectiveAt.Add(30*24*time.Hour).Format("2006-01-02"),
			record.Revision, record.KnownAt)
		return err
	case "assignments":
		return store.materializeAssignment(ctx, transaction, record)
	case "checklistTemplates":
		_, err := transaction.Exec(ctx, `
			INSERT INTO template_masters (
				id, title, owner_role, revision, created_at, updated_at
			) VALUES ($1, 'SYNTHETIC CHECKLIST TEMPLATE',
				'Admin Preview', 1, $2, $2)
		`, record.RecordID, record.KnownAt)
		return err
	case "checklistTemplateVersions":
		snapshot, _ := json.Marshal(record.Attributes)
		_, err := transaction.Exec(ctx, `
			INSERT INTO checklist_template_versions (
				id, template_id, version, title, snapshot, published_at
			) VALUES ($1, $2, $3,
				'SYNTHETIC CHECKLIST TEMPLATE VERSION', $4, $5)
		`, record.RecordID, attrString(record, "templateId"),
			record.Revision, snapshot, record.KnownAt)
		return err
	case "checklistQuestions":
		return store.materializeQuestion(ctx, transaction, record)
	case "inspectionPackages":
		snapshot, _ := json.Marshal(record.Attributes)
		digest := digestText(string(snapshot))
		_, err := transaction.Exec(ctx, `
			INSERT INTO inspection_packages (
				id, inspection_id, checklist_template_version_id,
				package_version, snapshot, package_digest, created_at
			) VALUES ($1, $2, $3, 1, $4, $5, $6)
		`, record.RecordID, attrString(record, "auditId"),
			attrString(record, "templateVersionId"), snapshot, digest,
			record.KnownAt)
		return err
	case "checklistResponses":
		return store.materializeResponse(ctx, transaction, record)
	case "potentialFindings":
		return store.materializePotentialFinding(
			ctx,
			transaction,
			record,
		)
	case "findings":
		return store.materializeFinding(ctx, transaction, record)
	case "capRevisions":
		return store.materializeCAP(ctx, transaction, record)
	case "objects":
		return nil
	case "objectVersions":
		_, err := transaction.Exec(ctx, `
			INSERT INTO object_metadata (
				id, aggregate_type, aggregate_id, object_key, filename,
				declared_media_type, detected_media_type, sha256,
				size_bytes, scan_status, organization_id, bucket_name,
				object_state, created_at
			) VALUES (
				$1, 'EVIDENCE', $2, $3, $4,
				'application/json', 'application/json', $5,
				$6, 'PENDING', $7, 'aviasurveil360-local-preprod',
				'QUARANTINED', $8
			)
		`, record.RecordID, attrString(record, "objectId"),
			"runs/"+store.runID+"/objects/"+record.RecordID+".json",
			record.RecordID+".json", attrString(record, "contentDigest"),
			attrInt64(record, "sizeBytes"), record.OrganizationID,
			record.KnownAt)
		return err
	case "evidenceVersions":
		return store.materializeEvidence(ctx, transaction, record)
	case "reviewDecisions":
		subjectID, err := store.actorSubject(ctx, transaction, record)
		if err != nil {
			return err
		}
		_, err = transaction.Exec(ctx, `
			INSERT INTO review_decisions (
				id, entity_type, entity_id, expected_revision,
				decision, reason, comment_to_auditee,
				internal_caa_note, decided_by_subject_id, decided_at
			) VALUES (
				$1, 'finding', $2, 1, $3, $4,
				'SYNTHETIC COMMENT TO AUDITEE',
				'SYNTHETIC INTERNAL CAA NOTE', $5, $6
			)
		`, record.RecordID, attrString(record, "recordId"),
			strings.ToUpper(attrString(record, "decision")),
			record.DecisionReason, subjectID, record.KnownAt)
		return err
	case "reportVersions":
		return store.materializeReport(ctx, transaction, record)
	case "communications":
		return store.materializeCommunication(ctx, transaction, record)
	case "notifications":
		return store.materializeNotification(ctx, transaction, record)
	case "outboxMessages":
		payload, _ := json.Marshal(record.Attributes)
		_, err := transaction.Exec(ctx, `
			INSERT INTO outbox_messages (
				id, topic, aggregate_type, aggregate_id, payload,
				available_at, idempotency_key, operation_id,
				correlation_id, created_at
			) VALUES (
				$1, 'preprod.connected', $2, $3, $4,
				$5, $6, $1, $1, $5
			)
		`, record.RecordID, attrString(record, "aggregateType"),
			attrString(record, "aggregateId"), payload, record.KnownAt,
			"preprod:"+store.runID+":"+record.RecordID)
		return err
	case "deliveryJobs":
		return store.materializeDeliveryJob(ctx, transaction, record)
	case "scannerJobs":
		return store.materializeScannerJob(ctx, transaction, record)
	case "renderJobs":
		return store.materializeRenderJob(ctx, transaction, record)
	case "calendarRecords":
		return store.materializeCalendar(ctx, transaction, record)
	case "offlineGrants":
		return store.materializeOfflineGrant(ctx, transaction, record)
	case "auditEvents":
		subjectID, err := store.actorSubject(ctx, transaction, record)
		if err != nil {
			return err
		}
		details, _ := json.Marshal(record.Attributes)
		_, err = transaction.Exec(ctx, `
			INSERT INTO audit_events (
				event_id, occurred_at, actor_subject_id, actor_role,
				organization_id, action, entity_type, entity_id,
				entity_version, reason, operation_id, correlation_id,
				request_id, details
			) VALUES (
				$1, $2, $3, 'admin', $4, 'PREPROD_CONNECTED_EVENT',
				$5, $6, 1, $7, $1, $1, $1, $8
			)
		`, record.RecordID, record.KnownAt, subjectID,
			record.OrganizationID, attrString(record, "entityType"),
			attrString(record, "entityId"), record.DecisionReason, details)
		return err
	case "syncChanges":
		subjectID, err := store.actorSubject(ctx, transaction, record)
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(record.Attributes)
		_, err = transaction.Exec(ctx, `
			INSERT INTO authorized_sync_changes (
				subject_id, organization_id, kind, entity_id,
				entity_revision, payload, changed_at, operation_id,
				correlation_id
			) VALUES ($1, $2, $3, $4, 1, $5, $6, $7, $7)
		`, subjectID, record.OrganizationID,
			attrString(record, "entityType"),
			attrString(record, "entityId"), payload, record.KnownAt,
			record.RecordID)
		return err
	default:
		return fmt.Errorf("unsupported materialization family %s", record.Family)
	}
}

func (store *PostgresStore) materializeMembership(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	membershipID := attrString(record, "membershipId")
	subjectID := attrString(record, "subjectId")
	roles := attrStrings(record, "roles")
	requestID := record.RecordID + "-request"
	if _, err := transaction.Exec(ctx, `
		INSERT INTO user_lifecycle_requests (
			id, subject_id, requested_action, requested_roles,
			requested_organization_id, status, idempotency_key,
			requested_by_subject_id, expected_membership_revision,
			reason, requested_effective_at, membership_id,
			resulting_membership_revision, provider_acknowledged_at,
			created_at, updated_at
		) VALUES (
			$1, $2, 'UPDATE_ROLES', $3, $4, 'SUCCEEDED', $5,
			$2, $6, $7, $8, $9, $10, $11, $11, $11
		)
	`, requestID, subjectID, roles, record.OrganizationID,
		"preprod:"+store.runID+":"+requestID, record.Revision-1,
		record.DecisionReason, record.EffectiveAt, membershipID,
		record.Revision, record.KnownAt); err != nil {
		return err
	}
	state := strings.ToUpper(strings.ReplaceAll(
		record.Distribution,
		"-",
		"_",
	))
	enabled := state != "SUSPENDED" && state != "DEACTIVATED"
	if _, err := transaction.Exec(ctx, `
		INSERT INTO desired_membership_versions (
			membership_id, subject_id, revision, membership_state,
			organization_id, roles, requested_by_subject_id, reason,
			source_request_id, requested_at, effective_at,
			observed_provider_enabled, observed_organization_id,
			observed_roles, observed_at, drift_state
		) VALUES (
			$1, $2, $3, $4, $5, $6, $2, $7, $8, $9, $10,
			$11, $5, $6, $12, 'IN_SYNC'
		)
	`, membershipID, subjectID, record.Revision, state,
		record.OrganizationID, roles, record.DecisionReason, requestID,
		record.EffectiveAt, record.EffectiveAt, enabled,
		record.KnownAt); err != nil {
		return err
	}
	_, err := transaction.Exec(ctx, `
		INSERT INTO desired_membership_sync (
			membership_id, subject_id, desired_revision,
			observed_provider_enabled, observed_organization_id,
			observed_roles, observed_at, drift_state
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'IN_SYNC')
		ON CONFLICT (membership_id) DO UPDATE SET
			desired_revision = EXCLUDED.desired_revision,
			observed_provider_enabled = EXCLUDED.observed_provider_enabled,
			observed_organization_id = EXCLUDED.observed_organization_id,
			observed_roles = EXCLUDED.observed_roles,
			observed_at = EXCLUDED.observed_at,
			drift_state = EXCLUDED.drift_state
	`, membershipID, subjectID, record.Revision, enabled,
		record.OrganizationID, roles, record.KnownAt)
	return err
}

func (store *PostgresStore) materializeIdentityFact(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
	kind, state string,
) error {
	membershipID := attrString(record, "membershipId")
	subjectID, err := subjectForMembership(ctx, transaction, membershipID)
	if err != nil {
		return err
	}
	var revision int64
	var organizationID string
	var roles []string
	if err := transaction.QueryRow(ctx, `
		SELECT revision, organization_id, roles
		FROM desired_membership_versions
		WHERE membership_id = $1
		ORDER BY revision DESC LIMIT 1
	`, membershipID).Scan(&revision, &organizationID, &roles); err != nil {
		return err
	}
	requestID := record.RecordID + "-request"
	action := "RESEND_INVITATION"
	if kind == "RECOVERY" {
		action = "RESET_PASSWORD"
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO user_lifecycle_requests (
			id, subject_id, requested_action, requested_roles,
			requested_organization_id, status, idempotency_key,
			requested_by_subject_id, expected_membership_revision,
			reason, membership_id, resulting_membership_revision,
			provider_acknowledged_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, 'SUCCEEDED', $6, $2, $7,
			$8, $9, $7, $10, $10, $10
		)
	`, requestID, subjectID, action, roles, organizationID,
		"preprod:"+store.runID+":"+requestID, revision,
		record.DecisionReason, membershipID, record.KnownAt); err != nil {
		return err
	}
	expiresAt := any(nil)
	if kind == "INVITATION" {
		expiresAt = record.EffectiveAt.Add(24 * time.Hour)
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO identity_action_facts (
			id, request_id, fact_sequence, membership_id, subject_id,
			action_kind, state, delivery_attempt, expires_at,
			provider_acknowledged_at, reason, created_at
		) VALUES ($1, $2, 1, $3, $4, $5, $6, 1, $7, $8, $9, $8)
	`, record.RecordID, requestID, membershipID, subjectID, kind,
		state, expiresAt, record.KnownAt, record.DecisionReason)
	return err
}

func (store *PostgresStore) materializeSession(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	membershipID := attrString(record, "membershipId")
	subjectID, err := subjectForMembership(ctx, transaction, membershipID)
	if err != nil {
		return err
	}
	var roles []string
	if err := transaction.QueryRow(ctx, `
		SELECT roles FROM desired_membership_versions
		WHERE membership_id = $1 AND revision = 1
	`, membershipID).Scan(&roles); err != nil {
		return err
	}
	state := attrString(record, "state")
	var revokedAt any
	authorityState := "ACTIVE"
	expiresAt := record.EffectiveAt.Add(8 * time.Hour)
	switch state {
	case "revoked":
		revokedAt = record.KnownAt
		authorityState = "REVOCATION_PENDING"
	case "expired":
		revokedAt = record.KnownAt
		expiresAt = record.EffectiveAt
		authorityState = "REVOCATION_PENDING"
	case "denied-stale-authority":
		revokedAt = record.KnownAt
		authorityState = "DENIED_STALE_AUTHORITY"
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO session_references (
			id, subject_id, organization_id, provider_session_id,
			expires_at, revoked_at, created_at, session_token_hash,
			csrf_token_hash, last_seen_at, absolute_expires_at, roles,
			membership_id, membership_revision, authority_observed_at,
			authority_state
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $7, $5, $10,
			$11, 1, $7, $12
		)
	`, record.RecordID, subjectID, record.OrganizationID,
		"provider-"+record.RecordID, expiresAt, revokedAt,
		record.KnownAt, digestText("session:"+record.RecordID),
		digestText("csrf:"+record.RecordID), roles, membershipID,
		authorityState)
	return err
}

func (store *PostgresStore) materializeAssignment(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	auditID := attrString(record, "auditId")
	membershipID := attrString(record, "membershipId")
	subjectID, err := subjectForMembership(ctx, transaction, membershipID)
	if err != nil {
		return err
	}
	assignmentID := "assignment-for-" + auditID
	if _, err := transaction.Exec(ctx, `
		INSERT INTO audit_assignments (
			id, inspection_id, organization_id, lead_subject_id,
			status, revision, created_at, updated_at
		)
		SELECT $1, inspection.id, inspection.organization_id, $2,
		       'ASSIGNED', 1, $3, $3
		FROM inspections inspection WHERE inspection.id = $4
		ON CONFLICT (inspection_id) DO NOTHING
	`, assignmentID, subjectID, record.KnownAt, auditID); err != nil {
		return err
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO audit_team_members (
			assignment_id, subject_id, member_role, revision, created_at
		) VALUES ($1, $2, 'INSPECTOR', 1, $3)
		ON CONFLICT (assignment_id, subject_id) DO NOTHING
	`, assignmentID, subjectID, record.KnownAt); err != nil {
		return err
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO audit_question_assignments (
			assignment_id, question_id, subject_id, revision, created_at
		) VALUES ($1, $2, $3, 1, $4)
	`, assignmentID, attrString(record, "questionId"), subjectID,
		record.KnownAt)
	return err
}

func (store *PostgresStore) materializeQuestion(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	subjectID, err := store.actorSubject(ctx, transaction, record)
	if err != nil {
		return err
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO question_versions (
			id, question_id, version, prompt, configured_reference,
			expected_evidence, created_by_subject_id, created_at
		) VALUES (
			$1, $1, 1, 'SYNTHETIC CHECKLIST QUESTION',
			'SYNTHETIC REGULATORY REFERENCE',
			'SYNTHETIC EXPECTED EVIDENCE', $2, $3
		)
	`, record.RecordID, subjectID, record.KnownAt); err != nil {
		return err
	}
	templateVersionID := attrString(record, "templateVersionId")
	var existing int
	if err := transaction.QueryRow(ctx, `
		SELECT COUNT(*) FROM template_version_questions
		WHERE template_version_id = $1
	`, templateVersionID).Scan(&existing); err != nil {
		return err
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO template_version_questions (
			template_version_id, question_version_id, position, created_at
		) VALUES ($1, $2, $3, $4)
	`, templateVersionID, record.RecordID, existing, record.KnownAt)
	return err
}

func (store *PostgresStore) materializeResponse(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	subjectID, err := subjectForMembership(
		ctx,
		transaction,
		attrString(record, "membershipId"),
	)
	if err != nil {
		return err
	}
	var packageID string
	if err := transaction.QueryRow(ctx, `
		SELECT id FROM inspection_packages
		WHERE inspection_id = $1 ORDER BY package_version DESC LIMIT 1
	`, attrString(record, "auditId")).Scan(&packageID); err != nil {
		return err
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO checklist_responses (
			id, inspection_id, package_id, question_id,
			assigned_inspector_subject_id, response_value,
			comment_to_auditee, internal_caa_note, revision, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, 'SYNTHETIC COMPLIANT RESPONSE',
			'SYNTHETIC COMMENT TO AUDITEE',
			'SYNTHETIC INTERNAL CAA NOTE', $6, $7
		)
	`, record.RecordID, attrString(record, "auditId"), packageID,
		attrString(record, "questionId"), subjectID, record.Revision,
		record.KnownAt)
	return err
}

func (store *PostgresStore) materializePotentialFinding(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	subjectID, err := store.actorSubject(ctx, transaction, record)
	if err != nil {
		return err
	}
	var organizationID string
	if err := transaction.QueryRow(ctx, `
		SELECT organization_id FROM inspections WHERE id = $1
	`, attrString(record, "auditId")).Scan(&organizationID); err != nil {
		return err
	}
	status := strings.ToUpper(strings.ReplaceAll(
		record.Distribution,
		"-",
		"_",
	))
	_, err = transaction.Exec(ctx, `
		INSERT INTO potential_findings (
			id, inspection_id, checklist_response_id, organization_id,
			status, finding_basis, expected_evidence, comment_to_auditee,
			internal_caa_note, revision, created_at, updated_at,
			question_id, title, description, created_by_subject_id
		) VALUES (
			$1, $2, $3, $4, $5, 'SYNTHETIC FINDING BASIS',
			'SYNTHETIC EXPECTED EVIDENCE',
			'SYNTHETIC COMMENT TO AUDITEE',
			'SYNTHETIC INTERNAL CAA NOTE', 1, $6, $6,
			$7, 'SYNTHETIC POTENTIAL FINDING',
			'SYNTHETIC POTENTIAL FINDING DESCRIPTION', $8
		)
	`, record.RecordID, attrString(record, "auditId"),
		attrString(record, "responseId"), organizationID, status,
		record.KnownAt, record.RecordID+"-question", subjectID)
	return err
}

func (store *PostgresStore) materializeFinding(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	status := strings.ToUpper(strings.ReplaceAll(
		record.Distribution,
		"-",
		"_",
	))
	closedAt := any(nil)
	closureBasis := any(nil)
	closureReason := any(nil)
	if strings.Contains(status, "CLOSED") {
		closedAt = record.KnownAt
		closureBasis = "SYNTHETIC_VERIFICATION"
		closureReason = record.DecisionReason
	}
	_, err := transaction.Exec(ctx, `
		INSERT INTO findings (
			id, reference, potential_finding_id, inspection_id,
			organization_id, severity, status, next_action, due_date,
			closure_basis, closure_reason, revision, created_at,
			updated_at, closed_at
		) VALUES (
			$1, $2, $3, $4, $5, 'MAJOR', $6,
			'SYNTHETIC NEXT ACTION', $7, $8, $9, 1, $10, $10, $11
		)
	`, record.RecordID, "SYN-"+record.RecordID,
		attrString(record, "potentialFindingId"),
		attrString(record, "auditId"), record.OrganizationID, status,
		record.EffectiveAt.Add(30*24*time.Hour).Format("2006-01-02"),
		closureBasis, closureReason, record.KnownAt, closedAt)
	return err
}

func (store *PostgresStore) materializeCAP(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	subjectID, err := store.actorSubject(ctx, transaction, record)
	if err != nil {
		return err
	}
	var organizationID string
	if err := transaction.QueryRow(ctx, `
		SELECT organization_id FROM findings WHERE id = $1
	`, attrString(record, "findingId")).Scan(&organizationID); err != nil {
		return err
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO cap_revisions (
			id, cap_id, finding_id, organization_id, revision, status,
			root_cause, corrective_action, preventive_action,
			target_completion_date, submitted_by_subject_id,
			submitted_at, responsible_person, comment_to_caa
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			'SYNTHETIC ROOT CAUSE', 'SYNTHETIC CORRECTIVE ACTION',
			'SYNTHETIC PREVENTIVE ACTION', $7, $8, $9,
			'SYNTHETIC RESPONSIBLE PERSON', 'SYNTHETIC COMMENT TO CAA'
		)
	`, record.RecordID, record.BusinessKey,
		attrString(record, "findingId"), organizationID, record.Revision,
		strings.ToUpper(strings.ReplaceAll(
			record.Distribution,
			"-",
			"_",
		)),
		record.EffectiveAt.Add(30*24*time.Hour).Format("2006-01-02"),
		subjectID, record.KnownAt)
	return err
}

func (store *PostgresStore) materializeEvidence(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	evidenceID := attrString(record, "evidenceId")
	var findingID string
	if err := transaction.QueryRow(ctx, `
		SELECT attributes->>'findingId'
		FROM preprod_loader.scenario_records
		WHERE run_id = $1 AND family = 'evidenceReferences'
		  AND record_id = $2
	`, store.runID, evidenceID).Scan(&findingID); err != nil {
		return err
	}
	var organizationID string
	if err := transaction.QueryRow(ctx, `
		SELECT organization_id FROM findings WHERE id = $1
	`, findingID).Scan(&organizationID); err != nil {
		return err
	}
	subjectID, err := store.actorSubject(ctx, transaction, record)
	if err != nil {
		return err
	}
	objectVersionID := attrString(record, "objectVersionId")
	var digest string
	var size int64
	if err := transaction.QueryRow(ctx, `
		SELECT sha256, size_bytes FROM object_metadata WHERE id = $1
	`, objectVersionID).Scan(&digest, &size); err != nil {
		return err
	}
	status := strings.ToUpper(strings.ReplaceAll(
		record.Distribution,
		"-",
		"_",
	))
	if _, err := transaction.Exec(ctx, `
		INSERT INTO evidence_versions (
			id, evidence_id, finding_id, organization_id, version,
			object_metadata_id, filename, media_type, sha256, size_bytes,
			status, submitted_by_subject_id, submitted_at, revision
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			'application/json', $8, $9, $10, $11, $12, $13
		)
	`, record.RecordID, evidenceID, findingID, organizationID,
		record.Revision, objectVersionID, record.RecordID+".json",
		digest, size, status, subjectID, record.KnownAt,
		record.Revision); err != nil {
		return err
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO evidence_version_states (
			evidence_version_id, upload_state, scan_state, review_state,
			canonical_object_metadata_id, revision, updated_at
		) VALUES ($1, 'COMPLETED', 'CLEAN', $2, $3, $4, $5)
	`, record.RecordID, status, objectVersionID, record.Revision,
		record.KnownAt)
	return err
}

func (store *PostgresStore) materializeReport(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	snapshot, _ := json.Marshal(map[string]any{
		"synthetic":     true,
		"predecessorId": record.PredecessorID,
		"effectiveAt":   record.EffectiveAt,
		"knownAt":       record.KnownAt,
		"reason":        record.DecisionReason,
	})
	status := strings.ToUpper(strings.ReplaceAll(
		record.Distribution,
		"-",
		"_",
	))
	if _, err := transaction.Exec(ctx, `
		INSERT INTO report_versions (
			id, report_id, inspection_id, version, status,
			snapshot, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, record.RecordID, record.BusinessKey,
		attrString(record, "auditId"), record.Revision, status, snapshot,
		record.KnownAt); err != nil {
		return err
	}
	_, err := transaction.Exec(ctx, `
		INSERT INTO report_approval_states (
			report_version_id, status, revision, issued_at, updated_at
		) VALUES ($1, $2, $3, $4, $5)
	`, record.RecordID, status, record.Revision,
		nullableTime(status == "ISSUED", record.KnownAt), record.KnownAt)
	return err
}

func (store *PostgresStore) materializeCommunication(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	subjectID, err := subjectForMembership(
		ctx,
		transaction,
		attrString(record, "senderMembershipId"),
	)
	if err != nil {
		return err
	}
	private := attrString(record, "visibility") == "caa-private"
	visibility := "AUDITEE_VISIBLE"
	audience := "AUDITEE"
	direction := "CAA_TO_AUDITEE"
	var organizationID any = record.OrganizationID
	if private {
		visibility = "INTERNAL_CAA"
		audience = "CAA"
		direction = "CAA_INTERNAL"
		organizationID = nil
	}
	threadID := record.RecordID + "-thread"
	if _, err := transaction.Exec(ctx, `
		INSERT INTO communication_threads (
			id, organization_id, visibility, subject, revision,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, 'SYNTHETIC CONNECTED COMMUNICATION',
			1, $4, $4
		)
	`, threadID, organizationID, visibility, record.KnownAt); err != nil {
		return err
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO communication_messages (
			id, thread_id, organization_id, visibility,
			sender_subject_id, audience, direction, subject, body,
			idempotency_key, revision, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			'SYNTHETIC CONNECTED COMMUNICATION',
			'SYNTHETIC COMMUNICATION BODY', $8, 1, $9
		)
	`, record.RecordID, threadID, organizationID, visibility,
		subjectID, audience, direction,
		"preprod:"+store.runID+":"+record.RecordID, record.KnownAt)
	return err
}

func (store *PostgresStore) materializeNotification(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	subjectID, err := subjectForMembership(
		ctx,
		transaction,
		attrString(record, "recipientMembershipId"),
	)
	if err != nil {
		return err
	}
	_, err = transaction.Exec(ctx, `
		INSERT INTO notification_records (
			id, recipient_subject_id, organization_id, title, body,
			related_entity_type, related_entity_id, deduplication_key,
			revision, created_at
		) VALUES (
			$1, $2, $3, 'SYNTHETIC CONNECTED NOTIFICATION',
			'SYNTHETIC NOTIFICATION BODY', 'AUDIT_EVENT', $4, $5, 1, $6
		)
	`, record.RecordID, subjectID, record.OrganizationID,
		attrString(record, "eventId"),
		"preprod:"+store.runID+":"+record.RecordID, record.KnownAt)
	return err
}

func (store *PostgresStore) materializeDeliveryJob(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	notificationID := attrString(record, "notificationId")
	var subjectID string
	if err := transaction.QueryRow(ctx, `
		SELECT recipient_subject_id FROM notification_records WHERE id = $1
	`, notificationID).Scan(&subjectID); err != nil {
		return err
	}
	state := attrString(record, "state")
	status := "FAILED"
	var providerMessageID, acceptedAt, nextAttemptAt, terminalAt any
	attempts := 1
	switch state {
	case "delivered":
		status = "DELIVERED"
		providerMessageID = "mailpit-" + record.RecordID
		acceptedAt = record.KnownAt
	case "delayed", "retrying":
		nextAttemptAt = record.KnownAt.Add(time.Minute)
	case "unavailable":
		status = "DEAD_LETTER"
		nextAttemptAt = nil
		terminalAt = record.KnownAt
		attempts = 3
	}
	_, err := transaction.Exec(ctx, `
		INSERT INTO notification_delivery_jobs (
			id, notification_id, recipient_subject_id, channel, status,
			idempotency_key, outbox_message_id, attempt_count, last_error,
			provider_message_id, accepted_at, next_attempt_at, terminal_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, 'EMAIL', $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $13
		)
	`, record.RecordID, notificationID, subjectID, status,
		"preprod:"+store.runID+":"+record.RecordID,
		attrString(record, "outboxId"), attempts,
		nullableString(status != "DELIVERED", "SYNTHETIC "+state),
		providerMessageID, acceptedAt, nextAttemptAt, terminalAt,
		record.KnownAt)
	return err
}

func (store *PostgresStore) materializeScannerJob(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	state := attrString(record, "processingState")
	scanStatus := map[string]string{
		"clean":       "CLEAN",
		"rejected":    "REJECTED",
		"expired":     "EXPIRED",
		"delayed":     "PENDING",
		"retrying":    "RETRYING",
		"unavailable": "UNAVAILABLE",
	}[state]
	_, err := transaction.Exec(ctx, `
		UPDATE object_metadata
		SET scan_status = $2,
		    scan_engine_version = 'synthetic-metadata-only',
		    scan_signature_version = 'synthetic-none',
		    scanned_at = $3,
		    object_state = CASE WHEN $2 = 'CLEAN'
		        THEN 'CANONICAL' ELSE object_state END
		WHERE id = $1
	`, attrString(record, "objectVersionId"), scanStatus,
		record.KnownAt)
	return err
}

func (store *PostgresStore) materializeRenderJob(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	reportVersionID := attrString(record, "reportVersionId")
	var organizationID string
	if err := transaction.QueryRow(ctx, `
		SELECT inspection.organization_id
		FROM report_versions report
		JOIN inspections inspection ON inspection.id = report.inspection_id
		WHERE report.id = $1
	`, reportVersionID).Scan(&organizationID); err != nil {
		return err
	}
	documentID := "document-" + reportVersionID
	if _, err := transaction.Exec(ctx, `
		INSERT INTO document_records (
			id, organization_id, kind, title, revision,
			created_at, updated_at
		) VALUES ($1, $2, 'REPORT', 'SYNTHETIC REPORT DOCUMENT',
			1, $3, $3)
		ON CONFLICT (id) DO NOTHING
	`, documentID, organizationID, record.KnownAt); err != nil {
		return err
	}
	state := strings.ToUpper(attrString(record, "state"))
	switch state {
	case "DELAYED":
		state = "PENDING"
	case "RETRYING", "UNAVAILABLE":
		state = "FAILED"
	}
	_, err := transaction.Exec(ctx, `
		INSERT INTO document_render_jobs (
			id, document_id, organization_id, requested_version,
			status, idempotency_key, input_snapshot, attempt_count,
			last_error, created_at, updated_at
		) VALUES (
			$1, $2, $3, 1, $4, $5,
			jsonb_build_object('reportVersionId', $6::text, 'synthetic', true),
			$7, $8, $9, $9
		)
	`, record.RecordID, documentID, organizationID, state,
		"preprod:"+store.runID+":"+record.RecordID, reportVersionID,
		boolInt(state == "FAILED"),
		nullableString(state == "FAILED", "SYNTHETIC RENDER STATE"),
		record.KnownAt)
	return err
}

func (store *PostgresStore) materializeCalendar(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	const ruleID = "preprod-connected-calendar-rule"
	if _, err := transaction.Exec(ctx, `
		INSERT INTO reminder_rules (
			id, label, offset_days, channel, status, revision,
			created_at, updated_at
		) VALUES (
			$1, 'SYNTHETIC CONNECTED CALENDAR', 7, 'IN_APP',
			'ACTIVE', 1, $2, $2
		)
		ON CONFLICT (id) DO NOTHING
	`, ruleID, record.KnownAt); err != nil {
		return err
	}
	notificationID := store.linkedScenarioRecord(
		ctx,
		transaction,
		"notifications",
		indexFromRecord(record),
	)
	var subjectID string
	if err := transaction.QueryRow(ctx, `
		SELECT recipient_subject_id
		FROM notification_records WHERE id = $1
	`, notificationID).Scan(&subjectID); err != nil {
		return err
	}
	_, err := transaction.Exec(ctx, `
		INSERT INTO reminder_dispatches (
			id, reminder_rule_id, entity_type, entity_id,
			recipient_subject_id, due_date, due_state,
			notification_id, dispatched_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, 'DUE', $7, $8
		)
	`, record.RecordID, ruleID, attrString(record, "recordType"),
		attrString(record, "recordId"), subjectID,
		record.EffectiveAt.Format("2006-01-02"), notificationID,
		record.KnownAt)
	return err
}

func (store *PostgresStore) materializeOfflineGrant(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) error {
	sessionID := attrString(record, "sessionId")
	var subjectID string
	if err := transaction.QueryRow(ctx, `
		SELECT subject_id FROM session_references WHERE id = $1
	`, sessionID).Scan(&subjectID); err != nil {
		return err
	}
	packageID := attrString(record, "packageId")
	var inspectionID, packageDigest string
	var packageVersion int
	if err := transaction.QueryRow(ctx, `
		SELECT inspection_id, package_version, package_digest
		FROM inspection_packages WHERE id = $1
	`, packageID).Scan(
		&inspectionID,
		&packageVersion,
		&packageDigest,
	); err != nil {
		return err
	}
	_, err := transaction.Exec(ctx, `
		INSERT INTO offline_grants (
			id, subject_id, device_id, package_id, inspection_id,
			assignment_revision, granted_at, expires_at, session_id,
			package_version, package_digest, allowed_command_types,
			assignment_scope, protocol_version, grant_token_hash
		) VALUES (
			$1, $2, $3, $4, $5, 1, $6, $7, $8, $9, $10,
			ARRAY['UPSERT_RESPONSE','CREATE_POTENTIAL_FINDING']::text[],
			'{"questionIds":[]}'::jsonb, 1, $11
		)
	`, record.RecordID, subjectID, "synthetic-device-"+record.RecordID,
		packageID, inspectionID, record.EffectiveAt,
		record.EffectiveAt.Add(8*time.Hour), sessionID, packageVersion,
		packageDigest, digestText("grant:"+record.RecordID))
	return err
}

func (store *PostgresStore) actorSubject(
	ctx context.Context,
	transaction pgx.Tx,
	record Record,
) (string, error) {
	return subjectForMembership(
		ctx,
		transaction,
		record.ActorMembershipID,
	)
}

func subjectForMembership(
	ctx context.Context,
	transaction pgx.Tx,
	membershipID string,
) (string, error) {
	var subjectID string
	err := transaction.QueryRow(ctx, `
		SELECT subject_id
		FROM desired_membership_versions
		WHERE membership_id = $1
		ORDER BY revision DESC LIMIT 1
	`, membershipID).Scan(&subjectID)
	return subjectID, err
}

func (store *PostgresStore) linkedScenarioRecord(
	ctx context.Context,
	transaction pgx.Tx,
	family string,
	index int,
) string {
	var recordID string
	err := transaction.QueryRow(ctx, `
		SELECT record_id
		FROM preprod_loader.scenario_records
		WHERE run_id = $1 AND family = $2
		ORDER BY record_id
		OFFSET $3 LIMIT 1
	`, store.runID, family,
		index%int(store.profile.ExpectedCounts[family])).Scan(&recordID)
	if err != nil {
		panic(err)
	}
	return recordID
}

func invitationFactState(distribution string) string {
	return map[string]string{
		"issued":            "ISSUED",
		"delivered":         "DELIVERY_ACCEPTED",
		"retryable-failure": "RETRYABLE_FAILURE",
		"expired":           "EXPIRED",
		"consumed":          "CONSUMED",
		"cancelled":         "CANCELLED",
	}[distribution]
}

func attrString(record Record, key string) string {
	value, _ := record.Attributes[key].(string)
	if strings.TrimSpace(value) == "" {
		panic(fmt.Sprintf(
			"scenario record %s/%s omits string attribute %s",
			record.Family,
			record.RecordID,
			key,
		))
	}
	return value
}

func attrStrings(record Record, key string) []string {
	switch values := record.Attributes[key].(type) {
	case []string:
		return append([]string(nil), values...)
	case []any:
		output := make([]string, len(values))
		for index, value := range values {
			output[index], _ = value.(string)
			if output[index] == "" {
				panic("scenario string-array attribute is malformed")
			}
		}
		return output
	default:
		panic("scenario string-array attribute is missing")
	}
}

func attrInt64(record Record, key string) int64 {
	switch value := record.Attributes[key].(type) {
	case int64:
		return value
	case float64:
		return int64(value)
	case json.Number:
		output, _ := value.Int64()
		return output
	default:
		panic("scenario integer attribute is missing")
	}
}

func digestText(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func nullableTime(include bool, value time.Time) any {
	if !include {
		return nil
	}
	return value
}

func nullableString(include bool, value string) any {
	if !include {
		return nil
	}
	return value
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func indexFromRecord(record Record) int {
	parts := strings.Split(record.RecordID, "-")
	if len(parts) == 0 {
		return 0
	}
	decoded, err := hex.DecodeString(parts[len(parts)-1])
	if err != nil || len(decoded) == 0 {
		return int(record.KnownAt.Unix())
	}
	var value int
	for _, item := range decoded {
		value = (value*256 + int(item)) & 0x7fffffff
	}
	return value
}
