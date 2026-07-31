package datafeed

import (
	"encoding/json"
	"testing"
	"time"

	feedstore "github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed/store/postgres"
)

func TestReconstructPersistedEventRevalidatesItsImmutableCanonicalDigest(t *testing.T) {
	now := time.Date(2026, 8, 1, 8, 0, 1, 0, time.UTC)
	event, err := BuildEvent(EventInput{
		EventID: "10000000-0000-4000-8000-000000000001", EventType: "audit.planned", TenantID: "tenant-contract-fixture", OwningOrganizationID: "organization-fixture", ActorOrganizationID: "caa-fixture", CorrelationID: "20000000-0000-4000-8000-000000000001", AggregateType: "audit", AggregateID: "audit-ref-001", AggregateRevision: 1, EffectiveAt: now.Add(-time.Second), KnownAt: now, OccurredAt: now.Add(-time.Second), EmittedAt: now, VisibilityPurposeCode: "regulated_oversight", EntityRefs: map[string]any{"audit_id": "audit-ref-001"}, StateAfter: "audit_planned", Payload: map[string]any{"audit_program_ref": "program-ref-001", "audit_scope_code": "airport_operations", "planned_start_at": "2026-08-01T08:00:00Z"},
	})
	if err != nil {
		t.Fatal(err)
	}
	eventID, _ := toPGUUID(event["event_id"].(string))
	correlationID, _ := toPGUUID(event["correlation_id"].(string))
	entityRefs, _ := json.Marshal(event["entity_refs"])
	payload, _ := json.Marshal(event["payload"])
	actor := event["actor_organization_id"].(string)
	row := feedstore.GetEventForScopedPublicationRow{
		EventID: eventID, ContractID: event["contract_id"].(string), ContractVersion: event["contract_version"].(string), SchemaVersion: event["schema_version"].(string), EventType: event["event_type"].(string), EventVersion: 1, SourceModule: event["source_module"].(string), SourceSystem: event["source_system"].(string), TenantID: event["tenant_id"].(string), OwningOrganizationID: event["owning_organization_id"].(string), ActorOrganizationID: &actor, VisibilityPurposeCode: event["visibility_purpose_code"].(string), CanonicalEventSha256: CanonicalDigest(event), PayloadSha256: event["payload_sha256"].(string), EntityRefs: entityRefs, StateAfter: event["state_after"].(string), EffectiveAt: timestamp(now.Add(-time.Second)), KnownAt: timestamp(now), OccurredAt: timestamp(now.Add(-time.Second)), EmittedAt: timestamp(now), AggregateType: event["aggregate_type"].(string), AggregateID: event["aggregate_id"].(string), AggregateRevision: 1, CorrelationID: correlationID,
	}
	reconstructed, err := ReconstructPersistedEvent(row, payload)
	if err != nil {
		t.Fatalf("reconstruct event: %v", err)
	}
	if got, err := EventContentDigest(reconstructed); err != nil || got == "" {
		t.Fatalf("content digest = %q, err = %v", got, err)
	}
	row.CanonicalEventSha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := ReconstructPersistedEvent(row, payload); err == nil {
		t.Fatal("mutated canonical digest was accepted")
	}
	row.CanonicalEventSha256 = CanonicalDigest(event)
	row.PayloadSha256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	corrupted := make(map[string]any, len(event))
	for key, value := range event {
		corrupted[key] = value
	}
	corrupted["payload_sha256"] = row.PayloadSha256
	row.CanonicalEventSha256 = CanonicalDigest(corrupted)
	if _, err := ReconstructPersistedEvent(row, payload); err == nil {
		t.Fatal("mutated payload digest was accepted")
	}
}
