package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/idempotency"
	fieldsync "github.com/MarlonJD/aviaSurveil360/apps/api/internal/sync"
)

func canonicalSyncGrant(t *testing.T, service *fieldsync.GrantService, operationID, deviceID string) fieldsync.OfflineGrant {
	t.Helper()
	grant, err := service.Issue(context.Background(), principal(
		"inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector,
	), fieldsync.CheckoutInput{
		OperationID: operationID, CorrelationID: operationID, PackageID: "package-cabin-001",
		ExpectedPackageVersion: 1, DeviceInstanceID: deviceID,
	})
	if err != nil {
		t.Fatalf("issue sync grant: %v", err)
	}
	return grant
}

func fieldOperation(t *testing.T, value any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode field operation: %v", err)
	}
	return encoded
}

func responseOperation(operationID, grantID, deviceID, answer string, baseRevision *int64) json.RawMessage {
	return fieldOperationRaw(map[string]any{
		"operationId": operationID, "protocolVersion": 1, "offlineGrantId": grantID,
		"packageId": "package-cabin-001", "packageVersion": 1,
		"entityId": "response-cabin-001", "commandType": "UPSERT_CHECKLIST_RESPONSE",
		"baseRevision": baseRevision, "deviceInstanceId": deviceID,
		"clientOccurredAt": "2099-01-01T00:00:00Z",
		"payload": map[string]any{
			"auditId": "audit-cabin-001", "questionId": "q-cabin-crew-training",
			"answer": answer, "comment": "Server validates this field operation.",
		},
	})
}

func fieldOperationRaw(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func TestFieldSyncPushIsTransactionalIdempotentAndServerAuthorized(t *testing.T) {
	pool := canonicalDatabase(t, "sync_push")
	ids := 0
	grantService := fieldsync.NewGrantService(pool, fieldsync.GrantDependencies{
		Clock:       func() time.Time { return canonicalNow },
		IDGenerator: func(string) string { return "grant-sync-001" },
	})
	grant := canonicalSyncGrant(t, grantService, "op-checkout-sync-001", "managed-device-sync-001")
	service := fieldsync.NewOperationService(pool, fieldsync.OperationDependencies{
		Clock: func() time.Time { return canonicalNow },
		IDGenerator: func(prefix string) string {
			ids++
			return fmt.Sprintf("%s-sync-%03d", prefix, ids)
		},
	})
	actor := principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector)
	base := int64(1)
	operation := responseOperation("op-sync-response-001", grant.ID, grant.DeviceInstanceID, "OBSERVATION", &base)

	first, err := service.Push(context.Background(), actor, operation)
	if err != nil {
		t.Fatalf("push field operation: %v", err)
	}
	if first.Status != fieldsync.PushAccepted || first.OperationID != "op-sync-response-001" ||
		first.AuthoritativeEntityID == nil || *first.AuthoritativeEntityID != "response-cabin-001" ||
		first.AuthoritativeRevision == nil || *first.AuthoritativeRevision != 2 {
		t.Fatalf("push result = %+v", first)
	}
	restartedService := fieldsync.NewOperationService(pool, fieldsync.OperationDependencies{
		Clock: func() time.Time { return canonicalNow },
		IDGenerator: func(string) string {
			t.Fatal("exact replay after service restart must not allocate a new entity ID")
			return ""
		},
	})
	replayed, err := restartedService.Push(context.Background(), actor, operation)
	if err != nil {
		t.Fatalf("lost-ack replay: %v", err)
	}
	if !reflect.DeepEqual(replayed, first) {
		t.Fatalf("exact replay = %+v, want %+v", replayed, first)
	}
	var sameSemanticPayload map[string]any
	if err := json.Unmarshal(operation, &sameSemanticPayload); err != nil {
		t.Fatalf("decode time-variant replay: %v", err)
	}
	sameSemanticPayload["clientOccurredAt"] = "2099-01-02T00:00:00Z"
	timeVariantReplay, err := restartedService.Push(context.Background(), actor, fieldOperation(t, sameSemanticPayload))
	if err != nil || !reflect.DeepEqual(timeVariantReplay, first) {
		t.Fatalf("server-ignored client time replay = %+v, err = %v, want %+v", timeVariantReplay, err, first)
	}

	var answer string
	var revision int64
	if err := pool.QueryRow(context.Background(), `SELECT response_value, revision FROM checklist_responses WHERE id = 'response-cabin-001'`).Scan(&answer, &revision); err != nil {
		t.Fatalf("read authoritative response: %v", err)
	}
	if answer != "OBSERVATION" || revision != 2 {
		t.Fatalf("authoritative response = %s@%d", answer, revision)
	}
	for table, expected := range map[string]int{
		"checklist_responses":     1,
		"audit_events":            2,
		"authorized_sync_changes": 2,
		"outbox_messages":         2,
		"idempotency_responses":   2,
	} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != expected {
			t.Fatalf("%s count = %d, want %d, err = %v", table, count, expected, err)
		}
	}

	changed := responseOperation("op-sync-response-001", grant.ID, grant.DeviceInstanceID, "COMPLIANT", &base)
	if _, err := service.Push(context.Background(), actor, changed); !errors.Is(err, idempotency.ErrOperationIDReuse) {
		t.Fatalf("same ID with changed semantic payload error = %v", err)
	}

	stale := int64(1)
	conflict, err := service.Push(context.Background(), actor,
		responseOperation("op-sync-response-stale", grant.ID, grant.DeviceInstanceID, "COMPLIANT", &stale))
	if err != nil {
		t.Fatalf("stale operation transport error: %v", err)
	}
	if conflict.Status != fieldsync.PushConflict || conflict.Conflict == nil ||
		conflict.Conflict.Code != fieldsync.ConflictStaleRevision ||
		conflict.Conflict.AuthoritativeRevision == nil || *conflict.Conflict.AuthoritativeRevision != 2 {
		t.Fatalf("typed stale conflict = %+v", conflict)
	}

	wrongDevice := responseOperation("op-sync-wrong-device", grant.ID, "client-device-override", "COMPLIANT", &revision)
	forbidden, err := service.Push(context.Background(), actor, wrongDevice)
	if err != nil || forbidden.Status != fieldsync.PushForbidden || forbidden.ErrorCode != fieldsync.ErrorGrantScope {
		t.Fatalf("device tamper result = %+v, err = %v", forbidden, err)
	}

	tampered := map[string]any{}
	if err := json.Unmarshal(operation, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered["actorSubjectId"] = "inspector-other"
	invalid, err := service.Push(context.Background(), actor, fieldOperation(t, tampered))
	if err != nil || invalid.Status != fieldsync.PushInvalid {
		t.Fatalf("actor-field injection result = %+v, err = %v", invalid, err)
	}
}

