package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
	"github.com/aviason/aviaSurveil/internal/potentialfindings"
)

func TestLostAcknowledgementReplaysOneCanonicalMutationAndTransitionEnvelope(t *testing.T) {
	pool := canonicalDatabase(t, "idempotent")
	service := testService(pool)
	actor := principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector)
	command := application.ConvertPotentialFindingCommand{
		OperationID: "op-convert-001", CorrelationID: "request-convert-001",
		PotentialFindingID: "potential-cabin-001", ExpectedRevision: 1, Severity: potentialfindings.SeverityLevel2Major,
	}

	first, err := service.ConvertPotentialFinding(context.Background(), actor, command)
	if err != nil {
		t.Fatalf("first conversion: %v", err)
	}
	replayed, err := service.ConvertPotentialFinding(context.Background(), actor, command)
	if err != nil {
		t.Fatalf("replayed conversion: %v", err)
	}
	if first != replayed {
		t.Fatalf("replayed response changed: %+v != %+v", first, replayed)
	}
	if first.FindingReference != "OPS-2026-001" || first.FindingStatus != "WAITING_FOR_CAP" {
		t.Fatalf("canonical Finding = %+v", first)
	}

	for table, expected := range map[string]int{"findings": 1, "audit_events": 1, "authorized_sync_changes": 1, "outbox_messages": 1, "idempotency_responses": 1} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != expected {
			t.Errorf("%s count = %d, want %d", table, count, expected)
		}
	}

	changedPayload := command
	changedPayload.Severity = potentialfindings.SeverityLevel1Critical
	if _, err := service.ConvertPotentialFinding(context.Background(), actor, changedPayload); !errors.Is(err, idempotency.ErrOperationIDReuse) {
		t.Fatalf("changed operation payload error = %v", err)
	}

	unauthorized := principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector)
	unauthorizedCommand := command
	unauthorizedCommand.OperationID = "op-convert-forbidden"
	if _, err := service.ConvertPotentialFinding(context.Background(), unauthorized, unauthorizedCommand); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("unauthorized conversion error = %v", err)
	}
	var events int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM audit_events").Scan(&events); err != nil || events != 1 {
		t.Fatalf("unauthorized transition created audit event: count=%d err=%v", events, err)
	}
}
