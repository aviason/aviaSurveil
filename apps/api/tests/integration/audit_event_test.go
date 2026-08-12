package integration_test

import (
	"context"
	"testing"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/potentialfindings"
)

func TestSuccessfulTransitionAuditEventRecordsExactAuthorityAndRevision(t *testing.T) {
	pool := canonicalDatabase(t, "audit_event")
	service := testService(pool)
	_, err := service.ConvertPotentialFinding(context.Background(), principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector), application.ConvertPotentialFindingCommand{
		OperationID: "op-audit-001", CorrelationID: "corr-audit-001", PotentialFindingID: "potential-cabin-001",
		ExpectedRevision: 1, Severity: potentialfindings.SeverityLevel2Major,
	})
	if err != nil {
		t.Fatalf("convert Potential Finding: %v", err)
	}

	var actor, role, organization, action, entityType, entityID, before, after, operationID, correlationID string
	var entityVersion int64
	if err := pool.QueryRow(context.Background(), `
		SELECT actor_subject_id, actor_role, organization_id, action, entity_type, entity_id,
		       before_status, after_status, entity_version, operation_id, correlation_id
		FROM audit_events
	`).Scan(&actor, &role, &organization, &action, &entityType, &entityID, &before, &after, &entityVersion, &operationID, &correlationID); err != nil {
		t.Fatalf("read audit event: %v", err)
	}
	if actor != "lead-001" || role != "leadInspector" || organization != "airline-xyz" || action != "potential_finding.converted" || entityType != "potential_finding" || entityID != "potential-cabin-001" {
		t.Fatalf("audit authority fields = %q %q %q %q %q %q", actor, role, organization, action, entityType, entityID)
	}
	if before != "PENDING_LEAD_REVIEW" || after != "CONVERTED" || entityVersion != 2 || operationID != "op-audit-001" || correlationID != "corr-audit-001" {
		t.Fatalf("audit transition fields = %q %q %d %q %q", before, after, entityVersion, operationID, correlationID)
	}
}
