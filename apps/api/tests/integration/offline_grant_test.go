package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
	fieldsync "github.com/aviason/aviaSurveil/internal/sync"
)

func TestOfflineGrantUsesServerAuthorityForExpiryAssignmentSessionAndDevice(t *testing.T) {
	pool := canonicalDatabase(t, "offline_grant")
	service := fieldsync.NewGrantService(pool, fieldsync.GrantDependencies{
		Clock:       func() time.Time { return canonicalNow },
		IDGenerator: func(string) string { return "grant-cabin-001" },
	})
	inspector := principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector)
	grant, err := service.Issue(context.Background(), inspector, fieldsync.CheckoutInput{
		OperationID: "op-checkout-001", CorrelationID: "corr-checkout-001", PackageID: "package-cabin-001", ExpectedPackageVersion: 1,
		DeviceInstanceID: "managed-device-001", ClaimedSubjectID: "client-cannot-override-subject",
	})
	if err != nil {
		t.Fatalf("issue grant: %v", err)
	}
	if grant.SubjectID != inspector.SubjectID || grant.DeviceInstanceID != "managed-device-001" || !grant.ExpiresAt.Equal(canonicalNow.Add(24*time.Hour)) {
		t.Fatalf("server-issued grant = %+v", grant)
	}
	var storedSubject, storedSession, storedDevice, storedPackage string
	var storedCommands []string
	if err := pool.QueryRow(context.Background(), `
		SELECT subject_id, session_id, device_id, package_id, allowed_command_types
		FROM offline_grants WHERE id = $1
	`, grant.ID).Scan(&storedSubject, &storedSession, &storedDevice, &storedPackage, &storedCommands); err != nil {
		t.Fatalf("read stored grant authority: %v", err)
	}
	if storedSubject != inspector.SubjectID || storedSession != inspector.SessionID || storedDevice != grant.DeviceInstanceID || storedPackage != grant.PackageID {
		t.Fatalf("stored grant scope = subject:%q session:%q device:%q package:%q", storedSubject, storedSession, storedDevice, storedPackage)
	}
	if len(storedCommands) == 0 || storedCommands[0] != "UPSERT_CHECKLIST_RESPONSE" {
		t.Fatalf("stored grant commands = %v", storedCommands)
	}
	replayed, err := service.Issue(context.Background(), inspector, fieldsync.CheckoutInput{
		OperationID: "op-checkout-001", CorrelationID: "different-transport-correlation", PackageID: "package-cabin-001",
		ExpectedPackageVersion: 1, DeviceInstanceID: "managed-device-001", ClaimedSubjectID: "another-client-claim",
	})
	if err != nil || replayed.ID != grant.ID || !replayed.ExpiresAt.Equal(grant.ExpiresAt) {
		t.Fatalf("lost-ack grant replay = %+v, err = %v", replayed, err)
	}
	for table, expected := range map[string]int{
		"offline_grants": 1, "audit_events": 1, "authorized_sync_changes": 1,
		"outbox_messages": 1, "idempotency_responses": 1,
	} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != expected {
			t.Fatalf("%s count = %d, want %d, err = %v", table, count, expected, err)
		}
	}
	changedPayload := fieldsync.CheckoutInput{
		OperationID: "op-checkout-001", CorrelationID: "corr-checkout-001", PackageID: "package-cabin-001",
		ExpectedPackageVersion: 1, DeviceInstanceID: "managed-device-002",
	}
	if _, err := service.Issue(context.Background(), inspector, changedPayload); !errors.Is(err, idempotency.ErrOperationIDReuse) {
		t.Fatalf("changed checkout operation error = %v", err)
	}
	if err := service.Authorize(context.Background(), inspector, fieldsync.AuthorizationInput{
		GrantID: grant.ID, PackageID: grant.PackageID, DeviceInstanceID: grant.DeviceInstanceID,
		ServerNow: grant.ExpiresAt.Add(4 * time.Minute),
	}); err != nil {
		t.Fatalf("stored subject/session/device/package scope rejected before command check: %v", err)
	}
	if err := service.Authorize(context.Background(), inspector, fieldsync.AuthorizationInput{
		GrantID: grant.ID, PackageID: grant.PackageID, DeviceInstanceID: grant.DeviceInstanceID,
		ServerNow: grant.ExpiresAt.Add(4 * time.Minute), CommandType: "UPSERT_CHECKLIST_RESPONSE",
	}); err != nil {
		t.Fatalf("grant inside clock-skew tolerance rejected: %v", err)
	}
	if err := service.Authorize(context.Background(), inspector, fieldsync.AuthorizationInput{
		GrantID: grant.ID, PackageID: grant.PackageID, DeviceInstanceID: grant.DeviceInstanceID,
		ServerNow: grant.ExpiresAt.Add(6 * time.Minute), CommandType: "UPSERT_CHECKLIST_RESPONSE",
	}); !errors.Is(err, fieldsync.ErrGrantExpired) {
		t.Fatalf("late grant error = %v", err)
	}
	otherUser := principal("inspector-other", "caa", "session-inspector", identity.RoleInspector)
	if err := service.Authorize(context.Background(), otherUser, fieldsync.AuthorizationInput{GrantID: grant.ID, PackageID: grant.PackageID, DeviceInstanceID: grant.DeviceInstanceID, ServerNow: canonicalNow}); !errors.Is(err, fieldsync.ErrGrantScope) {
		t.Fatalf("user-switch grant error = %v", err)
	}

	if _, err := pool.Exec(context.Background(), "UPDATE inspections SET assigned_inspector_subject_id = 'inspector-other', revision = revision + 1 WHERE id = 'audit-cabin-001'"); err != nil {
		t.Fatalf("change assignment: %v", err)
	}
	if err := service.Authorize(context.Background(), inspector, fieldsync.AuthorizationInput{GrantID: grant.ID, PackageID: grant.PackageID, DeviceInstanceID: grant.DeviceInstanceID, ServerNow: canonicalNow}); !errors.Is(err, fieldsync.ErrAssignmentChanged) {
		t.Fatalf("assignment-change grant error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE inspections SET assigned_inspector_subject_id = 'inspector-cabin-001', revision = 1 WHERE id = 'audit-cabin-001'"); err != nil {
		t.Fatalf("restore assignment: %v", err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE inspection_packages SET revoked_at = $1 WHERE id = 'package-cabin-001'", canonicalNow); err != nil {
		t.Fatalf("withdraw package: %v", err)
	}
	if err := service.Authorize(context.Background(), inspector, fieldsync.AuthorizationInput{GrantID: grant.ID, PackageID: grant.PackageID, DeviceInstanceID: grant.DeviceInstanceID, ServerNow: canonicalNow}); !errors.Is(err, fieldsync.ErrPackageRevoked) {
		t.Fatalf("package-withdrawal grant error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE inspection_packages SET revoked_at = NULL WHERE id = 'package-cabin-001'"); err != nil {
		t.Fatalf("restore package: %v", err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE session_references SET revoked_at = $1 WHERE id = 'session-inspector'", canonicalNow); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if err := service.Authorize(context.Background(), inspector, fieldsync.AuthorizationInput{GrantID: grant.ID, PackageID: grant.PackageID, DeviceInstanceID: grant.DeviceInstanceID, ServerNow: canonicalNow}); !errors.Is(err, fieldsync.ErrSessionRevoked) {
		t.Fatalf("logout/session-revoke grant error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE session_references SET revoked_at = NULL WHERE id = 'session-inspector'"); err != nil {
		t.Fatalf("restore session: %v", err)
	}
	if err := service.Revoke(context.Background(), inspector, grant.ID, "DEVICE_LOST"); err != nil {
		t.Fatalf("device-loss revoke: %v", err)
	}
	if err := service.Authorize(context.Background(), inspector, fieldsync.AuthorizationInput{GrantID: grant.ID, PackageID: grant.PackageID, DeviceInstanceID: grant.DeviceInstanceID, ServerNow: canonicalNow}); !errors.Is(err, fieldsync.ErrGrantRevoked) {
		t.Fatalf("device-loss grant error = %v", err)
	}
}