func TestFieldSyncPushRejectsStaleAuthorityAndInvalidDomainInput(t *testing.T) {
	tests := []struct {
		name             string
		mutation         string
		mutationArgument any
		answer           string
		expectedStatus   fieldsync.PushStatus
		expectedCode     string
		expectedConflict string
	}{
		{
			name: "expired grant", mutation: `UPDATE offline_grants SET expires_at = $1 WHERE id = 'grant-sync-authority'`,
			mutationArgument: canonicalNow.Add(-24 * time.Hour), answer: "COMPLIANT",
			expectedStatus: fieldsync.PushForbidden, expectedCode: fieldsync.ErrorGrantExpired,
		},
		{
			name: "revoked grant", mutation: `UPDATE offline_grants SET revoked_at = $1 WHERE id = 'grant-sync-authority'`,
			mutationArgument: canonicalNow, answer: "COMPLIANT",
			expectedStatus: fieldsync.PushForbidden, expectedCode: fieldsync.ErrorGrantRevoked,
		},
		{
			name: "changed assignment", mutation: `UPDATE inspections SET revision = revision + 1 WHERE id = 'audit-cabin-001'`,
			answer: "COMPLIANT", expectedStatus: fieldsync.PushConflict,
			expectedConflict: fieldsync.ConflictAssignmentChanged,
		},
		{
			name: "withdrawn package", mutation: `UPDATE inspection_packages SET revoked_at = $1 WHERE id = 'package-cabin-001'`,
			mutationArgument: canonicalNow, answer: "COMPLIANT", expectedStatus: fieldsync.PushConflict,
			expectedConflict: fieldsync.ConflictPackageRevoked,
		},
		{
			name: "server validation", answer: "UNSUPPORTED_ANSWER", expectedStatus: fieldsync.PushInvalid,
			expectedCode: fieldsync.ErrorValidationFailed,
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := canonicalDatabase(t, fmt.Sprintf("sync_authority_%d", index))
			grantService := fieldsync.NewGrantService(pool, fieldsync.GrantDependencies{
				Clock:       func() time.Time { return canonicalNow },
				IDGenerator: func(string) string { return "grant-sync-authority" },
			})
			grant := canonicalSyncGrant(t, grantService, "op-checkout-sync-authority", "managed-device-authority")
			if test.mutation != "" {
				var err error
				if test.mutationArgument == nil {
					_, err = pool.Exec(context.Background(), test.mutation)
				} else {
					_, err = pool.Exec(context.Background(), test.mutation, test.mutationArgument)
				}
				if err != nil {
					t.Fatalf("mutate server authority: %v", err)
				}
			}
			service := fieldsync.NewOperationService(pool, fieldsync.OperationDependencies{
				Clock: func() time.Time { return canonicalNow },
			})
			baseRevision := int64(1)
			result, err := service.Push(
				context.Background(),
				principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector),
				responseOperation("op-sync-authority-rejection", grant.ID, grant.DeviceInstanceID, test.answer, &baseRevision),
			)
			if err != nil {
				t.Fatalf("push rejected authority operation: %v", err)
			}
			if result.Status != test.expectedStatus || result.ErrorCode != test.expectedCode {
				t.Fatalf("authority result = %+v, want status %s code %q", result, test.expectedStatus, test.expectedCode)
			}
			if test.expectedConflict != "" && (result.Conflict == nil || result.Conflict.Code != test.expectedConflict) {
				t.Fatalf("authority conflict = %+v, want %s", result.Conflict, test.expectedConflict)
			}
		})
	}
}

func TestFieldSyncCausalPotentialFindingAttachmentAndChecklistSubmission(t *testing.T) {
	pool := canonicalDatabase(t, "sync_causal")
	if _, err := pool.Exec(context.Background(), "DELETE FROM potential_findings"); err != nil {
		t.Fatalf("remove canonical Potential Finding fixture: %v", err)
	}
	sequence := 0
	nextID := func(prefix string) string {
		sequence++
		return fmt.Sprintf("%s-causal-%03d", prefix, sequence)
	}
	grantService := fieldsync.NewGrantService(pool, fieldsync.GrantDependencies{
		Clock: func() time.Time { return canonicalNow }, IDGenerator: func(string) string { return "grant-sync-causal" },
	})
	grant := canonicalSyncGrant(t, grantService, "op-checkout-sync-causal", "managed-device-causal")
	service := fieldsync.NewOperationService(pool, fieldsync.OperationDependencies{
		Clock: func() time.Time { return canonicalNow }, IDGenerator: nextID,
	})
	actor := principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector)
	base := int64(1)
	responseResult, err := service.Push(context.Background(), actor,
		responseOperation("op-causal-response", grant.ID, grant.DeviceInstanceID, "NON_COMPLIANT", &base))
	if err != nil || responseResult.Status != fieldsync.PushAccepted {
		t.Fatalf("response prerequisite = %+v, err = %v", responseResult, err)
	}
	pfOperation := fieldOperation(t, map[string]any{
		"operationId": "op-causal-pf", "protocolVersion": 1, "offlineGrantId": grant.ID,
		"packageId": grant.PackageID, "packageVersion": 1, "entityId": "pf-local-causal",
		"commandType": "CREATE_POTENTIAL_FINDING", "baseRevision": nil,
		"deviceInstanceId": grant.DeviceInstanceID, "clientOccurredAt": canonicalNow.Add(-time.Hour).Format(time.RFC3339),
		"payload": map[string]any{
			"auditId": "audit-cabin-001", "questionId": "q-cabin-crew-training",
			"checklistResponseId": "response-cabin-001", "expectedChecklistResponseRevision": 2,
			"title": "Training record gap", "description": "Required crew record was unavailable.",
			"requiredComment": "Provide the current training record.",
		},
	})
	pf, err := service.Push(context.Background(), actor, pfOperation)
	if err != nil || pf.Status != fieldsync.PushAccepted || pf.AuthoritativeEntityID == nil || *pf.AuthoritativeEntityID == "pf-local-causal" {
		t.Fatalf("Potential Finding acknowledgement = %+v, err = %v", pf, err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE potential_findings
		SET status = 'RETURNED', revision = revision + 1, updated_at = $2
		WHERE id = $1
	`, *pf.AuthoritativeEntityID, canonicalNow); err != nil {
		t.Fatalf("mark Potential Finding returned: %v", err)
	}
	successorOperationID := "op-causal-pf-successor"
	successor := fieldOperation(t, map[string]any{
		"operationId": successorOperationID, "protocolVersion": 1, "offlineGrantId": grant.ID,
		"packageId": grant.PackageID, "packageVersion": 1, "entityId": "pf-local-causal-successor",
		"commandType": "CREATE_POTENTIAL_FINDING", "baseRevision": nil,
		"deviceInstanceId": grant.DeviceInstanceID, "clientOccurredAt": canonicalNow.Format(time.RFC3339),
		"payload": map[string]any{
			"auditId": "audit-cabin-001", "questionId": "q-cabin-crew-training",
			"checklistResponseId": "response-cabin-001", "expectedChecklistResponseRevision": 2,
			"title": "Corrected training record gap", "description": "The returned finding was corrected.",
			"requiredComment": "Provide the corrected current training record.",
		},
	})
	successorResult, err := service.Push(context.Background(), actor, successor)
	if err != nil || successorResult.Status != fieldsync.PushAccepted || successorResult.AuthoritativeEntityID == nil || *successorResult.AuthoritativeEntityID == *pf.AuthoritativeEntityID {
		t.Fatalf("returned Potential Finding successor = %+v, err = %v", successorResult, err)
	}
	var supersedes string
	if err := pool.QueryRow(context.Background(), `SELECT supersedes_potential_finding_id FROM potential_findings WHERE id = $1`, *successorResult.AuthoritativeEntityID).Scan(&supersedes); err != nil {
		t.Fatalf("read Potential Finding successor lineage: %v", err)
	}
	if supersedes != *pf.AuthoritativeEntityID {
		t.Fatalf("successor supersedes = %q, want %q", supersedes, *pf.AuthoritativeEntityID)
	}
	attachment := fieldOperation(t, map[string]any{
		"operationId": "op-causal-attachment", "protocolVersion": 1, "offlineGrantId": grant.ID,
		"packageId": grant.PackageID, "packageVersion": 1, "entityId": "attachment-local-causal",
		"commandType": "REGISTER_INSPECTION_ATTACHMENT", "baseRevision": nil,
		"deviceInstanceId": grant.DeviceInstanceID, "clientOccurredAt": canonicalNow.Format(time.RFC3339),
		"payload": map[string]any{
			"auditId": "audit-cabin-001", "checklistResponseId": "response-cabin-001",
			"potentialFindingOperationId": successorOperationID, "fileName": "crew-record.pdf",
			"mediaType": "application/pdf", "byteSize": 4, "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	})
	registered, err := service.Push(context.Background(), actor, attachment)
	if err != nil || registered.Status != fieldsync.PushAccepted || registered.AuthoritativeEntityID == nil {
		t.Fatalf("attachment registration = %+v, err = %v", registered, err)
	}
	var linkedPotential string
	if err := pool.QueryRow(context.Background(), `SELECT potential_finding_id FROM inspection_attachments WHERE id = $1`, *registered.AuthoritativeEntityID).Scan(&linkedPotential); err != nil {
		t.Fatalf("read attachment causal link: %v", err)
	}
	if linkedPotential != *successorResult.AuthoritativeEntityID {
		t.Fatalf("attachment Potential Finding = %q, want %q", linkedPotential, *successorResult.AuthoritativeEntityID)
	}

	pendingAttachmentSubmission, err := service.Push(context.Background(), actor, fieldOperation(t, map[string]any{
		"operationId": "op-causal-submit", "protocolVersion": 1, "offlineGrantId": grant.ID,
		"packageId": grant.PackageID, "packageVersion": 1, "entityId": "audit-cabin-001",
		"commandType": "SUBMIT_CHECKLIST", "baseRevision": 1,
		"deviceInstanceId": grant.DeviceInstanceID, "clientOccurredAt": canonicalNow.Format(time.RFC3339),
		"payload": map[string]any{"auditId": "audit-cabin-001"},
	}))
	if err != nil || pendingAttachmentSubmission.Status != fieldsync.PushInvalid {
		t.Fatalf("pending attachment checklist submission = %+v, err = %v", pendingAttachmentSubmission, err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO object_metadata (
			id, aggregate_type, aggregate_id, object_key, filename,
			declared_media_type, sha256, size_bytes, scan_status
		) VALUES ('metadata-causal', 'inspection_attachment', $1, 'canonical/metadata-causal',
			'crew-record.pdf', 'application/pdf',
			'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 4, 'CLEAN')
	`, *registered.AuthoritativeEntityID); err != nil {
		t.Fatalf("seed clean canonical object metadata: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE inspection_attachments
		SET upload_state = 'UPLOADED', scan_state = 'CLEAN',
			object_metadata_id = 'metadata-causal', canonical_object_metadata_id = 'metadata-causal',
			revision = revision + 1, updated_at = $2
		WHERE id = $1
	`, *registered.AuthoritativeEntityID, canonicalNow); err != nil {
		t.Fatalf("mark attachment clean: %v", err)
	}
	submitted, err := service.Push(context.Background(), actor, fieldOperation(t, map[string]any{
		"operationId": "op-causal-submit-clean", "protocolVersion": 1, "offlineGrantId": grant.ID,
		"packageId": grant.PackageID, "packageVersion": 1, "entityId": "audit-cabin-001",
		"commandType": "SUBMIT_CHECKLIST", "baseRevision": 1,
		"deviceInstanceId": grant.DeviceInstanceID, "clientOccurredAt": canonicalNow.Format(time.RFC3339),
		"payload": map[string]any{"auditId": "audit-cabin-001"},
	}))
	if err != nil || submitted.Status != fieldsync.PushAccepted || submitted.AuthoritativeRevision == nil || *submitted.AuthoritativeRevision != 2 {
		t.Fatalf("clean attachment checklist submission = %+v, err = %v", submitted, err)
	}
}

func TestFieldSyncPullUsesOpaqueScopedReplayableCursorAndSafeChanges(t *testing.T) {
	pool := canonicalDatabase(t, "sync_pull")
	grantService := fieldsync.NewGrantService(pool, fieldsync.GrantDependencies{
		Clock:       func() time.Time { return canonicalNow },
		IDGenerator: func(prefix string) string { return prefix + "-pull-001" },
	})
	grant := canonicalSyncGrant(t, grantService, "op-checkout-sync-pull", "managed-device-pull")
	service := fieldsync.NewOperationService(pool, fieldsync.OperationDependencies{
		Clock:       func() time.Time { return canonicalNow },
		IDGenerator: func(prefix string) string { return prefix + "-pull-result" },
	})
	actor := principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector)
	base := int64(1)
	if _, err := service.Push(context.Background(), actor,
		responseOperation("op-pull-response", grant.ID, grant.DeviceInstanceID, "OBSERVATION", &base)); err != nil {
		t.Fatalf("seed pull response: %v", err)
	}

	input := fieldsync.PullInput{PackageID: grant.PackageID, OfflineGrantID: grant.ID, DeviceInstanceID: grant.DeviceInstanceID, Limit: 1}
	first, err := service.Pull(context.Background(), actor, input)
	if err != nil {
		t.Fatalf("first pull: %v", err)
	}
	if len(first.Changes) != 1 || first.NextCursor == nil || !strings.HasPrefix(*first.NextCursor, "sync_") {
		t.Fatalf("first pull page = %+v", first)
	}
	replayed, err := service.Pull(context.Background(), actor, input)
	if err != nil || !reflect.DeepEqual(replayed, first) {
		t.Fatalf("cursor page replay = %+v, err = %v", replayed, err)
	}
	serialized, _ := json.Marshal(first.Changes)
	for _, forbidden := range []string{"internalCaaNote", "internal_caa_note", "actorRole", "organizationRisk"} {
		if strings.Contains(string(serialized), forbidden) {
			t.Fatalf("pull exposed forbidden key %q: %s", forbidden, serialized)
		}
	}

	otherGrantService := fieldsync.NewGrantService(pool, fieldsync.GrantDependencies{
		Clock: func() time.Time { return canonicalNow }, IDGenerator: func(string) string { return "grant-pull-other-device" },
	})
	otherGrant := canonicalSyncGrant(t, otherGrantService, "op-checkout-pull-other", "managed-device-pull-other")
	if _, err := service.Pull(context.Background(), actor, fieldsync.PullInput{
		PackageID: otherGrant.PackageID, OfflineGrantID: otherGrant.ID, DeviceInstanceID: otherGrant.DeviceInstanceID,
		Cursor: first.NextCursor, Limit: 1,
	}); !errors.Is(err, fieldsync.ErrCursorScope) {
		t.Fatalf("cursor scope mismatch error = %v", err)
	}

	resumed := fieldsync.NewOperationService(pool, fieldsync.OperationDependencies{
		Clock: func() time.Time { return canonicalNow }, IDGenerator: func(prefix string) string { return prefix + "-restart" },
	})
	afterRestart, err := resumed.Pull(context.Background(), actor, fieldsync.PullInput{
		PackageID: grant.PackageID, OfflineGrantID: grant.ID, DeviceInstanceID: grant.DeviceInstanceID,
		Cursor: first.NextCursor, Limit: 50,
	})
	if err != nil {
		t.Fatalf("pull after server restart: %v", err)
	}
	if afterRestart.ProjectionVersion != 1 {
		t.Fatalf("projection version = %d", afterRestart.ProjectionVersion)
	}

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO authorized_sync_changes (subject_id, organization_id, package_id, kind, entity_id, entity_revision, payload, changed_at)
		VALUES
		('inspector-cabin-001', 'airline-xyz', 'package-cabin-001', 'tombstone', 'response-deleted', 4,
		 '{"kind":"tombstone","entityType":"checklist_response","entityId":"response-deleted","revision":4}', $1),
		('inspector-cabin-001', 'airline-xyz', 'package-cabin-001', 'package_revoked', 'package-cabin-001', NULL,
		 '{"kind":"package_revoked","packageId":"package-cabin-001","reasonCode":"WITHDRAWN","revokedAt":"2026-07-21T12:00:00Z"}', $1)
	`, canonicalNow); err != nil {
		t.Fatalf("seed tombstone/revocation: %v", err)
	}
	page, err := resumed.Pull(context.Background(), actor, fieldsync.PullInput{
		PackageID: grant.PackageID, OfflineGrantID: grant.ID, DeviceInstanceID: grant.DeviceInstanceID,
		Cursor: afterRestart.NextCursor, Limit: 50,
	})
	if err != nil {
		t.Fatalf("pull typed terminal changes: %v", err)
	}
	joined, _ := json.Marshal(page.Changes)
	if !strings.Contains(string(joined), `"kind":"tombstone"`) || !strings.Contains(string(joined), `"kind":"package_revoked"`) {
		t.Fatalf("typed terminal changes = %s", joined)
	}

	if afterRestart.NextCursor != nil {
		var highWater int64
		if err := pool.QueryRow(context.Background(), `SELECT high_water_mark FROM sync_cursor_tokens WHERE token = $1`, *afterRestart.NextCursor).Scan(&highWater); err != nil {
			t.Fatalf("read cursor high-water mark: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM authorized_sync_changes WHERE subject_id = 'inspector-cabin-001' AND package_id = 'package-cabin-001' AND sequence_id <= $1`, highWater); err != nil {
			t.Fatalf("expire sync history: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO authorized_sync_changes (subject_id, organization_id, package_id, kind, entity_id, entity_revision, payload, changed_at)
			VALUES ('inspector-cabin-001', 'airline-xyz', 'package-cabin-001', 'tombstone', 'history-gap', 5,
			'{"kind":"tombstone","entityType":"potential_finding","entityId":"history-gap","revision":5}', $1)
		`, canonicalNow); err != nil {
			t.Fatalf("append post-expiry change: %v", err)
		}
		expired, err := resumed.Pull(context.Background(), actor, fieldsync.PullInput{
			PackageID: grant.PackageID, OfflineGrantID: grant.ID, DeviceInstanceID: grant.DeviceInstanceID,
			Cursor: afterRestart.NextCursor, Limit: 50,
		})
		if err != nil || !expired.ResnapshotRequired || len(expired.Changes) != 0 {
			t.Fatalf("history expiry page = %+v, err = %v", expired, err)
		}
	}
}
